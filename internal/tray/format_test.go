package tray

import (
	"math"
	"strings"
	"testing"
	"time"
)

func TestBar(t *testing.T) {
	cases := map[float64]string{
		0:   "░░░░░░░░░░",
		10:  "█░░░░░░░░░",
		71:  "███████░░░",
		100: "██████████",
		140: "██████████", // clamped rather than overflowing the menu row
		-5:  "░░░░░░░░░░",
	}
	for pct, want := range cases {
		if got := Bar(pct); got != want {
			t.Errorf("Bar(%v) = %q, want %q", pct, got, want)
		}
	}
}

func TestHumanDur(t *testing.T) {
	cases := map[time.Duration]string{
		0:                      "now",
		-time.Minute:           "now",
		12 * time.Minute:       "12m",
		113 * time.Minute:      "1h 53m",
		(6*24 + 9) * time.Hour: "6d 9h",
	}
	for d, want := range cases {
		if got := HumanDur(d); got != want {
			t.Errorf("HumanDur(%v) = %q, want %q", d, got, want)
		}
	}
}

// Rows must come out the same estimated width whether their left side is text
// or block characters - padding by character count is exactly what does not
// work here, so this guards against regressing to it.
func TestAlignEqualisesRowWidth(t *testing.T) {
	for _, pair := range [][2]string{
		{"5-Hour", "71%"},
		{"5-Hour", "100%"},
		{"███████░░░", "1h 53m"},
		{"█░░░░░░░░░", "6d 9h"},
		{"7-Day", "—"},
	} {
		got := align(pair[0], pair[1])
		if width := estimateWidth(got); math.Abs(width-rowWidth) > widthText {
			t.Errorf("align(%q, %q) estimates %.1f wide, want %.1f", pair[0], pair[1], width, rowWidth)
		}
		if !strings.HasPrefix(got, pair[0]) || !strings.HasSuffix(got, pair[1]) {
			t.Errorf("align(%q, %q) = %q", pair[0], pair[1], got)
		}
	}
}
