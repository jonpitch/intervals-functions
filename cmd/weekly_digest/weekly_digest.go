package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	intervals "intervals-functions/api"
	"intervals-functions/utils/ai"
	"intervals-functions/utils/ptr"
	"log"
	"os"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	toon "github.com/bug4fix/totoon/go"
	"github.com/joho/godotenv"
)

func main() {
	_, found := os.LookupEnv("IS_NETLIFY")
	if !found {
		fmt.Println("not netlify environment, loading .env")
		err := godotenv.Load("../../.env")
		if err != nil {
			log.Fatal("Error loading .env file")
		}

		_, err = weeklydigest()
		if err != nil {
			log.Fatalf("an error occurred: %v", err)
		}
	} else {
		lambda.Start(handler)
	}
}

// lambda function handler
func handler(_ events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	status, err := weeklydigest()
	return events.APIGatewayProxyResponse{
		StatusCode: status,
		Body:       "done",
	}, err
}

//go:embed weekly_digest_prompt.md
var weeklyDigestPrompt string

func weeklydigest() (int, error) {
	intervalsApiKey := os.Getenv("INTERVALS_API_KEY")
	intervalsAthleteID := os.Getenv("INTERVALS_ATHLETE_ID")
	anthropicApiKey := os.Getenv("ANTHROPIC_API_KEY")
	if intervalsApiKey == "" || intervalsAthleteID == "" || anthropicApiKey == "" {
		return 500, errors.New("INTERVALS_API_KEY, INTERVALS_ATHLETE_ID, or ANTHROPIC_API_KEY is not set")
	}

	intervalsClient := intervals.NewIntervalsClient(
		intervals.APIURL,
		intervalsApiKey,
		intervalsAthleteID,
	)

	today := time.Now()
	noteName := "🤖 Weekly Digest — " + today.Format("2006-01-02")

	// TODO check past week for context note
	fmt.Println("checking for existing weekly digest...")
	todaysEvents, err := intervalsClient.ListEventsForDateRange(today, today)
	if err != nil {
		return 500, err
	}

	for _, e := range todaysEvents {
		if e.Name == noteName {
			fmt.Println("weekly digest already exists for today, skipping")
			return 200, nil
		}
	}

	// weekly digest executes on monday.
	// start the day before, to prevent incomplete data
	// from being considered for insights
	yesterday := today.AddDate(0, 0, -1)
	fortyTwoDaysAgo := yesterday.AddDate(0, 0, -42)
	fmt.Println("getting wellness data...")
	wellness, err := intervalsClient.ListWellnessRecordsForDateRange(fortyTwoDaysAgo, yesterday)
	if err != nil {
		return 500, err
	}

	// remove wellness fields that aren't relevant to the weekly digest
	// and compute averages to reduce token usage
	fmt.Println("computing rolling averages...")
	wellness = trimWellnessRecords(wellness)
	windowAverages := windowAverages(wellness)
	rollingAverages := weeklyAverages(wellness)

	windowAveragesJson, err := json.Marshal(windowAverages)
	if err != nil {
		return 500, err
	}

	rollingAveragesJson, err := json.Marshal(rollingAverages)
	if err != nil {
		return 500, err
	}

	windowAveragesToon, err := toon.JSONToToon(string(windowAveragesJson))
	if err != nil {
		return 500, err
	}

	rollingAveragesToon, err := toon.JSONToToon(string(rollingAveragesJson))
	if err != nil {
		return 500, err
	}

	wellnessJson, err := json.Marshal(wellness)
	if err != nil {
		return 500, err
	}

	wellnessToon, err := toon.JSONToToon(string(wellnessJson))
	if err != nil {
		return 500, err
	}

	fmt.Println("get activities data...")
	activities, err := intervalsClient.ListActivitiesForDateRange(fortyTwoDaysAgo, yesterday)
	if err != nil {
		return 500, err
	}

	activitiesJson, err := json.Marshal(activities)
	if err != nil {
		return 500, err
	}

	activitiesToon, err := toon.JSONToToon(string(activitiesJson))
	if err != nil {
		return 500, err
	}

	fmt.Println("get events data...")
	events, err := intervalsClient.ListEventsForDateRange(fortyTwoDaysAgo, yesterday)
	if err != nil {
		return 500, err
	}

	events = trimEvents(events, yesterday.AddDate(0, 0, -8))
	eventsJson, err := json.Marshal(events)
	if err != nil {
		return 500, err
	}

	eventsToon, err := toon.JSONToToon(string(eventsJson))
	if err != nil {
		return 500, err
	}

	// make claude API request
	anthropicClient := anthropic.NewClient(
		option.WithAPIKey(anthropicApiKey),
	)

	userContent := ai.IntervalsUserContent(
		today,
		wellnessToon,
		windowAveragesToon,
		rollingAveragesToon,
		activitiesToon,
		eventsToon,
	)

	aiStart := time.Now()
	fmt.Println("sending to LLM...")
	message, err := anthropicClient.Messages.New(context.TODO(), anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeSonnet4_6,
		MaxTokens: 2048,
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
		return 500, err
	}

	modelResponse, err := ai.ExtractModelResponse(message)
	if err != nil {
		return 500, err
	}

	_, err = fmt.Printf("ai time: %f seconds\n", aiDuration)
	if err != nil {
		return 500, err
	}

	fmt.Println(ai.GetUsageStats(message.Usage))

	// add note for athlete
	err = intervalsClient.CreateEvent(intervals.Event{
		Date:        today.Format("2006-01-02T00:00:00"),
		Name:        noteName,
		Category:    intervals.Note,
		Description: modelResponse,
	})

	if err != nil {
		return 500, err
	}

	fmt.Println("complete")
	return 200, nil
}

