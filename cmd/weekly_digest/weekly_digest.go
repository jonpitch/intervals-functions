package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	intervals "intervals-functions/api"
	"intervals-functions/utils/ai"
	"log"
	"os"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	toon "github.com/bug4fix/totoon/go"
	"github.com/joho/godotenv"
)

//go:embed weekly_digest_prompt.md
var weeklyDigestPrompt string

func main() {
	// update this based on netlify or not
	err := godotenv.Load("../../.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	intervalsApiKey := os.Getenv("INTERVALS_API_KEY")
	intervalsAthleteID := os.Getenv("INTERVALS_ATHLETE_ID")

	intervalsClient := intervals.NewIntervalsClient(
		intervals.APIURL,
		intervalsApiKey,
		intervalsAthleteID,
	)

	// future features:
	// - quarterly digest
	// - annual digest / compare data year-over-year
	today := time.Now()
	oneMonthAgo := time.Now().AddDate(0, -1, 0)
	wellness, err := intervalsClient.ListWellnessRecordsForDateRange(oneMonthAgo, today)
	if err != nil {
		log.Fatal(err)
	}

	// remove wellness fields that aren't relevant to the weekly digest
	wellness = trimWellnessRecords(wellness)
	wellnessJson, err := json.Marshal(wellness)
	if err != nil {
		log.Fatal(err)
	}

	wellnessToon, err := toon.JSONToToon(string(wellnessJson))
	if err != nil {
		log.Fatal(err)
	}

	// note - no activities endpoint as csv. do it myself to save tokens?
	activities, err := intervalsClient.ListActivitiesForDateRange(oneMonthAgo, today)
	if err != nil {
		log.Fatal(err)
	}

	activitiesJson, err := json.Marshal(activities)
	if err != nil {
		log.Fatal(err)
	}

	activitiesToon, err := toon.JSONToToon(string(activitiesJson))
	if err != nil {
		log.Fatal(err)
	}

	events, err := intervalsClient.ListEventsForDateRange(oneMonthAgo, today)
	if err != nil {
		log.Fatal(err)
	}

	events = trimEvents(events, 255, today.AddDate(0, 0, -8))
	eventsJson, err := json.Marshal(events)
	if err != nil {
		log.Fatal(err)
	}

	eventsToon, err := toon.JSONToToon(string(eventsJson))
	if err != nil {
		log.Fatal(err)
	}

	// make claude API request
	anthropicApiKey := os.Getenv("ANTHROPIC_API_KEY")
	anthropicClient := anthropic.NewClient(
		option.WithAPIKey(anthropicApiKey),
	)

	userContent := ai.IntervalsUserContent(
		today,
		wellnessToon,
		activitiesToon,
		eventsToon,
	)

	fmt.Println(userContent)

	// ~30-35 seconds
	// 1st pass (json, 0.05) - input: 10579, output: 1024 (limit 1024) - output stopped about halfway
	// 2nd pass (csv, 0.04) - input: 9294, output: 1259 (limit 2048) - output finished
	// 3rd pass (toon, 0.05) - input: 6640, output: 1427 (limit 2048) -output finished
	// 4th pass, cache, haiku (toon, 0.01) - input: 6580, output: 1098 (limit 2048) -output finished in 20s

	aiStart := time.Now()
	message, err := anthropicClient.Messages.New(context.TODO(), anthropic.MessageNewParams{
		Model:        anthropic.ModelClaudeSonnet4_6,
		MaxTokens:    2048,
		CacheControl: anthropic.NewCacheControlEphemeralParam(),
		System: []anthropic.TextBlockParam{
			{Text: weeklyDigestPrompt},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userContent)),
		},
	})
	aiEnd := time.Now()
	aiDuration := aiEnd.Sub(aiStart).Seconds()

	if err != nil {
		log.Fatal(err)
	}

	modelResponse, err := ai.ExtractModelResponse(message)
	if err != nil {
		log.Fatal(err)
	}

	_, err = fmt.Printf("ai time: %f seconds\n", aiDuration)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(ai.GetUsageStats(message.Usage))

	// add note for athlete
	noteName := "🤖 Weekly Digest — " + today.Format("2006-01-02")
	err = intervalsClient.CreateEvent(intervals.Event{
		Date:        today.Format("2006-01-02T00:00:00"),
		Name:        noteName,
		Category:    intervals.Note,
		Description: modelResponse,
	})

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("complete")
}

// trimWellnessRecords will set attributes to nil that aren't relevant for AI analysis.
// this is done to reduce input token size
func trimWellnessRecords(wellness []intervals.WellnessRecord) []intervals.WellnessRecord {
	for i := range wellness {
		wellness[i].BodyBatterMax = nil
		wellness[i].BodyBatteryMin = nil
		wellness[i].Carbohydrates = nil
		wellness[i].Fat = nil
		wellness[i].HighStressSeconds = nil
		wellness[i].KCalConsumed = nil
		wellness[i].LowStressSeconds = nil
		wellness[i].MediumStressSeconds = nil
		wellness[i].OxygenSaturation = nil
		wellness[i].OxygenSaturation = nil
		wellness[i].Protein = nil
		wellness[i].RestStressSeconds = nil
		wellness[i].SleepAwakeTimeSeconds = nil
		wellness[i].SleepDeepTimeSeconds = nil
		wellness[i].SleepLightTimeSeconds = nil
		wellness[i].SleepRemTimeSeconds = nil
	}
	return wellness
}

// trimEvents will clamp event content to a character limit to reduce input tokens.
// it will also drop out any weekly digest event that isn't from last week.
func trimEvents(
	events []intervals.Event,
	userInputLimit int,
	weekAgo time.Time,
) []intervals.Event {
	updated := []intervals.Event{}
	for _, e := range events {
		if e.Category != intervals.Note {
			e.Description = firstNBytes(e.Description, userInputLimit)
			updated = append(updated, e)
			continue
		} else {
			if strings.Contains(e.Name, "Weekly Digest") {
				// if older weekly digest - throw away
				d, err := time.Parse("2006-01-02", e.Date)
				if err != nil {
					log.Println(err)
					continue
				}

				if d.Before(weekAgo) {
					continue
				}

				// if previous weekly digest, use <!-- carryover only
				_, after, found := strings.Cut(e.Description, "<!-- carryover")
				if !found {
					e.Description = firstNBytes(e.Description, userInputLimit)
					updated = append(updated, e)
					continue
				} else {
					e.Description = after
					updated = append(updated, e)
					continue
				}

			} else {
				e.Description = firstNBytes(e.Description, userInputLimit)
				updated = append(updated, e)
				continue
			}
		}
	}

	return updated
}

// firstNBytes gets the first bytes of a string without allocating memory
func firstNBytes(s string, n int) string {
	i := 0
	for j := range s {
		if i == n {
			return s[:j]
		}
		i++
	}
	return s
}
