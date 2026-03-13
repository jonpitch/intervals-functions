package main

import (
	"fmt"
	intervals "intervals-functions/api"
	"intervals-functions/utils/ptr"
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

	// weight URL is slightly different
	urls, err = buildGarminURLs(WeightURL, from, to)
	assert.NoError(t, err)
	assert.Equal(t, []string{
		fmt.Sprintf("%s%s/2026-01-01/2026-01-28?includeAll=true", GarminAPI, WeightURL),
		fmt.Sprintf("%s%s/2026-01-29/2026-02-25?includeAll=true", GarminAPI, WeightURL),
		fmt.Sprintf("%s%s/2026-02-26/2026-03-01?includeAll=true", GarminAPI, WeightURL),
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
		result := garminStressToIntervalsStress(c.Garmin)
		assert.Equal(t, c.Expected, result)
	}
}

func TestGarminStressAccumulator(t *testing.T) {
	overallStress := garminStressToIntervalsStress(20)
	timeA := GarminDate{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	timeB := GarminDate{Time: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)}

	cases := []struct {
		Entries  []StressEntry
		Wellness map[GarminDate]intervals.WellnessRecord
		Expected map[GarminDate]intervals.WellnessRecord
	}{
		{
			// nothing provided
			Entries:  []StressEntry{},
			Wellness: map[GarminDate]intervals.WellnessRecord{},
			Expected: map[GarminDate]intervals.WellnessRecord{},
		},
		{
			// update an existing record
			Entries: []StressEntry{
				{
					Date: timeA,
					Value: StressValue{
						OverallStressLevel:          20,
						RestStressDurationSeconds:   100,
						LowStressDurationSeconds:    200,
						MediumStressDurationSeconds: 300,
						HighStressDurationSeconds:   400,
					},
				},
			},
			Wellness: map[GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:             intervals.WellnessRecordID("2026-01-01"),
					BodyBatteryMin: ptr.Int(30),
					BodyBatterMax:  ptr.Int(100),
				},
			},
			Expected: map[GarminDate]intervals.WellnessRecord{
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
			Entries: []StressEntry{
				{
					Date: timeB,
					Value: StressValue{
						OverallStressLevel:          20,
						RestStressDurationSeconds:   100,
						LowStressDurationSeconds:    200,
						MediumStressDurationSeconds: 300,
						HighStressDurationSeconds:   400,
					},
				},
			},
			Wellness: map[GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:             intervals.WellnessRecordID("2026-01-01"),
					BodyBatteryMin: ptr.Int(30),
					BodyBatterMax:  ptr.Int(100),
				},
			},
			Expected: map[GarminDate]intervals.WellnessRecord{
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
			Entries: []StressEntry{
				{
					Date: timeA,
					Value: StressValue{
						OverallStressLevel:          20,
						RestStressDurationSeconds:   100,
						LowStressDurationSeconds:    200,
						MediumStressDurationSeconds: 300,
						HighStressDurationSeconds:   400,
					},
				},
			},
			Wellness: map[GarminDate]intervals.WellnessRecord{
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
			Expected: map[GarminDate]intervals.WellnessRecord{
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

func TestGarminRespirationAccumulator(t *testing.T) {
	timeA := GarminDate{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	timeB := GarminDate{Time: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)}

	cases := []struct {
		Entries  []RespirationEntry
		Wellness map[GarminDate]intervals.WellnessRecord
		Expected map[GarminDate]intervals.WellnessRecord
	}{
		{
			// nothing provided
			Entries:  []RespirationEntry{},
			Wellness: map[GarminDate]intervals.WellnessRecord{},
			Expected: map[GarminDate]intervals.WellnessRecord{},
		},
		{
			// update an existing record
			Entries: []RespirationEntry{
				{
					Date:                    timeA,
					AverageSleepRespiration: 14.0,
				},
			},
			Wellness: map[GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:             intervals.WellnessRecordID("2026-01-01"),
					BodyBatteryMin: ptr.Int(30),
					BodyBatterMax:  ptr.Int(100),
				},
			},
			Expected: map[GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:             intervals.WellnessRecordID("2026-01-01"),
					BodyBatteryMin: ptr.Int(30),
					BodyBatterMax:  ptr.Int(100),
					Respiration:    ptr.Float(14.0),
				},
			},
		},
		{
			// add a new record
			Entries: []RespirationEntry{
				{
					Date:                    timeB,
					AverageSleepRespiration: 14.0,
				},
			},
			Wellness: map[GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:             intervals.WellnessRecordID("2026-01-01"),
					BodyBatteryMin: ptr.Int(30),
					BodyBatterMax:  ptr.Int(100),
				},
			},
			Expected: map[GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:             intervals.WellnessRecordID("2026-01-01"),
					BodyBatteryMin: ptr.Int(30),
					BodyBatterMax:  ptr.Int(100),
				},
				timeB: {
					ID:          intervals.WellnessRecordID("2026-02-01"),
					Respiration: ptr.Float(14.0),
				},
			},
		},
		{
			// overwrite an existing record
			Entries: []RespirationEntry{
				{
					Date:                    timeA,
					AverageSleepRespiration: 14.0,
				},
			},
			Wellness: map[GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:          intervals.WellnessRecordID("2026-01-01"),
					Respiration: ptr.Float(9.0),
				},
			},
			Expected: map[GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:          intervals.WellnessRecordID("2026-01-01"),
					Respiration: ptr.Float(14.0),
				},
			},
		},
	}

	for _, c := range cases {
		result := garminRespirationAccumulator(c.Entries, c.Wellness)
		assert.Equal(t, c.Expected, result)
	}
}

