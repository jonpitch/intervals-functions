package main

import (
	"intervals-functions/api/garmin"
	"intervals-functions/api/intervals"
	"intervals-functions/utils/connect"
	"intervals-functions/utils/format"
	"intervals-functions/utils/ptr"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDailySummaryAccumulator(t *testing.T) {
	overallStress := connect.GarminStressToIntervalsStress(20)
	timeA := format.GarminDate{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	timeB := format.GarminDate{Time: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)}

	cases := []struct {
		Entries  []garmin.DailySummaryData
		Wellness map[format.GarminDate]intervals.WellnessRecord
		Expected map[format.GarminDate]intervals.WellnessRecord
	}{
		{
			// nothing provided
			Entries:  []garmin.DailySummaryData{},
			Wellness: map[format.GarminDate]intervals.WellnessRecord{},
			Expected: map[format.GarminDate]intervals.WellnessRecord{},
		},
		{
			// update an existing record
			Entries: []garmin.DailySummaryData{
				{
					CalendarDate: timeA,
					BodyBattery: garmin.BodyBattery{
						MinValue: 30,
						MaxValue: 100,
					},
					Stress: garmin.Stress{
						AverageLevel:                 20,
						RestDurationInMillis:         100 * 1000,
						LowStressDurationInMillis:    200 * 1000,
						MediumStressDurationInMillis: 300 * 1000,
						HighStressDurationInMillis:   400 * 1000,
					},
					HeartRate: garmin.HeartRate{
						RestingValue: 40,
					},
				},
			},
			Wellness: map[format.GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:          intervals.WellnessRecordID("2026-01-01"),
					Respiration: ptr.Float(12),
				},
			},
			Expected: map[format.GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:                  intervals.WellnessRecordID("2026-01-01"),
					Respiration:         ptr.Float(12.0),
					BodyBatteryMin:      ptr.Int(30),
					BodyBatteryMax:      ptr.Int(100),
					Stress:              &overallStress,
					RestStressSeconds:   ptr.Int(100),
					LowStressSeconds:    ptr.Int(200),
					MediumStressSeconds: ptr.Int(300),
					HighStressSeconds:   ptr.Int(400),
					RestingHr:           ptr.Int(40),
				},
			},
		},
		{
			// add a new record
			Entries: []garmin.DailySummaryData{
				{
					CalendarDate: timeB,
					BodyBattery: garmin.BodyBattery{
						MinValue: 25,
						MaxValue: 95,
					},
					Stress: garmin.Stress{
						AverageLevel:                 20,
						RestDurationInMillis:         100 * 1000,
						LowStressDurationInMillis:    200 * 1000,
						MediumStressDurationInMillis: 300 * 1000,
						HighStressDurationInMillis:   400 * 1000,
					},
					HeartRate: garmin.HeartRate{
						RestingValue: 40,
					},
				},
			},
			Wellness: map[format.GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:             intervals.WellnessRecordID("2026-01-01"),
					BodyBatteryMin: ptr.Int(10),
					BodyBatteryMax: ptr.Int(90),
				},
			},
			Expected: map[format.GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:             intervals.WellnessRecordID("2026-01-01"),
					BodyBatteryMin: ptr.Int(10),
					BodyBatteryMax: ptr.Int(90),
				},
				timeB: {
					ID:                  intervals.WellnessRecordID("2026-02-01"),
					BodyBatteryMin:      ptr.Int(25),
					BodyBatteryMax:      ptr.Int(95),
					Stress:              &overallStress,
					RestStressSeconds:   ptr.Int(100),
					LowStressSeconds:    ptr.Int(200),
					MediumStressSeconds: ptr.Int(300),
					HighStressSeconds:   ptr.Int(400),
					RestingHr:           ptr.Int(40),
				},
			},
		},
		{
			// overwrite an existing record
			Entries: []garmin.DailySummaryData{
				{
					CalendarDate: timeA,
					BodyBattery: garmin.BodyBattery{
						MinValue: 45,
						MaxValue: 65,
					},
					Stress: garmin.Stress{
						AverageLevel:                 20,
						RestDurationInMillis:         100 * 1000,
						LowStressDurationInMillis:    200 * 1000,
						MediumStressDurationInMillis: 300 * 1000,
						HighStressDurationInMillis:   400 * 1000,
					},
					HeartRate: garmin.HeartRate{
						RestingValue: 40,
					},
				},
			},
			Wellness: map[format.GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:                  intervals.WellnessRecordID("2026-01-01"),
					BodyBatteryMin:      ptr.Int(32),
					BodyBatteryMax:      ptr.Int(67),
					RestStressSeconds:   ptr.Int(900),
					LowStressSeconds:    ptr.Int(800),
					MediumStressSeconds: ptr.Int(700),
					HighStressSeconds:   ptr.Int(600),
					RestingHr:           ptr.Int(50),
				},
			},
			Expected: map[format.GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:                  intervals.WellnessRecordID("2026-01-01"),
					BodyBatteryMin:      ptr.Int(45),
					BodyBatteryMax:      ptr.Int(65),
					Stress:              &overallStress,
					RestStressSeconds:   ptr.Int(100),
					LowStressSeconds:    ptr.Int(200),
					MediumStressSeconds: ptr.Int(300),
					HighStressSeconds:   ptr.Int(400),
					RestingHr:           ptr.Int(40),
				},
			},
		},
	}

	for _, c := range cases {
		result := garminDailySummaryAccumulator(c.Entries, c.Wellness)
		assert.Equal(t, c.Expected, result)
	}
}

