package main

import (
	"errors"
	"fmt"
	"intervals-functions/api/garmin"
	"log"
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

	fmt.Println("-- sleep")
	fmt.Printf("date: %s\n", sleepResponse[0].CalendarDate)
	fmt.Printf("light: %d\n", sleepResponse[0].LightSleepSeconds)
	fmt.Printf("deep: %d\n", sleepResponse[0].DeepSleepSeconds)
	fmt.Printf("rem: %d\n", sleepResponse[0].RemSleepSeconds)
	fmt.Printf("awake: %d\n", sleepResponse[0].AwakeSleepSeconds)
	fmt.Printf("baseline: %d\n", sleepResponse[0].SleepNeed.BaselineMinutes)
	fmt.Printf("actual: %d\n", sleepResponse[0].SleepNeed.ActualMinutes)
	fmt.Printf("spo2: %f\n", sleepResponse[0].AverageSpo2)
	fmt.Printf("hr: %f\n", sleepResponse[0].AverageHeartRate)
	fmt.Printf("quality: %s\n", sleepResponse[0].SleepScores.Overall.Qaulifier)
	fmt.Printf("score: %d\n", sleepResponse[0].SleepScores.Overall.Value)
	fmt.Println("-- body battery")
	fmt.Printf("date: %s\n", dailySummaryResponse[0].CalendarDate)
	fmt.Printf("min bat: %d\n", dailySummaryResponse[0].BodyBattery.MinValue)
	fmt.Printf("max bat: %d\n", dailySummaryResponse[0].BodyBattery.MaxValue)
	fmt.Println("-- stress")
	fmt.Printf("date: %s\n", dailySummaryResponse[0].CalendarDate)
	fmt.Printf("overall: %d\n", dailySummaryResponse[0].Stress.TotalDurationInMillis)
	fmt.Printf("rest: %d\n", dailySummaryResponse[0].Stress.RestDurationInMillis)
	fmt.Printf("low: %d\n", dailySummaryResponse[0].Stress.LowStressDurationInMillis)
	fmt.Printf("med: %d\n", dailySummaryResponse[0].Stress.MediumStressDurationInMillis)
	fmt.Printf("high: %d\n", dailySummaryResponse[0].Stress.HighStressDurationInMillis)
	fmt.Println("-- weight")
	fmt.Printf("date: %s\n", weightResponse[0].LatestWeight.CalendarDate)
	fmt.Printf("weight: %f\n", weightResponse[0].LatestWeight.WeightInGrams)
	fmt.Printf("muscle: %f\n", weightResponse[0].LatestWeight.MuscleMassInGrams)
	fmt.Printf("fat: %f\n", weightResponse[0].LatestWeight.BodyFat)
	fmt.Println("-- hrv")
	fmt.Printf("date: %s\n", hrvResponse[0].CalendarDate)
	fmt.Printf("last night avg: %f\n", hrvResponse[0].LastNightAvg)

	// TODO upate intervals wellness record

	return 200, nil
}
