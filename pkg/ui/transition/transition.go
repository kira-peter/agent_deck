// Package transition plays the second half of the menu ⇄ view page
// transition on a tcell screen: the line the selected menu entry became
// splits in two that move to the top and bottom edges, uncovering the new
// view between them while the old menu stays visible outside them. Played
// backwards it closes over the view, ending on that line.
package transition

import (
	"time"

	"github.com/gdamore/tcell/v2"
)

// SplitDuration is how long the split takes.
const SplitDuration = 260 * time.Millisecond

const frameTime = 16 * time.Millisecond

var lineStyle = tcell.StyleDefault.Foreground(tcell.ColorLightGreen).Bold(true)

// Cell is one character of a captured screen.
type Cell struct {
	R     rune
	Width int
	Style tcell.Style
}

// Frame is a captured screen, row by row, with the row the line sits on.
type Frame struct {
	Rows [][]Cell
	Line int
}

// pending is the menu frame handed from the menu to the next view, nil when
// there is no transition to play.
var pending *Frame

// Hand stores the menu frame for the view that opens next.
func Hand(f *Frame) { pending = f }

// Take returns and clears the stored menu frame.
func Take() *Frame {
	f := pending
	pending = nil
	return f
}

// Capture copies what screen currently holds (after the view has drawn into
// it, before Show) so it can be blended with the menu frame.
func Capture(screen tcell.Screen) [][]Cell {
	w, h := screen.Size()
	rows := make([][]Cell, h)
	for y := range rows {
		rows[y] = make([]Cell, w)
		for x := 0; x < w; x++ {
			r, _, st, cw := screen.GetContent(x, y)
			rows[y][x] = Cell{R: r, Width: cw, Style: st}
		}
	}
	return rows
}

// ShowLine shows the menu frame as handed over (the full line), so the screen
// stays on it while the view loads.
func ShowLine(screen tcell.Screen, menu *Frame) {
	w, h := screen.Size()
	for y := 0; y < h; y++ {
		drawRow(screen, y, w, menu.Rows)
	}
	screen.Show()
}

// Open plays the split forward: view is uncovered from the line outwards. It
// ends on the whole view, with the lines gone off the edges.
func Open(screen tcell.Screen, menu *Frame, view [][]Cell) {
	play(screen, menu, view, false)
	w, h := screen.Size()
	for y := 0; y < h; y++ {
		drawRow(screen, y, w, view)
	}
	screen.Show()
}

// Close plays the split backwards: the lines close over view onto the line.
func Close(screen tcell.Screen, menu *Frame, view [][]Cell) {
	play(screen, menu, view, true)
}

func play(screen tcell.Screen, menu *Frame, view [][]Cell, reverse bool) {
	w, h := screen.Size()
	line := min(max(menu.Line, 0), h-1)
	start := time.Now()
	for {
		t := float64(time.Since(start)) / float64(SplitDuration)
		if t > 1 {
			t = 1
		}
		p := ease(t)
		if reverse {
			p = 1 - p
		}
		top := line - int(p*float64(line)+0.5)
		bottom := line + int(p*float64(h-1-line)+0.5)

		for y := 0; y < h; y++ {
			switch {
			case y == top || y == bottom:
				for x := 0; x < w; x++ {
					screen.SetContent(x, y, '━', nil, lineStyle)
				}
			case y > top && y < bottom:
				drawRow(screen, y, w, view)
			default:
				drawRow(screen, y, w, menu.Rows)
			}
		}
		screen.Show()
		if t >= 1 {
			return
		}
		next := start.Add(time.Duration(float64(SplitDuration) * t)).Add(frameTime)
		if d := time.Until(next); d > 0 {
			time.Sleep(d)
		}
	}
}

// drawRow copies row y of src onto the screen; missing cells are blank.
func drawRow(screen tcell.Screen, y, w int, src [][]Cell) {
	for x := 0; x < w; x++ {
		c := Cell{R: ' ', Width: 1}
		if y < len(src) && x < len(src[y]) {
			c = src[y][x]
		}
		if c.Width == 0 && c.R == 0 {
			continue // right half of a wide rune drawn just before
		}
		r := c.R
		if r == 0 {
			r = ' '
		}
		screen.SetContent(x, y, r, nil, c.Style)
		if c.Width > 1 {
			x += c.Width - 1 // the wide rune covers the next cell(s)
		}
	}
}

// ease is a smooth in-out curve on [0, 1].
func ease(t float64) float64 {
	t = min(max(t, 0), 1)
	if t < 0.5 {
		return 4 * t * t * t
	}
	f := 2*t - 2
	return 1 + f*f*f/2
}
