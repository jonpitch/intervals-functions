package main

import (
	"errors"
	"fmt"
	"intervals-functions/api/garmin"
	"intervals-functions/api/intervals"
	"intervals-functions/utils/connect"
	"intervals-functions/utils/ptr"
	"log"
	"math"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
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

		_, err = garminToIntervals()
		if err != nil {
			log.Fatalf("an error occurred: %v", err)
		}
	} else {
		lambda.Start(handler)
	}
}

// lambda function handler
func handler(_ events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	status, err := garminToIntervals()
	return events.APIGatewayProxyResponse{
		StatusCode: status,
		Body:       "done",
	}, err
}

func garminToIntervals() (int, error) {
	garminApiKey := os.Getenv("GARMIN_API_KEY")
	if garminApiKey == "" {
		return 500, errors.New("GARMIN_API_KEY is not set")
	}

	intervalsAthleteID := os.Getenv("INTERVALS_ATHLETE_ID")
	intervalsApiKey := os.Getenv("INTERVALS_API_KEY")
	if intervalsAthleteID == "" || intervalsApiKey == "" {
		return 500, errors.New("INTERVALS_ATHLETE_ID or INTERVALS_API_KEY is not set")
	}

	garminClient, err := garmin.NewGarminClient(garminApiKey)
	if err != nil {
		return 500, fmt.Errorf("create Garmin client: %w", err)
	}

	yesterday := time.Now().AddDate(0, 0, -1)
	sleepResponse, err := garminClient.GetDailySleepSummariesByDate(yesterday, yesterday)
	if err != nil {
		return 500, fmt.Errorf("get daily sleep: %w", err)
	}
	if len(sleepResponse) == 0 {
		return 500, errors.New("get daily sleep: no sleep data returned")
	}

	dailySummaryResponse, err := garminClient.GetDailySummary(yesterday, yesterday)
	if err != nil {
		return 500, fmt.Errorf("get daily summary: %w", err)
	}

	weightResponse, err := garminClient.GetWeight(yesterday, yesterday)
	if err != nil {
		return 500, fmt.Errorf("get weight: %w", err)
	}

	hrvResponse, err := garminClient.GetHrvDailySummaries(yesterday, yesterday)
	if err != nil {
		return 500, fmt.Errorf("get hrv: %w", err)
	}

	fmt.Printf("get intervals wellness record for %v\n", yesterday.Format("2006-01-02"))
	intervalsClient := intervals.NewIntervalsClient(intervals.APIURL, intervalsApiKey, intervalsAthleteID)
	wellness, err := intervalsClient.GetWellnessRecord(yesterday)
	if err != nil {
		return 500, err
	}

	fmt.Println("updating intervals wellness record...")
	stressAvg := connect.GarminStressToIntervalsStress(dailySummaryResponse[0].Stress.AverageLevel)
	err = intervalsClient.UpdateWellnessRecord(intervals.WellnessRecord{
		ID: wellness.ID,
		// body battery
		BodyBatteryMin: ptr.Int(dailySummaryResponse[0].BodyBattery.MinValue),
		BodyBatteryMax: ptr.Int(dailySummaryResponse[0].BodyBattery.MaxValue),
		// sleep
		SleepLightTimeSeconds: ptr.Int(sleepResponse[0].LightSleepSeconds),
		SleepDeepTimeSeconds:  ptr.Int(sleepResponse[0].DeepSleepSeconds),
		SleepRemTimeSeconds:   ptr.Int(sleepResponse[0].RemSleepSeconds),
		SleepAwakeTimeSeconds: ptr.Int(sleepResponse[0].AwakeSleepSeconds),
		SleepNeedMinutes:      ptr.Int(sleepResponse[0].SleepNeed.ActualMinutes),
		SleepQuality:          connect.GarminSleepQualityToIntervalsSleepQuality(sleepResponse[0].SleepScores.Overall.Qaulifier),
		SleepScore:            ptr.Float(float64(sleepResponse[0].SleepScores.Overall.Value)),
		// stress
		Stress:              &stressAvg,
		RestStressSeconds:   ptr.Int(dailySummaryResponse[0].Stress.RestDurationInMillis / 1000),
		LowStressSeconds:    ptr.Int(dailySummaryResponse[0].Stress.LowStressDurationInMillis / 1000),
		MediumStressSeconds: ptr.Int(dailySummaryResponse[0].Stress.MediumStressDurationInMillis / 1000),
		HighStressSeconds:   ptr.Int(dailySummaryResponse[0].Stress.HighStressDurationInMillis / 1000),
		// weight
		BodyFat:    ptr.Float(weightResponse[0].LatestWeight.BodyFat),
		MuscleMass: ptr.Float(weightResponse[0].LatestWeight.MuscleMassInGrams / 1000),
		Weight:     ptr.Float(weightResponse[0].LatestWeight.WeightInGrams / 1000),
		// heart
		RestingHr:        ptr.Int(int(math.Round(dailySummaryResponse[0].HeartRate.RestingValue))),
		HrvRmssd:         ptr.Float(hrvResponse[0].LastNightAvg),
		OxygenSaturation: ptr.Float(sleepResponse[0].AverageSpo2),
	})

	if err != nil {
		return 500, err
	}

	return 200, nil
}
