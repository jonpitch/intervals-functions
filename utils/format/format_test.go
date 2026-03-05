package format

import (
	"intervals-functions/utils/ptr"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFloatPtr(t *testing.T) {
	var a *float64
	b := ptr.Float(3.14)

	assert.Equal(t, "nil", FloatPtr(a))
	assert.Equal(t, "3.1", FloatPtr(b))
}
