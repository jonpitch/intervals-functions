package intervals

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

var APIURL = "https://intervals.icu/api/v1"

// documentation: https://forum.intervals.icu/t/api-access-to-intervals-icu/609
type IntervalsClient struct {
	url       string
	apiKey    string
	athleteID string
}

func NewIntervalsClient(url string, apiKey string, athleteID string) IntervalsClient {
	return IntervalsClient{url, apiKey, athleteID}
}

type WellnessRecordID string

// Optional fields use omitempty so nil pointers are omitted from JSON. Without it, they marshal as
// null and the intervals.icu API treats null as clearing the attribute (problematic for partial updates).
type WellnessRecord struct {
	ID               WellnessRecordID `json:"id"`
	KCalConsumed     *float64         `json:"kcalConsumed,omitempty"`
	Carbohydrates    *float64         `json:"carbohydrates,omitempty"`
	Protein          *float64         `json:"protein,omitempty"`
	Fat              *float64         `json:"fatTotal,omitempty"`
	OxygenSaturation *float64         `json:"spO2,omitempty"`
	Respiration      *float64         `json:"respiration,omitempty"`
	Stress           *StressLevel     `json:"stress,omitempty"`
	SleepScore       *float64         `json:"sleepScore,omitempty"`
	SleepSeconds     *int             `json:"sleepSecs,omitempty"`
	SleepQuality     *SleepQuality    `json:"sleepQuality,omitempty"`
	HrvRmssd         *float64         `json:"hrv,omitempty"`
	RestingHr        *int             `json:"restingHR,omitempty"`
	Weight           *float64         `json:"weight,omitempty"` // stored in user's measurement preference (kg, lbs)

	// custom attributes
	BodyBatteryMin        *int `json:"BodyBatteryMin,omitempty"`
	BodyBatterMax         *int `json:"BodyBatteryMax,omitempty"`
	RestStressSeconds     *int `json:"StressRestSeconds,omitempty"`
	LowStressSeconds      *int `json:"StressLowSeconds,omitempty"`
	MediumStressSeconds   *int `json:"StressMediumSeconds,omitempty"`
	HighStressSeconds     *int `json:"StressHighSeconds,omitempty"`
	SleepNeedMinutes      *int `json:"SleepNeedMinutes,omitempty"`
	SleepRemTimeSeconds   *int `json:"SleepRemSeconds,omitempty"`
	SleepDeepTimeSeconds  *int `json:"SleepDeepSeconds,omitempty"`
	SleepLightTimeSeconds *int `json:"SleepLightSeconds,omitempty"`
	SleepAwakeTimeSeconds *int `json:"SleepAwakeSeconds,omitempty"`
}

type StressLevel int

const (
	LowStress     StressLevel = 1
	AvgStress     StressLevel = 2
	HighStress    StressLevel = 3
	ExtremeStress StressLevel = 4
)

type SleepQuality int

const (
	GreatSleepQuality   SleepQuality = 1
	GoodSleepQuality    SleepQuality = 2
	AverageSleepQuality SleepQuality = 3
	PoorSleepQuality    SleepQuality = 4
)

type Activity struct {
	ID                        string  `json:"id"`
	Type                      string  `json:"type"`
	Date                      string  `json:"start_date_local"`
	TrainingLoad              int     `json:"icu_training_load"`
	Atl                       float64 `json:"icu_atl"`
	Ctl                       float64 `json:"icu_ctl"`
	ElapsedTime               int     `json:"elapsed_time"`
	Name                      string  `json:"name"`
	AverageTemp               float64 `json:"average_temp"`
	MinTemp                   int     `json:"min_temp"`
	MaxTemp                   int     `json:"max_temp"`
	Rpe                       int     `json:"icu_rpe"` // user supplied rpe
	KgLifted                  float64 `json:"kg_lifted"`
	Decoupling                float64 `json:"decoupling"`
	PowerLoad                 int     `json:"power_load"`
	HrLoad                    int     `json:"hr_load"`
	PaceLoad                  int     `json:"pace_load"`
	SessionRpe                int     `json:"session_rpe"` // rpe x session load
	Distance                  float64 `json:"distance"`
	LactateThresholdHeartRate int     `json:"lthr"`
	RollingFtp                int     `json:"icu_rolling_ftp"` // eFTP
	Ftp                       int     `json:"icu_ftp"`         // user ftp

	// TODO get some concept of FTP, thresholds, etc. for comparison?
}

type Event struct {
	ID          int    `json:"id"`
	Date        string `json:"start_date_local"`
	Type        string `json:"type"`     // enum
	Category    string `json:"category"` // enum
	Name        string `json:"name"`
	Description string `json:"description"`
}

