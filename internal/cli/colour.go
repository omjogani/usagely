package cli

import (
	"github.com/fatih/color"

	"github.com/omjogani/usagely/internal/tray"
)

var (
	bold   = color.New(color.Bold).SprintFunc()
	faint  = color.New(color.Faint).SprintFunc()
	cyan   = color.New(color.FgCyan).SprintFunc()
	green  = color.New(color.FgGreen).SprintFunc()
	yellow = color.New(color.FgYellow).SprintFunc()
	red    = color.New(color.FgRed).SprintFunc()
)

func level(pct float64) func(...any) string {
	switch {
	case pct >= tray.CriticalAt:
		return red
	case pct >= tray.WarnAt:
		return yellow
	default:
		return green
	}
}
