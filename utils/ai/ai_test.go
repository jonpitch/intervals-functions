package ai

import (
	"fmt"
	"testing"
	"time"

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

// TODO see what this structure actually looks like
// func TestExtractText(t *testing.T) {
// 	message := anthropic.Message{
// 		Content: []anthropic.ContentBlockUnion{
// 			{
// 				Type: "thinking",
// 				Text: "pondering...",
// 			},
// 			{
// 				Type: "text",
// 				Text: "model response",
// 			},
// 			{
// 				Type: "tool_use",
// 				Text: "used xyz tools",
// 			},
// 		},
// 	}
// 	result, err := ExtractText(&message)
// 	assert.NoError(t, err)
// 	assert.Equal(t, "model response", result)
// }
