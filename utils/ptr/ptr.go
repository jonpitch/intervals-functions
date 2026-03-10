package ptr

// Float is a helper to create a pointer to a float
func Float(f float64) *float64 {
	return &f
}

// Int is a helper to create a pointer to an int
func Int(i int) *int {
	return &i
}
