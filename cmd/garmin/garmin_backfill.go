package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	intervals "intervals-functions/api"
	"intervals-functions/utils/ptr"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type WeightResponse struct {
	DailyWeightSummaries []WeightSummary `json:"dailyWeightSummaries"`
}

// people can measure themselves multiple times throughout the day.
// intervals.icu doesn't support multiple weights, and keeps the most recent measurement.
// we'll use latestWeight for consistency rather than choosing min/max or some other measurement.
type WeightSummary struct {
	SummaryDate  GarminDate   `json:"summaryDate"`
	LatestWeight LatestWeight `json:"latestWeight"`
}

type LatestWeight struct {
	Weight float64 `json:"weight"`
}

type WeightUnits int

const (
	Metric   WeightUnits = iota
	Imperial             = iota
)

type SleepResponse struct {
	IndividualStats []SleepEntry `json:"individualStats"`
}

type SleepEntry struct {
	Date   GarminDate `json:"calendarDate"`
	Values SleepValue `json:"values"`
}

type GarminSleepQuality string

const (
	Excellent GarminSleepQuality = "EXCELLENT"
	Good      GarminSleepQuality = "GOOD"
	Fair      GarminSleepQuality = "FAIR"
	Poor      GarminSleepQuality = "POOR"
)

type SleepValue struct {
	RemTimeSeconds        int                `json:"remTime"`
	RestingHeartRate      int                `json:"restingHeartRate"`
	TotalSleepTimeSeconds int                `json:"totalSleepTimeInSeconds"`
	DeepTimeSeconds       int                `json:"deepTime"`
	AwakeTimeSeconds      int                `json:"awakeTime"`
	SleepQuality          GarminSleepQuality `json:"sleepScoreQuality"`
	Spo2                  float64            `json:"spO2"`
	LightTimeSeconds      int                `json:"lightTime"`
	AverageOvernightHrv   float64            `json:"avgOvernightHrv"`
	SleepNeedMinutes      int                `json:"sleepNeed"`
	SleepScore            int                `json:"sleepScore"`
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
	WeightURL      GarminAPIEndpoint = "/weight-service/weight/range"
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

type GarminOverallAndIndividualResponseData interface {
	SleepResponse | WeightResponse
}

type GarminArrayResponseData interface {
	BodyBatteryEntry | RespirationEntry | StressEntry
}

type GarminRequest struct {
	Csrf        string
	Cookie      string
	FromDateStr string
	ToDateStr   string
}

func NewGarminRequest(csrf string, cookie string, fromDateStr string, toDateString string) GarminRequest {
	return GarminRequest{
		Csrf:        csrf,
		Cookie:      cookie,
		FromDateStr: fromDateStr,
		ToDateStr:   toDateString,
	}
}

/*
backfill data from garmin connect to intervals.icu.

signature:
go run garmin_backfill.go -from=YYYY-MM-DD -to=YYYY-MM-DD -athleteId=XYZ -apiKey=ABC -metric -dry-run bodybattery,stress,respiration,sleep,weight
*/
func main() {
	const curlPath = "./request/curl.txt"
	curlContents, err := os.ReadFile(curlPath)
	if err != nil {
		log.Fatal(err)
	}

	fromDateStr := flag.String("from", "", "start date of backfill (YYYY-MM-DD format)")
	toDateStr := flag.String("to", "", "last day of backfill (YYYY-MM-DD format)")
	intervalsAthleteID := flag.String("athleteId", "", "intervals.icu athlete ID")
	intervalsApiKey := flag.String("apiKey", "", "intervals.icu API key")
	metric := flag.Bool("metric", false, "Use metric units (g, kg)")
	dryRun := flag.Bool("dry-run", false, "Do not persist data to intervals.icu, just print")
	flag.Parse()

	args := flag.Args()
	if len(args) != 1 {
		log.Fatal("usage: garmin_backfill <garmin metrics> -from=X -to=Y -athleteId=A -apiKey=B --metric --dry-run")
	}

	if intervalsApiKey == nil || intervalsAthleteID == nil {
		log.Fatal("intervals API key and athlete ID are required")
	}

	if fromDateStr == nil || toDateStr == nil {
		log.Fatal("from date and to date are required, and in the format of YYYY-MM-DD")
	}

	_, err = validDate(*fromDateStr)
	if err != nil {
		log.Fatal("from date is invalid, use YYYY-MM-DD format")
	}

	_, err = validDate(*toDateStr)
	if err != nil {
		log.Fatal("to date is invalid, use YYYY-MM-DD format")
	}

	fetchBodyBattery := false
	fetchStress := false
	fetchRespiration := false
	fetchSleep := false
	fetchWeight := false

	garminArgs := args[0]
	garminOptions := strings.Split(garminArgs, ",")
	for _, o := range garminOptions {
		switch o {
		case "bodybattery":
			fetchBodyBattery = true
		case "stress":
			fetchStress = true
		case "respiration":
			fetchRespiration = true
		case "sleep":
			fetchSleep = true
		case "weight":
			fetchWeight = true
		default:
			log.Fatalf("unknown garmin option")
		}
	}

	csrf, cookie, err := getGarminHeaders(curlContents)
	if err != nil {
		log.Fatal(err)
	}

	records := map[GarminDate]intervals.WellnessRecord{}

	// accumulate all wellness data before updating in intervals
	garminRequest := NewGarminRequest(csrf, cookie, *fromDateStr, *toDateStr)

	if fetchBodyBattery {
		records, err = getGarminArrayData(
			garminRequest,
			BodyBatteryURL,
			records,
			garminBodyBatteryAccumulator,
		)
		if err != nil {
			log.Fatal(err)
		}
	}

	if fetchRespiration {
		records, err = getGarminArrayData(
			garminRequest,
			RespirationURL,
			records,
			garminRespirationAccumulator,
		)
		if err != nil {
			log.Fatal(err)
		}
	}

	if fetchStress {
		records, err = getGarminArrayData(
			garminRequest,
			StressURL,
			records,
			garminStressAccumulator,
		)
		if err != nil {
			log.Fatal(err)
		}
	}

	if fetchSleep {
		records, err = getGarminOveralAndIndividualData(
			garminRequest,
			SleepURL,
			records,
			func(resp SleepResponse) []SleepEntry { return resp.IndividualStats },
			garminSleepAccumulator,
		)
		if err != nil {
			log.Fatal(err)
		}
	}

	if fetchWeight {
		var units WeightUnits
		if metric != nil && *metric {
			units = Metric
		} else {
			units = Imperial
		}

		records, err = getGarminOveralAndIndividualData(
			garminRequest,
			WeightURL,
			records,
			func(resp WeightResponse) []WeightSummary { return resp.DailyWeightSummaries },
			garminWeightAccumulatorWithUnits(units),
		)
		if err != nil {
			log.Fatal(err)
		}
	}

	// collect wellness records to bulk update
	var wellness []intervals.WellnessRecord
	for _, w := range records {
		wellness = append(wellness, w)
		if *dryRun {
			data, err := json.MarshalIndent(w, "", "  ")
			if err != nil {
				fmt.Printf("error marshalling wellness record: %v", err)
			}
			fmt.Println(string(data))
		}
	}

	if !*dryRun {
		intervalsClient := intervals.NewIntervalsClient(intervals.APIURL, *intervalsApiKey, *intervalsAthleteID)
		err = intervalsClient.BulkUpdateWellnessRecord(wellness)
		if err != nil {
			log.Fatal(err)
		}
	}

	log.Println("done")
}

// getGarminArrayData make requests to the Garmin API specified for the date ranges specified.
// this is meant to handle garmin API endpoints that return an array of records
func getGarminArrayData[T GarminArrayResponseData](
	r GarminRequest,
	endpoint GarminAPIEndpoint,
	wellness map[GarminDate]intervals.WellnessRecord,
	accumulate func([]T, map[GarminDate]intervals.WellnessRecord) map[GarminDate]intervals.WellnessRecord,
) (map[GarminDate]intervals.WellnessRecord, error) {
	urls, err := buildGarminURLs(endpoint, r.FromDateStr, r.ToDateStr)
	if err != nil {
		return wellness, err
	}

	var data []T
	var request *http.Request
	client := &http.Client{}
	for _, url := range urls {
		log.Printf("[%s] fetching %s...\n", garminApiToLabel(endpoint), url)
		request, err = http.NewRequest("GET", url, nil)
		if err != nil {
			return wellness, err
		}

		request.Header.Set("connect-csrf-token", r.Csrf)
		request.Header.Set("cookie", r.Cookie)
		resp, err := client.Do(request)
		if err != nil {
			return wellness, err
		}
		defer resp.Body.Close()
		bodyText, err := io.ReadAll(resp.Body)
		if err != nil {
			return wellness, err
		}

		var records []T
		err = json.Unmarshal(bodyText, &records)
		if err != nil {
			return wellness, err
		}

		data = append(data, records...)
	}

	return accumulate(data, wellness), nil
}

// getGarminOveralAndIndividualData make requests to the Garmin API specified for the date ranges specified.
// this is meant to handle garmin API endpoints that return an object with "overallStats" and "individualStats" data.
// this accumulator only handles individual stats
func getGarminOveralAndIndividualData[T GarminOverallAndIndividualResponseData, E any](
	r GarminRequest,
	endpoint GarminAPIEndpoint,
	wellness map[GarminDate]intervals.WellnessRecord,
	extract func(T) []E,
	accumulate func([]E, map[GarminDate]intervals.WellnessRecord) map[GarminDate]intervals.WellnessRecord,
) (map[GarminDate]intervals.WellnessRecord, error) {
	urls, err := buildGarminURLs(endpoint, r.FromDateStr, r.ToDateStr)
	if err != nil {
		return wellness, err
	}

	var data []E
	var request *http.Request
	client := &http.Client{}
	for _, url := range urls {
		log.Printf("[%s] fetching %s...\n", garminApiToLabel(endpoint), url)
		request, err = http.NewRequest("GET", url, nil)
		if err != nil {
			return wellness, err
		}

		request.Header.Set("connect-csrf-token", r.Csrf)
		request.Header.Set("cookie", r.Cookie)
		resp, err := client.Do(request)
		if err != nil {
			return wellness, err
		}
		defer resp.Body.Close()
		bodyText, err := io.ReadAll(resp.Body)
		if err != nil {
			return wellness, err
		}

		var payload T
		err = json.Unmarshal(bodyText, &payload)
		if err != nil {
			return wellness, err
		}

		data = append(data, extract(payload)...)
	}

	return accumulate(data, wellness), nil
}

// garminWeightAccumulatorWithUnits converts WeightSummary records to intervals.WellnessRecord and accumulates
// them on the provided map
func garminWeightAccumulatorWithUnits(units WeightUnits) func(
	[]WeightSummary,
	map[GarminDate]intervals.WellnessRecord,
) map[GarminDate]intervals.WellnessRecord {
	return func(
		weight []WeightSummary,
		records map[GarminDate]intervals.WellnessRecord,
	) map[GarminDate]intervals.WellnessRecord {
		for _, w := range weight {
			val := w.LatestWeight.Weight
			switch units {
			case Metric:
				val = val / 1000.0
			case Imperial:
				val = val / 16.0
			}

			if record, exists := records[w.SummaryDate]; exists {
				record.Weight = ptr.Float(val)
				records[w.SummaryDate] = record
			} else {
				records[w.SummaryDate] = intervals.WellnessRecord{
					ID:     intervals.WellnessRecordID(w.SummaryDate.Format("2006-01-02")),
					Weight: ptr.Float(val),
				}
			}
		}

		return records
	}
}

// garminStressAccumulator converts StressEntry records to intervals.WellnessRecord and accumulates
// them on the provided map
func garminStressAccumulator(
	stress []StressEntry,
	records map[GarminDate]intervals.WellnessRecord,
) map[GarminDate]intervals.WellnessRecord {
	for _, b := range stress {
		if record, exists := records[b.Date]; exists {
			record.LowStressSeconds = ptr.Int(b.Value.LowStressDurationSeconds)
			record.MediumStressSeconds = ptr.Int(b.Value.MediumStressDurationSeconds)
			record.HighStressSeconds = ptr.Int(b.Value.HighStressDurationSeconds)
			record.RestStressSeconds = ptr.Int(b.Value.RestStressDurationSeconds)
			overall := garminStressToIntervalsStress(b.Value.OverallStressLevel)
			record.Stress = &overall
			records[b.Date] = record
		} else {
			overall := garminStressToIntervalsStress(b.Value.OverallStressLevel)
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

	return records
}

// garminRespirationAccumulator converts RespirationEntry records to intervals.WellnessRecord and accumulates
// them on the provided map
func garminRespirationAccumulator(
	respiration []RespirationEntry,
	records map[GarminDate]intervals.WellnessRecord,
) map[GarminDate]intervals.WellnessRecord {
	for _, b := range respiration {
		if record, exists := records[b.Date]; exists {
			record.Respiration = ptr.Float(b.AverageSleepRespiration)
			records[b.Date] = record
		} else {
			records[b.Date] = intervals.WellnessRecord{
				ID:          intervals.WellnessRecordID(b.Date.Format("2006-01-02")),
				Respiration: ptr.Float(b.AverageSleepRespiration),
			}
		}
	}

	return records
}

// garminSleepAccumulator converts SleepEntry records to intervals.WellnessRecord and accumulates
// them on the provided map
func garminSleepAccumulator(
	sleep []SleepEntry,
	records map[GarminDate]intervals.WellnessRecord,
) map[GarminDate]intervals.WellnessRecord {
	for _, s := range sleep {
		quality := garminSleepQualityToIntervalsSleepQuality(s.Values.SleepQuality)
		if record, exists := records[s.Date]; exists {
			record.SleepScore = ptr.Int(s.Values.SleepScore)
			record.SleepSeconds = ptr.Int(s.Values.TotalSleepTimeSeconds)
			record.SleepQuality = &quality
			record.HrvRmssd = ptr.Float(s.Values.AverageOvernightHrv)
			record.RestingHr = ptr.Int(s.Values.RestingHeartRate)
			record.OxygenSaturation = ptr.Float(s.Values.Spo2)
			record.SleepNeedMinutes = ptr.Int(s.Values.SleepNeedMinutes)
			record.SleepRemTimeSeconds = ptr.Int(s.Values.RemTimeSeconds)
			record.SleepDeepTimeSeconds = ptr.Int(s.Values.DeepTimeSeconds)
			record.SleepLightTimeSeconds = ptr.Int(s.Values.LightTimeSeconds)
			record.SleepAwakeTimeSeconds = ptr.Int(s.Values.AwakeTimeSeconds)
			records[s.Date] = record
		} else {
			records[s.Date] = intervals.WellnessRecord{
				ID:                    intervals.WellnessRecordID(s.Date.Format("2006-01-02")),
				SleepScore:            ptr.Int(s.Values.SleepScore),
				SleepSeconds:          ptr.Int(s.Values.TotalSleepTimeSeconds),
				SleepQuality:          &quality,
				HrvRmssd:              ptr.Float(s.Values.AverageOvernightHrv),
				RestingHr:             ptr.Int(s.Values.RestingHeartRate),
				OxygenSaturation:      ptr.Float(s.Values.Spo2),
				SleepNeedMinutes:      ptr.Int(s.Values.SleepNeedMinutes),
				SleepRemTimeSeconds:   ptr.Int(s.Values.RemTimeSeconds),
				SleepDeepTimeSeconds:  ptr.Int(s.Values.DeepTimeSeconds),
				SleepLightTimeSeconds: ptr.Int(s.Values.LightTimeSeconds),
				SleepAwakeTimeSeconds: ptr.Int(s.Values.AwakeTimeSeconds),
			}
		}
	}

	return records
}

// garminBodyBatteryAccumulator converts BodyBatteryEntry records to intervals.WellnessRecord and accumulates
// them on the provided map
func garminBodyBatteryAccumulator(
	battery []BodyBatteryEntry,
	records map[GarminDate]intervals.WellnessRecord,
) map[GarminDate]intervals.WellnessRecord {
	for _, b := range battery {
		if record, exists := records[b.Date]; exists {
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

	return records
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

		if endpoint == WeightURL {
			url += "?includeAll=true"
		}

		urls = append(urls, url)
		curStart = curEnd.AddDate(0, 0, 1)
	}

	return urls, nil
}

// garminStressToIntervalsStress maps a numerical garmin overall stress score to an intervals stress value
func garminStressToIntervalsStress(stress int) intervals.StressLevel {
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

// garminSleepQualityToIntervalsSleepQuality maps a garmin sleep quality string to intervals sleep quality value
func garminSleepQualityToIntervalsSleepQuality(q GarminSleepQuality) intervals.SleepQuality {
	switch q {
	case Excellent:
		return intervals.GreatSleepQuality
	case Good:
		return intervals.GoodSleepQuality
	case Fair:
		return intervals.AverageSleepQuality
	default:
		return intervals.PoorSleepQuality
	}
}

// garminApiToLabel is a helper to print out a user-friendly name of a given garmin API endpoint
func garminApiToLabel(api GarminAPIEndpoint) string {
	switch api {
	case BodyBatteryURL:
		return "body battery"
	case RespirationURL:
		return "respiration"
	case StressURL:
		return "stress"
	case SleepURL:
		return "sleep"
	case WeightURL:
		return "weight"
	default:
		return ""
	}
}

// validDate checks the command supplied args for dates are in the correct format
func validDate(d string) (time.Time, error) {
	return time.Parse("2006-01-02", d)
}
