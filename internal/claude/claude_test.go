package claude

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

func TestParse(t *testing.T) {
	got, ok := Parse([]byte(samplePayload), time.Unix(500, 0))
	if !ok {
		t.Fatal("Parse reported no rate limits in a payload that has them")
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
// response of a session. Those payloads must be reported as "nothing here" so
// the caller keeps its previous snapshot instead of blanking it.
func TestParseRejectsPayloadsWithoutRateLimits(t *testing.T) {
	for name, payload := range map[string]string{
		"no rate_limits": `{"model": {"display_name": "Opus"}}`,
		"empty object":   `{"rate_limits": {}}`,
		"not json":       `not json at all`,
		"empty input":    ``,
	} {
		if _, ok := Parse([]byte(payload), time.Unix(500, 0)); ok {
			t.Errorf("%s: Parse reported rate limits where there are none", name)
		}
	}
}

func TestCacheRoundTrip(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	want, _ := Parse([]byte(samplePayload), time.Unix(500, 0))
	if err := Write(want); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got.FiveHour == nil || got.FiveHour.UsedPercent != want.FiveHour.UsedPercent {
		t.Errorf("five_hour = %+v, want %+v", got.FiveHour, want.FiveHour)
	}
	if got.UpdatedAt != want.UpdatedAt {
		t.Errorf("updated_at = %d, want %d", got.UpdatedAt, want.UpdatedAt)
	}
}

// Each window has a fixed end, so a stale percentage past ResetsAt must read as
// zero rather than over-reporting between sessions.
func TestWindowLive(t *testing.T) {
	w := &Window{UsedPercent: 71, ResetsAt: 1000}

	pct, in, ok := w.Live(time.Unix(400, 0))
	if !ok || pct != 71 || in != 600*time.Second {
		t.Errorf("before reset: got %v%% in %v (ok=%v), want 71%% in 10m", pct, in, ok)
	}

	if pct, _, ok := w.Live(time.Unix(1001, 0)); !ok || pct != 0 {
		t.Errorf("after reset: got %v%% (ok=%v), want 0%%", pct, ok)
	}

	var absent *Window
	if _, _, ok := absent.Live(time.Unix(400, 0)); ok {
		t.Error("a missing window reported itself as present")
	}
}
