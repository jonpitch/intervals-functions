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
			ID:                    intervals.WellnessRecordID("abc"),
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
			SleepNeedMinutes:      ptr.Int(33),
			SleepRemTimeSeconds:   ptr.Int(43),
			SleepDeepTimeSeconds:  ptr.Int(53),
			SleepLightTimeSeconds: ptr.Int(63),
			SleepAwakeTimeSeconds: ptr.Int(73),
		},
	}, result)
}

func TestWindowAverages(t *testing.T) {
	cases := []struct {
		records  []intervals.WellnessRecord
		expected WindowAverage
	}{
		{
			records: []intervals.WellnessRecord{
				{
					RestingHr:             ptr.Int(1),
					HrvRmssd:              ptr.Float(10),
					SleepScore:            ptr.Float(100.0),
					Respiration:           ptr.Float(9),
					BodyBatteryMin:        ptr.Int(33),
					BodyBatterMax:         ptr.Int(100),
					SleepNeedMinutes:      ptr.Int(400),
					SleepRemTimeSeconds:   ptr.Int(4000),
					SleepDeepTimeSeconds:  ptr.Int(3000),
					SleepLightTimeSeconds: ptr.Int(5000),
					SleepAwakeTimeSeconds: ptr.Int(600),
					Vo2Max:                ptr.Float(51.0),
					CyclingVo2Max:         ptr.Float(53.0),
				},
				{
					RestingHr:             ptr.Int(2),
					HrvRmssd:              ptr.Float(20),
					SleepScore:            ptr.Float(60.0),
					Respiration:           ptr.Float(8),
					BodyBatteryMin:        ptr.Int(25),
					BodyBatterMax:         ptr.Int(95),
					SleepNeedMinutes:      ptr.Int(420),
					SleepRemTimeSeconds:   ptr.Int(4200),
					SleepDeepTimeSeconds:  ptr.Int(3200),
					SleepLightTimeSeconds: ptr.Int(5200),
					SleepAwakeTimeSeconds: ptr.Int(800),
					Vo2Max:                ptr.Float(53.0),
					CyclingVo2Max:         ptr.Float(55.0),
				},
			},
			expected: WindowAverage{
				RestingHeartRate: AveragedAttribute{
					Average: ptr.Float(1.5),
					Count:   2,
				},
				Hrv: AveragedAttribute{
					Average: ptr.Float(15.0),
					Count:   2,
				},
				SleepScore: AveragedAttribute{
					Average: ptr.Float(80.0),
					Count:   2,
				},
				Respiration: AveragedAttribute{
					Average: ptr.Float(8.5),
					Count:   2,
				},
				BodyBatteryMin: AveragedAttribute{
					Average: ptr.Float(29.0),
					Count:   2,
				},
				BodyBatteryMax: AveragedAttribute{
					Average: ptr.Float(97.5),
					Count:   2,
				},
				SleepNeedMinutes: AveragedAttribute{
					Average: ptr.Float(410.0),
					Count:   2,
				},
				SleepRemTimeSeconds: AveragedAttribute{
					Average: ptr.Float(4100.0),
					Count:   2,
				},
				SleepDeepTimeSeconds: AveragedAttribute{
					Average: ptr.Float(3100.0),
					Count:   2,
				},
				SleepLightTimeSeconds: AveragedAttribute{
					Average: ptr.Float(5100.0),
					Count:   2,
				},
				SleepAwakeTimeSeconds: AveragedAttribute{
					Average: ptr.Float(700.0),
					Count:   2,
				},
				Vo2Max: AveragedAttribute{
					Average: ptr.Float(52.0),
					Count:   2,
				},
				CyclingVo2Max: AveragedAttribute{
					Average: ptr.Float(54.0),
					Count:   2,
				},
			},
		},
		{
			records: []intervals.WellnessRecord{},
			expected: WindowAverage{
				RestingHeartRate: AveragedAttribute{
					Average: nil,
					Count:   0,
				},
			},
		},
	}

	for _, c := range cases {
		result := windowAverages(c.records)
		assert.Equal(t, c.expected, result)
	}
}

