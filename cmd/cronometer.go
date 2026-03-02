package main

import (
	"context"
	"fmt"
	intervals "intervals-functions/api"
	"intervals-functions/utils/csv"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/jrmycanady/gocronometer"
)

type DateRange struct {
	Start time.Time
	End   time.Time
}

func main() {
	// TODO use an environment agnostic approach in order to convert this to a lambda
	err := godotenv.Load("../.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	username := os.Getenv("CRONOMETER_USERNAME")
	password := os.Getenv("CRONOMETER_PASSWORD")
	if username == "" || password == "" {
		log.Fatal("CRONOMETER_USERNAME or CRONOMETER_PASSWORD is not set")
	}

	intervalsAthleteID := os.Getenv("INTERVALS_ATHLETE_ID")
	intervalsApiKey := os.Getenv("INTERVALS_API_KEY")
	if intervalsAthleteID == "" || intervalsApiKey == "" {
		log.Fatal("INTERVALS_ATHLETE_ID or INTERVALS_API_KEY is not set")
	}

	c := gocronometer.NewClient(nil)
	if err := c.Login(context.Background(), username, password); err != nil {
		log.Fatalf("failed to login: %v", err)
	}

	// get nutrition info for yesterday
	now := time.Now()
	dateRange := getYesterdayDateRange(now)
	rawNutritionCSV, err := c.ExportDailyNutrition(context.Background(), dateRange.Start, dateRange.End)
	if err != nil {
		log.Fatalf("failed to retrieve daily nutrition data: %v", err)
	}

	totals, err := csv.ParseCronometerDailyTotals(rawNutritionCSV)
	if err != nil {
		log.Fatalf("failed to parse daily totals: %v", err)
	}

	fmt.Printf("Kcal: %d\n", *totals.Kcal)
	fmt.Printf("Carbs (g): %d\n", *totals.Carbs)
	fmt.Printf("Protein (g): %d\n", *totals.Protein)
	fmt.Printf("Fat (g): %d\n", *totals.Fat)

	intervalsClient := intervals.NewIntervalsClient(intervals.APIURL, intervalsApiKey, intervalsAthleteID)
	wellness, err := intervalsClient.GetWellnessRecord(dateRange.Start)
	if err != nil {
		log.Fatalf("failed to get wellness record: %v", err)
	}

	err = intervalsClient.UpdateWellnessRecord(intervals.WellnessRecord{
		ID:            wellness.ID,
		KCalConsumed:  *totals.Kcal,
		Carbohydrates: *totals.Carbs,
		Protein:       *totals.Protein,
		Fat:           *totals.Fat,
	})

	if err != nil {
		log.Fatalf("failed to update wellness record: %v", err)
	}

	fmt.Print("done")
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
