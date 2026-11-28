package visual

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestDialogButtonSweepKeepsGeometry(t *testing.T) {
	dialog := Dialog{
		Title: "Confirm", Message: "Choose an action", Hint: "Enter",
		Buttons:  []DialogButton{{Label: "Back"}, {Label: "Quit", Danger: true}},
		Selected: 1, OpenAge: -1, FocusAge: -1,
	}
	type cell struct {
		rune
		style DialogStyle
	}
	const width, height = 70, 17
	render := func(d Dialog) [height][width]cell {
		var frame [height][width]cell
		d.Draw(width, height, Ink(), utf8.RuneCountInString, testDialogTruncate, func(x, y int, text string, style DialogStyle) {
			for _, glyph := range text {
				frame[y][x] = cell{glyph, style}
				x++
			}
		})
		return frame
	}
	rest := render(dialog)
	dialog.FocusAge = ButtonSweepDuration / 2
	sweep := render(dialog)
	changed := 0
	for y, row := range rest {
		for x, old := range row {
			next := sweep[y][x]
			if old.rune != next.rune || old.style.Background != next.style.Background {
				t.Fatal("button sweep should leave glyphs and surfaces in place")
			}
			if old.style.Foreground != next.style.Foreground {
				if old.style.Foreground != Ink().Danger || !strings.ContainsRune("Quit", old.rune) {
					t.Fatal("sweep should affect only the focused button label")
				}
				changed++
			}
		}
	}
	if changed == 0 {
		t.Fatal("expected visible sweep at the middle of its travel")
	}
	dialog.OpenAge = 0
	open := render(dialog)
	if open[height/2][width/2].rune != '─' || open[height/2-1][width/2].rune != 0 {
		t.Fatal("opening should start as a single centered line")
	}
}

func TestDialogFitsSmallTerminals(t *testing.T) {
	d := Dialog{Title: "Confirm", Message: "Choose an action", Buttons: []DialogButton{{Label: "Back"}, {Label: "Quit"}}, OpenAge: -1}
	for w := 1; w <= 58; w++ {
		for h := 1; h <= 12; h++ {
			d.Draw(w, h, Ink(), utf8.RuneCountInString, testDialogTruncate, func(x, y int, text string, style DialogStyle) {
				if x < 0 || y < 0 || x+utf8.RuneCountInString(text) > w || y >= h {
					t.Fatalf("span %q at (%d,%d) extends outside %dx%d", text, x, y, w, h)
				}
			})
		}
	}
}

func testDialogTruncate(text string, width int) string {
	runes := []rune(text)
	return string(runes[:min(width, len(runes))])
}
