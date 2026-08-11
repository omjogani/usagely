package tray

import (
	"image/color"
	"testing"
)

func TestIconsAreValidPNGs(t *testing.T) {
	for _, pct := range []float64{noData, 0, 71, 100} {
		for name, got := range map[string][]byte{"ringIcon": ringIcon(pct), "dotIcon": dotIcon(pct)} {
			if len(got) < 8 || string(got[1:4]) != "PNG" {
				t.Errorf("%s(%v) did not produce a PNG (%d bytes)", name, pct, len(got))
			}
		}
	}
}

func TestLevelColour(t *testing.T) {
	cases := map[float64]color.RGBA{
		noData:         colourUnknown,
		0:              colourOK,
		69.9:           colourOK,
		WarnAt:         colourWarn,
		CriticalAt - 1: colourWarn,
		CriticalAt:     colourCritical,
		100:            colourCritical,
	}
	for pct, want := range cases {
		if got := levelColour(pct); got != want {
			t.Errorf("levelColour(%v) = %v, want %v", pct, got, want)
		}
	}
}
