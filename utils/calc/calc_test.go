package calc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAverage(t *testing.T) {
	empty := []float64{}
	result := Average(empty)
	assert.Equal(t, 0.0, result)

	numbers := []float64{1.1, 2.2, 3.3, 4.4}
	result = Average(numbers)
	assert.Equal(t, 2.75, result)
}

func TestRoundToTenth(t *testing.T) {
	cases := []struct {
		Val      float64
		Expected float64
	}{
		{Val: 1.123, Expected: 1.1},
		{Val: 2.009, Expected: 2.0},
		{Val: 3.901, Expected: 3.9},
		{Val: 4.999, Expected: 5.0},
		{Val: 5.099, Expected: 5.1},
		{Val: 6.0, Expected: 6.0},
		{Val: 7.949, Expected: 7.9},
	}

	for _, c := range cases {
		result := RoundToTenth(c.Val)
		assert.Equal(t, c.Expected, result)
	}
}
