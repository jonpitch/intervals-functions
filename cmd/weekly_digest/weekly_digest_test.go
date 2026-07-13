package main

import (
	intervals "intervals-functions/api"
	"intervals-functions/utils/ptr"
	"testing"

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
			ID:             intervals.WellnessRecordID("abc"),
			Respiration:    ptr.Float(6.1),
			Stress:         &stress,
			SleepScore:     ptr.Float(7.1),
			SleepSeconds:   ptr.Int(81),
			SleepQuality:   &sleep,
			HrvRmssd:       ptr.Float(9.1),
			RestingHr:      ptr.Int(11),
			Weight:         ptr.Float(1.2),
			Soreness:       ptr.Int(22),
			Fatigue:        ptr.Int(32),
			Mood:           ptr.Int(42),
			Motivation:     ptr.Int(52),
			Injury:         ptr.Int(62),
			BodyBatteryMin: ptr.Int(72),
			BodyBatterMax:  ptr.Int(82),
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
					RestingHr:      ptr.Int(1),
					HrvRmssd:       ptr.Float(10),
					SleepScore:     ptr.Float(100.0),
					Respiration:    ptr.Float(9),
					BodyBatteryMin: ptr.Int(33),
					BodyBatterMax:  ptr.Int(100),
				},
				{
					RestingHr:      ptr.Int(2),
					HrvRmssd:       ptr.Float(20),
					SleepScore:     ptr.Float(60.0),
					Respiration:    ptr.Float(8),
					BodyBatteryMin: ptr.Int(25),
					BodyBatterMax:  ptr.Int(95),
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
