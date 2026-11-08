package garmin

import (
	"encoding/json"
	"testing"
)

func TestDailySleepResponseUnmarshalsFloatAverageHeartRate(t *testing.T) {
	var response DailySleepResponse
	if err := json.Unmarshal([]byte(`{"avgHeartRate":45.0}`), &response); err != nil {
		t.Fatalf("unmarshal daily sleep response: %v", err)
	}

	if response.AverageHeartRate != 45 {
		t.Errorf("AverageHeartRate = %v, want 45", response.AverageHeartRate)
	}
}
