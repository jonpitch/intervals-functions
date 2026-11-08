package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"intervals-functions/api/garmin"
	"intervals-functions/api/intervals"
	"intervals-functions/utils/connect"
	"intervals-functions/utils/format"
	"intervals-functions/utils/ptr"
	"log"
	"math"
	"os"
	"strings"
	"time"
)

/*
backfill data from garmin connect to intervals.icu.

signature:
go run garmin_backfill.go -from=YYYY-MM-DD -to=YYYY-MM-DD -athleteId=XYZ -apiKey=ABC -dry-run daily,sleep,weight,hrv
*/
func main() {
	garminApiKey := os.Getenv("GARMIN_API_KEY")
	if garminApiKey == "" {
		log.Fatal("GARMIN_API_KEY is not set")
	}

	fromDateStr := flag.String("from", "", "start date of backfill (YYYY-MM-DD format)")
	toDateStr := flag.String("to", "", "last day of backfill (YYYY-MM-DD format)")
	intervalsAthleteID := flag.String("athleteId", "", "intervals.icu athlete ID")
	intervalsApiKey := flag.String("apiKey", "", "intervals.icu API key")
	dryRun := flag.Bool("dry-run", false, "Do not persist data to intervals.icu, just print")
	flag.Parse()

	args := flag.Args()
	if len(args) != 1 {
		log.Fatal("usage: garmin_backfill -from=X -to=Y -athleteId=A -apiKey=B --dry-run <garmin metrics>")
	}

	if intervalsApiKey == nil || intervalsAthleteID == nil {
		log.Fatal("intervals API key and athlete ID are required")
	}

	if fromDateStr == nil || toDateStr == nil {
		log.Fatal("from date and to date are required, and in the format of YYYY-MM-DD")
	}

	from, err := validDate(*fromDateStr)
	if err != nil {
		log.Fatal("from date is invalid, use YYYY-MM-DD format")
	}

	to, err := validDate(*toDateStr)
	if err != nil {
		log.Fatal("to date is invalid, use YYYY-MM-DD format")
	}

	garminClient, err := garmin.NewGarminClient(garminApiKey)
	if err != nil {
		log.Fatal(err)
	}

	dateRange := NewGarminDateRange(from, to)

	fetchDailySummary := false
	fetchSleep := false
	fetchWeight := false
	fetchHrv := false

	garminArgs := args[0]
	garminOptions := strings.Split(garminArgs, ",")
	for _, o := range garminOptions {
		switch o {
		case "daily":
			fetchDailySummary = true
		case "sleep":
			fetchSleep = true
		case "weight":
			fetchWeight = true
		case "hrv":
			fetchHrv = true
		default:
			log.Fatalf("unknown garmin option")
		}
	}

	records := map[format.GarminDate]intervals.WellnessRecord{}

	if fetchDailySummary {
		records, err = GarminApiDateRangeRequest(
			dateRange,
			garminClient.GetDailySummary,
			records,
			garminDailySummaryAccumulator,
		)

		if err != nil {
			log.Fatal(err)
		}
	}

	if fetchSleep {
		records, err = GarminApiDateRangeRequest(
			dateRange,
			garminClient.GetDailySleepSummariesByDate,
			records,
			garminDailySleepAccumulator,
		)

		if err != nil {
			log.Fatal(err)
		}
	}

	if fetchWeight {
		records, err = GarminApiDateRangeRequest(
			dateRange,
			garminClient.GetWeight,
			records,
			garminWeightAccumulator,
		)

		if err != nil {
			log.Fatal(err)
		}
	}

	if fetchHrv {
		records, err = GarminApiDateRangeRequest(
			dateRange,
			garminClient.GetHrvDailySummaries,
			records,
			garminHrvAccumulator,
		)

		if err != nil {
			log.Fatal(err)
		}
	}

	// collect wellness records to bulk update
	var wellness []intervals.WellnessRecord
	for _, w := range records {
		wellness = append(wellness, w)
		if *dryRun {
			data, err := json.MarshalIndent(w, "", "  ")
			if err != nil {
				fmt.Printf("error marshalling wellness record: %v", err)
			}
			fmt.Println(string(data))
		}
	}

	if !*dryRun {
		intervalsClient := intervals.NewIntervalsClient(intervals.APIURL, *intervalsApiKey, *intervalsAthleteID)
		err = intervalsClient.BulkUpdateWellnessRecord(wellness)
		if err != nil {
			log.Fatal(err)
		}
	}

	log.Println("done")
}