func TestWeeklyAverages(t *testing.T) {
	// note that weeklyAverages calls windowAverages, so these test cases
	// are focused on dates and grouping and wellness attributes, averages, totals, etc.
	// are handled in TestWindowAverages
	cases := []struct {
		records  []intervals.WellnessRecord
		expected []WeeklyAverages
	}{
		{
			records: []intervals.WellnessRecord{
				{
					ID:        intervals.WellnessRecordID("2026-01-01"),
					RestingHr: ptr.Int(1),
				},
				{
					ID:        intervals.WellnessRecordID("2026-01-02"),
					RestingHr: ptr.Int(2),
				},
			},
			expected: []WeeklyAverages{
				{
					Period: Week{
						Year: 2026,
						Week: 1,
					},
					Averages: WindowAverage{
						RestingHeartRate: AveragedAttribute{
							Average: ptr.Float(1.5),
							Count:   2,
						},
					},
				},
			},
		},
		{
			records:  []intervals.WellnessRecord{},
			expected: []WeeklyAverages{},
		},
		{
			records: []intervals.WellnessRecord{
				{
					ID: intervals.WellnessRecordID("2026-01-01"),
				},
				{
					ID: intervals.WellnessRecordID("2026-01-02"),
				},
			},
			expected: []WeeklyAverages{
				{
					Period: Week{
						Year: 2026,
						Week: 1,
					},
					Averages: WindowAverage{},
				},
			},
		},
		{
			records: []intervals.WellnessRecord{
				{
					ID:        intervals.WellnessRecordID("2026-01-01"),
					RestingHr: ptr.Int(1),
				},
				{
					ID:        intervals.WellnessRecordID("2026-02-01"),
					RestingHr: ptr.Int(2),
				},
			},
			expected: []WeeklyAverages{
				{
					Period: Week{
						Year: 2026,
						Week: 1,
					},
					Averages: WindowAverage{
						RestingHeartRate: AveragedAttribute{
							Average: ptr.Float(1.0),
							Count:   1,
						},
					},
				},
				{
					Period: Week{
						Year: 2026,
						Week: 5,
					},
					Averages: WindowAverage{
						RestingHeartRate: AveragedAttribute{
							Average: ptr.Float(2.0),
							Count:   1,
						},
					},
				},
			},
		},
	}

	for _, c := range cases {
		result := weeklyAverages(c.records)
		assert.Equal(t, c.expected, result)
	}
}

func TestTrimEvents(t *testing.T) {
	today := time.Now()
	dateCutoff := today.AddDate(0, 0, -8)
	oldDate := today.AddDate(0, 0, -43).Format("2006-01-02T00:00:00")
	recentDate := today.AddDate(0, 0, -7).Format("2006-01-02T00:00:00")
	events := []intervals.Event{
		// included, content trimmed
		{
			Category:    intervals.Note,
			Name:        "non digest",
			Date:        recentDate,
			Description: "here is a poem i wrote: be excellent to each other",
		},
		// excluded entirely
		{
			Category:    intervals.Note,
			Name:        "Weekly Digest",
			Date:        oldDate,
			Description: "a previous weekly digest",
		},
		// carryover not found in digest
		{
			Category:    intervals.Note,
			Name:        "Weekly Digest",
			Date:        recentDate,
			Description: "before content to ignore here is important context to use",
		},
		// included, only use carryover content, all of it
		{
			Category:    intervals.Note,
			Name:        "Weekly Digest",
			Date:        recentDate,
			Description: "before content to ignore <!-- carryover here is important context to use",
		},
	}

	result := trimEvents(events, dateCutoff)
	assert.Equal(t, []intervals.Event{
		{
			Category:    intervals.Note,
			Name:        "Weekly Digest",
			Date:        recentDate,
			Description: " here is important context to use",
		},
	}, result)
}
