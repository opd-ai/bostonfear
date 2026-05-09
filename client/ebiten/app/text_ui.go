package app

import (
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
)

// Fallback to basicfont if embedded font fails
var uiTextFace font.Face = basicfont.Face7x13

// Font sizes for different UI contexts
const (
	defaultFontSize = 16.0 // pixels
	smallFontSize   = 12.0
	largeFontSize   = 20.0
)

func drawUIText(dst *ebiten.Image, value string, x, y int, clr color.Color) {
	metrics := uiTextFace.Metrics()
	baseline := y + metrics.Ascent.Ceil()
	text.Draw(dst, value, uiTextFace, x, baseline, clr)
}

// wrapText breaks a line into multiple lines to fit within maxWidth.
func wrapText(value string, maxWidth int) []string {
	if maxWidth <= 0 {
		return []string{value}
	}

	if value == "" {
		return []string{""}
	}

	words := strings.Fields(value)
	if len(words) == 0 {
		return []string{""}
	}

	lines := []string{}
	currentLine := ""

	for _, word := range words {
		/* Try adding the word to the current line */
		testLine := currentLine
		if testLine != "" {
			testLine += " "
		}
		testLine += word

		if textWidth(testLine) <= maxWidth {
			currentLine = testLine
		} else {
			/* Word doesn't fit; save current and start new */
			if currentLine != "" {
				lines = append(lines, currentLine)
			}
			currentLine = word
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines
}

// drawWrappedText renders multi-line text starting at (x, y), advancing
// y by line height for each line. Returns the final y position.
func drawWrappedText(dst *ebiten.Image, value string, maxWidth int, x, y int, clr color.Color) int {
	lines := wrapText(value, maxWidth)
	metrics := uiTextFace.Metrics()
	lineHeight := metrics.Height.Ceil()

	for _, line := range lines {
		drawUIText(dst, line, x, y, clr)
		y += lineHeight
	}

	return y
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

// textHeight returns the height in pixels for a single line of text using uiTextFace.
func textHeight() int {
	metrics := uiTextFace.Metrics()
	return metrics.Height.Ceil()
}
