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

type WellnessRecord struct {
	ID            WellnessRecordID `json:"id"`
	KCalConsumed  int              `json:"kcalConsumed"`
	Carbohydrates int              `json:"carbohydrates"`
	Protein       int              `json:"protein"`
	Fat           int              `json:"fatTotal"`
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
