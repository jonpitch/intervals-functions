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
	SleepScore            float64            `json:"sleepScore"`
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
	Headers     map[string]string
	FromDateStr string
	ToDateStr   string
}

// NewGarminRequest copies parsed cURL headers and applies defaults needed for Garmin API requests.
func NewGarminRequest(headers map[string]string, fromDateStr string, toDateString string) GarminRequest {
	h := make(map[string]string, len(headers)+1)
	for k, v := range headers {
		h[k] = v
	}
	return GarminRequest{
		Headers:     h,
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
	dryRun := flag.Bool("dry-run", false, "Do not persist data to intervals.icu, just print")
	flag.Parse()

	args := flag.Args()
	if len(args) != 1 {
		log.Fatal("usage: garmin_backfill -from=X -to=Y -athleteId=A -apiKey=B --dry-run <garmin metrics>")
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

	headers, err := getGarminHeaders(curlContents)
	if err != nil {
		log.Fatal(err)
	}

	records := map[GarminDate]intervals.WellnessRecord{}

	// accumulate all wellness data before updating in intervals
	garminRequest := NewGarminRequest(headers, *fromDateStr, *toDateStr)

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
		records, err = getGarminOveralAndIndividualData(
			garminRequest,
			WeightURL,
			records,
			func(resp WeightResponse) []WeightSummary { return resp.DailyWeightSummaries },
			garminWeightAccumulator,
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
	for _, url := range urls {
		log.Printf("[%s] fetching %s...\n", garminApiToLabel(endpoint), url)
		bodyText, err := garminFetchGET(url, r)
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
	for _, url := range urls {
		log.Printf("[%s] fetching %s...\n", garminApiToLabel(endpoint), url)
		bodyText, err := garminFetchGET(url, r)
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

// garminWeightAccumulator converts WeightSummary records to intervals.WellnessRecord and accumulates
// them on the provided map
func garminWeightAccumulator(
	weight []WeightSummary,
	records map[GarminDate]intervals.WellnessRecord,
) map[GarminDate]intervals.WellnessRecord {
	for _, w := range weight {
		weightInGrams := w.LatestWeight.Weight
		weight := weightInGrams / 1000
		if record, exists := records[w.SummaryDate]; exists {
			record.Weight = ptr.Float(weight)
			records[w.SummaryDate] = record
		} else {
			records[w.SummaryDate] = intervals.WellnessRecord{
				ID:     intervals.WellnessRecordID(w.SummaryDate.Format("2006-01-02")),
				Weight: ptr.CoalesceFloat(weight),
			}
		}
	}

	return records
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
				Respiration: ptr.CoalesceFloat(b.AverageSleepRespiration),
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
		if record, exists := records[s.Date]; exists {
			record.SleepScore = ptr.Float(s.Values.SleepScore)
			record.SleepSeconds = ptr.Int(s.Values.TotalSleepTimeSeconds)
			record.SleepQuality = garminSleepQualityToIntervalsSleepQuality(s.Values.SleepQuality)
			record.HrvRmssd = ptr.CoalesceFloat(s.Values.AverageOvernightHrv)
			record.RestingHr = ptr.Int(s.Values.RestingHeartRate)
			record.OxygenSaturation = ptr.CoalesceFloat(s.Values.Spo2)
			record.SleepNeedMinutes = ptr.Int(s.Values.SleepNeedMinutes)
			record.SleepRemTimeSeconds = ptr.Int(s.Values.RemTimeSeconds)
			record.SleepDeepTimeSeconds = ptr.Int(s.Values.DeepTimeSeconds)
			record.SleepLightTimeSeconds = ptr.Int(s.Values.LightTimeSeconds)
			record.SleepAwakeTimeSeconds = ptr.Int(s.Values.AwakeTimeSeconds)
			records[s.Date] = record
		} else {
			records[s.Date] = intervals.WellnessRecord{
				ID:                    intervals.WellnessRecordID(s.Date.Format("2006-01-02")),
				SleepScore:            ptr.Float(s.Values.SleepScore),
				SleepSeconds:          ptr.Int(s.Values.TotalSleepTimeSeconds),
				SleepQuality:          garminSleepQualityToIntervalsSleepQuality(s.Values.SleepQuality),
				HrvRmssd:              ptr.CoalesceFloat(s.Values.AverageOvernightHrv),
				RestingHr:             ptr.CoalesceInt(s.Values.RestingHeartRate),
				OxygenSaturation:      ptr.CoalesceFloat(s.Values.Spo2),
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

// garminFetchGET performs a GET with the headers from r (including defaults from NewGarminRequest).
func garminFetchGET(url string, r GarminRequest) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range r.Headers {
		req.Header.Set(k, v)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// getGarminHeaders parses -H 'name: value' lines and -b cookie from a garmin connect cURL request.
// Header names are canonicalized; the cookie from -b is stored under the Cookie key.
func getGarminHeaders(fileContents []byte) (map[string]string, error) {
	headers := make(map[string]string)
	lines := strings.SplitSeq(string(fileContents), "\n")
	for line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "-H ") {
			rest := strings.TrimPrefix(line, "-H ")
			rest = strings.Trim(rest, `'" \`)
			parts := strings.SplitN(rest, ":", 2)
			if len(parts) != 2 {
				continue
			}
			name := http.CanonicalHeaderKey(strings.TrimSpace(parts[0]))
			value := strings.TrimSpace(parts[1])
			headers[name] = value
		}

		if after, ok := strings.CutPrefix(line, "-b "); ok {
			cookie := strings.Trim(after, `'" \`)
			headers[http.CanonicalHeaderKey("cookie")] = cookie
		}
	}

	csrf := headers[http.CanonicalHeaderKey("connect-csrf-token")]
	cookie := headers[http.CanonicalHeaderKey("cookie")]
	if csrf == "" || cookie == "" {
		return nil, errors.New("failed to find csrf token or cookie")
	}

	return headers, nil
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
func garminSleepQualityToIntervalsSleepQuality(q GarminSleepQuality) *intervals.SleepQuality {
	var quality *intervals.SleepQuality
	switch q {
	case Excellent:
		quality := intervals.GreatSleepQuality
		return &quality
	case Good:
		quality := intervals.GoodSleepQuality
		return &quality
	case Fair:
		quality := intervals.AverageSleepQuality
		return &quality
	case Poor:
		quality := intervals.PoorSleepQuality
		return &quality
	default:
		return quality
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
