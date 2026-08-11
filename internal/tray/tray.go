// Package tray puts the usage indicator in the desktop panel.
//
// The indicator speaks StatusNotifierItem over D-Bus, which is the tray
// protocol KDE, XFCE, Cinnamon, COSMIC and GNOME's AppIndicator extension all
// understand. The popup is a D-Bus menu drawn by the desktop itself, so it
// lands correctly under the icon on both X11 and Wayland — at the cost of
// being plain text, since a menu row is a string rather than a widget.
//
// Each window takes two rows: label and percentage, then the bar and its
// countdown underneath. Values are right-aligned by estimating each row's
// rendered width, since the menu font is proportional and counting characters
// does not line anything up. See align.
package tray

import (
	"fmt"
	"math"
	"strings"
	"time"

	"fyne.io/systray"

	"github.com/omjogani/usagely/internal/claude"
)

// refreshInterval covers two jobs at once: noticing that the hook wrote a new
// snapshot, and ticking the "resets in" countdowns down between snapshots.
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

// Run displays the indicator and blocks until the user quits it.
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

	// Refresh re-reads the cache now rather than waiting out the tick. It
	// cannot pull newer numbers than Claude Code has written, so the tooltip
	// says what it actually does.
	refreshItem := systray.AddMenuItem("Refresh", "Re-read the latest captured usage")
	quit := systray.AddMenuItem("Quit", "Stop Usagely")

	systray.SetTooltip("Usagely — Claude Code usage")
	apply(items, time.Now())

	go func() {
		ticker := time.NewTicker(refreshInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				apply(items, time.Now())
			case <-refreshItem.ClickedCh:
				apply(items, time.Now())
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

// apply pushes a freshly rendered snapshot into the menu and panel icon.
func apply(items [rowCount]*systray.MenuItem, now time.Time) {
	snapshot, err := claude.Read()
	if err != nil {
		snapshot = claude.Snapshot{}
	}
	rows, panel := Render(snapshot, now)

	for i, text := range rows {
		items[i].SetTitle(text)
	}

	// The header carries the status dot as a menu icon. Menu rows are plain
	// text with no markup, so a coloured dot has to be an image; icon-data on
	// the row is the one piece of colour the D-Bus menu protocol allows.
	items[rowHeader].SetIcon(dotIcon(panel))

	if panel > noData {
		systray.SetTitle(fmt.Sprintf("%.0f%%", panel))
	} else {
		systray.SetTitle("")
	}
	systray.SetIcon(ringIcon(panel))
}

// Render turns a snapshot into the menu rows and the percentage the panel
// shows. Exported so `usagely status` prints exactly what the tray displays —
// without it, "the tray looks wrong" can only be debugged by squinting at it.
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

// window renders one rate-limit window as its label row and bar row, and
// raises panel to its percentage so the icon reflects whichever limit is
// closest to being hit.
func window(label string, w *claude.Window, now time.Time, panel *float64) (labelRow, barRow string) {
	pct, resetsIn, ok := w.Live(now)
	if !ok {
		return align(label, "—"), ""
	}
	*panel = max(*panel, pct)
	return align(label, fmt.Sprintf("%.0f%%", pct)),
		align(bar(pct), humanDur(resetsIn))
}

// align puts value at the right of a fixed column.
//
// Desktop menus render in a proportional font, so padding by character count
// does not line up: a block character is far wider than the space we pad with,
// which is why bar rows drift right of label rows. Instead we estimate each
// row's width and pad to a target.
//
// Widths are in units of one padding space, eyeballed against GNOME's default
// UI font. They are calibration, not truth — if your rows sit short or long,
// these four numbers are the knobs, and rowWidth is the one to try first.
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

func bar(pct float64) string {
	filled := min(max(int(pct/100*barCells+0.5), 0), barCells)
	return strings.Repeat("█", filled) + strings.Repeat("░", barCells-filled)
}

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
