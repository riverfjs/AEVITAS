package usagehud

import (
	"strings"
	"testing"

	"github.com/riverfjs/agentsdk-go/pkg/api"
)

func TestUsageBarEmojiLevels(t *testing.T) {
	tests := []struct {
		name  string
		ratio float64
		want  string
	}{
		{name: "0 percent", ratio: 0.0, want: "⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜"},
		{name: "25 percent", ratio: 0.25, want: "🟩🟩🟨⬜⬜⬜⬜⬜⬜⬜"},
		{name: "50 percent", ratio: 0.50, want: "🟩🟩🟩🟩🟩⬜⬜⬜⬜⬜"},
		{name: "75 percent", ratio: 0.75, want: "🟩🟩🟩🟩🟩🟩🟩🟨⬜⬜"},
		{name: "100 percent", ratio: 1.0, want: "🟩🟩🟩🟩🟩🟩🟩🟩🟩🟩"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := usageBar(tc.ratio, 10)
			if got != tc.want {
				t.Fatalf("usageBar(%v) = %q, want %q", tc.ratio, got, tc.want)
			}
		})
	}
}

func TestFormatIncludesEmojiBarAndTotals(t *testing.T) {
	stats := &api.SessionTokenStats{
		TotalInput:   120,
		TotalOutput:  30,
		TotalTokens:  150,
		CacheCreated: 10,
		CacheRead:    20,
	}
	got := Format("📊 Usage", stats, 70000, 200000) // 35%
	if !strings.Contains(got, "🟩🟩🟩🟨⬜⬜⬜⬜⬜⬜") {
		t.Fatalf("expected emoji usage bar in output, got %q", got)
	}
	if !strings.Contains(got, "Total billed tokens: 150") {
		t.Fatalf("expected totals in output, got %q", got)
	}
}
