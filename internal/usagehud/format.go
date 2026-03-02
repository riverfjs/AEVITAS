package usagehud

import (
	"fmt"
	"strings"

	"github.com/riverfjs/agentsdk-go/pkg/api"
)

// Format builds a unified usage HUD text.
// inputTokens represents the token count used to estimate context window usage.
func Format(title string, stats *api.SessionTokenStats, inputTokens int, contextWindowTokens int) string {
	if stats == nil {
		return ""
	}
	bar := "N/A"
	percent := "N/A"
	detail := "N/A"
	if inputTokens > 0 && contextWindowTokens > 0 {
		ratio := float64(inputTokens) / float64(contextWindowTokens)
		if ratio < 0 {
			ratio = 0
		}
		pct := ratio * 100
		if pct > 999 {
			pct = 999
		}
		percent = fmt.Sprintf("%.1f%%", pct)
		detail = fmt.Sprintf("%d/%d", inputTokens, contextWindowTokens)
		bar = usageBar(ratio, 10)
	}
	return fmt.Sprintf(
		"%s\nContext window: %s %s (%s)\nTotal billed tokens: %d\nInput: %d | Output: %d | Cache: %d | Total: %d",
		title,
		bar,
		percent,
		detail,
		stats.TotalTokens,
		stats.TotalInput,
		stats.TotalOutput,
		stats.CacheRead+stats.CacheCreated,
		stats.TotalTokens,
	)
}

func usageBar(ratio float64, width int) string {
	if width <= 0 {
		return ""
	}
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	var b strings.Builder
	progress := ratio * float64(width)
	for i := 0; i < width; i++ {
		cellEnd := float64(i + 1)
		if progress >= cellEnd {
			b.WriteString("🟩")
			continue
		}
		if progress >= cellEnd-0.5 {
			b.WriteString("🟨")
			continue
		}
		b.WriteString("⬜")
	}
	return b.String()
}
