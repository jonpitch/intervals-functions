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
		result := garminStressToIntervalsStress(c.Garmin)
		assert.Equal(t, c.Expected, result)
	}
}

func TestGarminStressAccumulator(t *testing.T) {
	overallStress := garminStressToIntervalsStress(20)
	timeA := format.GarminDate{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	timeB := format.GarminDate{Time: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)}

	cases := []struct {
		Entries  []garmin.DailyStressResponse
		Wellness map[format.GarminDate]intervals.WellnessRecord
		Expected map[format.GarminDate]intervals.WellnessRecord
	}{
		{
			// nothing provided
			Entries:  []garmin.DailyStressResponse{},
			Wellness: map[format.GarminDate]intervals.WellnessRecord{},
			Expected: map[format.GarminDate]intervals.WellnessRecord{},
		},
		{
			// update an existing record
			Entries: []garmin.DailyStressResponse{
				{
					CalendarDate: timeA,
					Values: garmin.StressValues{
						OverallStressLevel:          20,
						RestStressDurationSeconds:   100,
						LowStressDurationSeconds:    200,
						MediumStressDurationSeconds: 300,
						HighStressDurationSeconds:   400,
					},
				},
			},
			Wellness: map[format.GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:             intervals.WellnessRecordID("2026-01-01"),
					BodyBatteryMin: ptr.Int(30),
					BodyBatterMax:  ptr.Int(100),
				},
			},
			Expected: map[format.GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:                  intervals.WellnessRecordID("2026-01-01"),
					BodyBatteryMin:      ptr.Int(30),
					BodyBatterMax:       ptr.Int(100),
					Stress:              &overallStress,
					RestStressSeconds:   ptr.Int(100),
					LowStressSeconds:    ptr.Int(200),
					MediumStressSeconds: ptr.Int(300),
					HighStressSeconds:   ptr.Int(400),
				},
			},
		},
		{
			// add a new record
			Entries: []garmin.DailyStressResponse{
				{
					CalendarDate: timeB,
					Values: garmin.StressValues{
						OverallStressLevel:          20,
						RestStressDurationSeconds:   100,
						LowStressDurationSeconds:    200,
						MediumStressDurationSeconds: 300,
						HighStressDurationSeconds:   400,
					},
				},
			},
			Wellness: map[format.GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:             intervals.WellnessRecordID("2026-01-01"),
					BodyBatteryMin: ptr.Int(30),
					BodyBatterMax:  ptr.Int(100),
				},
			},
			Expected: map[format.GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:             intervals.WellnessRecordID("2026-01-01"),
					BodyBatteryMin: ptr.Int(30),
					BodyBatterMax:  ptr.Int(100),
				},
				timeB: {
					ID:                  intervals.WellnessRecordID("2026-02-01"),
					Stress:              &overallStress,
					RestStressSeconds:   ptr.Int(100),
					LowStressSeconds:    ptr.Int(200),
					MediumStressSeconds: ptr.Int(300),
					HighStressSeconds:   ptr.Int(400),
				},
			},
		},
		{
			// overwrite an existing record
			Entries: []garmin.DailyStressResponse{
				{
					CalendarDate: timeA,
					Values: garmin.StressValues{
						OverallStressLevel:          20,
						RestStressDurationSeconds:   100,
						LowStressDurationSeconds:    200,
						MediumStressDurationSeconds: 300,
						HighStressDurationSeconds:   400,
					},
				},
			},
			Wellness: map[format.GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:                  intervals.WellnessRecordID("2026-01-01"),
					BodyBatteryMin:      ptr.Int(30),
					BodyBatterMax:       ptr.Int(100),
					RestStressSeconds:   ptr.Int(900),
					LowStressSeconds:    ptr.Int(800),
					MediumStressSeconds: ptr.Int(700),
					HighStressSeconds:   ptr.Int(600),
				},
			},
			Expected: map[format.GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:                  intervals.WellnessRecordID("2026-01-01"),
					BodyBatteryMin:      ptr.Int(30),
					BodyBatterMax:       ptr.Int(100),
					Stress:              &overallStress,
					RestStressSeconds:   ptr.Int(100),
					LowStressSeconds:    ptr.Int(200),
					MediumStressSeconds: ptr.Int(300),
					HighStressSeconds:   ptr.Int(400),
				},
			},
		},
	}

	for _, c := range cases {
		result := garminStressAccumulator(c.Entries, c.Wellness)
		assert.Equal(t, c.Expected, result)
	}
}