type AveragedAttribute struct {
	Average *float64
	Count   int
}

type WindowAverage struct {
	RestingHeartRate      AveragedAttribute
	Hrv                   AveragedAttribute
	SleepScore            AveragedAttribute
	Respiration           AveragedAttribute
	BodyBatteryMin        AveragedAttribute
	BodyBatteryMax        AveragedAttribute
	SleepNeedMinutes      AveragedAttribute
	SleepRemTimeSeconds   AveragedAttribute
	SleepDeepTimeSeconds  AveragedAttribute
	SleepLightTimeSeconds AveragedAttribute
	SleepAwakeTimeSeconds AveragedAttribute
	Vo2Max                AveragedAttribute
	CyclingVo2Max         AveragedAttribute
	Weight                AveragedAttribute
	MuscleMass            AveragedAttribute
	BodyFat               AveragedAttribute
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
	sleepNeedMinutesSum := 0
	sleepNeedMinutesTotal := 0
	sleepRemSum := 0
	sleepRemTotal := 0
	sleepDeepSum := 0
	sleepDeepTotal := 0
	sleepLightSum := 0
	sleepLightTotal := 0
	sleepAwakeSum := 0
	sleepAwakeTotal := 0
	vo2MaxSum := 0.0
	vo2MaxTotal := 0
	cyclingVo2MaxSum := 0.0
	cyclingVo2MaxTotal := 0
	weightSum := 0.0
	weightTotal := 0
	muscleMassSum := 0.0
	muscleMassTotal := 0
	bodyFatSum := 0.0
	bodyFatTotal := 0

	var avgRestingHr *float64
	var avgHrv *float64
	var avgSleepScore *float64
	var avgRespiration *float64
	var avgBodyBatteryMin *float64
	var avgBodyBatterMax *float64
	var avgSleepNeedMinutes *float64
	var avgSleepRem *float64
	var avgSleepDeep *float64
	var avgSleepLight *float64
	var avgSleepAwake *float64
	var avgVo2Max *float64
	var avgCyclingVo2Max *float64
	var avgWeight *float64
	var avgMuscleMass *float64
	var avgBodyFat *float64
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
		if w.SleepNeedMinutes != nil {
			sleepNeedMinutesSum += *w.SleepNeedMinutes
			sleepNeedMinutesTotal++
		}
		if w.SleepRemTimeSeconds != nil {
			sleepRemSum += *w.SleepRemTimeSeconds
			sleepRemTotal++
		}
		if w.SleepDeepTimeSeconds != nil {
			sleepDeepSum += *w.SleepDeepTimeSeconds
			sleepDeepTotal++
		}
		if w.SleepLightTimeSeconds != nil {
			sleepLightSum += *w.SleepLightTimeSeconds
			sleepLightTotal++
		}
		if w.SleepAwakeTimeSeconds != nil {
			sleepAwakeSum += *w.SleepAwakeTimeSeconds
			sleepAwakeTotal++
		}
		if w.Vo2Max != nil {
			vo2MaxSum += *w.Vo2Max
			vo2MaxTotal++
		}
		if w.CyclingVo2Max != nil {
			cyclingVo2MaxSum += *w.CyclingVo2Max
			cyclingVo2MaxTotal++
		}
		if w.Weight != nil {
			weightSum += *w.Weight
			weightTotal++
		}
		if w.MuscleMass != nil {
			muscleMassSum += *w.MuscleMass
			muscleMassTotal++
		}
		if w.BodyFat != nil {
			bodyFatSum += *w.BodyFat
			bodyFatTotal++
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
	if sleepNeedMinutesTotal != 0 {
		avgSleepNeedMinutes = ptr.Float(float64(sleepNeedMinutesSum) / float64(sleepNeedMinutesTotal))
	}
	if sleepRemTotal != 0 {
		avgSleepRem = ptr.Float(float64(sleepRemSum) / float64(sleepRemTotal))
	}
	if sleepDeepTotal != 0 {
		avgSleepDeep = ptr.Float(float64(sleepDeepSum) / float64(sleepDeepTotal))
	}
	if sleepLightTotal != 0 {
		avgSleepLight = ptr.Float(float64(sleepLightSum) / float64(sleepLightTotal))
	}
	if sleepAwakeTotal != 0 {
		avgSleepAwake = ptr.Float(float64(sleepAwakeSum) / float64(sleepAwakeTotal))
	}
	if vo2MaxTotal != 0 {
		avgVo2Max = ptr.Float(vo2MaxSum / float64(vo2MaxTotal))
	}
	if cyclingVo2MaxTotal != 0 {
		avgCyclingVo2Max = ptr.Float(cyclingVo2MaxSum / float64(cyclingVo2MaxTotal))
	}
	if weightTotal != 0 {
		avgWeight = ptr.Float(weightSum / float64(weightTotal))
	}
	if muscleMassTotal != 0 {
		avgMuscleMass = ptr.Float(muscleMassSum / float64(muscleMassTotal))
	}
	if bodyFatTotal != 0 {
		avgBodyFat = ptr.Float(bodyFatSum / float64(bodyFatTotal))
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
		SleepNeedMinutes: AveragedAttribute{
			Average: avgSleepNeedMinutes,
			Count:   sleepNeedMinutesTotal,
		},
		SleepRemTimeSeconds: AveragedAttribute{
			Average: avgSleepRem,
			Count:   sleepRemTotal,
		},
		SleepDeepTimeSeconds: AveragedAttribute{
			Average: avgSleepDeep,
			Count:   sleepDeepTotal,
		},
		SleepLightTimeSeconds: AveragedAttribute{
			Average: avgSleepLight,
			Count:   sleepLightTotal,
		},
		SleepAwakeTimeSeconds: AveragedAttribute{
			Average: avgSleepAwake,
			Count:   sleepAwakeTotal,
		},
		Vo2Max: AveragedAttribute{
			Average: avgVo2Max,
			Count:   vo2MaxTotal,
		},
		CyclingVo2Max: AveragedAttribute{
			Average: avgCyclingVo2Max,
			Count:   cyclingVo2MaxTotal,
		},
		Weight: AveragedAttribute{
			Average: avgWeight,
			Count:   weightTotal,
		},
		MuscleMass: AveragedAttribute{
			Average: avgMuscleMass,
			Count:   muscleMassTotal,
		},
		BodyFat: AveragedAttribute{
			Average: avgBodyFat,
			Count:   bodyFatTotal,
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
		wellness[i].Protein = nil
		wellness[i].RestStressSeconds = nil
	}
	return wellness
}

// trimEvents will clamp event content to a character limit to reduce input tokens.
// it will also drop out any weekly digest event that isn't from last week.
func trimEvents(
	events []intervals.Event,
	weekAgo time.Time,
) []intervals.Event {
	updated := []intervals.Event{}
	for _, e := range events {
		if e.Category != intervals.Note {
			continue
		}

		if strings.Contains(e.Name, "Weekly Digest") {
			// if older weekly digest - throw away
			d, err := time.Parse("2006-01-02T15:04:05", e.Date)
			if err != nil {
				log.Println(err)
				continue
			}

			if d.Before(weekAgo) {
				continue
			}

			// if previous weekly digest, use <!-- carryover only
			// ignore everything else
			_, after, found := strings.Cut(e.Description, "<!-- carryover")
			if !found {
				continue
			} else {
				e.Description = after
				updated = append(updated, e)
				continue
			}

		} else {
			continue
		}
	}

	return updated
}
