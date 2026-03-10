package main

import (
	"encoding/json"
	"errors"
	"fmt"
	intervals "intervals-functions/api"
	"intervals-functions/utils/ptr"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type SleepResponse struct {
	IndividualStats []SleepEntry `json:"individualStats"`
}

type SleepEntry struct {
	Date   GarminDate `json:"calendarDate"`
	Values SleepValue `json:"values"`
}

type SleepQuality string

const (
	Excellent SleepQuality = "EXCELLENT"
	Good      SleepQuality = "GOOD"
	Fair      SleepQuality = "FAIR"
	Poor      SleepQuality = "POOR"
)

type SleepValue struct {
	RemTimeSeconds        int          `json:"remTime"`
	RestingHeartRate      int          `json:"restingHeartRate"`
	Respiration           float64      `json:"respiration"`
	TotalSleepTimeSeconds int          `json:"totalSleepTimeInSeconds"`
	DeepTimeSeconds       int          `json:"deepTime"`
	AwakeTimeSeconds      int          `json:"awakeTime"`
	SleepQuality          SleepQuality `json:"sleepScoreQuality"`
	Spo2                  float64      `json:"spO2"`
	LightTimeSeconds      int          `json:"lightTime"`
	AverageOvernightHrv   float64      `json:"avgOvernightHrv"`
}

type RespirationEntry struct {
	Date                    GarminDate `json:"calendarDate"`
	AverageSleepRespiration float64    `json:"avgSleepRespiration"`
}

type StressEntry struct {
	Date  GarminDate  `json:"calendarDate"`
	Value StressValue `json:"values"`
}

type StressValue struct {
	HighStressDurationSeconds   int `json:"highStressDuration"`
	LowStressDurationSeconds    int `json:"lowStressDuration"`
	OverallStressLevel          int `json:"overallStressLevel"`
	RestStressDurationSeconds   int `json:"restStressDuration"`
	MediumStressDurationSeconds int `json:"mediumStressDuration"`
}

type BodyBatteryEntry struct {
	Date  GarminDate       `json:"calendarDate"`
	Value BodyBatteryValue `json:"values"`
}

type BodyBatteryValue struct {
	LowBodyBattery  int `json:"lowBodyBattery"`
	HighBodyBattery int `json:"highBodyBattery"`
}

const GarminAPI = "https://connect.garmin.com/gc-api"

type GarminAPIEndpoint string

const (
	BodyBatteryURL GarminAPIEndpoint = "/usersummary-service/stats/bodybattery/daily"
	RespirationURL GarminAPIEndpoint = "/usersummary-service/stats/respiration/daily"
	StressURL      GarminAPIEndpoint = "/usersummary-service/stats/stress/daily"
	SleepURL       GarminAPIEndpoint = "/sleep-service/stats/sleep/daily"
)

type GarminDate struct {
	time.Time
}

const garminDateLayout = "2006-01-02"

func (d *GarminDate) UnmarshalJSON(b []byte) error {
	s := string(b)
	s = s[1 : len(s)-1]
	if s == "" {
		d.Time = time.Time{}
		return nil
	}
	parsed, err := time.Parse(garminDateLayout, s)
	if err != nil {
		return err
	}
	d.Time = parsed
	return nil
}

func (d GarminDate) MarshalJSON() ([]byte, error) {
	if d.IsZero() {
		return []byte("null"), nil
	}
	return []byte(`"` + d.Format(garminDateLayout) + `"`), nil
}

/*
go run garmin_backfill.go bodybattery from to --dry-run
- always provide from to for simplicify
- <from> is the furthest point back in time
- <to> is when to stop
- option to dry run
*/
func main() {
	const curlPath = "./request/curl.txt"
	curlContents, err := os.ReadFile(curlPath)
	if err != nil {
		log.Fatal(err)
	}

	err = godotenv.Load("../../.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	intervalsAthleteID := os.Getenv("INTERVALS_ATHLETE_ID")
	intervalsApiKey := os.Getenv("INTERVALS_API_KEY")
	if intervalsAthleteID == "" || intervalsApiKey == "" {
		log.Fatal("INTERVALS_ATHLETE_ID or INTERVALS_API_KEY is not set")
	}

	csrf, cookie, err := getGarminHeaders(curlContents)
	if err != nil {
		log.Fatal(err)
	}

	fromDateStr := "2026-01-01"
	toDateStr := "2026-03-01"

	records := map[GarminDate]intervals.WellnessRecord{}

	// accumulate all wellness data before updating in intervals
	records, err = getBodyBatteryData(csrf, cookie, fromDateStr, toDateStr, records)
	if err != nil {
		log.Fatal(err)
	}

	records, err = getRespirationData(csrf, cookie, fromDateStr, toDateStr, records)
	if err != nil {
		log.Fatal(err)
	}

	records, err = getStressData(csrf, cookie, fromDateStr, toDateStr, records)
	if err != nil {
		log.Fatal(err)
	}

	// TODO weight - needs some thought. there's min/max, latest weight. weights appear to be in grams?

	// some of these attributes require custom wellness attributes
	// TODO sleep

	// collect wellness records to bulk update
	var wellness []intervals.WellnessRecord
	for _, w := range records {
		wellness = append(wellness, w)
	}

	intervalsClient := intervals.NewIntervalsClient(intervals.APIURL, intervalsApiKey, intervalsAthleteID)
	err = intervalsClient.BulkUpdateWellnessRecord(wellness)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("done")
}

// garmingStressToIntervalsStress maps a numerical garmin overall stress score to an intervals stress value
func garmingStressToIntervalsStress(stress int) intervals.StressLevel {
	if stress <= 25 {
		return intervals.LowStress
	} else if stress > 25 && stress <= 50 {
		return intervals.AvgStress
	} else if stress > 50 && stress <= 75 {
		return intervals.HighStress
	} else {
		return intervals.ExtremeStress
	}
}

func getStressData(
	csrf string,
	cookie string,
	fromDateStr string,
	toDateStr string,
	records map[GarminDate]intervals.WellnessRecord,
) (map[GarminDate]intervals.WellnessRecord, error) {
	urls, err := buildGarminURLs(StressURL, fromDateStr, toDateStr)
	if err != nil {
		log.Fatal(err)
	}

	var request *http.Request
	client := &http.Client{}
	for _, url := range urls {
		log.Printf("[stress] fetching %s...\n", url)
		request, err = http.NewRequest("GET", url, nil)
		if err != nil {
			log.Fatal(err)
		}

		request.Header.Set("connect-csrf-token", csrf)
		request.Header.Set("cookie", cookie)
		resp, err := client.Do(request)
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()
		bodyText, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}

		var stress []StressEntry
		err = json.Unmarshal(bodyText, &stress)
		if err != nil {
			log.Fatal(err)
		}

		for _, b := range stress {
			if _, exists := records[b.Date]; exists {
				record := records[b.Date]
				record.LowStressSeconds = ptr.Int(b.Value.LowStressDurationSeconds)
				record.MediumStressSeconds = ptr.Int(b.Value.MediumStressDurationSeconds)
				record.HighStressSeconds = ptr.Int(b.Value.HighStressDurationSeconds)
				record.RestStressSeconds = ptr.Int(b.Value.RestStressDurationSeconds)
				overall := garmingStressToIntervalsStress(b.Value.OverallStressLevel)
				record.Stress = &overall
				records[b.Date] = record
			} else {
				overall := garmingStressToIntervalsStress(b.Value.OverallStressLevel)
				records[b.Date] = intervals.WellnessRecord{
					ID:                  intervals.WellnessRecordID(b.Date.Format("2006-01-02")),
					LowStressSeconds:    ptr.Int(b.Value.LowStressDurationSeconds),
					MediumStressSeconds: ptr.Int(b.Value.MediumStressDurationSeconds),
					HighStressSeconds:   ptr.Int(b.Value.HighStressDurationSeconds),
					RestStressSeconds:   ptr.Int(b.Value.RestStressDurationSeconds),
					Stress:              &overall,
				}
			}
		}
	}

	return records, nil
}

// getBodyBatteryData will fetch all respiration data for the specified date range
// and return an updated map of wellness records
func getRespirationData(
	csrf string,
	cookie string,
	fromDateStr string,
	toDateStr string,
	records map[GarminDate]intervals.WellnessRecord,
) (map[GarminDate]intervals.WellnessRecord, error) {
	urls, err := buildGarminURLs(RespirationURL, fromDateStr, toDateStr)
	if err != nil {
		log.Fatal(err)
	}

	var request *http.Request
	client := &http.Client{}
	for _, url := range urls {
		log.Printf("[respiration] fetching %s...\n", url)
		request, err = http.NewRequest("GET", url, nil)
		if err != nil {
			log.Fatal(err)
		}

		request.Header.Set("connect-csrf-token", csrf)
		request.Header.Set("cookie", cookie)
		resp, err := client.Do(request)
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()
		bodyText, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}

		var respiration []RespirationEntry
		err = json.Unmarshal(bodyText, &respiration)
		if err != nil {
			log.Fatal(err)
		}

		for _, b := range respiration {
			if _, exists := records[b.Date]; exists {
				record := records[b.Date]
				record.Respiration = ptr.Float(b.AverageSleepRespiration)
				records[b.Date] = record
			} else {
				records[b.Date] = intervals.WellnessRecord{
					ID:          intervals.WellnessRecordID(b.Date.Format("2006-01-02")),
					Respiration: ptr.Float(b.AverageSleepRespiration),
				}
			}
		}
	}

	return records, nil
}

