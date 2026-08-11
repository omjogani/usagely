package tray

import (
	"fmt"
	"time"

	"github.com/omjogani/usagely/internal/claude"
)

const (
	rowHeader = iota
	rowFiveLabel
	rowFiveBar
	rowSevenLabel
	rowSevenBar
	rowUpdated
	rowCount
)

// Render turns a snapshot into the menu rows and the percentage the panel
// shows. Exported so `usagely status` can print what the tray is displaying.
func Render(snapshot claude.Snapshot, now time.Time) (rows [rowCount]string, panel float64) {
	panel = noData

	if snapshot.UpdatedAt == 0 {
		return [rowCount]string{
			rowHeader:     "Claude Code",
			rowFiveLabel:  align("5-Hour", "—"),
			rowFiveBar:    "",
			rowSevenLabel: align("7-Day", "—"),
			rowSevenBar:   "",
			rowUpdated:    "Waiting for first session",
		}, panel
	}

	rows[rowHeader] = "Claude Code"
	rows[rowFiveLabel], rows[rowFiveBar] = windowRows("5-Hour", snapshot.FiveHour, now)
	rows[rowSevenLabel], rows[rowSevenBar] = windowRows("7-Day", snapshot.SevenDay, now)
	rows[rowUpdated] = "Updated " + time.Unix(snapshot.UpdatedAt, 0).Format("15:04")
	return rows, closestToLimit(now, snapshot.FiveHour, snapshot.SevenDay)
}

func windowRows(label string, w *claude.Window, now time.Time) (labelRow, barRow string) {
	pct, resetsIn, ok := w.Live(now)
	if !ok {
		return align(label, "—"), ""
	}
	return align(label, fmt.Sprintf("%.0f%%", pct)),
		align(Bar(pct), HumanDur(resetsIn))
}

func closestToLimit(now time.Time, windows ...*claude.Window) float64 {
	highest := noData
	for _, w := range windows {
		if pct, _, ok := w.Live(now); ok {
			highest = max(highest, pct)
		}
	}
	return highest
}
