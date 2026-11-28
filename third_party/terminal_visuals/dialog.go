package visual

import (
	"strings"
	"time"
)

const DialogOpenDuration = 200 * time.Millisecond
const ButtonSweepDuration = 1100 * time.Millisecond

// DialogButton describes the label and emphasis of a framed action.
type DialogButton struct {
	Label  string
	Danger bool
}

// Dialog describes a centered panel with a row of keyboard-focused buttons.
// Negative animation ages render the settled appearance.
type Dialog struct {
	Title, Message, Hint string
	Buttons              []DialogButton
	Selected             int
	OpenAge, FocusAge    time.Duration
}

// DialogStyle colors a text span drawn on the panel surface.
type DialogStyle struct {
	Foreground, Background RGB
	Bold                   bool
}

// Draw emits layered spans in terminal coordinates. Callers provide text width
// and truncation so the layout uses the same cell metrics as their renderer.
func (d Dialog) Draw(width, height int, palette Palette, measure func(string) int, truncate func(string, int) string, paint func(x, y int, text string, style DialogStyle)) {
	if width < 1 || height < 1 || len(d.Buttons) == 0 {
		return
	}
	selected := min(max(d.Selected, 0), len(d.Buttons)-1)
	boxWidth, boxHeight := min(52, max(width-2, 1)), min(11, height)
	x, y := (width-boxWidth)/2, (height-boxHeight)/2
	style := func(fg RGB, bold bool) DialogStyle { return DialogStyle{fg, palette.Surface, bold} }
	center := func(col, row, span int, text string, ink DialogStyle) {
		text = truncate(text, span)
		paint(col+(span-measure(text))/2, row, text, ink)
	}
	if boxWidth < 8 || boxHeight < 5 {
		paint(x, height/2, strings.Repeat(" ", boxWidth), style(palette.Text, false))
		center(x, height/2, boxWidth, d.Buttons[selected].Label, style(palette.Accent, true))
		return
	}
	if d.OpenAge >= 0 && d.OpenAge < DialogOpenDuration {
		p := EaseInOutCubic(float64(d.OpenAge) / float64(DialogOpenDuration))
		top, bottom := SplitLines(boxHeight, p)
		for row := top + 1; row < bottom; row++ {
			paint(x, y+row, strings.Repeat(" ", boxWidth), style(palette.Text, false))
		}
		line := strings.Repeat("─", boxWidth)
		color := style(Mix(palette.Highlight, palette.Border, p), false)
		paint(x, y+top, line, color)
		if bottom != top {
			paint(x, y+bottom, line, color)
		}
		return
	}
	drawDialogBox(x, y, boxWidth, boxHeight, style(palette.Border, false), paint)
	title := " " + truncate(d.Title, boxWidth-6) + " "
	paint(x+2, y, title, style(palette.Text, true))
	if boxHeight >= 7 {
		center(x+2, y+boxHeight-2, boxWidth-4, d.Hint, style(palette.Muted, false))
	}
	gap := 4
	count := len(d.Buttons)
	maxButton := (boxWidth - 2 - gap*(count-1)) / count
	framed := boxHeight >= 9 && maxButton >= 6
	messageRow := min(3, boxHeight-4)
	if framed {
		messageRow = min(messageRow, boxHeight-7)
	}
	center(x+2, y+messageRow, boxWidth-4, d.Message, style(palette.Text, false))
	if !framed {
		gap = 1
		maxButton = (boxWidth - 2 - gap*(count-1)) / count
	}
	if maxButton < 3 {
		paint(x+1, y+boxHeight-2, strings.Repeat(" ", boxWidth-2), style(palette.Text, false))
		center(x+1, y+boxHeight-2, boxWidth-2, d.Buttons[selected].Label, style(palette.Accent, true))
		return
	}
	buttonWidth := 0
	for _, button := range d.Buttons {
		buttonWidth = max(buttonWidth, measure(button.Label)+8)
	}
	buttonWidth = min(buttonWidth+buttonWidth%2, maxButton)
	left := x + (boxWidth-count*buttonWidth-gap*(count-1))/2
	for i, button := range d.Buttons {
		fg := palette.Muted
		if i == selected {
			fg = palette.Accent
			if button.Danger {
				fg = palette.Danger
			}
		}
		col := left + i*(buttonWidth+gap)
		if !framed {
			center(col, y+boxHeight-3, buttonWidth, button.Label, style(fg, i == selected))
			continue
		}
		row := y + boxHeight - 6
		drawDialogBox(col, row, buttonWidth, 3, style(fg, false), paint)
		innerWidth := buttonWidth - 2
		label := truncate(button.Label, innerWidth)
		offset := (innerWidth - measure(label)) / 2
		band := (Sweep{Duration: ButtonSweepDuration, Radius: 3.2}).At(d.FocusAge, innerWidth)
		if i != selected {
			paint(col+1+offset, row+1, label, style(fg, false))
			continue
		}
		for _, r := range label {
			glyph := string(r)
			cells := measure(glyph)
			light := max(band.Strength(offset), band.Strength(offset+cells-1))
			paint(col+1+offset, row+1, glyph, style(Mix(fg, palette.Highlight, light), true))
			offset += cells
		}
	}
}

func drawDialogBox(x, y, width, height int, style DialogStyle, paint func(int, int, string, DialogStyle)) {
	line := strings.Repeat("─", width-2)
	blank := "│" + strings.Repeat(" ", width-2) + "│"
	paint(x, y, "╭"+line+"╮", style)
	for row := 1; row < height-1; row++ {
		paint(x, y+row, blank, style)
	}
	paint(x, y+height-1, "╰"+line+"╯", style)
}