func TestGarminBodyBatteryAccumulator(t *testing.T) {
	timeA := GarminDate{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	timeB := GarminDate{Time: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)}

	cases := []struct {
		Entries  []BodyBatteryEntry
		Wellness map[GarminDate]intervals.WellnessRecord
		Expected map[GarminDate]intervals.WellnessRecord
	}{
		{
			// nothing provided
			Entries:  []BodyBatteryEntry{},
			Wellness: map[GarminDate]intervals.WellnessRecord{},
			Expected: map[GarminDate]intervals.WellnessRecord{},
		},
		{
			// update an existing record
			Entries: []BodyBatteryEntry{
				{
					Date: timeA,
					Value: BodyBatteryValue{
						LowBodyBattery:  30,
						HighBodyBattery: 100,
					},
				},
			},
			Wellness: map[GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:          intervals.WellnessRecordID("2026-01-01"),
					Respiration: ptr.Float(12),
				},
			},
			Expected: map[GarminDate]intervals.WellnessRecord{
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
			Entries: []BodyBatteryEntry{
				{
					Date: timeB,
					Value: BodyBatteryValue{
						LowBodyBattery:  25,
						HighBodyBattery: 95,
					},
				},
			},
			Wellness: map[GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:             intervals.WellnessRecordID("2026-01-01"),
					BodyBatteryMin: ptr.Int(10),
					BodyBatterMax:  ptr.Int(90),
				},
			},
			Expected: map[GarminDate]intervals.WellnessRecord{
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
			Entries: []BodyBatteryEntry{
				{
					Date: timeA,
					Value: BodyBatteryValue{
						LowBodyBattery:  45,
						HighBodyBattery: 65,
					},
				},
			},
			Wellness: map[GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:             intervals.WellnessRecordID("2026-01-01"),
					BodyBatteryMin: ptr.Int(32),
					BodyBatterMax:  ptr.Int(67),
				},
			},
			Expected: map[GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:             intervals.WellnessRecordID("2026-01-01"),
					BodyBatteryMin: ptr.Int(45),
					BodyBatterMax:  ptr.Int(65),
				},
			},
		},
	}

	for _, c := range cases {
		result := garminBodyBatteryAccumulator(c.Entries, c.Wellness)
		assert.Equal(t, c.Expected, result)
	}
}

func TestSleepAccumulator(t *testing.T) {
	timeA := GarminDate{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	timeB := GarminDate{Time: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)}
	greatSleepQuality := intervals.GreatSleepQuality

	cases := []struct {
		Response SleepResponse
		Wellness map[GarminDate]intervals.WellnessRecord
		Expected map[GarminDate]intervals.WellnessRecord
	}{
		{
			// nothing provided
			Response: SleepResponse{},
			Wellness: map[GarminDate]intervals.WellnessRecord{},
			Expected: map[GarminDate]intervals.WellnessRecord{},
		},
		{
			// update an existing record
			Response: SleepResponse{
				IndividualStats: []SleepEntry{
					{
						Date: timeA,
						Values: SleepValue{
							RemTimeSeconds:        1,
							RestingHeartRate:      40,
							TotalSleepTimeSeconds: 100,
							DeepTimeSeconds:       2,
							AwakeTimeSeconds:      3,
							SleepQuality:          Excellent,
							Spo2:                  95.0,
							LightTimeSeconds:      4,
							AverageOvernightHrv:   66,
							SleepNeedMinutes:      9,
							SleepScore:            99,
						},
					},
				},
			},
			Wellness: map[GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:             intervals.WellnessRecordID("2026-01-01"),
					BodyBatteryMin: ptr.Int(11),
					BodyBatterMax:  ptr.Int(97),
				},
			},
			Expected: map[GarminDate]intervals.WellnessRecord{
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
					SleepScore:            ptr.Int(99),
				},
			},
		},
		{
			// add a new record
			Response: SleepResponse{
				IndividualStats: []SleepEntry{
					{
						Date: timeB,
						Values: SleepValue{
							RemTimeSeconds:        2,
							RestingHeartRate:      41,
							TotalSleepTimeSeconds: 101,
							DeepTimeSeconds:       21,
							AwakeTimeSeconds:      31,
							SleepQuality:          Excellent,
							Spo2:                  0.0,
							LightTimeSeconds:      5,
							AverageOvernightHrv:   0.0,
							SleepNeedMinutes:      10,
							SleepScore:            100,
						},
					},
				},
			},
			Wellness: map[GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:             intervals.WellnessRecordID("2026-01-01"),
					BodyBatteryMin: ptr.Int(10),
					BodyBatterMax:  ptr.Int(90),
				},
			},
			Expected: map[GarminDate]intervals.WellnessRecord{
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
					SleepScore:            ptr.Int(100),
				},
			},
		},
		{
			// overwrite an existing record
			Response: SleepResponse{
				IndividualStats: []SleepEntry{
					{
						Date: timeA,
						Values: SleepValue{
							RemTimeSeconds:        1,
							RestingHeartRate:      40,
							TotalSleepTimeSeconds: 100,
							DeepTimeSeconds:       2,
							AwakeTimeSeconds:      3,
							SleepQuality:          Excellent,
							Spo2:                  95.0,
							LightTimeSeconds:      4,
							AverageOvernightHrv:   66,
							SleepNeedMinutes:      9,
							SleepScore:            99,
						},
					},
				},
			},
			Wellness: map[GarminDate]intervals.WellnessRecord{
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
					SleepScore:            ptr.Int(100),
				},
			},
			Expected: map[GarminDate]intervals.WellnessRecord{
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
					SleepScore:            ptr.Int(99),
				},
			},
		},
	}

	for _, c := range cases {
		result := garminSleepAccumulator(c.Response.IndividualStats, c.Wellness)
		assert.Equal(t, c.Expected, result)
	}
}

