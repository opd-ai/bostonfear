package ui

import (
	"image/color"
	"math"
)

// ContrastRatio computes WCAG-style contrast ratio between two colors.
func ContrastRatio(a, b color.Color) float64 {
	al := relativeLuminance(a)
	bl := relativeLuminance(b)
	if al < bl {
		al, bl = bl, al
	}
	return (al + 0.05) / (bl + 0.05)
}

func relativeLuminance(c color.Color) float64 {
	r, g, b, _ := c.RGBA()
	rf := channelLuminance(float64(r>>8) / 255.0)
	gf := channelLuminance(float64(g>>8) / 255.0)
	bf := channelLuminance(float64(b>>8) / 255.0)
	return 0.2126*rf + 0.7152*gf + 0.0722*bf
}

func channelLuminance(v float64) float64 {
	if v <= 0.03928 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}