type GarminApiArrayResponseData interface {
	garmin.DailySleepResponse | garmin.DailyWeightSummary | garmin.DailySummaryData | garmin.HrvSummary
}

type GarminDateRange struct {
	From time.Time
	To   time.Time
}

// NewGarminDateRange API requests that take a date range
func NewGarminDateRange(from time.Time, to time.Time) GarminDateRange {
	return GarminDateRange{
		From: from,
		To:   to,
	}
}

// GarminApiDateRangeRequest make a garmin API request for a given date range and accumulate wellness data
func GarminApiDateRangeRequest[T GarminApiArrayResponseData](
	request GarminDateRange,
	apiRequest func(startDate time.Time, endDate time.Time) ([]T, error),
	wellness map[format.GarminDate]intervals.WellnessRecord,
	accumulate func([]T, map[format.GarminDate]intervals.WellnessRecord) map[format.GarminDate]intervals.WellnessRecord,
) (map[format.GarminDate]intervals.WellnessRecord, error) {
	response, err := apiRequest(request.From, request.To)
	if err != nil {
		return wellness, err
	}

	return accumulate(response, wellness), nil
}

// garminRespirationAccumulator converts DailySleepResponse records to intervals.WellnessRecord and accumulates
// them on the provided map
func garminDailySleepAccumulator(
	sleepResponses []garmin.DailySleepResponse,
	records map[format.GarminDate]intervals.WellnessRecord,
) map[format.GarminDate]intervals.WellnessRecord {
	for _, b := range sleepResponses {
		if record, exists := records[b.CalendarDate]; exists {
			record.Respiration = ptr.Float(b.AverageRespiration)
			record.SleepScore = ptr.Float(float64(b.SleepScores.Overall.Value))
			record.SleepSeconds = ptr.Int(b.SleepTimeSeconds)
			record.SleepQuality = connect.GarminSleepQualityToIntervalsSleepQuality(b.SleepScores.Overall.Qaulifier)
			record.OxygenSaturation = ptr.CoalesceFloat(b.AverageSpo2)
			record.SleepNeedMinutes = ptr.Int(b.SleepNeed.ActualMinutes)
			record.SleepRemTimeSeconds = ptr.Int(b.RemSleepSeconds)
			record.SleepDeepTimeSeconds = ptr.Int(b.DeepSleepSeconds)
			record.SleepLightTimeSeconds = ptr.Int(b.LightSleepSeconds)
			record.SleepAwakeTimeSeconds = ptr.Int(b.AwakeSleepSeconds)

			records[b.CalendarDate] = record
		} else {
			records[b.CalendarDate] = intervals.WellnessRecord{
				ID:                    intervals.WellnessRecordID(b.CalendarDate.Format("2006-01-02")),
				Respiration:           ptr.CoalesceFloat(b.AverageRespiration),
				SleepScore:            ptr.Float(float64(b.SleepScores.Overall.Value)),
				SleepSeconds:          ptr.Int(b.SleepTimeSeconds),
				SleepQuality:          connect.GarminSleepQualityToIntervalsSleepQuality(b.SleepScores.Overall.Qaulifier),
				OxygenSaturation:      ptr.CoalesceFloat(b.AverageSpo2),
				SleepNeedMinutes:      ptr.Int(b.SleepNeed.ActualMinutes),
				SleepRemTimeSeconds:   ptr.Int(b.RemSleepSeconds),
				SleepDeepTimeSeconds:  ptr.Int(b.DeepSleepSeconds),
				SleepLightTimeSeconds: ptr.Int(b.LightSleepSeconds),
				SleepAwakeTimeSeconds: ptr.Int(b.AwakeSleepSeconds),
			}
		}
	}

	return records
}

