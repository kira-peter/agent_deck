// Package textwidth draws and measures text by terminal display width.
package textwidth

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
)

// cond measures display width the way terminals draw it: CJK is two cells, but
// "ambiguous" glyphs such as ─ and ↑ stay one cell even on a Chinese locale.
var cond = func() *runewidth.Condition {
	c := runewidth.NewCondition()
	c.EastAsianWidth = false
	return c
}()

// StringWidth returns the number of terminal cells text occupies.
func StringWidth(text string) int {
	return cond.StringWidth(text)
}

// Draw writes text at (x, y), advancing by each rune's display width so that
// wide (CJK) characters do not get a blank cell after them.
func Draw(screen tcell.Screen, x, y int, text string, style tcell.Style) {
	for _, r := range text {
		screen.SetContent(x, y, r, nil, style)
		if w := cond.RuneWidth(r); w > 1 {
			x += w
		} else {
			x++
		}
	}
}

// DrawCentered writes text centered on a screen of the given width.
func DrawCentered(screen tcell.Screen, width, y int, text string, style tcell.Style) {
	Draw(screen, max((width-StringWidth(text))/2, 0), y, text, style)
}

// PadRight pads text with spaces on the right up to width display cells.
func PadRight(text string, width int) string {
	if gap := width - StringWidth(text); gap > 0 {
		return text + strings.Repeat(" ", gap)
	}
	return text
}