func TestSleepAccumulator(t *testing.T) {
	timeA := format.GarminDate{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	timeB := format.GarminDate{Time: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)}
	greatSleepQuality := intervals.GreatSleepQuality

	cases := []struct {
		Response []garmin.DailySleepResponse
		Wellness map[format.GarminDate]intervals.WellnessRecord
		Expected map[format.GarminDate]intervals.WellnessRecord
	}{
		{
			// nothing provided
			Response: []garmin.DailySleepResponse{},
			Wellness: map[format.GarminDate]intervals.WellnessRecord{},
			Expected: map[format.GarminDate]intervals.WellnessRecord{},
		},
		{
			// update an existing record
			Response: []garmin.DailySleepResponse{
				{
					CalendarDate:      timeA,
					RemSleepSeconds:   1,
					SleepTimeSeconds:  100,
					DeepSleepSeconds:  2,
					AwakeSleepSeconds: 3,
					SleepScores: garmin.SleepScores{
						Overall: garmin.OverallSleepScore{
							Value:     99,
							Qaulifier: connect.Excellent,
						},
					},
					AverageSpo2:       95.0,
					LightSleepSeconds: 4,
					SleepNeed: garmin.SleepNeed{
						ActualMinutes: 9,
					},
				},
			},
			Wellness: map[format.GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:             intervals.WellnessRecordID("2026-01-01"),
					BodyBatteryMin: ptr.Int(11),
					BodyBatteryMax: ptr.Int(97),
				},
			},
			Expected: map[format.GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:                    intervals.WellnessRecordID("2026-01-01"),
					BodyBatteryMin:        ptr.Int(11),
					BodyBatteryMax:        ptr.Int(97),
					SleepRemTimeSeconds:   ptr.Int(1),
					SleepSeconds:          ptr.Int(100),
					SleepDeepTimeSeconds:  ptr.Int(2),
					SleepAwakeTimeSeconds: ptr.Int(3),
					SleepQuality:          &greatSleepQuality,
					OxygenSaturation:      ptr.Float(95.0),
					SleepLightTimeSeconds: ptr.Int(4),
					HrvRmssd:              ptr.Float(66),
					SleepNeedMinutes:      ptr.Int(9),
					SleepScore:            ptr.Float(99.0),
				},
			},
		},
		{
			// add a new record
			Response: []garmin.DailySleepResponse{
				{
					CalendarDate:      timeB,
					RemSleepSeconds:   2,
					SleepTimeSeconds:  101,
					DeepSleepSeconds:  21,
					AwakeSleepSeconds: 31,
					SleepScores: garmin.SleepScores{
						Overall: garmin.OverallSleepScore{
							Value:     100,
							Qaulifier: connect.Excellent,
						},
					},
					AverageSpo2:       95.0,
					LightSleepSeconds: 5,
					SleepNeed: garmin.SleepNeed{
						ActualMinutes: 10,
					},
				},
			},
			Wellness: map[format.GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:             intervals.WellnessRecordID("2026-01-01"),
					BodyBatteryMin: ptr.Int(10),
					BodyBatteryMax: ptr.Int(90),
				},
			},
			Expected: map[format.GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:             intervals.WellnessRecordID("2026-01-01"),
					BodyBatteryMin: ptr.Int(10),
					BodyBatteryMax: ptr.Int(90),
				},
				timeB: {
					ID:                    intervals.WellnessRecordID("2026-02-01"),
					SleepRemTimeSeconds:   ptr.Int(2),
					SleepSeconds:          ptr.Int(101),
					SleepDeepTimeSeconds:  ptr.Int(21),
					SleepAwakeTimeSeconds: ptr.Int(31),
					SleepQuality:          &greatSleepQuality,
					SleepLightTimeSeconds: ptr.Int(5),
					SleepNeedMinutes:      ptr.Int(10),
					SleepScore:            ptr.Float(100.0),
				},
			},
		},
		{
			// overwrite an existing record
			Response: []garmin.DailySleepResponse{
				{
					CalendarDate:      timeA,
					RemSleepSeconds:   1,
					SleepTimeSeconds:  100,
					DeepSleepSeconds:  2,
					AwakeSleepSeconds: 3,
					SleepScores: garmin.SleepScores{
						Overall: garmin.OverallSleepScore{
							Value:     99,
							Qaulifier: connect.Excellent,
						},
					},
					AverageSpo2:       95.0,
					LightSleepSeconds: 4,
					SleepNeed: garmin.SleepNeed{
						ActualMinutes: 9,
					},
				},
			},
			Wellness: map[format.GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:                    intervals.WellnessRecordID("2026-01-01"),
					SleepRemTimeSeconds:   ptr.Int(2),
					SleepSeconds:          ptr.Int(101),
					SleepDeepTimeSeconds:  ptr.Int(21),
					SleepAwakeTimeSeconds: ptr.Int(31),
					SleepQuality:          &greatSleepQuality,
					OxygenSaturation:      ptr.Float(96.0),
					SleepLightTimeSeconds: ptr.Int(5),
					HrvRmssd:              ptr.Float(67),
					SleepNeedMinutes:      ptr.Int(10),
					SleepScore:            ptr.Float(100.0),
				},
			},
			Expected: map[format.GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:                    intervals.WellnessRecordID("2026-01-01"),
					SleepRemTimeSeconds:   ptr.Int(1),
					SleepSeconds:          ptr.Int(100),
					SleepDeepTimeSeconds:  ptr.Int(2),
					SleepAwakeTimeSeconds: ptr.Int(3),
					SleepQuality:          &greatSleepQuality,
					OxygenSaturation:      ptr.Float(95.0),
					SleepLightTimeSeconds: ptr.Int(4),
					HrvRmssd:              ptr.Float(66),
					SleepNeedMinutes:      ptr.Int(9),
					SleepScore:            ptr.Float(99.0),
				},
			},
		},
	}

	for _, c := range cases {
		result := garminDailySleepAccumulator(c.Response, c.Wellness)
		assert.Equal(t, c.Expected, result)
	}
}

