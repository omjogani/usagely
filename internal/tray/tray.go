// Package tray shows the usage indicator via StatusNotifierItem over D-Bus.
package tray

import (
	"fmt"
	"time"

	"fyne.io/systray"

	"github.com/omjogani/usagely/internal/claude"
)

const (
	refreshInterval = 10 * time.Second
	pollInterval    = time.Minute
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
	caches := newCacheSection()

	systray.AddSeparator()
	items[rowUpdated] = addDisabled("")

	refreshItem := systray.AddMenuItem("Refresh", "Fetch the latest usage now")
	quit := systray.AddMenuItem("Quit", "Stop Usagely")

	systray.SetTooltip("Usagely - Claude Code usage")
	redraw := func(now time.Time) {
		refreshFromCache(items, now)
		caches.refresh(now)
	}
	redraw(time.Now())

	warnings := newWarner()

	go func() {
		ticker := time.NewTicker(refreshInterval)
		poll := time.NewTicker(pollInterval)
		defer ticker.Stop()
		defer poll.Stop()

		fetchAndRefresh(items, redraw)
		warnings.check(time.Now())
		for {
			select {
			case <-ticker.C:
				redraw(time.Now())
			case <-poll.C:
				fetchAndRefresh(items, redraw)
				warnings.check(time.Now())
			case <-refreshItem.ClickedCh:
				fetchAndRefresh(items, redraw)
			case <-quit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

func fetchAndRefresh(items [rowCount]*systray.MenuItem, redraw func(time.Time)) {
	if snapshot, err := claude.Fetch(time.Now()); err == nil {
		_ = claude.Write(snapshot)
	}
	redraw(time.Now())
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

	// Menu rows are plain text, so the header's colour has to be an image.
	items[rowHeader].SetIcon(dotIcon(panel))

	if panel > noData {
		systray.SetTitle(fmt.Sprintf("%.0f%%", panel))
	} else {
		systray.SetTitle("")
	}
	systray.SetIcon(ringIcon(panel))
}