// GetWellnessRecord sends a GET request to
//
//	https://intervals.icu/api/v1/athlete/{id}/wellness/{date}
//
// api documentation: https://intervals.icu/api-docs.html#get-/api/v1/athlete/-id-/wellness/-date-
func (c IntervalsClient) GetWellnessRecord(date time.Time) (WellnessRecord, error) {
	dateStr := date.Format("2006-01-02") // ISO-8601 calendar date
	url := fmt.Sprintf(c.url+"/athlete/%s/wellness/%s", c.athleteID, dateStr)

	resp, err := get(url, c.apiKey)
	if err != nil {
		return WellnessRecord{}, fmt.Errorf("get wellness record error: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return WellnessRecord{}, fmt.Errorf("read response body failed: %w", err)
	}

	var wellness WellnessRecord
	err = json.Unmarshal(body, &wellness)
	if err != nil {
		return WellnessRecord{}, fmt.Errorf("unmarshal response body failed: %w", err)
	}

	return wellness, nil
}

// UpdateWellnessRecord sends a PUT request to
//
//	https://intervals.icu/api/v1/athlete/{id}/wellness
//
// api documentation: https://intervals.icu/api-docs.html#put-/api/v1/athlete/-id-/wellness/-date-
func (c IntervalsClient) UpdateWellnessRecord(wellness WellnessRecord) error {
	url := fmt.Sprintf(c.url+"/athlete/%s/wellness/%s", c.athleteID, string(wellness.ID))

	body, err := json.Marshal(wellness)
	if err != nil {
		return fmt.Errorf("marshal wellness payload: %w", err)
	}

	resp, err := put(url, c.apiKey, body)
	if err != nil {
		return fmt.Errorf("update wellness record error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("intervals.icu API returned status %s", resp.Status)
	}

	return nil
}

// BulkUpdateWellnessRecord allows for the update of multiple wellness record updates in 1 request
//
//	https://intervals.icu/api/v1/athlete/{id}/wellness-bulk
//
// api documentation: https://intervals.icu/api-docs.html#put-/api/v1/athlete/-id-/wellness-bulk
func (c IntervalsClient) BulkUpdateWellnessRecord(wellness []WellnessRecord) error {
	url := fmt.Sprintf(c.url+"/athlete/%s/wellness-bulk", c.athleteID)

	body, err := json.Marshal(wellness)
	if err != nil {
		return fmt.Errorf("marshal wellness payload: %w", err)
	}

	resp, err := put(url, c.apiKey, body)
	if err != nil {
		return fmt.Errorf("bulk update wellness records error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("intervals.icu API returned status %s", resp.Status)
	}

	return nil
}

// https://intervals.icu/api-docs.html#get-/api/v1/athlete/-id-/wellness-ext-
func (c IntervalsClient) ListWellnessRecordsForDateRange(
	oldest time.Time,
	newest time.Time,
) ([]WellnessRecord, error) {
	oldestStr := oldest.Format("2006-01-02")
	newestStr := newest.Format("2006-01-02")
	url := fmt.Sprintf(
		c.url+"/athlete/%s/wellness?oldest=%s&newest=%s",
		c.athleteID,
		oldestStr,
		newestStr,
	)

	resp, err := get(url, c.apiKey)
	if err != nil {
		return []WellnessRecord{}, fmt.Errorf("list wellness records error: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return []WellnessRecord{}, fmt.Errorf("read response body failed: %w", err)
	}

	var wellness []WellnessRecord
	err = json.Unmarshal(body, &wellness)
	if err != nil {
		return []WellnessRecord{}, fmt.Errorf("unmarshal wellness records response body failed: %w", err)
	}

	return wellness, nil
}

// https://intervals.icu/api-docs.html#get-/api/v1/athlete/-id-/activities
func (c IntervalsClient) ListActivitiesForDateRange(
	oldest time.Time,
	newest time.Time,
) ([]Activity, error) {
	oldestStr := oldest.Format("2006-01-02")
	newestStr := newest.Format("2006-01-02")
	url := fmt.Sprintf(
		c.url+"/athlete/%s/activities?oldest=%s&newest=%s",
		c.athleteID,
		oldestStr,
		newestStr,
	)

	resp, err := get(url, c.apiKey)
	if err != nil {
		return []Activity{}, fmt.Errorf("list activities error: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return []Activity{}, fmt.Errorf("read activities body failed: %w", err)
	}

	var activities []Activity
	err = json.Unmarshal(body, &activities)
	if err != nil {
		return []Activity{}, fmt.Errorf("unmarshal activities response body failed: %w", err)
	}

	return activities, nil
}

// https://intervals.icu/api-docs.html#get-/api/v1/athlete/-id-/events-format-
func (c IntervalsClient) ListEventsForDateRange(
	oldest time.Time,
	newest time.Time,
) ([]Event, error) {
	oldestStr := oldest.Format("2006-01-02")
	newestStr := newest.Format("2006-01-02")
	url := fmt.Sprintf(
		c.url+"/athlete/%s/events?oldest=%s&newest=%s&category=%s",
		c.athleteID,
		oldestStr,
		newestStr,
		// TODO params?
		"NOTE,RACE_A,RACE_B,RACE_C,SEASON_START,HOLIDAY,SICK,INJURED",
	)

	resp, err := get(url, c.apiKey)
	if err != nil {
		return []Event{}, fmt.Errorf("list events error: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return []Event{}, fmt.Errorf("read events body failed: %w", err)
	}

	var events []Event
	err = json.Unmarshal(body, &events)
	if err != nil {
		return []Event{}, fmt.Errorf("unmarshal events response body failed: %w", err)
	}

	return events, nil
}

// get sends a GET request to the given url
func get(url string, apiKey string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth("API_KEY", apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// put sends a PUT request to the given url with the given api key and body
func put(url string, apiKey string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth("API_KEY", apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
