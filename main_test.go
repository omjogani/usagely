package main

import (
	"testing"
	"time"
)

// A trimmed copy of what Claude Code actually sends the status line.
const samplePayload = `{
  "model": {"display_name": "Opus"},
  "context_window": {"used_percentage": 8},
  "rate_limits": {
    "five_hour": {"used_percentage": 71, "resets_at": 1000},
    "seven_day": {"used_percentage": 10.4, "resets_at": 9000}
  }
}`

func TestCaptureRoundTrip(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	capture([]byte(samplePayload), time.Unix(500, 0))

	got, err := readSnapshot()
	if err != nil {
		t.Fatalf("readSnapshot: %v", err)
	}
	if got.FiveHour == nil || got.FiveHour.UsedPercent != 71 || got.FiveHour.ResetsAt != 1000 {
		t.Errorf("five_hour = %+v, want 71%% resetting at 1000", got.FiveHour)
	}
	if got.SevenDay == nil || got.SevenDay.UsedPercent != 10.4 {
		t.Errorf("seven_day = %+v, want 10.4%%", got.SevenDay)
	}
	if got.UpdatedAt != 500 {
		t.Errorf("updated_at = %d, want 500", got.UpdatedAt)
	}
}

// rate_limits is absent for non-subscription sessions and before the first API
// response of a session. Those payloads must not wipe a good snapshot.
func TestCaptureKeepsSnapshotWhenRateLimitsAbsent(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	capture([]byte(samplePayload), time.Unix(500, 0))
	capture([]byte(`{"model": {"display_name": "Opus"}}`), time.Unix(600, 0))
	capture([]byte(`not json at all`), time.Unix(700, 0))

	got, err := readSnapshot()
	if err != nil {
		t.Fatalf("readSnapshot: %v", err)
	}
	if got.FiveHour == nil || got.FiveHour.UsedPercent != 71 {
		t.Errorf("five_hour = %+v, want the earlier 71%% to survive", got.FiveHour)
	}
	if got.UpdatedAt != 500 {
		t.Errorf("updated_at = %d, want the snapshot untouched at 500", got.UpdatedAt)
	}
}

// Each window has a fixed end, so a stale percentage past resets_at reads as
// zero rather than over-reporting.
func TestLiveZeroesAfterReset(t *testing.T) {
	w := &Window{UsedPercent: 71, ResetsAt: 1000}

	pct, in, ok := live(w, time.Unix(400, 0))
	if !ok || pct != 71 || in != 600*time.Second {
		t.Errorf("before reset: got %v%% in %v (ok=%v), want 71%% in 10m", pct, in, ok)
	}

	pct, _, ok = live(w, time.Unix(1001, 0))
	if !ok || pct != 0 {
		t.Errorf("after reset: got %v%% (ok=%v), want 0%%", pct, ok)
	}

	if _, _, ok := live(nil, time.Unix(400, 0)); ok {
		t.Error("missing window reported as present")
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

func TestBar(t *testing.T) {
	cases := map[float64]string{
		0:   "░░░░░░░░░░",
		10:  "█░░░░░░░░░",
		71:  "███████░░░",
		100: "██████████",
		140: "██████████", // clamped rather than overflowing the menu row
	}
	for pct, want := range cases {
		if got := bar(pct); got != want {
			t.Errorf("bar(%v) = %q, want %q", pct, got, want)
		}
	}
}

// install stores the previous status line inside the settings.json command
// string, so quoting has to survive a round trip.
func TestShellQuoteRoundTrip(t *testing.T) {
	for _, s := range []string{
		"bash ~/.claude/statusline-command.sh",
		`sh -c 'echo hi'`,
		`awk '{print $1}' | tr -d "'"`,
	} {
		got, ok := shellUnquote(shellQuote(s))
		if !ok || got != s {
			t.Errorf("round trip of %q gave %q (ok=%v)", s, got, ok)
		}
	}
}
