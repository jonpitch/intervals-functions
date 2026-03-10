package main

import (
	"fmt"
	intervals "intervals-functions/api"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGarmingHeaders_NoneFound(t *testing.T) {
	curl := `curl 'some-url' \
	-H 'accept: */*' \
	-H 'user-agent: something-wild'`

	csrf, cookie, err := getGarminHeaders([]byte(curl))
	assert.Error(t, err)
	assert.Equal(t, "", csrf)
	assert.Equal(t, "", cookie)
}

func TestGarminHeaders_OneFound(t *testing.T) {
	curl := `curl 'some-url' \
	-H 'accept: */*' \
	-H 'user-agent: something-wild' \
	-H 'connect-csrf-token: abc123'`

	csrf, cookie, err := getGarminHeaders([]byte(curl))
	assert.Error(t, err)
	assert.Equal(t, "", csrf)
	assert.Equal(t, "", cookie)
}

func TestGarminHeaders(t *testing.T) {
	curl := `curl 'some-url' \
	-H 'accept: */*' \
	-H 'user-agent: something-wild' \
	-H 'connect-csrf-token: abc123' \
	-b 'a_whole_bunch_of_nonsense'`

	csrf, cookie, err := getGarminHeaders([]byte(curl))
	assert.NoError(t, err)
	assert.Equal(t, "abc123", csrf)
	assert.Equal(t, "a_whole_bunch_of_nonsense", cookie)
}

func TestGarminUrls_ErrorHandling(t *testing.T) {
	today := time.Now()
	cases := []struct {
		From string
		To   string
	}{
		{From: "", To: "2026-01-01"},
		{From: "2026-01-01", To: ""},
		{From: "", To: ""},
		{From: "01-01-1984", To: "2026-01-01"},
		{From: "2026-01-01", To: "01-01-1984"},
		{From: today.AddDate(0, 0, 1).Format("2026-01-01"), To: "2026-01-01"},
		{From: "2026-01-01", To: today.AddDate(0, 0, 1).Format("2026-01-01")},
		{From: "2026-01-01", To: "2026-01-01"},
	}

	for _, c := range cases {
		_, err := buildGarminURLs(SleepURL, c.From, c.To)
		assert.Error(t, err)
	}
}

func TestGarminUrls(t *testing.T) {
	from := "2026-01-01"
	to := "2026-03-01"

	urls, err := buildGarminURLs(SleepURL, from, to)
	assert.NoError(t, err)
	assert.Equal(t, []string{
		fmt.Sprintf("%s%s/2026-01-01/2026-01-28", GarminAPI, SleepURL),
		fmt.Sprintf("%s%s/2026-01-29/2026-02-25", GarminAPI, SleepURL),
		fmt.Sprintf("%s%s/2026-02-26/2026-03-01", GarminAPI, SleepURL),
	}, urls)

	// from/to within 28 day window
	to = "2026-01-15"
	urls, err = buildGarminURLs(SleepURL, from, to)
	assert.NoError(t, err)
	assert.Equal(t, []string{
		fmt.Sprintf("%s%s/2026-01-01/2026-01-15", GarminAPI, SleepURL),
	}, urls)
}

func TestGarminStressToIntervalsStress(t *testing.T) {
	cases := []struct {
		Garmin   int
		Expected intervals.StressLevel
	}{
		{Garmin: -1, Expected: intervals.LowStress},
		{Garmin: 0, Expected: intervals.LowStress},
		{Garmin: 25, Expected: intervals.LowStress},
		{Garmin: 26, Expected: intervals.AvgStress},
		{Garmin: 50, Expected: intervals.AvgStress},
		{Garmin: 51, Expected: intervals.HighStress},
		{Garmin: 75, Expected: intervals.HighStress},
		{Garmin: 76, Expected: intervals.ExtremeStress},
		{Garmin: 100, Expected: intervals.ExtremeStress},
		{Garmin: 101, Expected: intervals.ExtremeStress},
	}

	for _, c := range cases {
		result := garmingStressToIntervalsStress(c.Garmin)
		assert.Equal(t, c.Expected, result)
	}
}
