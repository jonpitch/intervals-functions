package format

import (
	"fmt"
	"time"
)

// FloatPtr prints the value of a float pointer
func FloatPtr(f *float64) string {
	if f == nil {
		return "nil"
	}

	return fmt.Sprintf("%.1f", *f)
}

// GarminDate is a wrapper around time.Time that implements custom JSON marshaling and unmarshaling
type GarminDate struct {
	time.Time
}

const garminDateLayout = "2006-01-02"

func (d *GarminDate) UnmarshalJSON(b []byte) error {
	s := string(b)
	s = s[1 : len(s)-1]
	if s == "" {
		d.Time = time.Time{}
		return nil
	}
	parsed, err := time.Parse(garminDateLayout, s)
	if err != nil {
		return err
	}
	d.Time = parsed
	return nil
}

func (d GarminDate) MarshalJSON() ([]byte, error) {
	if d.IsZero() {
		return []byte("null"), nil
	}
	return []byte(`"` + d.Format(garminDateLayout) + `"`), nil
}
