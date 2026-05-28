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
