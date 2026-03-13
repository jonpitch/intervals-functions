package ptr

// Float is a helper to create a pointer to a float
func Float(f float64) *float64 {
	return &f
}

// Int is a helper to create a pointer to an int
func Int(i int) *int {
	return &i
}

// CoalesceFloat will return a pointer to a float if it's not equal to 0.0
func CoalesceFloat(f float64) *float64 {
	if f != 0.0 {
		return &f
	}

	return nil
}

// CoalesceInt will return a pointer to an int if it's not equal to 0
func CoalesceInt(i int) *int {
	if i != 0 {
		return &i
	}

	return nil
}
