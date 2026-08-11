package tray

import (
	"testing"
	"time"

	"github.com/omjogani/usagely/internal/claude"
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
		if got := bar(pct); got != want {
			t.Errorf("bar(%v) = %q, want %q", pct, got, want)
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
		if got := humanDur(d); got != want {
			t.Errorf("humanDur(%v) = %q, want %q", d, got, want)
		}
	}
}

// The panel icon should reflect whichever limit is closest to being hit, so a
// row raises worst but never lowers it.
func TestRowTracksWorst(t *testing.T) {
	now := time.Unix(1000, 0)
	five := &claude.Window{UsedPercent: 71, ResetsAt: 1000 + 6780}
	seven := &claude.Window{UsedPercent: 10, ResetsAt: 1000 + 550000}

	worst := noData
	if got := row("5-Hour ", five, now, &worst); got != "5-Hour   ███████░░░   71%   resets in 1h 53m" {
		t.Errorf("row = %q", got)
	}
	if worst != 71 {
		t.Errorf("worst = %v, want 71", worst)
	}

	row("7-Day  ", seven, now, &worst)
	if worst != 71 {
		t.Errorf("worst = %v after a lower window, want it to stay at 71", worst)
	}

	var absent *claude.Window
	if got := row("7-Day  ", absent, now, &worst); got != "7-Day    —" {
		t.Errorf("absent window rendered as %q", got)
	}
}

func TestRingIconIsValidPNG(t *testing.T) {
	for _, pct := range []float64{noData, 0, 71, 100} {
		got := ringIcon(pct)
		if len(got) < 8 || string(got[1:4]) != "PNG" {
			t.Errorf("ringIcon(%v) did not produce a PNG (%d bytes)", pct, len(got))
		}
	}
}
