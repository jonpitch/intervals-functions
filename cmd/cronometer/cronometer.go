package main

import (
	"context"
	"errors"
	"fmt"
	intervals "intervals-functions/api"
	"intervals-functions/utils/calc"
	"intervals-functions/utils/csv"
	"intervals-functions/utils/format"
	"intervals-functions/utils/ptr"
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/jrmycanady/gocronometer"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type DateRange struct {
	Start time.Time
	End   time.Time
}

func main() {
	_, found := os.LookupEnv("NETLIFY")
	if !found {
		fmt.Println("not netlify environment, loading .env")
		err := godotenv.Load("../../.env")
		if err != nil {
			log.Fatal("Error loading .env file")
		}

		_, err = cronometerToIntervals()
		if err != nil {
			log.Fatalf("an error occurred: %v", err)
		}
	} else {
		lambda.Start(handler)
	}
}

// lambda function handler
func handler(_ events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	status, err := cronometerToIntervals()
	return events.APIGatewayProxyResponse{
		StatusCode: status,
		Body:       "done",
	}, err
}

// extract yesterday's nutrition data from cronometer and update yesterday's wellness record in intervals
func cronometerToIntervals() (int, error) {
	username := os.Getenv("CRONOMETER_USERNAME")
	password := os.Getenv("CRONOMETER_PASSWORD")
	if username == "" || password == "" {
		return 500, errors.New("CRONOMETER_USERNAME or CRONOMETER_PASSWORD is not set")
	}

	intervalsAthleteID := os.Getenv("INTERVALS_ATHLETE_ID")
	intervalsApiKey := os.Getenv("INTERVALS_API_KEY")
	if intervalsAthleteID == "" || intervalsApiKey == "" {
		return 500, errors.New("INTERVALS_ATHLETE_ID or INTERVALS_API_KEY is not set")
	}

	cronometer := gocronometer.NewClient(nil)
	if err := cronometer.Login(context.Background(), username, password); err != nil {
		return 500, err
	}

	// get nutrition info for yesterday
	now := time.Now()
	dateRange := getYesterdayDateRange(now)
	rawNutritionCSV, err := cronometer.ExportDailyNutrition(context.Background(), dateRange.Start, dateRange.End)
	if err != nil {
		return 500, err
	}

	// gocronometer doesn't provide a parser for nutrition data. gocronometer.ParseServingsExport is close
	fmt.Printf("parsing cronomoter daily totals for %v...\n", dateRange.Start.Format("2006-01-02"))
	totals, err := csv.ParseCronometerDailyTotals(rawNutritionCSV)
	if err != nil {
		return 500, err
	}

	fmt.Printf(
		"daily totals: %sk %sc %sp %sf\n",
		format.FloatPtr(totals.Kcal),
		format.FloatPtr(totals.Carbs),
		format.FloatPtr(totals.Protein),
		format.FloatPtr(totals.Fat),
	)

	// get biometrics info for yesterday
	// NOTE: intervals.icu garmin sync does not include oxygen saturation or respiration rate.
	// the garmin sync with cronometer does include this data, but it's a series rather than a single metric.
	// this will be removed if intervals.icu adds support for spo2 or respiration in its native sync.
	rawBiometrics, err := cronometer.ExportBiometrics(context.Background(), dateRange.Start, dateRange.End)
	if err != nil {
		return 500, err
	}

	biometrics, err := gocronometer.ParseBiometricRecordsExport(strings.NewReader(rawBiometrics), nil)
	if err != nil {
		return 500, err
	}

	spo2 := []float64{}
	respiration := []float64{}
	for _, b := range biometrics {
		switch b.Metric {
		case "Oxygen Saturation (SPO2) (Garmin)":
			spo2 = append(spo2, b.Amount)
		case "Respiration Rate (Garmin)":
			respiration = append(respiration, b.Amount)
		}
	}

	avgSpo2 := calc.Average(spo2)
	roundedSpo2 := calc.RoundToTenth(avgSpo2)

	avgRespiration := calc.Average(respiration)
	roundedRespiration := calc.RoundToTenth(avgRespiration)
	fmt.Printf("average spo2: %.1f - average respiration: %.1f\n", roundedSpo2, roundedRespiration)

	fmt.Printf("get intervals wellness record for %v\n", dateRange.Start.Format("2006-01-02"))
	intervalsClient := intervals.NewIntervalsClient(intervals.APIURL, intervalsApiKey, intervalsAthleteID)
	wellness, err := intervalsClient.GetWellnessRecord(dateRange.Start)
	if err != nil {
		return 500, err
	}

	fmt.Println("updating intervals wellness record...")
	err = intervalsClient.UpdateWellnessRecord(intervals.WellnessRecord{
		ID:               wellness.ID,
		KCalConsumed:     totals.Kcal,
		Carbohydrates:    totals.Carbs,
		Protein:          totals.Protein,
		Fat:              totals.Fat,
		OxygenSaturation: ptr.Float(roundedSpo2),
		Respiration:      ptr.Float(roundedRespiration),
	})

	if err != nil {
		return 500, err
	}

	return 200, nil
}

// get the start and end of the day for the previous day
func getYesterdayDateRange(now time.Time) DateRange {
	yesterday := now.AddDate(0, 0, -1)
	year := yesterday.Year()
	month := yesterday.Month()
	day := yesterday.Day()

	return DateRange{
		Start: time.Date(year, month, day, 0, 0, 0, 0, time.UTC),
		End:   time.Date(year, month, day, 23, 59, 59, 0, time.UTC),
	}
}
