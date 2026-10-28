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

	// TODO decision - make requests to summary APIs or individual?
	// there's no vo2 max?
	fmt.Println(sleepResponse.SleepTimeSeconds)

	return 200, nil
}
