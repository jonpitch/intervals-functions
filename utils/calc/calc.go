package calc

import "math"

// Average returns the average of a slice of floats
func Average(list []float64) float64 {
	if len(list) == 0 {
		return 0.0
	}

	sum := 0.0
	for _, l := range list {
		sum += l
	}

	return sum / float64(len(list))
}

// RoundToTenth rounds a float to the tenth place
func RoundToTenth(val float64) float64 {
	return math.Round(val*10) / 10
}
