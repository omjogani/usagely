package tray

import (
	"image/color"
	"math"
	"strings"
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

// Rows must come out the same estimated width whether their left side is text
// or block characters — padding by character count is exactly what does not
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

func snapshot(now time.Time) claude.Snapshot {
	return claude.Snapshot{
		FiveHour:  &claude.Window{UsedPercent: 71, ResetsAt: now.Unix() + 6780},
		SevenDay:  &claude.Window{UsedPercent: 10, ResetsAt: now.Unix() + 550000},
		UpdatedAt: now.Unix(),
	}
}

// Each window occupies a label row and a bar row underneath it.
func TestRenderLayout(t *testing.T) {
	now := time.Unix(1786454714, 0)
	rows, panel := Render(snapshot(now), now)

	if want := align("5-Hour", "71%"); rows[rowFiveLabel] != want {
		t.Errorf("five label row = %q, want %q", rows[rowFiveLabel], want)
	}
	if want := align("███████░░░", "1h 53m"); rows[rowFiveBar] != want {
		t.Errorf("five bar row = %q, want %q", rows[rowFiveBar], want)
	}
	if want := align("█░░░░░░░░░", "6d 8h"); rows[rowSevenBar] != want {
		t.Errorf("seven bar row = %q, want %q", rows[rowSevenBar], want)
	}
	if !strings.HasPrefix(rows[rowUpdated], "Updated ") {
		t.Errorf("updated row = %q", rows[rowUpdated])
	}

	// The panel tracks whichever limit is closest to being hit.
	if panel != 71 {
		t.Errorf("panel = %v, want 71", panel)
	}
}

// An empty cache must not render as a confident zero.
func TestRenderWithoutData(t *testing.T) {
	rows, panel := Render(claude.Snapshot{}, time.Unix(1786454714, 0))

	if panel != noData {
		t.Errorf("panel = %v, want noData", panel)
	}
	if !strings.Contains(rows[rowFiveLabel], "—") {
		t.Errorf("five label row = %q, want a dash", rows[rowFiveLabel])
	}
	if rows[rowFiveBar] != "" || rows[rowSevenBar] != "" {
		t.Error("bar rows should be blank when there is nothing to draw")
	}
}

// A window the payload omitted shows a dash rather than a fabricated zero.
func TestRenderHandlesAbsentWindow(t *testing.T) {
	now := time.Unix(1786454714, 0)
	s := snapshot(now)
	s.SevenDay = nil

	rows, panel := Render(s, now)
	if want := align("7-Day", "—"); rows[rowSevenLabel] != want {
		t.Errorf("seven label row = %q, want %q", rows[rowSevenLabel], want)
	}
	if rows[rowSevenBar] != "" {
		t.Errorf("seven bar row = %q, want blank", rows[rowSevenBar])
	}
	if panel != 71 {
		t.Errorf("panel = %v, want the remaining window's 71", panel)
	}
}

func TestIconsAreValidPNGs(t *testing.T) {
	for _, pct := range []float64{noData, 0, 71, 100} {
		for name, got := range map[string][]byte{"ringIcon": ringIcon(pct), "dotIcon": dotIcon(pct)} {
			if len(got) < 8 || string(got[1:4]) != "PNG" {
				t.Errorf("%s(%v) did not produce a PNG (%d bytes)", name, pct, len(got))
			}
		}
	}
}

// The header dot is the only thing carrying urgency at a glance, so the
// boundaries it changes colour on are worth pinning down.
func TestLevelColour(t *testing.T) {
	cases := map[float64]color.RGBA{
		noData:         colourUnknown,
		0:              colourOK,
		69.9:           colourOK,
		WarnAt:         colourWarn,
		CriticalAt - 1: colourWarn,
		CriticalAt:     colourCritical,
		100:            colourCritical,
	}
	for pct, want := range cases {
		if got := levelColour(pct); got != want {
			t.Errorf("levelColour(%v) = %v, want %v", pct, got, want)
		}
	}
}
