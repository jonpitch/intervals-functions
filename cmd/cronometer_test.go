package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetYesterdayDateRange(t *testing.T) {
	now := time.Now()
	dateRange := getYesterdayDateRange(now)
	assert.Equal(t, time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, time.UTC), dateRange.Start)
	assert.Equal(t, time.Date(now.Year(), now.Month(), now.Day()-1, 23, 59, 59, 0, time.UTC), dateRange.End)
	assert.True(t, dateRange.Start.Before(dateRange.End))
}