// getBodyBatteryData will fetch all body battery data for the specified date range
// and return an updated map of wellness records
func getBodyBatteryData(
	csrf string,
	cookie string,
	fromDateStr string,
	toDateStr string,
	records map[GarminDate]intervals.WellnessRecord,
) (map[GarminDate]intervals.WellnessRecord, error) {
	urls, err := buildGarminURLs(BodyBatteryURL, fromDateStr, toDateStr)
	if err != nil {
		log.Fatal(err)
	}

	var request *http.Request
	client := &http.Client{}
	for _, url := range urls {
		log.Printf("[body battery] fetching %s...\n", url)
		request, err = http.NewRequest("GET", url, nil)
		if err != nil {
			log.Fatal(err)
		}

		request.Header.Set("connect-csrf-token", csrf)
		request.Header.Set("cookie", cookie)
		resp, err := client.Do(request)
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()
		bodyText, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}

		var battery []BodyBatteryEntry
		err = json.Unmarshal(bodyText, &battery)
		if err != nil {
			log.Fatal(err)
		}

		for _, b := range battery {
			if _, exists := records[b.Date]; exists {
				record := records[b.Date]
				record.BodyBatteryMin = ptr.Int(b.Value.LowBodyBattery)
				record.BodyBatterMax = ptr.Int(b.Value.HighBodyBattery)
				records[b.Date] = record
			} else {
				records[b.Date] = intervals.WellnessRecord{
					ID:             intervals.WellnessRecordID(b.Date.Format("2006-01-02")),
					BodyBatteryMin: ptr.Int(b.Value.LowBodyBattery),
					BodyBatterMax:  ptr.Int(b.Value.HighBodyBattery),
				}
			}
		}
	}

	return records, nil
}

