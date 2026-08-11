// Package tray shows the usage indicator via StatusNotifierItem over D-Bus.
package tray

import (
	"fmt"
	"time"

	"fyne.io/systray"

	"github.com/omjogani/usagely/internal/claude"
)

const refreshInterval = 10 * time.Second

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

	// Menu rows are plain text, so the header's colour has to be an image.
	items[rowHeader].SetIcon(dotIcon(panel))

	if panel > noData {
		systray.SetTitle(fmt.Sprintf("%.0f%%", panel))
	} else {
		systray.SetTitle("")
	}
	systray.SetIcon(ringIcon(panel))
}
