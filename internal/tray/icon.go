package tray

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
)

const noData = -1.0

const (
	WarnAt     = 70.0
	CriticalAt = 90.0
)

var (
	colourTrack    = color.RGBA{R: 0x88, G: 0x8D, B: 0x96, A: 0x8C}
	colourUnknown  = color.RGBA{R: 0x88, G: 0x8D, B: 0x96, A: 0xFF}
	colourOK       = color.RGBA{R: 0x5B, G: 0xBF, B: 0x6A, A: 0xFF}
	colourWarn     = color.RGBA{R: 0xE8, G: 0x83, B: 0x3A, A: 0xFF}
	colourCritical = color.RGBA{R: 0xE5, G: 0x48, B: 0x3C, A: 0xFF}
)

func ringIcon(pct float64) []byte {
	const (
		size        = 64
		outerRadius = 30.0
		innerRadius = 20.0
	)

	img := image.NewRGBA(image.Rect(0, 0, size, size))
	centre := float64(size)/2 - 0.5
	fill := levelColour(pct)
	fraction := pct / 100

	for y := range size {
		for x := range size {
			dx, dy := float64(x)-centre, float64(y)-centre
			distance := math.Hypot(dx, dy)
			if distance < innerRadius || distance > outerRadius {
				continue
			}
			// Angle measured clockwise from 12 o'clock, normalised to 0..1.
			angle := math.Atan2(dx, -dy)
			if angle < 0 {
				angle += 2 * math.Pi
			}
			if pct > 0 && angle/(2*math.Pi) <= fraction {
				img.SetRGBA(x, y, fill)
			} else {
				img.SetRGBA(x, y, colourTrack)
			}
		}
	}

	return encodePNG(img)
}

func dotIcon(pct float64) []byte {
	const (
		size   = 64
		radius = 22.0
	)

	img := image.NewRGBA(image.Rect(0, 0, size, size))
	centre := float64(size)/2 - 0.5
	fill := levelColour(pct)

	for y := range size {
		for x := range size {
			if math.Hypot(float64(x)-centre, float64(y)-centre) <= radius {
				img.SetRGBA(x, y, fill)
			}
		}
	}

	return encodePNG(img)
}

func encodePNG(img image.Image) []byte {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil
	}
	return buf.Bytes()
}

func levelColour(pct float64) color.RGBA {
	switch {
	case pct < 0:
		return colourUnknown
	case pct >= CriticalAt:
		return colourCritical
	case pct >= WarnAt:
		return colourWarn
	default:
		return colourOK
	}
}
