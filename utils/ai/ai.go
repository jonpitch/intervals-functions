package ai

import (
	"fmt"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
)

// IntervalsUserContent is a helper function to concatenate some data together in a structured way for AI
func IntervalsUserContent(today time.Time, wellness string, activities string, events string) string {
	return fmt.Sprintf(
		`<today>%s</today><wellness>%s</wellness><activities>%s</activities><events>%s</events>`,
		today,
		wellness,
		activities,
		events,
	)
}

// ExtractText will pull out the claude response from an anthropic.Message
func ExtractText(message *anthropic.Message) (string, error) {
	var sb strings.Builder

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