func TestGarminWeightAccumulator(t *testing.T) {
	timeA := format.GarminDate{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	timeB := format.GarminDate{Time: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)}

	cases := []struct {
		Entries  []garmin.DailyWeightSummary
		Wellness map[format.GarminDate]intervals.WellnessRecord
		Expected map[format.GarminDate]intervals.WellnessRecord
	}{
		{
			// nothing provided
			Entries:  []garmin.DailyWeightSummary{},
			Wellness: map[format.GarminDate]intervals.WellnessRecord{},
			Expected: map[format.GarminDate]intervals.WellnessRecord{},
		},
		{
			// update an existing record
			Entries: []garmin.DailyWeightSummary{
				{
					LatestWeight: garmin.LatestWeight{
						CalendarDate:      timeA,
						WeightInGrams:     2000,
						MuscleMassInGrams: 32000,
					},
				},
			},
			Wellness: map[format.GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:             intervals.WellnessRecordID("2026-01-01"),
					BodyBatteryMin: ptr.Int(30),
					BodyBatteryMax: ptr.Int(100),
				},
			},
			Expected: map[format.GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:             intervals.WellnessRecordID("2026-01-01"),
					BodyBatteryMin: ptr.Int(30),
					BodyBatteryMax: ptr.Int(100),
					Weight:         ptr.Float(2),
					MuscleMass:     ptr.Float(32),
				},
			},
		},
		{
			// add a new record
			Entries: []garmin.DailyWeightSummary{
				{
					LatestWeight: garmin.LatestWeight{
						CalendarDate:      timeB,
						WeightInGrams:     3000,
						MuscleMassInGrams: 28000,
					},
				},
			},
			Wellness: map[format.GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:             intervals.WellnessRecordID("2026-01-01"),
					BodyBatteryMin: ptr.Int(30),
					BodyBatteryMax: ptr.Int(100),
				},
			},
			Expected: map[format.GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:             intervals.WellnessRecordID("2026-01-01"),
					BodyBatteryMin: ptr.Int(30),
					BodyBatteryMax: ptr.Int(100),
				},
				timeB: {
					ID:         intervals.WellnessRecordID("2026-02-01"),
					Weight:     ptr.Float(3),
					MuscleMass: ptr.Float(28),
				},
			},
		},
		{
			// overwrite an existing record
			Entries: []garmin.DailyWeightSummary{
				{
					LatestWeight: garmin.LatestWeight{
						CalendarDate:      timeA,
						WeightInGrams:     8000,
						MuscleMassInGrams: 35000,
					},
				},
			},
			Wellness: map[format.GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:         intervals.WellnessRecordID("2026-01-01"),
					Weight:     ptr.Float(5.0),
					MuscleMass: ptr.Float(30.0),
				},
			},
			Expected: map[format.GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:         intervals.WellnessRecordID("2026-01-01"),
					Weight:     ptr.Float(8.0),
					MuscleMass: ptr.Float(35.0),
				},
			},
		},
		{
			// a weigh-in that doesn't report body composition (e.g. a scale without that
			// feature) must not clobber an existing MuscleMass/BodyFat with an explicit zero
			Entries: []garmin.DailyWeightSummary{
				{
					LatestWeight: garmin.LatestWeight{
						CalendarDate:  timeA,
						WeightInGrams: 8000,
					},
				},
			},
			Wellness: map[format.GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:         intervals.WellnessRecordID("2026-01-01"),
					Weight:     ptr.Float(5.0),
					MuscleMass: ptr.Float(30.0),
				},
			},
			Expected: map[format.GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:         intervals.WellnessRecordID("2026-01-01"),
					Weight:     ptr.Float(8.0),
					MuscleMass: ptr.Float(30.0),
				},
			},
		},
	}

	for _, c := range cases {
		result := garminWeightAccumulator(c.Entries, c.Wellness)
		assert.Equal(t, c.Expected, result)
	}
}

