package main

import (
	"context"
	"fmt"
	intervals "intervals-functions/api"
	"intervals-functions/utils/csv"
	"intervals-functions/utils/format"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/jrmycanady/gocronometer"
)

// Backfill intervals with cronometer data. meant to be used as a one-off.
//
//	go run backfill.go from to
//	go run backfill.go 2026-01-01 2026-02-01
func main() {
	if len(os.Args) < 3 {
		log.Fatal("usage: backfill YYYY-MM-DD YYYY-MM-DD")
	}

	fromStr := os.Args[1]
	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		log.Fatalf("invalid from date %q: %v", fromStr, err)
	}

	toStr := os.Args[2]
	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		log.Fatalf("invalid to date %q: %v", toStr, err)
	}

	if !from.Before(to) {
		log.Fatalf("to date must be after from date (from %v, to %v)", from, to)
	}

	err = godotenv.Load("../../.env")
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

	cronometer := gocronometer.NewClient(nil)
	if err := cronometer.Login(context.Background(), username, password); err != nil {
		log.Fatalf("failed to login: %v", err)
	}

	for day := from; !day.After(to); day = day.AddDate(0, 0, 1) {
		fmt.Printf("export daily nutrition for date: %s\n", day.Format("2006-01-02"))
		rawNutritionCSV, err := cronometer.ExportDailyNutrition(
			context.Background(),
			time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC),
			time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC),
		)
		if err != nil {
			fmt.Printf("failed to parse daily nutrition: %v", err)
			continue
		}

		fmt.Println("parse daily nutrition...")
		totals, err := csv.ParseCronometerDailyTotals(rawNutritionCSV)
		if err != nil {
			fmt.Printf("failed to parse daily totals: %v", err)
			continue
		}

		fmt.Printf(
			"daily totals: %sk %sc %sp %sf\n",
			format.FloatPtr(totals.Kcal),
			format.FloatPtr(totals.Carbs),
			format.FloatPtr(totals.Protein),
			format.FloatPtr(totals.Fat),
		)

		fmt.Println("get intervals wellness record...")
		intervalsClient := intervals.NewIntervalsClient(intervals.APIURL, intervalsApiKey, intervalsAthleteID)
		wellness, err := intervalsClient.GetWellnessRecord(day)
		if err != nil {
			fmt.Printf("failed to get wellness record: %v", err)
			continue
		}

		fmt.Println("updating intervals wellness record...")
		err = intervalsClient.UpdateWellnessRecord(intervals.WellnessRecord{
			ID:            wellness.ID,
			KCalConsumed:  totals.Kcal, // coalesce
			Carbohydrates: totals.Carbs,
			Protein:       totals.Protein,
			Fat:           totals.Fat,
		})

		if err != nil {
			fmt.Printf("failed to update wellness record: %v", err)
			continue
		}
	}

	fmt.Println("done")
}
