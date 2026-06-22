package main

import (
	intervals "intervals-functions/api"
	"intervals-functions/utils/ptr"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTrimWellnessRecords(t *testing.T) {
	stress := intervals.HighStress
	sleep := intervals.GoodSleepQuality
	records := []intervals.WellnessRecord{
		{
			ID:                    intervals.WellnessRecordID("abc"),
			KCalConsumed:          ptr.Float(1.1),
			Carbohydrates:         ptr.Float(2.1),
			Protein:               ptr.Float(3.1),
			Fat:                   ptr.Float(4.1),
			OxygenSaturation:      ptr.Float(5.1),
			Respiration:           ptr.Float(6.1),
			Stress:                &stress,
			SleepScore:            ptr.Float(7.1),
			SleepSeconds:          ptr.Int(81),
			SleepQuality:          &sleep,
			HrvRmssd:              ptr.Float(9.1),
			RestingHr:             ptr.Int(11),
			Weight:                ptr.Float(1.2),
			Soreness:              ptr.Int(22),
			Fatigue:               ptr.Int(32),
			Mood:                  ptr.Int(42),
			Motivation:            ptr.Int(52),
			Injury:                ptr.Int(62),
			BodyBatteryMin:        ptr.Int(72),
			BodyBatterMax:         ptr.Int(82),
			RestStressSeconds:     ptr.Int(92),
			LowStressSeconds:      ptr.Int(102),
			MediumStressSeconds:   ptr.Int(13),
			HighStressSeconds:     ptr.Int(23),
			SleepNeedMinutes:      ptr.Int(33),
			SleepRemTimeSeconds:   ptr.Int(43),
			SleepDeepTimeSeconds:  ptr.Int(53),
			SleepLightTimeSeconds: ptr.Int(63),
			SleepAwakeTimeSeconds: ptr.Int(73),
		},
	}

	result := trimWellnessRecords(records)
	assert.Equal(t, []intervals.WellnessRecord{
		{
			ID:               intervals.WellnessRecordID("abc"),
			Respiration:      ptr.Float(6.1),
			Stress:           &stress,
			SleepScore:       ptr.Float(7.1),
			SleepSeconds:     ptr.Int(81),
			SleepQuality:     &sleep,
			HrvRmssd:         ptr.Float(9.1),
			RestingHr:        ptr.Int(11),
			Weight:           ptr.Float(1.2),
			Soreness:         ptr.Int(22),
			Fatigue:          ptr.Int(32),
			Mood:             ptr.Int(42),
			Motivation:       ptr.Int(52),
			Injury:           ptr.Int(62),
			SleepNeedMinutes: ptr.Int(33),
		},
	}, result)
}

func TestTrimEvents(t *testing.T) {
	today := time.Now()
	dateCutoff := today.AddDate(0, 0, -8)
	userInputLimit := 10
	events := []intervals.Event{
		// content trimmed
		{
			Category:    intervals.Injured,
			Description: "i got injured bro",
		},
		// included as is
		{
			Category:    intervals.RaceA,
			Description: "won",
		},
		// included, content trimmed
		{
			Category:    intervals.Note,
			Name:        "non digest",
			Description: "here is a poem i wrote: be excellent to each other",
		},
		// excluded entirely
		{
			Category:    intervals.Note,
			Name:        "Weekly Digest",
			Date:        today.AddDate(0, 0, -30).Format("2006-01-02"),
			Description: "a previous weekly digest",
		},
		// carryover not found in digest
		{
			Category:    intervals.Note,
			Name:        "Weekly Digest",
			Date:        today.AddDate(0, 0, -7).Format("2006-01-02"),
			Description: "before content to ignore here is important context to use",
		},
		// included, only use carryover content, all of it
		{
			Category:    intervals.Note,
			Name:        "Weekly Digest",
			Date:        today.AddDate(0, 0, -7).Format("2006-01-02"),
			Description: "before content to ignore <!-- carryover here is important context to use",
		},
	}

	result := trimEvents(events, userInputLimit, dateCutoff)
	assert.Equal(t, []intervals.Event{
		{
			Category:    intervals.Injured,
			Description: "i got inju",
		},
		{
			Category:    intervals.RaceA,
			Description: "won",
		},
		{
			Category:    intervals.Note,
			Name:        "non digest",
			Description: "here is a ",
		},
		{
			Category:    intervals.Note,
			Name:        "Weekly Digest",
			Date:        today.AddDate(0, 0, -7).Format("2006-01-02"),
			Description: "before con",
		},
		{
			Category:    intervals.Note,
			Name:        "Weekly Digest",
			Date:        today.AddDate(0, 0, -7).Format("2006-01-02"),
			Description: " here is important context to use",
		},
	}, result)
}
