package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"time"

	"fyne.io/systray"
)

// The menu is drawn by the desktop from a D-Bus description, so every row is
// plain text in the system UI font. One row per window keeps it readable
// without depending on column alignment a proportional font won't give us.
type menu struct {
	header  *systray.MenuItem
	five    *systray.MenuItem
	seven   *systray.MenuItem
	updated *systray.MenuItem
}

func runTray() {
	systray.Run(onReady, func() {})
}

func onReady() {
	m := menu{}
	m.header = addDisabled("Claude Code")
	systray.AddSeparator()
	m.five = addDisabled("5-Hour   —")
	m.seven = addDisabled("7-Day    —")
	systray.AddSeparator()
	m.updated = addDisabled("")
	quit := systray.AddMenuItem("Quit", "Stop Usagely")

	systray.SetTooltip("Usagely — Claude Code usage")
	m.refresh(time.Now())

	go func() {
		// One ticker covers both jobs: noticing a new snapshot, and ticking
		// the "resets in" countdowns down between snapshots.
		t := time.NewTicker(10 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				m.refresh(time.Now())
			case <-quit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

func addDisabled(title string) *systray.MenuItem {
	it := systray.AddMenuItem(title, "")
	it.Disable()
	return it
}

func (m menu) refresh(now time.Time) {
	s, err := readSnapshot()
	if err != nil {
		m.header.SetTitle("Claude Code — waiting for first session")
		m.five.SetTitle("5-Hour   —")
		m.seven.SetTitle("7-Day    —")
		m.updated.SetTitle("No data yet")
		systray.SetTitle("")
		systray.SetIcon(ringIcon(-1))
		return
	}

	worst := -1.0
	m.header.SetTitle("Claude Code")
	m.five.SetTitle(row("5-Hour ", s.FiveHour, now, &worst))
	m.seven.SetTitle(row("7-Day  ", s.SevenDay, now, &worst))
	m.updated.SetTitle("Updated " + time.Unix(s.UpdatedAt, 0).Format("15:04"))

	if worst >= 0 {
		systray.SetTitle(fmt.Sprintf("%.0f%%", worst))
	} else {
		systray.SetTitle("")
	}
	systray.SetIcon(ringIcon(worst))
}

// row renders one window, and tracks the highest percentage seen so the panel
// icon reflects whichever limit you are closest to hitting.
func row(label string, w *Window, now time.Time, worst *float64) string {
	pct, resetsIn, ok := live(w, now)
	if !ok {
		return label + "  —"
	}
	if pct > *worst {
		*worst = pct
	}
	return fmt.Sprintf("%s  %s  %3.0f%%   resets in %s", label, bar(pct), pct, humanDur(resetsIn))
}

// ---------------------------------------------------------------- icon

// ringIcon draws a donut gauge as PNG. Drawn at 64px and left for the panel to
// scale down, which smooths the edges for free. A negative pct means no data.
func ringIcon(pct float64) []byte {
	const (
		size   = 64
		rOuter = 30.0
		rInner = 20.0
	)
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	centre := float64(size)/2 - 0.5
	track := color.RGBA{R: 0x88, G: 0x8D, B: 0x96, A: 0x8C}
	fill := levelColour(pct)
	frac := pct / 100

	for y := range size {
		for x := range size {
			dx, dy := float64(x)-centre, float64(y)-centre
			d := math.Hypot(dx, dy)
			if d < rInner || d > rOuter {
				continue
			}
			// Angle measured clockwise from 12 o'clock, normalised to 0..1.
			a := math.Atan2(dx, -dy)
			if a < 0 {
				a += 2 * math.Pi
			}
			if pct > 0 && a/(2*math.Pi) <= frac {
				img.SetRGBA(x, y, fill)
			} else {
				img.SetRGBA(x, y, track)
			}
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil
	}
	return buf.Bytes()
}

func levelColour(pct float64) color.RGBA {
	switch {
	case pct < 0:
		return color.RGBA{R: 0x88, G: 0x8D, B: 0x96, A: 0xFF}
	case pct >= 90:
		return color.RGBA{R: 0xE5, G: 0x48, B: 0x3C, A: 0xFF}
	case pct >= 70:
		return color.RGBA{R: 0xE8, G: 0x83, B: 0x3A, A: 0xFF}
	default:
		return color.RGBA{R: 0x5B, G: 0xBF, B: 0x6A, A: 0xFF}
	}
}
