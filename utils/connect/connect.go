package connect

import (
	"intervals-functions/api/intervals"
)

type GarminSleepQuality string

const (
	Excellent GarminSleepQuality = "EXCELLENT"
	Good      GarminSleepQuality = "GOOD"
	Fair      GarminSleepQuality = "FAIR"
	Poor      GarminSleepQuality = "POOR"
)

// GarminStressToIntervalsStress maps a numerical garmin overall stress score to an intervals stress value
func GarminStressToIntervalsStress(stress int) intervals.StressLevel {
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

// GarminSleepQualityToIntervalsSleepQuality maps a garmin sleep quality string to intervals sleep quality value
func GarminSleepQualityToIntervalsSleepQuality(q GarminSleepQuality) *intervals.SleepQuality {
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
