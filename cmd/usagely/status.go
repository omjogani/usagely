package main

import (
	"fmt"
	"time"

	"github.com/omjogani/usagely/internal/claude"
	"github.com/omjogani/usagely/internal/tray"
)

const statusRule = "────────────────────────────────────────────"

func runStatus() error {
	snapshot, err := claude.Read()
	if err != nil {
		return fmt.Errorf("no data yet (%s): run a Claude Code session first", claude.CachePath())
	}

	now := time.Now()
	_, panel := tray.Render(snapshot, now)
	captured := time.Unix(snapshot.UpdatedAt, 0)

	fmt.Println(bold("Claude Code"))
	fmt.Println(faint(statusRule))
	fmt.Println(statusRow("5-Hour", snapshot.FiveHour, now))
	fmt.Println(statusRow("7-Day", snapshot.SevenDay, now))
	fmt.Println(faint(statusRule))
	if panel >= 0 {
		fmt.Printf("%s %s\n", faint("panel     "), level(panel)(fmt.Sprintf("%.0f%%", panel)))
	}
	fmt.Printf("%s %s\n", faint("source    "), claude.CachePath())
	fmt.Printf("%s %s %s\n", faint("captured  "), captured.Format("15:04:05"),
		faint(fmt.Sprintf("(%s ago)", now.Sub(captured).Round(time.Second))))
	return nil
}

func statusRow(label string, w *claude.Window, now time.Time) string {
	pct, resetsIn, ok := w.Live(now)
	if !ok {
		return fmt.Sprintf("%-8s %s", label, faint("not reported"))
	}

	paint := level(pct)
	resets := "resets in " + tray.HumanDur(resetsIn)
	if resetsIn <= 0 {
		resets = "reset"
	}
	return fmt.Sprintf("%-8s %s %s   %s",
		label, paint(tray.Bar(pct)), paint(fmt.Sprintf("%4.0f%%", pct)), faint(resets))
}