func TestDailySummaryAccumulator(t *testing.T) {
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
					ID:             intervals.WellnessRecordID("2026-01-01"),
					Respiration:    ptr.Float(12.0),
					BodyBatteryMin: ptr.Int(30),
					BodyBatterMax:  ptr.Int(100),
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
				},
			},
			Wellness: map[format.GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:             intervals.WellnessRecordID("2026-01-01"),
					BodyBatteryMin: ptr.Int(10),
					BodyBatterMax:  ptr.Int(90),
				},
			},
			Expected: map[format.GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:             intervals.WellnessRecordID("2026-01-01"),
					BodyBatteryMin: ptr.Int(10),
					BodyBatterMax:  ptr.Int(90),
				},
				timeB: {
					ID:             intervals.WellnessRecordID("2026-02-01"),
					BodyBatteryMin: ptr.Int(25),
					BodyBatterMax:  ptr.Int(95),
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
				},
			},
			Wellness: map[format.GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:             intervals.WellnessRecordID("2026-01-01"),
					BodyBatteryMin: ptr.Int(32),
					BodyBatterMax:  ptr.Int(67),
				},
			},
			Expected: map[format.GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:             intervals.WellnessRecordID("2026-01-01"),
					BodyBatteryMin: ptr.Int(45),
					BodyBatterMax:  ptr.Int(65),
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
					AverageHeartRate:  40,
					SleepTimeSeconds:  100,
					DeepSleepSeconds:  2,
					AwakeSleepSeconds: 3,
					SleepQuality:      connect.Excellent,
					// Spo2:                  95.0,
					LightSleepSeconds: 4,
					// AverageOvernightHrv:   66,
					SleepNeed: garmin.SleepNeed{
						ActualMinutes: 9,
					},
					// SleepScore: 99,
				},
			},
			Wellness: map[format.GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:             intervals.WellnessRecordID("2026-01-01"),
					BodyBatteryMin: ptr.Int(11),
					BodyBatterMax:  ptr.Int(97),
				},
			},
			Expected: map[format.GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:                    intervals.WellnessRecordID("2026-01-01"),
					BodyBatteryMin:        ptr.Int(11),
					BodyBatterMax:         ptr.Int(97),
					SleepRemTimeSeconds:   ptr.Int(1),
					RestingHr:             ptr.Int(40),
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
					AverageHeartRate:  41,
					SleepTimeSeconds:  101,
					DeepSleepSeconds:  21,
					AwakeSleepSeconds: 31,
					SleepQuality:      connect.Excellent,
					// Spo2:                  0.0,
					LightSleepSeconds: 5,
					// AverageOvernightHrv:   0.0,
					SleepNeed: garmin.SleepNeed{
						ActualMinutes: 10,
					},
					// SleepScore:            100,
				},
			},
			Wellness: map[format.GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:             intervals.WellnessRecordID("2026-01-01"),
					BodyBatteryMin: ptr.Int(10),
					BodyBatterMax:  ptr.Int(90),
				},
			},
			Expected: map[format.GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:             intervals.WellnessRecordID("2026-01-01"),
					BodyBatteryMin: ptr.Int(10),
					BodyBatterMax:  ptr.Int(90),
				},
				timeB: {
					ID:                    intervals.WellnessRecordID("2026-02-01"),
					SleepRemTimeSeconds:   ptr.Int(2),
					RestingHr:             ptr.Int(41),
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
					AverageHeartRate:  40,
					SleepTimeSeconds:  100,
					DeepSleepSeconds:  2,
					AwakeSleepSeconds: 3,
					SleepQuality:      connect.Excellent,
					// Spo2:                  95.0,
					LightSleepSeconds: 4,
					// AverageOvernightHrv:   66,
					SleepNeed: garmin.SleepNeed{
						ActualMinutes: 9,
					},
					// SleepScore:            99,
				},
			},
			Wellness: map[format.GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:                    intervals.WellnessRecordID("2026-01-01"),
					SleepRemTimeSeconds:   ptr.Int(2),
					RestingHr:             ptr.Int(41),
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
					RestingHr:             ptr.Int(40),
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

func TestGarminWeightAccumulator_Metric(t *testing.T) {
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
					BodyBatterMax:  ptr.Int(100),
				},
			},
			Expected: map[format.GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:             intervals.WellnessRecordID("2026-01-01"),
					BodyBatteryMin: ptr.Int(30),
					BodyBatterMax:  ptr.Int(100),
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
					BodyBatterMax:  ptr.Int(100),
				},
			},
			Expected: map[format.GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:             intervals.WellnessRecordID("2026-01-01"),
					BodyBatteryMin: ptr.Int(30),
					BodyBatterMax:  ptr.Int(100),
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

func TestGarminSleepQualityToIntervals(t *testing.T) {
	great := intervals.GreatSleepQuality
	good := intervals.GoodSleepQuality
	average := intervals.AverageSleepQuality
	poor := intervals.PoorSleepQuality

	cases := []struct {
		Garmin   connect.GarminSleepQuality
		Expected *intervals.SleepQuality
	}{
		{Garmin: connect.Excellent, Expected: &great},
		{Garmin: connect.Good, Expected: &good},
		{Garmin: connect.Fair, Expected: &average},
		{Garmin: connect.Poor, Expected: &poor},
		{Garmin: "", Expected: nil},
	}

	for _, c := range cases {
		result := garminSleepQualityToIntervalsSleepQuality(c.Garmin)
		assert.Equal(t, c.Expected, result)
	}
}
