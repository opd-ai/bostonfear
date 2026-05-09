package app

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
)

var uiTextFace font.Face = basicfont.Face7x13

func drawUIText(dst *ebiten.Image, value string, x, y int, clr color.Color) {
	metrics := uiTextFace.Metrics()
	baseline := y + metrics.Ascent.Ceil()
	text.Draw(dst, value, uiTextFace, x, baseline, clr)
}

func trimToWidth(value string, maxWidth int) string {
	if maxWidth <= 0 || value == "" {
		return ""
	}
	if textWidth(value) <= maxWidth {
		return value
	}

	const ellipsis = "..."
	ellipsisWidth := textWidth(ellipsis)
	if ellipsisWidth >= maxWidth {
		return ""
	}

	limit := maxWidth - ellipsisWidth
	runes := []rune(value)
	for len(runes) > 0 && textWidth(string(runes)) > limit {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + ellipsis
}

func textWidth(value string) int {
	return font.MeasureString(uiTextFace, value).Ceil()
}