// getGarminHeaders pulls just the minimum required values from a garmin connect cURL request
func getGarminHeaders(fileContents []byte) (csrfToken string, cookie string, err error) {
	lines := strings.SplitSeq(string(fileContents), "\n")
	for line := range lines {
		line = strings.TrimSpace(line)

		// Header form: -H 'connect-csrf-token: <value>' \
		if strings.HasPrefix(line, "-H 'connect-csrf-token:") {
			line = strings.TrimPrefix(line, "-H ")
			line = strings.Trim(line, `'" \`)
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				csrfToken = strings.TrimSpace(parts[1])
			}
		}

		if after, ok := strings.CutPrefix(line, "-b "); ok {
			cookie = after
			cookie = strings.Trim(cookie, `'" \`)
		}
	}

	if csrfToken == "" || cookie == "" {
		return "", "", errors.New("failed to find csrf token or cookie")
	}

	return csrfToken, cookie, nil
}

// buildGarminURLs will build the Garmin API URL for a specific date range
// note: the Garmin API only allows for windows of 28 days
func buildGarminURLs(endpoint GarminAPIEndpoint, startStr string, endStr string) ([]string, error) {
	const layout = "2006-01-02"
	start, err := time.Parse(layout, startStr)
	if err != nil {
		return nil, fmt.Errorf("invalid start date %q: %w", startStr, err)
	}

	end, err := time.Parse(layout, endStr)
	if err != nil {
		return nil, fmt.Errorf("invalid end date %q: %w", endStr, err)
	}

	today := time.Now().Truncate(24 * time.Hour)
	if start.After(today) {
		return nil, fmt.Errorf("start date %s cannot be after today (%s)", startStr, today.Format(layout))
	}

	if end.After(today) {
		return nil, fmt.Errorf("end date %s cannot be after today (%s)", endStr, today.Format(layout))
	}

	if start.Equal(end) {
		return nil, fmt.Errorf("start date %s is the same as end date %s", startStr, endStr)
	}

	// garmin only accepts date ranges of 28 days
	var urls []string
	for curStart := start; !curStart.After(end); {
		curEnd := curStart.AddDate(0, 0, 27)
		if curEnd.After(end) {
			curEnd = end
		}

		url := fmt.Sprintf(
			"%s%s/%s/%s",
			GarminAPI,
			endpoint,
			curStart.Format(layout),
			curEnd.Format(layout),
		)

		urls = append(urls, url)
		curStart = curEnd.AddDate(0, 0, 1)
	}

	return urls, nil
}
