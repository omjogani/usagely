package tray

import (
	"strings"
	"testing"
	"time"

	"github.com/omjogani/usagely/internal/claude"
)

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
