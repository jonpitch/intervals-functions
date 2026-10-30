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
	sleepResponse, err := garminClient.GetDailySleep(yesterday)
	if err != nil {
		return 500, fmt.Errorf("get daily sleep: %w", err)
	}

	dailySummaryResponse, err := garminClient.GetDailySummary(yesterday, yesterday)
	if err != nil {
		return 500, fmt.Errorf("get daily summary: %w", err)
	}

	stressResponse, err := garminClient.GetDailyStress(yesterday, yesterday)
	if err != nil {
		return 500, fmt.Errorf("get daily stress: %w", err)
	}

	weightResponse, err := garminClient.GetWeight(yesterday, yesterday)
	if err != nil {
		return 500, fmt.Errorf("get weight: %w", err)
	}

	fmt.Println("-- sleep")
	fmt.Printf("light: %d\n", sleepResponse.LightSleepSeconds)
	fmt.Printf("deep: %d\n", sleepResponse.DeepSleepSeconds)
	fmt.Printf("rem: %d\n", sleepResponse.RemSleepSeconds)
	fmt.Printf("awake: %d\n", sleepResponse.AwakeSleepSeconds)
	fmt.Printf("baseline: %d\n", sleepResponse.SleepNeed.BaselineMinutes)
	fmt.Printf("actual: %d\n", sleepResponse.SleepNeed.ActualMinutes)
	fmt.Println("-- body battery")
	fmt.Printf("min bat: %d\n", dailySummaryResponse.Data[0].BodyBattery.MinValue)
	fmt.Printf("max bat: %d\n", dailySummaryResponse.Data[0].BodyBattery.MaxValue)
	fmt.Println("-- stress")
	fmt.Printf("overall: %d\n", stressResponse[0].Values.OverallStressLevel)
	fmt.Printf("rest: %d\n", stressResponse[0].Values.RestStressDurationSeconds)
	fmt.Printf("low: %d\n", stressResponse[0].Values.LowStressDurationSeconds)
	fmt.Printf("med: %d\n", stressResponse[0].Values.MediumStressDurationSeconds)
	fmt.Printf("high: %d\n", stressResponse[0].Values.HighStressDurationSeconds)
	fmt.Println("-- weight")
	fmt.Printf("weight: %f\n", weightResponse.DailyWeightSummaries[0].LatestWeight.WeightInGrams)
	fmt.Printf("muscle: %d\n", weightResponse.DailyWeightSummaries[0].LatestWeight.MuscleMassInGrams)
	fmt.Printf("fat: %f\n", weightResponse.DailyWeightSummaries[0].LatestWeight.BodyFat)

	// TODO upate intervals wellness record

	return 200, nil
}
