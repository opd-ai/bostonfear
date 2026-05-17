package app

import (
	"image/color"
	"log"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
)

type textScale string

const (
	textScaleDisplay textScale = "display-32"
	textScaleTitle   textScale = "title-24"
	textScaleHeader  textScale = "header-18"
	textScaleBody    textScale = "body-14"
	textScaleCaption textScale = "caption-12"
)

// Fallback to basicfont if TTF initialization fails.
var (
	uiTextFace  font.Face = basicfont.Face7x13
	uiTextFaces           = map[textScale]font.Face{
		textScaleDisplay: basicfont.Face7x13,
		textScaleTitle:   basicfont.Face7x13,
		textScaleHeader:  basicfont.Face7x13,
		textScaleBody:    basicfont.Face7x13,
		textScaleCaption: basicfont.Face7x13,
	}
)

// Font sizes for different UI contexts
const (
	defaultFontSize = 16.0 // pixels
	smallFontSize   = 12.0
	largeFontSize   = 20.0
)

func init() {
	base, err := opentype.Parse(goregular.TTF)
	if err != nil {
		log.Printf("ui text font parse failed, using bitmap fallback: %v", err)
		return
	}
	build := func(size float64) font.Face {
		face, ferr := opentype.NewFace(base, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull})
		if ferr != nil {
			log.Printf("ui text face init failed (size %.0f), using fallback: %v", size, ferr)
			return basicfont.Face7x13
		}
		return face
	}
	uiTextFaces = map[textScale]font.Face{
		textScaleDisplay: build(32),
		textScaleTitle:   build(24),
		textScaleHeader:  build(18),
		textScaleBody:    build(14),
		textScaleCaption: build(12),
	}
	uiTextFace = uiTextFaces[textScaleBody]
}

func drawUIText(dst *ebiten.Image, value string, x, y int, clr color.Color) {
	drawUITextScaled(dst, value, x, y, clr, textScaleBody)
}

func drawUITextScaled(dst *ebiten.Image, value string, x, y int, clr color.Color, scale textScale) {
	face := textFace(scale)
	metrics := face.Metrics()
	baseline := y + metrics.Ascent.Ceil()
	text.Draw(dst, value, face, x, baseline, clr)
}

func textFace(scale textScale) font.Face {
	if face, ok := uiTextFaces[scale]; ok && face != nil {
		return face
	}
	if uiTextFace != nil {
		return uiTextFace
	}
	return basicfont.Face7x13
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
func drawWrappedText(dst *ebiten.Image, value string, maxWidth, x, y int, clr color.Color) int {
	lines := wrapText(value, maxWidth)
	metrics := textFace(textScaleBody).Metrics()
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
	return font.MeasureString(textFace(textScaleBody), value).Ceil()
}

// textHeight returns the height in pixels for a single line of text using uiTextFace.
func textHeight() int {
	metrics := textFace(textScaleBody).Metrics()
	return metrics.Height.Ceil()
}
