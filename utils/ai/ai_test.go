package ai

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/stretchr/testify/assert"
)

func TestIntervalsUserContent(t *testing.T) {
	date := time.Date(2026, 1, 1, 12, 12, 12, 12, time.UTC)
	wellness := "wellness"
	events := "events"
	activities := "activities"

	result := IntervalsUserContent(date, wellness, activities, events)
	assert.Equal(
		t,
		fmt.Sprintf("<today>%s</today><wellness>%s</wellness><activities>%s</activities><events>%s</events>", date, wellness, activities, events),
		result,
	)
}

// mustContentBlock builds a ContentBlockUnion the same way the SDK does when
// it unmarshals a real API response: from raw JSON, not a struct literal.
// AsAny()/AsText() etc. read from the union's internal JSON.raw field, which
// only gets populated via UnmarshalJSON — setting Type/Text directly on a
// struct literal leaves JSON.raw empty and AsText() returns a zero TextBlock.
func mustContentBlock(t *testing.T, rawJSON string) anthropic.ContentBlockUnion {
	t.Helper()
	var block anthropic.ContentBlockUnion
	if err := json.Unmarshal([]byte(rawJSON), &block); err != nil {
		t.Fatalf("failed to unmarshal content block: %v", err)
	}
	return block
}

func TestExtractModelResponse(t *testing.T) {
	message := anthropic.Message{
		Content: []anthropic.ContentBlockUnion{
			mustContentBlock(t, `{"type":"thinking","thinking":"pondering..."}`),
			mustContentBlock(t, `{"type":"text","text":"model response"}`),
			mustContentBlock(t, `{"type":"tool_use","id":"toolu_1","name":"xyz","input":{}}`),
		},
	}

	result, err := ExtractModelResponse(&message)
	assert.NoError(t, err)
	assert.Equal(t, "model response", result)
}

func TestExtractModelResponse_NilMessage(t *testing.T) {
	result, err := ExtractModelResponse(nil)
	assert.Error(t, err)
	assert.Empty(t, result)
}

func TestExtractModelResponse_NoTextBlocks(t *testing.T) {
	message := anthropic.Message{
		Content: []anthropic.ContentBlockUnion{
			mustContentBlock(t, `{"type":"tool_use","id":"toolu_1","name":"xyz","input":{}}`),
		},
	}

	result, err := ExtractModelResponse(&message)
	assert.Error(t, err)
	assert.Empty(t, result)
}

func TestGetUsageStats(t *testing.T) {
	usage := anthropic.Usage{
		InputTokens:              1,
		OutputTokens:             2,
		CacheReadInputTokens:     3,
		CacheCreationInputTokens: 4,
	}
	result := GetUsageStats(usage)
	assert.Equal(t, "tokens — input: 1, output: 2, cache_read: 3, cache_creation: 4", result)
}
