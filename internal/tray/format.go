package tray

import (
	"fmt"
	"math"
	"strings"
	"time"
)

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

func Bar(pct float64) string {
	filled := min(max(int(pct/100*barCells+0.5), 0), barCells)
	return strings.Repeat("█", filled) + strings.Repeat("░", barCells-filled)
}

func HumanDur(d time.Duration) string {
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
