package ai

import (
	"fmt"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
)

// IntervalsUserContent is a helper function to concatenate some data together in a structured way for AI
func IntervalsUserContent(
	today time.Time,
	wellness string,
	windowAverages string,
	rollingAverages string,
	activities string,
) string {
	return fmt.Sprintf(
		`<today>%s</today><wellness>%s</wellness><averages>%s</averages><rolling-averages>%s</rolling-averages><activities>%s</activities>`,
		today,
		wellness,
		windowAverages,
		rollingAverages,
		activities,
	)
}

// ExtractModelResponse will pull out the claude response from an anthropic.Message
func ExtractModelResponse(message *anthropic.Message) (string, error) {
	var sb strings.Builder
	if message == nil {
		return "", fmt.Errorf("empty model response")
	}

	for _, block := range message.Content {
		switch variant := block.AsAny().(type) {
		case anthropic.TextBlock:
			fmt.Println(variant)
			sb.WriteString(variant.Text)
		default:
			continue
		}
	}

	if sb.Len() == 0 {
		return "", fmt.Errorf("no text content in response")
	}

	return sb.String(), nil
}

// GetUsageStats will return usage metrics for logging
func GetUsageStats(usage anthropic.Usage) string {
	return fmt.Sprintf(
		"tokens — input: %d, output: %d, cache_read: %d, cache_creation: %d",
		usage.InputTokens,
		usage.OutputTokens,
		usage.CacheReadInputTokens,
		usage.CacheCreationInputTokens,
	)
}
