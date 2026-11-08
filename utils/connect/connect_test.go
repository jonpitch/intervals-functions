package connect

import (
	"intervals-functions/api/intervals"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
		result := GarminStressToIntervalsStress(c.Garmin)
		assert.Equal(t, c.Expected, result)
	}
}

func TestGarminSleepQualityToIntervals(t *testing.T) {
	great := intervals.GreatSleepQuality
	good := intervals.GoodSleepQuality
	average := intervals.AverageSleepQuality
	poor := intervals.PoorSleepQuality

	cases := []struct {
		Garmin   GarminSleepQuality
		Expected *intervals.SleepQuality
	}{
		{Garmin: Excellent, Expected: &great},
		{Garmin: Good, Expected: &good},
		{Garmin: Fair, Expected: &average},
		{Garmin: Poor, Expected: &poor},
		{Garmin: "", Expected: nil},
	}

	for _, c := range cases {
		result := GarminSleepQualityToIntervalsSleepQuality(c.Garmin)
		assert.Equal(t, c.Expected, result)
	}
}
