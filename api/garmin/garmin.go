package garmin

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const garminTokenURL = "https://personal.connectapi.garmin.com/personalapitoken-service/personal-api/personal-api-key/token"
const garminAPIURL = "https://personal.connectapi.garmin.com"

type GarminClient struct {
	token string
}

func NewGarminClient(apiKey string) (GarminClient, error) {
	token, err := requestToken(apiKey)
	if err != nil {
		return GarminClient{}, fmt.Errorf("create Garmin client: %w", err)
	}

	return GarminClient{token: token}, nil
}

type DailySleepResponse struct {
	CalendarDate       string `json:"calendarDate"`
	SleepTimeSeconds   int    `json:"sleepTimeSeconds"`
	AverageRespiration int    `json:"averageRespirationValue"`
	DeepSleepSeconds   int    `json:"deepSleepSeconds"`
	LightSleepSeconds  int    `json:"lightSleepSeconds"`
	RemSleepSeconds    int    `json:"remSleepSeconds"`
	AwakeSleepSeconds  int    `json:"awakeSleepSeconds"`
	// TODO investigate sleep need shape
}

type DailyBodyBatteryResponse []struct {
	// TODO not sure this is right or if we want to use another endpoint
	BodyBatteryValuesArray []struct {
		BodyBatteryLevel int `json:"bodyBatteryLevel"`
	} `json:"bodyBatteryValuesArray"`
}

type DailyStressResponse []struct {
	// TODO need to investigate - not documented response
	Values struct {
		OverallStressLevel int `json:"overallStressLevel"`
	} `json:"values"`
}

type WeightResponse struct {
	// TODO this response doesn't look right
	DailyWeightSummaries []struct {
		LatestWeight struct {
			Weight     float64 `json:"weight"`
			BodyFat    float64 `json:"bodyFat"`
			MuscleMass float64 `json:"muscleMass"`
		} `json:"latestWeight"`
	} `json:"dailyWeightSummaries"`
}

// GetDailySleep retrieves the daily sleep data for a specific date from the Garmin API.
// documentation: https://developer.garmin.com/gc-developer-program/personal/swagger-docs/
func (c GarminClient) GetDailySleep(date time.Time) (DailySleepResponse, error) {
	query := url.Values{}
	query.Set("date", date.Format("2006-01-02"))
	requestURL := garminAPIURL + "/sleep-service/sleep/dailySleep" + "?" + query.Encode()

	dailySleep, err := getJSON[DailySleepResponse](requestURL, c.token, "daily sleep")
	if err != nil {
		return DailySleepResponse{}, fmt.Errorf("get daily sleep: %w", err)
	}

	return dailySleep, nil
}

// GetDailyBodyBattery retrieves body battery reports for a specific date.
// documentation: https://developer.garmin.com/gc-developer-program/personal/swagger-docs/
func (c GarminClient) GetDailyBodyBattery(startDate time.Time, endDate time.Time) (DailyBodyBatteryResponse, error) {
	requestURL := dateRangeURL("/wellness-service/wellness/bodyBattery/reports/daily", startDate, endDate)

	bodyBattery, err := getJSON[DailyBodyBatteryResponse](requestURL, c.token, "daily body battery")
	if err != nil {
		return nil, fmt.Errorf("get daily body battery: %w", err)
	}

	return bodyBattery, nil
}

// GetDailyStress retrieves stress reports for a date range.
// documentation: https://developer.garmin.com/gc-developer-program/personal/swagger-docs/
func (c GarminClient) GetDailyStress(startDate time.Time, endDate time.Time) (DailyStressResponse, error) {
	requestURL := dateRangeURL("/usersummary-service/stats/stress/daily", startDate, endDate)

	stress, err := getJSON[DailyStressResponse](requestURL, c.token, "daily stress")
	if err != nil {
		return nil, fmt.Errorf("get daily stress: %w", err)
	}

	return stress, nil
}

// GetWeight retrieves weight data for a date range.
// documentation: https://developer.garmin.com/gc-developer-program/personal/swagger-docs/
func (c GarminClient) GetWeight(startDate time.Time, endDate time.Time) (WeightResponse, error) {
	requestURL := dateRangeURL("/weight-service/weight/all", startDate, endDate)

	weight, err := getJSON[WeightResponse](requestURL, c.token, "weight")
	if err != nil {
		return WeightResponse{}, fmt.Errorf("get weight: %w", err)
	}

	return weight, nil
}

// dateRangeURL constructs a URL for Garmin API endpoints that require a start and end date.
func dateRangeURL(endpoint string, startDate time.Time, endDate time.Time) string {
	const dateLayout = "2006-01-02"
	return garminAPIURL + endpoint + "/" + startDate.Format(dateLayout) + "/" + endDate.Format(dateLayout)
}

// getJSON is a generic function to make a GET request to the Garmin API and unmarshal the JSON response into the specified type T.
func getJSON[T any](requestURL, apiToken, resource string) (T, error) {
	var result T
	resp, err := get(requestURL, apiToken)
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return result, fmt.Errorf("Garmin %s request returned status %s", resource, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return result, fmt.Errorf("read Garmin %s response: %w", resource, err)
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return result, fmt.Errorf("unmarshal Garmin %s response: %w", resource, err)
	}

	return result, nil
}

// requestToken requests a Garmin token using a personal API key
func requestToken(apiKey string) (string, error) {
	req, err := http.NewRequest(http.MethodPost, garminTokenURL, nil)
	if err != nil {
		return "", fmt.Errorf("create Garmin token request: %w", err)
	}
	req.Header.Set("Authorization", "ApiKey "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send Garmin token request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read Garmin token response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("Garmin token request returned status %s", resp.Status)
	}

	return string(body), nil
}

// get is a generic function to make a GET request to the Garmin API
func get(requestURL, apiToken string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create Garmin API request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiToken)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send Garmin API request: %w", err)
	}
	return resp, nil
}