func TestGarminWeightAccumulator_Metric(t *testing.T) {
	timeA := GarminDate{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	timeB := GarminDate{Time: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)}

	cases := []struct {
		Entries  []WeightSummary
		Wellness map[GarminDate]intervals.WellnessRecord
		Expected map[GarminDate]intervals.WellnessRecord
	}{
		{
			// nothing provided
			Entries:  []WeightSummary{},
			Wellness: map[GarminDate]intervals.WellnessRecord{},
			Expected: map[GarminDate]intervals.WellnessRecord{},
		},
		{
			// update an existing record
			Entries: []WeightSummary{
				{
					SummaryDate: timeA,
					LatestWeight: LatestWeight{
						Weight: 2000,
					},
				},
			},
			Wellness: map[GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:             intervals.WellnessRecordID("2026-01-01"),
					BodyBatteryMin: ptr.Int(30),
					BodyBatterMax:  ptr.Int(100),
				},
			},
			Expected: map[GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:             intervals.WellnessRecordID("2026-01-01"),
					BodyBatteryMin: ptr.Int(30),
					BodyBatterMax:  ptr.Int(100),
					Weight:         ptr.Float(2),
				},
			},
		},
		{
			// add a new record
			Entries: []WeightSummary{
				{
					SummaryDate: timeB,
					LatestWeight: LatestWeight{
						Weight: 3000,
					},
				},
			},
			Wellness: map[GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:             intervals.WellnessRecordID("2026-01-01"),
					BodyBatteryMin: ptr.Int(30),
					BodyBatterMax:  ptr.Int(100),
				},
			},
			Expected: map[GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:             intervals.WellnessRecordID("2026-01-01"),
					BodyBatteryMin: ptr.Int(30),
					BodyBatterMax:  ptr.Int(100),
				},
				timeB: {
					ID:     intervals.WellnessRecordID("2026-02-01"),
					Weight: ptr.Float(3),
				},
			},
		},
		{
			// overwrite an existing record
			Entries: []WeightSummary{
				{
					SummaryDate: timeA,
					LatestWeight: LatestWeight{
						Weight: 8000,
					},
				},
			},
			Wellness: map[GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:     intervals.WellnessRecordID("2026-01-01"),
					Weight: ptr.Float(5.0),
				},
			},
			Expected: map[GarminDate]intervals.WellnessRecord{
				timeA: {
					ID:     intervals.WellnessRecordID("2026-01-01"),
					Weight: ptr.Float(8.0),
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
	cases := []struct {
		Garmin   GarminSleepQuality
		Expected intervals.SleepQuality
	}{
		{Garmin: Excellent, Expected: intervals.GreatSleepQuality},
		{Garmin: Good, Expected: intervals.GoodSleepQuality},
		{Garmin: Fair, Expected: intervals.AverageSleepQuality},
		{Garmin: Poor, Expected: intervals.PoorSleepQuality},
	}

	for _, c := range cases {
		result := garminSleepQualityToIntervalsSleepQuality(c.Garmin)
		assert.Equal(t, c.Expected, result)
	}
}

func TestGarminApiEndpointLabel(t *testing.T) {
	cases := []struct {
		Endpoint GarminAPIEndpoint
		Expected string
	}{
		{Endpoint: BodyBatteryURL, Expected: "body battery"},
		{Endpoint: StressURL, Expected: "stress"},
		{Endpoint: RespirationURL, Expected: "respiration"},
		{Endpoint: SleepURL, Expected: "sleep"},
		{Endpoint: WeightURL, Expected: "weight"},
	}

	for _, c := range cases {
		result := garminApiToLabel(c.Endpoint)
		assert.Equal(t, c.Expected, result)
	}
}
