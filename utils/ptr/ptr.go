package ptr

// Float is a helper to create a pointer to a float
func Float(f float64) *float64 {
	return &f
}

// Int is a helper to create a pointer to an int
func Int(i int) *int {
	return &i
}

func CoalesceFloat(f float64) *float64 {
	if f != 0.0 {
		return &f
	}

	return nil
}
