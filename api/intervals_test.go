package intervals

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetWellnessRecord(t *testing.T) {
	var (
		ExpectedMethod string
		ExpectedPath   string
		ExpectedAuth   string
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ExpectedMethod = r.Method
		ExpectedPath = r.URL.Path
		ExpectedAuth = r.Header.Get("Authorization")
		defer r.Body.Close()

		// Return a mock response body so we can verify unmarshalling.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(WellnessRecord{
			ID:            WellnessRecordID("888"),
			KCalConsumed:  1,
			Protein:       2,
			Carbohydrates: 3,
			Fat:           4,
		})
	}))
	defer server.Close()

	athleteID := "i1234"
	apiKey := "xyz999"
	auth := base64.StdEncoding.EncodeToString([]byte("API_KEY:" + apiKey))
	date := time.Date(2026, 2, 26, 0, 0, 0, 0, time.UTC)

	client := IntervalsClient{server.URL, apiKey, athleteID}
	body, err := client.GetWellnessRecord(date)
	assert.NoError(t, err)
	assert.Equal(t, http.MethodGet, ExpectedMethod)
	assert.Equal(t, "/athlete/"+athleteID+"/wellness/2026-02-26", ExpectedPath)
	assert.Equal(t, "Basic "+auth, ExpectedAuth)
	assert.Equal(t, WellnessRecord{
		ID:            WellnessRecordID("888"),
		KCalConsumed:  1,
		Protein:       2,
		Carbohydrates: 3,
		Fat:           4,
	}, body)
}

func TestUpdateWellnessRecord(t *testing.T) {
	var (
		ExpectedMethod string
		ExpectedPath   string
		ExpectedAuth   string
		ExpectedBody   WellnessRecord
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ExpectedMethod = r.Method
		ExpectedPath = r.URL.Path
		ExpectedAuth = r.Header.Get("Authorization")
		defer r.Body.Close()
		_ = json.NewDecoder(r.Body).Decode(&ExpectedBody)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	wellness := WellnessRecord{
		ID:            WellnessRecordID("666"),
		KCalConsumed:  100,
		Carbohydrates: 200,
		Protein:       300,
		Fat:           400,
	}

	athleteID := "i1234"
	apiKey := "xyz999"
	auth := base64.StdEncoding.EncodeToString([]byte("API_KEY:" + apiKey))

	client := IntervalsClient{server.URL, apiKey, athleteID}
	err := client.UpdateWellnessRecord(wellness)
	assert.NoError(t, err)
	assert.Equal(t, http.MethodPut, ExpectedMethod)
	assert.Equal(t, "/athlete/"+athleteID+"/wellness/666", ExpectedPath)
	assert.Equal(t, "Basic "+auth, ExpectedAuth)
	assert.Equal(t, wellness, ExpectedBody)
}
