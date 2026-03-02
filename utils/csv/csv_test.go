package csv

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseDailyTotals_NoSummaryRow(t *testing.T) {
	const csvData = `Day,Group,Food Name
2026-02-26,"pre workout","Coffee"`

	_, err := ParseCronometerDailyTotals(csvData)
	assert.Errorf(t, err, "expected error for csv without summary row, got nil")
}

func TestParseDailyTotals_MissingColumns(t *testing.T) {
	// Missing the Protein (g) column in the header.
	const csvData = `Date,Energy (kcal),Carbs (g),Fat (g)
2026-02-26,2000,250,70`

	_, err := ParseCronometerDailyTotals(csvData)
	assert.Errorf(t, err, "expected error for csv missing required columns, got nil")
}

func TestParseDailyTotals_AllColumnsZero(t *testing.T) {
	const csvData = `Date,Energy (kcal),Carbs (g),Protein (g),Fat (g)
2026-02-26,,,,`

	totals, err := ParseCronometerDailyTotals(csvData)
	assert.NoError(t, err, "expected no error, got %v", err)
	assert.Nil(t, totals.Kcal)
	assert.Nil(t, totals.Carbs)
	assert.Nil(t, totals.Protein)
	assert.Nil(t, totals.Fat)
}

func TestParseDailyTotals_SomeColumnsZero(t *testing.T) {
	const csvData = `Date,Energy (kcal),Carbs (g),Protein (g),Fat (g)
2026-02-26,100.1,,300.7,`

	totals, err := ParseCronometerDailyTotals(csvData)
	assert.NoError(t, err, "expected no error, got %v", err)
	assert.Equal(t, 100, *totals.Kcal)
	assert.Nil(t, totals.Carbs)
	assert.Equal(t, 301, *totals.Protein)
	assert.Nil(t, totals.Fat)
}

func TestParseDailyTotals_AllColumns(t *testing.T) {
	const csvData = `Date,Energy (kcal),Carbs (g),Protein (g),Fat (g)
2026-02-26,100.1,200.2,300.5,400.99`

	totals, err := ParseCronometerDailyTotals(csvData)
	assert.NoError(t, err, "expected no error, got %v", err)
	assert.Equal(t, 100, *totals.Kcal)
	assert.Equal(t, 200, *totals.Carbs)
	assert.Equal(t, 301, *totals.Protein)
	assert.Equal(t, 401, *totals.Fat)
}
