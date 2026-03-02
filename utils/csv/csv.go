package csv

import (
	"encoding/csv"
	"fmt"
	"math"
	"strconv"
	"strings"
)

type CronometerDailyTotals struct {
	Kcal    *int
	Carbs   *int
	Protein *int
	Fat     *int
}

const (
	KcalHeader    = "Energy (kcal)"
	CarbsHeader   = "Carbs (g)"
	ProteinHeader = "Protein (g)"
	FatHeader     = "Fat (g)"
)

func ParseCronometerDailyTotals(csvData string) (*CronometerDailyTotals, error) {
	r := csv.NewReader(strings.NewReader(csvData))
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parse csv: %w", err)
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("csv does not contain daily totals row")
	}

	// The last two rows are the daily totals header and data.
	header := records[len(records)-2]
	row := records[len(records)-1]

	// Helper to find column index by name.
	colIndex := func(name string) (int, error) {
		for i, h := range header {
			if h == name {
				return i, nil
			}
		}
		return 0, fmt.Errorf("column %q not found", name)
	}

	kcalIdx, err := colIndex(KcalHeader)
	if err != nil {
		return nil, err
	}
	carbsIdx, err := colIndex(CarbsHeader)
	if err != nil {
		return nil, err
	}
	proteinIdx, err := colIndex(ProteinHeader)
	if err != nil {
		return nil, err
	}
	fatIdx, err := colIndex(FatHeader)
	if err != nil {
		return nil, err
	}

	parse := func(idx int) (*int, error) {
		if idx >= len(row) {
			return nil, fmt.Errorf("index %d out of range", idx)
		}
		if row[idx] == "" {
			return nil, nil
		}
		val, err := strconv.ParseFloat(row[idx], 64)
		asInt := int(math.Round(val))
		return &asInt, err
	}

	kcal, err := parse(kcalIdx)
	if err != nil {
		fmt.Println(fmt.Errorf("parse kcal: %w", err))
	}
	carbs, err := parse(carbsIdx)
	if err != nil {
		fmt.Println(fmt.Errorf("parse carbs: %w", err))
	}
	protein, err := parse(proteinIdx)
	if err != nil {
		fmt.Println(fmt.Errorf("parse protein: %w", err))
	}
	fat, err := parse(fatIdx)
	if err != nil {
		fmt.Println(fmt.Errorf("parse fat: %w", err))
	}

	return &CronometerDailyTotals{
		Kcal:    kcal,
		Carbs:   carbs,
		Protein: protein,
		Fat:     fat,
	}, nil
}