// garminWeightAccumulator converts WeightSummary records to intervals.WellnessRecord and accumulates
// them on the provided map
func garminWeightAccumulator(
	weight []garmin.DailyWeightSummary,
	records map[format.GarminDate]intervals.WellnessRecord,
) map[format.GarminDate]intervals.WellnessRecord {
	for _, w := range weight {
		weightInGrams := w.LatestWeight.WeightInGrams
		weight := weightInGrams / 1000
		muscleMassInGrams := w.LatestWeight.MuscleMassInGrams
		muscleMass := muscleMassInGrams / 1000
		if record, exists := records[w.LatestWeight.CalendarDate]; exists {
			record.Weight = ptr.Float(weight)
			if muscleMass != 0 {
				record.MuscleMass = ptr.Float(muscleMass)
			}
			records[w.LatestWeight.CalendarDate] = record
		} else {
			records[w.LatestWeight.CalendarDate] = intervals.WellnessRecord{
				ID:         intervals.WellnessRecordID(w.LatestWeight.CalendarDate.Format("2006-01-02")),
				Weight:     ptr.CoalesceFloat(weight),
				MuscleMass: ptr.CoalesceFloat(muscleMass),
			}
		}
	}

	return records
}

// garminDailySummaryAccumulator converts DailySummaryData records to intervals.WellnessRecord and accumulates
// them on the provided map
func garminDailySummaryAccumulator(
	battery []garmin.DailySummaryData,
	records map[format.GarminDate]intervals.WellnessRecord,
) map[format.GarminDate]intervals.WellnessRecord {
	for _, b := range battery {
		if record, exists := records[b.CalendarDate]; exists {
			record.BodyBatteryMin = ptr.Int(b.BodyBattery.MinValue)
			record.BodyBatteryMax = ptr.Int(b.BodyBattery.MaxValue)
			record.LowStressSeconds = ptr.Int(b.Stress.LowStressDurationInMillis / 1000)
			record.MediumStressSeconds = ptr.Int(b.Stress.MediumStressDurationInMillis / 1000)
			record.HighStressSeconds = ptr.Int(b.Stress.HighStressDurationInMillis / 1000)
			record.RestStressSeconds = ptr.Int(b.Stress.RestDurationInMillis / 1000)
			overall := connect.GarminStressToIntervalsStress(b.Stress.AverageLevel)
			record.Stress = &overall
			record.RestingHr = ptr.Int(int(math.Round(b.HeartRate.RestingValue)))
			records[b.CalendarDate] = record
		} else {
			stressLevel := connect.GarminStressToIntervalsStress(b.Stress.AverageLevel)
			records[b.CalendarDate] = intervals.WellnessRecord{
				ID:                  intervals.WellnessRecordID(b.CalendarDate.Format("2006-01-02")),
				BodyBatteryMin:      ptr.Int(b.BodyBattery.MinValue),
				BodyBatteryMax:      ptr.Int(b.BodyBattery.MaxValue),
				LowStressSeconds:    ptr.Int(b.Stress.LowStressDurationInMillis / 1000),
				MediumStressSeconds: ptr.Int(b.Stress.MediumStressDurationInMillis / 1000),
				HighStressSeconds:   ptr.Int(b.Stress.HighStressDurationInMillis / 1000),
				RestStressSeconds:   ptr.Int(b.Stress.RestDurationInMillis / 1000),
				Stress:              &stressLevel,
				RestingHr:           ptr.Int(int(math.Round(b.HeartRate.RestingValue))),
			}
		}
	}

	return records
}

// garminHrvAccumulator converts HrvSummary records to intervals.WellnessRecord and accumulates
// them on the provided map
func garminHrvAccumulator(
	hrv []garmin.HrvSummary,
	records map[format.GarminDate]intervals.WellnessRecord,
) map[format.GarminDate]intervals.WellnessRecord {
	for _, h := range hrv {
		if record, exists := records[h.CalendarDate]; exists {
			record.HrvRmssd = ptr.Float(h.LastNightAvg)
			records[h.CalendarDate] = record
		} else {
			records[h.CalendarDate] = intervals.WellnessRecord{
				ID:       intervals.WellnessRecordID(h.CalendarDate.Format("2006-01-02")),
				HrvRmssd: ptr.CoalesceFloat(h.LastNightAvg),
			}
		}
	}

	return records
}

// validDate checks the command supplied args for dates are in the correct format
func validDate(d string) (time.Time, error) {
	return time.Parse("2006-01-02", d)
}
