package ptr

import intervals "intervals-functions/api"

// Float is a helper to create a pointer to a float
func Float(f float64) *float64 {
	return &f
}

// Int is a helper to create a pointer to an int
func Int(i int) *int {
	return &i
}

// StressLevel is a helper to create a pointer to a intervals.StressLevel
func StressLevel(s intervals.StressLevel) *intervals.StressLevel {
	return &s
}