func TestHrvAccumulator(t *testing.T) {
	timeA := format.GarminDate{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	timeB := format.GarminDate{Time: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)}

	cases := []struct {
		Entries  []garmin.HrvSummary
		Wellness map[format.GarminDate]intervals.WellnessRecord
		Expected map[format.GarminDate]intervals.WellnessRecord
	}{
		{
			// nothing provided
			Entries:  []garmin.HrvSummary{},
			Wellness: map[format.GarminDate]intervals.WellnessRecord{},
			Expected: map[format.GarminDate]intervals.WellnessRecord{},
		},
		{
			// update an existing record
			Entries: []garmin.HrvSummary{
				{
					CalendarDate: timeA,
					LastNightAvg: 66.0,
				},
			},
			Wellness: map[format.GarminDate]intervals.WellnessRecord{
				timeA: {
					ID: intervals.WellnessRecordID("2026-01-01"),
				},
			},
			Expected: map[format.GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:       intervals.WellnessRecordID("2026-01-01"),
					HrvRmssd: ptr.Float(66.0),
				},
			},
		},
		{
			// add a new record
			Entries: []garmin.HrvSummary{
				{
					CalendarDate: timeB,
					LastNightAvg: 67.0,
				},
			},
			Wellness: map[format.GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:       intervals.WellnessRecordID("2026-01-01"),
					HrvRmssd: ptr.Float(65.0),
				},
			},
			Expected: map[format.GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:       intervals.WellnessRecordID("2026-01-01"),
					HrvRmssd: ptr.Float(65.0),
				},
				timeB: {
					ID:       intervals.WellnessRecordID("2026-02-01"),
					HrvRmssd: ptr.Float(67.0),
				},
			},
		},
		{
			// overwrite an existing record
			Entries: []garmin.HrvSummary{
				{
					CalendarDate: timeA,
					LastNightAvg: 66.0,
				},
			},
			Wellness: map[format.GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:       intervals.WellnessRecordID("2026-01-01"),
					HrvRmssd: ptr.Float(65.0),
				},
			},
			Expected: map[format.GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:       intervals.WellnessRecordID("2026-01-01"),
					HrvRmssd: ptr.Float(66.0),
				},
			},
		},
	}

	for _, c := range cases {
		result := garminHrvAccumulator(c.Entries, c.Wellness)
		assert.Equal(t, c.Expected, result)
	}
}
