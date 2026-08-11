// Package tray puts the usage indicator in the desktop panel.
//
// The indicator speaks StatusNotifierItem over D-Bus, which is the tray
// protocol KDE, XFCE, Cinnamon, COSMIC and GNOME's AppIndicator extension all
// understand. The popup is a D-Bus menu drawn by the desktop itself, so it
// lands correctly under the icon on both X11 and Wayland — at the cost of
// being plain text, since a menu is a list of strings rather than widgets.
package tray

import (
	"fmt"
	"strings"
	"time"

	"fyne.io/systray"

	"github.com/omjogani/usagely/internal/claude"
)

// refreshInterval covers two jobs at once: noticing that the hook wrote a new
// snapshot, and ticking the "resets in" countdowns down between snapshots.
const refreshInterval = 10 * time.Second

// Run displays the indicator and blocks until the user quits it.
func Run() {
	systray.Run(onReady, func() {})
}

// menu holds the rows we rewrite on every refresh. Each row is one window,
// because desktop menus render in a proportional font and columns split
// across separate rows would not line up.
type menu struct {
	header  *systray.MenuItem
	five    *systray.MenuItem
	seven   *systray.MenuItem
	updated *systray.MenuItem
}

func onReady() {
	m := menu{header: addDisabled("Claude Code")}
	systray.AddSeparator()
	m.five = addDisabled("5-Hour   —")
	m.seven = addDisabled("7-Day    —")
	systray.AddSeparator()
	m.updated = addDisabled("")
	// Refresh re-reads the cache now rather than waiting out the tick. It
	// cannot pull newer numbers than Claude Code has written, so the tooltip
	// says what it actually does.
	refreshItem := systray.AddMenuItem("Refresh", "Re-read the latest captured usage")
	quit := systray.AddMenuItem("Quit", "Stop Usagely")

	systray.SetTooltip("Usagely — Claude Code usage")
	m.refresh(time.Now())

	go func() {
		ticker := time.NewTicker(refreshInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				m.refresh(time.Now())
			case <-refreshItem.ClickedCh:
				m.refresh(time.Now())
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

// Render turns a snapshot into the four menu rows and the percentage the panel
// shows. Exported so `usagely status` prints exactly what the tray displays —
// without it, "the tray looks wrong" can only be debugged by squinting at it.
func Render(snapshot claude.Snapshot, now time.Time) (rows [4]string, panel float64) {
	panel = noData
	rows[0] = "Claude Code"
	rows[1] = row("5-Hour ", snapshot.FiveHour, now, &panel)
	rows[2] = row("7-Day  ", snapshot.SevenDay, now, &panel)
	rows[3] = "Updated " + time.Unix(snapshot.UpdatedAt, 0).Format("15:04")
	return rows, panel
}

func (m menu) refresh(now time.Time) {
	snapshot, err := claude.Read()
	if err != nil {
		m.header.SetTitle("Claude Code — waiting for first session")
		m.five.SetTitle("5-Hour   —")
		m.seven.SetTitle("7-Day    —")
		m.updated.SetTitle("No data yet")
		systray.SetTitle("")
		systray.SetIcon(ringIcon(noData))
		return
	}

	rows, panel := Render(snapshot, now)
	m.header.SetTitle(rows[0])
	m.five.SetTitle(rows[1])
	m.seven.SetTitle(rows[2])
	m.updated.SetTitle(rows[3])

	if panel > noData {
		systray.SetTitle(fmt.Sprintf("%.0f%%", panel))
	} else {
		systray.SetTitle("")
	}
	systray.SetIcon(ringIcon(panel))
}

// row renders one window and raises worst to its percentage, so the panel icon
// ends up reflecting whichever limit is closest to being hit.
func row(label string, w *claude.Window, now time.Time, worst *float64) string {
	pct, resetsIn, ok := w.Live(now)
	if !ok {
		return label + "  —"
	}
	*worst = max(*worst, pct)
	return fmt.Sprintf("%s  %s  %3.0f%%   resets in %s", label, bar(pct), pct, humanDur(resetsIn))
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
