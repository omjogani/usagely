package main

import (
	"strings"
	"testing"
	"time"

	"github.com/fatih/color"

	"github.com/omjogani/usagely/internal/claude"
	"github.com/omjogani/usagely/internal/tray"
)

func TestStatusRow(t *testing.T) {
	color.NoColor = true
	now := time.Unix(1000, 0)

	for name, tc := range map[string]struct {
		window *claude.Window
		want   string
	}{
		"live":         {&claude.Window{UsedPercent: 71, ResetsAt: 4600}, "5-Hour   ███████░░░   71%   resets in 1h 0m"},
		"past reset":   {&claude.Window{UsedPercent: 71, ResetsAt: 900}, "5-Hour   ░░░░░░░░░░    0%   reset"},
		"not reported": {nil, "5-Hour   not reported"},
	} {
		if got := statusRow("5-Hour", tc.window, now); got != tc.want {
			t.Errorf("%s:\n got %q\nwant %q", name, got, tc.want)
		}
	}
}

func TestLevelMatchesRingThresholds(t *testing.T) {
	color.NoColor = false
	defer func() { color.NoColor = true }()

	for pct, want := range map[float64]string{
		0:                   "32", // green
		tray.WarnAt:         "33", // yellow
		tray.CriticalAt:     "31", // red
		tray.CriticalAt - 1: "33",
	} {
		if got := level(pct)("x"); !strings.Contains(got, "\x1b["+want+"m") {
			t.Errorf("level(%v) = %q, want colour %s", pct, got, want)
		}
	}
}
