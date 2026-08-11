// Package claude reads Claude Code's local usage data.
//
// Claude Code passes a JSON payload to whatever command is configured as its
// status line, and that payload carries the current rate-limit windows. This
// package parses that payload, persists it, and owns the settings.json wiring
// that puts us in the status line slot.
//
// Nothing here reads credentials or makes a network call. That is deliberate:
// Anthropic's OAuth refresh tokens are single-use and rotating, so a tool that
// refreshes them races Claude Code and logs the user out.
package claude

import (
	"encoding/json"
	"time"
)

// Window is one rate-limit window as Claude Code reports it.
type Window struct {
	UsedPercent float64 `json:"used_percentage"`
	ResetsAt    int64   `json:"resets_at"` // unix seconds
}

// Snapshot is a point-in-time reading of every window, as persisted for the
// tray to read back.
type Snapshot struct {
	FiveHour  *Window `json:"five_hour,omitempty"`
	SevenDay  *Window `json:"seven_day,omitempty"`
	UpdatedAt int64   `json:"updated_at"`
}

// Live reports a window's state as of now.
//
// Each window has a fixed end rather than rolling, so once ResetsAt has passed
// the correct reading is zero — holding the stale percentage would over-report
// between sessions. ok is false for a window the payload did not include.
//
// The nil receiver is valid, so callers can pass an absent window straight in.
func (w *Window) Live(now time.Time) (pct float64, resetsIn time.Duration, ok bool) {
	if w == nil {
		return 0, 0, false
	}
	until := time.Unix(w.ResetsAt, 0).Sub(now)
	if until <= 0 {
		return 0, 0, true
	}
	return w.UsedPercent, until, true
}

// Parse extracts the rate-limit windows from a status line payload.
//
// ok is false when the payload carries no rate limits at all, which happens
// for non-subscription sessions and before the first API response of a
// session. Callers should keep their previous snapshot in that case rather
// than storing an empty one.
func Parse(input []byte, now time.Time) (snap Snapshot, ok bool) {
	var payload struct {
		RateLimits struct {
			FiveHour *Window `json:"five_hour"`
			SevenDay *Window `json:"seven_day"`
		} `json:"rate_limits"`
	}
	if err := json.Unmarshal(input, &payload); err != nil {
		return Snapshot{}, false
	}
	if payload.RateLimits.FiveHour == nil && payload.RateLimits.SevenDay == nil {
		return Snapshot{}, false
	}
	return Snapshot{
		FiveHour:  payload.RateLimits.FiveHour,
		SevenDay:  payload.RateLimits.SevenDay,
		UpdatedAt: now.Unix(),
	}, true
}
