// Package tray shows the usage indicator via StatusNotifierItem over D-Bus.
package tray

import (
	"fmt"
	"math"
	"strings"
	"time"

	"fyne.io/systray"

	"github.com/omjogani/usagely/internal/claude"
)

const refreshInterval = 10 * time.Second

// Row positions in the menu.
const (
	rowHeader = iota
	rowFiveLabel
	rowFiveBar
	rowSevenLabel
	rowSevenBar
	rowUpdated
	rowCount
)

func Run() {
	systray.Run(onReady, func() {})
}

func onReady() {
	items := [rowCount]*systray.MenuItem{}
	items[rowHeader] = addDisabled("Claude Code")
	systray.AddSeparator()
	items[rowFiveLabel] = addDisabled("5-Hour")
	items[rowFiveBar] = addDisabled("")
	items[rowSevenLabel] = addDisabled("7-Day")
	items[rowSevenBar] = addDisabled("")
	systray.AddSeparator()
	items[rowUpdated] = addDisabled("")

	refreshItem := systray.AddMenuItem("Refresh", "Re-read the latest captured usage")
	quit := systray.AddMenuItem("Quit", "Stop Usagely")

	systray.SetTooltip("Usagely — Claude Code usage")
	refreshFromCache(items, time.Now())

	go func() {
		ticker := time.NewTicker(refreshInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				refreshFromCache(items, time.Now())
			case <-refreshItem.ClickedCh:
				refreshFromCache(items, time.Now())
			case <-quit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

func addDisabled(title string) *systray.MenuItem {
	item := systray.AddMenuItem(title, "")
	item.Disable()
	return item
}

func refreshFromCache(items [rowCount]*systray.MenuItem, now time.Time) {
	snapshot, err := claude.Read()
	if err != nil {
		snapshot = claude.Snapshot{}
	}
	rows, panel := Render(snapshot, now)

	for i, text := range rows {
		items[i].SetTitle(text)
	}

	items[rowHeader].SetIcon(dotIcon(panel))

	if panel > noData {
		systray.SetTitle(fmt.Sprintf("%.0f%%", panel))
	} else {
		systray.SetTitle("")
	}
	systray.SetIcon(ringIcon(panel))
}

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
	rows[rowFiveLabel], rows[rowFiveBar] = window("5-Hour", snapshot.FiveHour, now, &panel)
	rows[rowSevenLabel], rows[rowSevenBar] = window("7-Day", snapshot.SevenDay, now, &panel)
	rows[rowUpdated] = "Updated " + time.Unix(snapshot.UpdatedAt, 0).Format("15:04")
	return rows, panel
}

func window(label string, w *claude.Window, now time.Time, panel *float64) (labelRow, barRow string) {
	pct, resetsIn, ok := w.Live(now)
	if !ok {
		return align(label, "—"), ""
	}
	*panel = max(*panel, pct)
	return align(label, fmt.Sprintf("%.0f%%", pct)),
		align(bar(pct), humanDur(resetsIn))
}

const (
	padSpace   = " " // one plain space: the unit the widths below are measured in
	widthBlock = 2.6 // █ and ░
	widthText  = 1.9 // letters, digits, punctuation
	rowWidth   = 52.0
)

func align(label, value string) string {
	gap := int(math.Round(rowWidth - estimateWidth(label) - estimateWidth(value)))
	return label + strings.Repeat(padSpace, max(gap, 1)) + value
}

func estimateWidth(s string) float64 {
	var total float64
	for _, r := range s {
		switch r {
		case ' ':
			total++
		case '█', '░':
			total += widthBlock
		default:
			total += widthText
		}
	}
	return total
}

const barCells = 10

func Bar(pct float64) string { return bar(pct) }

func bar(pct float64) string {
	filled := min(max(int(pct/100*barCells+0.5), 0), barCells)
	return strings.Repeat("█", filled) + strings.Repeat("░", barCells-filled)
}

func HumanDur(d time.Duration) string { return humanDur(d) }

func humanDur(d time.Duration) string {
	if d <= 0 {
		return "now"
	}
	hours := int(d.Hours())
	switch {
	case hours >= 24:
		return fmt.Sprintf("%dd %dh", hours/24, hours%24)
	case hours >= 1:
		return fmt.Sprintf("%dh %dm", hours, int(d.Minutes())%60)
	default:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
}
