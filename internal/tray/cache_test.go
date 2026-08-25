package tray

import (
	"strings"
	"testing"
	"time"

	"github.com/omjogani/usagely/internal/claude"
)

func warmSession(project string, cached int, left time.Duration, running bool) claude.Session {
	now := time.Now()
	return claude.Session{
		Project:  project,
		Cached:   cached,
		TTL:      time.Hour,
		LastSeen: now.Add(left - time.Hour),
		Running:  running,
	}
}

func TestCacheRowsFormatting(t *testing.T) {
	now := time.Now()
	rows := cacheRows([]claude.Session{warmSession("Usagely", 180100, 59*time.Minute, true)}, now)

	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	if !strings.HasPrefix(rows[0].title, "Usagely") {
		t.Errorf("title = %q, want it to lead with the project", rows[0].title)
	}
	if !strings.Contains(rows[0].title, "180k") {
		t.Errorf("title = %q, want a compact token count", rows[0].title)
	}
	if !strings.HasSuffix(rows[0].title, "58m") && !strings.HasSuffix(rows[0].title, "59m") {
		t.Errorf("title = %q, want the countdown at the right edge", rows[0].title)
	}
	if !strings.Contains(rows[0].tooltip, "Running") || !strings.Contains(rows[0].tooltip, "180,100") {
		t.Errorf("tooltip = %q, want state and the exact token count", rows[0].tooltip)
	}
}

func TestCacheRowsTooltipTellsClosedSessionsToResume(t *testing.T) {
	rows := cacheRows([]claude.Session{warmSession("shapehill", 197771, 8*time.Minute, false)}, time.Now())
	if !strings.Contains(rows[0].tooltip, "Resume it") {
		t.Errorf("tooltip = %q, want it to say to resume", rows[0].tooltip)
	}
}

func TestCacheRowsUrgencyRisesTowardExpiry(t *testing.T) {
	now := time.Now()
	fresh := cacheRows([]claude.Session{warmSession("A", 50000, 55*time.Minute, true)}, now)[0]
	dying := cacheRows([]claude.Session{warmSession("B", 50000, 3*time.Minute, true)}, now)[0]

	if fresh.urgency >= WarnAt {
		t.Errorf("fresh urgency = %.0f, want below the warn threshold", fresh.urgency)
	}
	if dying.urgency < CriticalAt {
		t.Errorf("dying urgency = %.0f, want at or above critical", dying.urgency)
	}
}

func TestCacheRowsSummarisesOverflowInsteadOfDroppingIt(t *testing.T) {
	now := time.Now()
	sessions := []claude.Session{
		warmSession("A", 50000, 5*time.Minute, true),
		warmSession("B", 50000, 15*time.Minute, true),
		warmSession("C", 50000, 25*time.Minute, true),
		warmSession("D", 50000, 35*time.Minute, true),
		warmSession("E", 50000, 45*time.Minute, true),
	}
	rows := cacheRows(sessions, now)

	if len(rows) != maxCacheRows {
		t.Fatalf("got %d rows, want %d", len(rows), maxCacheRows)
	}
	if !strings.HasPrefix(rows[len(rows)-1].title, "+3 more") {
		t.Errorf("last row = %q, want it to account for the 3 rows it stands in for", rows[len(rows)-1].title)
	}
}

func TestCacheRowsIgnoresLapsedCaches(t *testing.T) {
	now := time.Now()
	sessions := []claude.Session{
		warmSession("Gone", 90000, -2*time.Minute, false),
		warmSession("Warm", 90000, 20*time.Minute, true),
	}
	rows := cacheRows(sessions, now)

	if len(rows) != 1 || !strings.HasPrefix(rows[0].title, "Warm") {
		t.Fatalf("rows = %+v, want only the warm session", rows)
	}
}

func TestCacheRowsEmptyWhenNothingIsWarm(t *testing.T) {
	if rows := cacheRows(nil, time.Now()); len(rows) != 0 {
		t.Errorf("got %d rows, want none so the section stays hidden", len(rows))
	}
}

func TestRuleMatchesRowWidth(t *testing.T) {
	got := estimateWidth(rule())
	if got < rowWidth-widthText || got > rowWidth+widthText {
		t.Errorf("rule renders at width %.1f, want about %.1f so it spans the menu", got, rowWidth)
	}
}
