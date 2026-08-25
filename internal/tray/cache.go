package tray

import (
	"fmt"
	"math"
	"strings"
	"time"

	"fyne.io/systray"

	"github.com/omjogani/usagely/internal/claude"
)

const maxCacheRows = 3

type cacheSection struct {
	divider *systray.MenuItem
	label   *systray.MenuItem
	rows    [maxCacheRows]*systray.MenuItem
}

func newCacheSection() *cacheSection {
	section := &cacheSection{
		divider: addDisabled(rule()),
		label:   addDisabled("Prompt cache"),
	}
	for i := range section.rows {
		section.rows[i] = addDisabled("")
	}
	section.hide()
	return section
}

func (c *cacheSection) hide() {
	c.divider.Hide()
	c.label.Hide()
	for _, row := range c.rows {
		row.Hide()
	}
}

func (c *cacheSection) refresh(now time.Time) {
	rows := cacheRows(claude.Sessions(now), now)
	if len(rows) == 0 {
		c.hide()
		return
	}

	c.divider.Show()
	c.label.Show()
	for i, item := range c.rows {
		if i >= len(rows) {
			item.Hide()
			continue
		}
		item.SetTitle(rows[i].title)
		item.SetTooltip(rows[i].tooltip)
		item.SetIcon(dotIcon(rows[i].urgency))
		item.Show()
	}
}

func rule() string {
	return strings.Repeat("─", int(math.Round(rowWidth/widthText)))
}

type cacheRow struct {
	title   string
	tooltip string
	urgency float64
}

func cacheRows(sessions []claude.Session, now time.Time) []cacheRow {
	var rows []cacheRow
	for _, session := range sessions {
		left := session.ExpiresIn(now)
		if left <= 0 {
			continue
		}
		if len(rows) == maxCacheRows-1 && len(sessions) > maxCacheRows {
			break
		}
		rows = append(rows, cacheRow{
			title:   align(session.Project, fmt.Sprintf("%s · %s", compactTokens(session.Cached), HumanDur(left))),
			tooltip: cacheTooltip(session, left),
			urgency: spent(left, session.TTL),
		})
		if len(rows) == maxCacheRows {
			break
		}
	}

	if hidden := countLive(sessions, now) - len(rows); hidden > 0 {
		rows = append(rows, cacheRow{
			title:   align(fmt.Sprintf("+%d more", hidden), ""),
			tooltip: "Run \"usagely sessions\" to see every warm cache",
			urgency: noData,
		})
	}
	return rows
}

func countLive(sessions []claude.Session, now time.Time) int {
	live := 0
	for _, session := range sessions {
		if session.ExpiresIn(now) > 0 {
			live++
		}
	}
	return live
}

func spent(left, ttl time.Duration) float64 {
	if ttl <= 0 {
		return noData
	}
	return (1 - left.Seconds()/ttl.Seconds()) * 100
}

func cacheTooltip(session claude.Session, left time.Duration) string {
	state := "Closed"
	action := "Resume it"
	if session.Running {
		state = "Running"
		action = "Send a message there"
	}
	return fmt.Sprintf("%s · %s tokens cached, warm for another %s. %s to keep reading the cache at 0.1x input price.",
		state, thousands(session.Cached), HumanDur(left), action)
}

func compactTokens(n int) string {
	if n < 1000 {
		return fmt.Sprint(n)
	}
	return fmt.Sprintf("%dk", (n+500)/1000)
}
