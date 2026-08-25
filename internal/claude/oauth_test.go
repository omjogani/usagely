package claude

import (
	"os"
	"strconv"
	"testing"
	"time"
)

func TestParseUsage(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)

	t.Run("maps utilization and RFC 3339 resets", func(t *testing.T) {
		snap, err := ParseUsage([]byte(`{
			"five_hour": {"utilization": 4.0, "resets_at": "2026-08-25T21:09:59.745852+00:00"},
			"seven_day": {"utilization": 7.5, "resets_at": "2026-08-26T18:59:59+00:00"}
		}`), now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if snap.FiveHour.UsedPercent != 4 {
			t.Errorf("five-hour percent = %v, want 4", snap.FiveHour.UsedPercent)
		}
		if got, want := snap.FiveHour.ResetsAt, int64(1787692199); got != want {
			t.Errorf("five-hour reset = %d, want %d", got, want)
		}
		if snap.SevenDay.UsedPercent != 7.5 {
			t.Errorf("seven-day percent = %v, want 7.5", snap.SevenDay.UsedPercent)
		}
		if snap.UpdatedAt != now.Unix() {
			t.Errorf("updated at = %d, want %d", snap.UpdatedAt, now.Unix())
		}
	})

	t.Run("keeps a window whose reset is null", func(t *testing.T) {
		snap, err := ParseUsage([]byte(`{"five_hour": {"utilization": 0.0, "resets_at": null}}`), now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if snap.FiveHour == nil || snap.FiveHour.ResetsAt != 0 {
			t.Fatalf("five-hour = %+v, want a window with a zero reset", snap.FiveHour)
		}
		if snap.SevenDay != nil {
			t.Errorf("seven-day = %+v, want nil for an absent window", snap.SevenDay)
		}
	})

	t.Run("errors when no window is reported", func(t *testing.T) {
		if _, err := ParseUsage([]byte(`{"five_hour": null, "seven_day": null}`), now); err == nil {
			t.Error("expected an error for a response with no windows")
		}
	})

	t.Run("errors on malformed JSON", func(t *testing.T) {
		if _, err := ParseUsage([]byte(`not json`), now); err == nil {
			t.Error("expected an error for malformed JSON")
		}
	})
}

func TestAccessTokenRejectsExpired(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", dir)

	expired := time.Now().Add(-time.Hour).UnixMilli()
	write(t, dir+"/.credentials.json", `{"claudeAiOauth":{"accessToken":"sk-x","expiresAt":`+itoa(expired)+`}}`)

	if _, err := accessToken(time.Now()); err == nil {
		t.Error("expected an expired token to be rejected rather than sent")
	}

	valid := time.Now().Add(time.Hour).UnixMilli()
	write(t, dir+"/.credentials.json", `{"claudeAiOauth":{"accessToken":"sk-x","expiresAt":`+itoa(valid)+`}}`)

	token, err := accessToken(time.Now())
	if err != nil || token != "sk-x" {
		t.Errorf("accessToken = %q, %v; want the live token", token, err)
	}
}

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }
