package format

import "fmt"

// FloatPtr prints the value of a float pointer
func FloatPtr(f *float64) string {
	if f == nil {
		return "nil"
	}

	return fmt.Sprintf("%.1f", *f)
}
