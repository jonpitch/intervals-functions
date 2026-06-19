package main

import (
	"context"
	"encoding/json"
	"fmt"
	intervals "intervals-functions/api"
	"intervals-functions/utils/ai"
	"log"
	"os"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/joho/godotenv"
)

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
	// wellness, err := intervalsClient.ListWellnessRecordsForDateRange(oneMonthAgo, today)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// remove wellness fields that aren't relevant to the weekly digest
	// wellness = trimWellnessRecords(wellness)

	// wellnessJson, err := json.Marshal(wellness)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// get input for claude: wellness, activities, events
	wellness, err := intervalsClient.ListWellnessRecordsForDateRangeAsCsv(oneMonthAgo, today)
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

	events, err := intervalsClient.ListEventsForDateRange(oneMonthAgo, today)
	if err != nil {
		log.Fatal(err)
	}

	eventsJson, err := json.Marshal(events)
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
		wellness,
		string(activitiesJson),
		string(eventsJson),
	)

	fmt.Println(userContent)

	// ~30-35 seconds
	// 1st pass (json, 0.05) - input: 10579, output: 1024 (limit 1024) - output stopped about halfway
	// 2nd pass (csv, 0.04) - input: 9294, output: 1259 (limit 2048) - output finished

	message, err := anthropicClient.Messages.New(context.TODO(), anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeSonnet4_6,
		MaxTokens: 2048,
		System: []anthropic.TextBlockParam{
			{Text: ai.WeeklyDigestPrompt},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userContent)),
		},
	})

	if err != nil {
		log.Fatal(err)
	}

	modelResponse, err := ai.ExtractText(message)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%+v\n", message.Content)
	fmt.Printf("%+v\n", message.Usage)
	fmt.Println(modelResponse)

	// add note for athlete
	noteName := "🤖 Weekly Digest — " + today.Format("2006-01-02")
	err = intervalsClient.CreateEvent(intervals.Event{
		Date:        today.Format("2006-01-02T00:00:00"),
		Name:        noteName,
		Category:    "NOTE", // TODO enum
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
