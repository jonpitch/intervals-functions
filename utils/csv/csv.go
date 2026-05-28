package csv

import (
	"encoding/csv"
	"fmt"
	"math"
	"strconv"
	"strings"
)

type CronometerDailyTotals struct {
	Kcal    *float64
	Carbs   *float64
	Protein *float64
	Fat     *float64
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

	// Cronometer exports a single header row followed by per-group rows, and (usually)
	// a "Total" row. Older exports sometimes included a second header near the end;
	// we don't rely on that.
	header := records[0]

	// Helper to find column index by name.
	colIndex := func(name string) (int, error) {
		normalize := func(s string) string {
			// Some exports include a UTF-8 BOM at the beginning of the first header.
			s = strings.TrimPrefix(s, "\ufeff")
			return strings.TrimSpace(s)
		}
		name = normalize(name)
		for i, h := range header {
			if normalize(h) == name {
				return i, nil
			}
		}
		return 0, fmt.Errorf("column %q not found", name)
	}

	// Find the totals row. Prefer Group == "Total"; fall back to the last row.
	var row []string
	if groupIdx, err := colIndex("Group"); err == nil {
		for _, rec := range records[1:] {
			if groupIdx < len(rec) && strings.TrimSpace(rec[groupIdx]) == "Total" {
				row = rec
				break
			}
		}
	}
	if row == nil {
		row = records[len(records)-1]
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

	parse := func(idx int) (*float64, error) {
		if idx >= len(row) {
			return nil, fmt.Errorf("index %d out of range", idx)
		}
		if row[idx] == "" {
			return nil, nil
		}
		val, err := strconv.ParseFloat(row[idx], 64)
		round := math.Round(val)
		return &round, err
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
