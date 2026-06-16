package main

import (
	"encoding/json"
	"fmt"
	intervals "intervals-functions/api"
	"log"
	"os"
	"time"

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

	today := time.Now()
	oneMonthAgo := time.Now().AddDate(0, -1, 0)
	wellness, err := intervalsClient.ListWellnessRecordsForDateRange(oneMonthAgo, today)
	if err != nil {
		log.Fatal(err)
	}

	wellnessJson, err := json.Marshal(wellness)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(wellnessJson))

	activities, err := intervalsClient.ListActivitiesForDateRange(oneMonthAgo, today)
	if err != nil {
		log.Fatal(err)
	}

	activitiesJson, err := json.Marshal(activities)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(activitiesJson))

	events, err := intervalsClient.ListEventsForDateRange(oneMonthAgo, today)
	if err != nil {
		log.Fatal(err)
	}

	eventsJson, err := json.Marshal(events)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(eventsJson))

	// make claude API request, need claude prompt and data as strings?
	// add note to intervals with claude response

	// future features:
	// - quarterly digest
	// - annual digest / compare data year-over-year
}
