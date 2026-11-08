package garmin

import (
	"encoding/json"
	"fmt"
	"intervals-functions/utils/connect"
	"intervals-functions/utils/format"
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

type SleepNeed struct {
	BaselineMinutes int `json:"baseline"`
	ActualMinutes   int `json:"actual"`
}

type OverallSleepScore struct {
	Value     int                        `json:"value"`
	Qaulifier connect.GarminSleepQuality `json:"qualifierKey"`
}

type SleepScores struct {
	Overall OverallSleepScore `json:"overall"`
}

type DailySleepResponse struct {
	CalendarDate       format.GarminDate `json:"calendarDate"`
	SleepTimeSeconds   int               `json:"sleepTimeSeconds"`
	AverageRespiration float64           `json:"averageRespirationValue"`
	DeepSleepSeconds   int               `json:"deepSleepSeconds"`
	LightSleepSeconds  int               `json:"lightSleepSeconds"`
	RemSleepSeconds    int               `json:"remSleepSeconds"`
	AwakeSleepSeconds  int               `json:"awakeSleepSeconds"`
	SleepNeed          SleepNeed         `json:"sleepNeed"`
	AverageSpo2        float64           `json:"averageSpO2Value"`
	AverageHeartRate   float64           `json:"avgHeartRate"`
	SleepScores        SleepScores       `json:"sleepScores"`
}

type DailySummaryData struct {
	CalendarDate format.GarminDate `json:"calendarDate"`
	BodyBattery  BodyBattery       `json:"bodybattery"`
	Stress       Stress            `json:"stress"`
}

type BodyBattery struct {
	MinValue int `json:"minValue"`
	MaxValue int `json:"maxValue"`
}

type Stress struct {
	AverageLevel                 int `json:"avgLevel"`
	TotalDurationInMillis        int `json:"totalDurationInMillis"`
	RestDurationInMillis         int `json:"restDurationInMillis"`
	LowStressDurationInMillis    int `json:"lowStressDurationInMillis"`
	MediumStressDurationInMillis int `json:"mediumStressDurationInMillis"`
	HighStressDurationInMillis   int `json:"highStressDurationInMillis"`
}

type DailySummaryResponse struct {
	Data []DailySummaryData `json:"data"`
}

type LatestWeight struct {
	CalendarDate      format.GarminDate `json:"calendarDate"`
	WeightInGrams     float64           `json:"weight"`
	BodyFat           float64           `json:"bodyFat"`
	MuscleMassInGrams float64           `json:"muscleMass"`
}

type DailyWeightSummary struct {
	LatestWeight LatestWeight `json:"latestWeight"`
}

type WeightResponse struct {
	DailyWeightSummaries []DailyWeightSummary `json:"dailyWeightSummaries"`
}

// GetDailySleepSummariesByDate retrieves daily sleep data for a date range from the Garmin API.
// documentation: https://developer.garmin.com/gc-developer-program/personal/swagger-docs/
func (c GarminClient) GetDailySleepSummariesByDate(startDate time.Time, endDate time.Time) ([]DailySleepResponse, error) {
	query := url.Values{}
	query.Set("startDate", startDate.Format("2006-01-02"))
	query.Set("endDate", endDate.Format("2006-01-02"))
	requestURL := garminAPIURL + "/sleep-service/sleep/dailySleepSummariesByDate?" + query.Encode()

	dailySleep, err := getJSON[[]DailySleepResponse](requestURL, c.token, "daily sleep")
	if err != nil {
		return nil, fmt.Errorf("get daily sleep: %w", err)
	}

	return dailySleep, nil
}

// GetDailySummary retrieves daily summaries for a date range.
// documentation: https://developer.garmin.com/gc-developer-program/personal/swagger-docs/
func (c GarminClient) GetDailySummary(startDate time.Time, endDate time.Time) ([]DailySummaryData, error) {
	query := url.Values{}
	query.Set("startDate", startDate.Format("2006-01-02"))
	query.Set("endDate", endDate.Format("2006-01-02"))
	requestURL := garminAPIURL + "/usersummary-service/v2/dailysummaries" + "?" + query.Encode()

	dailySummary, err := getJSON[DailySummaryResponse](requestURL, c.token, "daily summary")
	if err != nil {
		return []DailySummaryData{}, fmt.Errorf("get daily summary: %w", err)
	}

	return dailySummary.Data, nil
}

// GetWeight retrieves weight data for a date range.
// documentation: https://developer.garmin.com/gc-developer-program/personal/swagger-docs/
func (c GarminClient) GetWeight(startDate time.Time, endDate time.Time) ([]DailyWeightSummary, error) {
	requestURL := dateRangeURL("/weight-service/weight/range", startDate, endDate)

	weight, err := getJSON[WeightResponse](requestURL, c.token, "weight")
	if err != nil {
		return []DailyWeightSummary{}, fmt.Errorf("get weight: %w", err)
	}

	return weight.DailyWeightSummaries, nil
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
