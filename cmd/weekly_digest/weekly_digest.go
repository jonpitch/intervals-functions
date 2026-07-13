package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	intervals "intervals-functions/api"
	"intervals-functions/utils/ai"
	"intervals-functions/utils/ptr"
	"log"
	"os"
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
	fortyTwoDaysAgo := time.Now().AddDate(0, 0, -42)
	wellness, err := intervalsClient.ListWellnessRecordsForDateRange(fortyTwoDaysAgo, today)
	if err != nil {
		log.Fatal(err)
	}

	// remove wellness fields that aren't relevant to the weekly digest
	// and compute averages
	wellness = trimWellnessRecords(wellness)
	windowAverages := windowAverages(wellness)
	rollingAverages := weeklyAverages(wellness)

	windowAveragesJson, err := json.Marshal(windowAverages)
	if err != nil {
		log.Fatal(err)
	}

	rollingAveragesJson, err := json.Marshal(rollingAverages)
	if err != nil {
		log.Fatal(err)
	}

	windowAveragesToon, err := toon.JSONToToon(string(windowAveragesJson))
	if err != nil {
		log.Fatal(err)
	}

	rollingAveragesToon, err := toon.JSONToToon(string(rollingAveragesJson))
	if err != nil {
		log.Fatal(err)
	}

	// TODO 42 day averages of fitness/ctl/atl?
	// TODO rolling averages of fitness/ctl/atl?

	wellnessJson, err := json.Marshal(wellness)
	if err != nil {
		log.Fatal(err)
	}

	wellnessToon, err := toon.JSONToToon(string(wellnessJson))
	if err != nil {
		log.Fatal(err)
	}

	// note - no activities endpoint as csv. do it myself to save tokens?
	activities, err := intervalsClient.ListActivitiesForDateRange(fortyTwoDaysAgo, today)
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

	// make claude API request
	anthropicApiKey := os.Getenv("ANTHROPIC_API_KEY")
	anthropicClient := anthropic.NewClient(
		option.WithAPIKey(anthropicApiKey),
	)

	userContent := ai.IntervalsUserContent(
		today,
		wellnessToon,
		windowAveragesToon,
		rollingAveragesToon,
		activitiesToon,
	)

	fmt.Println(userContent)

	aiStart := time.Now()
	message, err := anthropicClient.Messages.New(context.TODO(), anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeSonnet4_6,
		MaxTokens: 2048,
		// CacheControl: anthropic.NewCacheControlEphemeralParam(),
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

	fmt.Println(modelResponse)

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

type AveragedAttribute struct {
	Average *float64
	Count   int
}

type WindowAverage struct {
	RestingHeartRate AveragedAttribute
	Hrv              AveragedAttribute
	SleepScore       AveragedAttribute
	Respiration      AveragedAttribute
	BodyBatteryMin   AveragedAttribute
	BodyBatteryMax   AveragedAttribute
}

// windowAverages computes averages for specific wellness attributes across all wellness records
func windowAverages(wellness []intervals.WellnessRecord) WindowAverage {
	restingHrSum := 0
	restingHrTotal := 0
	hrvSum := 0.0
	hrvTotal := 0
	sleepScoreSum := 0.0
	sleepScoreTotal := 0
	respirationSum := 0.0
	respirationTotal := 0
	bodyBatteryMinSum := 0
	bodyBatteryMinTotal := 0
	bodyBatteryMaxSum := 0
	bodyBatterMaxTotal := 0

	var avgRestingHr *float64
	var avgHrv *float64
	var avgSleepScore *float64
	var avgRespiration *float64
	var avgBodyBatteryMin *float64
	var avgBodyBatterMax *float64
	for _, w := range wellness {
		if w.RestingHr != nil {
			restingHrSum += *w.RestingHr
			restingHrTotal++
		}
		if w.HrvRmssd != nil {
			hrvSum += *w.HrvRmssd
			hrvTotal++
		}
		if w.SleepScore != nil {
			sleepScoreSum += *w.SleepScore
			sleepScoreTotal++
		}
		if w.BodyBatteryMin != nil {
			bodyBatteryMinSum += *w.BodyBatteryMin
			bodyBatteryMinTotal++
		}
		if w.BodyBatterMax != nil {
			bodyBatteryMaxSum += *w.BodyBatterMax
			bodyBatterMaxTotal++
		}
		if w.Respiration != nil {
			respirationSum += *w.Respiration
			respirationTotal++
		}
	}

	if restingHrTotal != 0 {
		avgRestingHr = ptr.Float(float64(restingHrSum) / float64(restingHrTotal))
	}
	if hrvTotal != 0 {
		avgHrv = ptr.Float(hrvSum / float64(hrvTotal))
	}
	if sleepScoreTotal != 0 {
		avgSleepScore = ptr.Float(sleepScoreSum / float64(sleepScoreTotal))
	}
	if bodyBatteryMinTotal != 0 {
		avgBodyBatteryMin = ptr.Float(float64(bodyBatteryMinSum) / float64(bodyBatteryMinTotal))
	}
	if bodyBatterMaxTotal != 0 {
		avgBodyBatterMax = ptr.Float(float64(bodyBatteryMaxSum) / float64(bodyBatterMaxTotal))
	}
	if respirationTotal != 0 {
		avgRespiration = ptr.Float(respirationSum / float64(respirationTotal))
	}

	return WindowAverage{
		RestingHeartRate: AveragedAttribute{
			Average: avgRestingHr,
			Count:   restingHrTotal,
		},
		Hrv: AveragedAttribute{
			Average: avgHrv,
			Count:   hrvTotal,
		},
		SleepScore: AveragedAttribute{
			Average: avgSleepScore,
			Count:   sleepScoreTotal,
		},
		Respiration: AveragedAttribute{
			Average: avgRespiration,
			Count:   respirationTotal,
		},
		BodyBatteryMin: AveragedAttribute{
			Average: avgBodyBatteryMin,
			Count:   bodyBatteryMinTotal,
		},
		BodyBatteryMax: AveragedAttribute{
			Average: avgBodyBatterMax,
			Count:   bodyBatterMaxTotal,
		},
	}
}

// used to group wellness records.
// capturing year and week, to handle edge cases in ISO week
// that wrap years
type Week struct {
	Year int
	Week int
}

type WeeklyAverages struct {
	Period   Week
	Averages WindowAverage
}

// weeklyAverages groups wellness records by week and computes average for that time period
func weeklyAverages(wellness []intervals.WellnessRecord) []WeeklyAverages {
	if len(wellness) == 0 {
		return []WeeklyAverages{}
	}

	grouped := map[Week][]intervals.WellnessRecord{}
	for _, w := range wellness {
		t, err := time.Parse("2006-01-02", string(w.ID))
		if err != nil {
			continue
		}

		year, week := t.ISOWeek()
		if val, ok := grouped[Week{Year: year, Week: week}]; !ok {
			averages := []intervals.WellnessRecord{}
			averages = append(averages, w)

			grouped[Week{Year: year, Week: week}] = averages
		} else {
			grouped[Week{Year: year, Week: week}] = append(val, w)
		}
	}

	result := []WeeklyAverages{}
	for key, g := range grouped {
		result = append(result, WeeklyAverages{
			Period:   key,
			Averages: windowAverages(g),
		})
	}

	return result
}

// trimWellnessRecords will set attributes to nil that aren't relevant for AI analysis.
// this is done to reduce input token size
func trimWellnessRecords(wellness []intervals.WellnessRecord) []intervals.WellnessRecord {
	for i := range wellness {
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
		wellness[i].SleepNeedMinutes = nil
		wellness[i].SleepAwakeTimeSeconds = nil
		wellness[i].SleepDeepTimeSeconds = nil
		wellness[i].SleepLightTimeSeconds = nil
		wellness[i].SleepRemTimeSeconds = nil
	}
	return wellness
}
