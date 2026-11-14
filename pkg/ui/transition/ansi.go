package transition

import (
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
)

var cond = func() *runewidth.Condition {
	c := runewidth.NewCondition()
	c.EastAsianWidth = false
	return c
}()

// FromANSI turns terminal rows written with SGR colour codes (as Bubble Tea
// views produce them) into cells, padded or cut to width.
func FromANSI(lines []string, width int) [][]Cell {
	rows := make([][]Cell, len(lines))
	for y, line := range lines {
		row := make([]Cell, 0, width)
		style := tcell.StyleDefault
		for i := 0; i < len(line); {
			if line[i] == 0x1b && i+1 < len(line) && line[i+1] == '[' {
				end := strings.IndexByte(line[i:], 'm')
				if end < 0 {
					break
				}
				style = applySGR(style, line[i+2:i+end])
				i += end + 1
				continue
			}
			r, size := decodeRune(line[i:])
			i += size
			w := cond.RuneWidth(r)
			if w == 0 {
				continue
			}
			if len(row)+w > width {
				break
			}
			row = append(row, Cell{R: r, Width: w, Style: style})
			for k := 1; k < w; k++ {
				row = append(row, Cell{Width: 0, Style: style}) // covered cell
			}
		}
		for len(row) < width {
			row = append(row, Cell{R: ' ', Width: 1})
		}
		rows[y] = row
	}
	return rows
}

func decodeRune(s string) (rune, int) {
	for _, r := range s {
		return r, len(string(r))
	}
	return ' ', 1
}

// applySGR applies the subset of SGR parameters the menu uses.
func applySGR(st tcell.Style, params string) tcell.Style {
	if params == "" {
		return tcell.StyleDefault
	}
	for _, p := range strings.Split(params, ";") {
		n, err := strconv.Atoi(p)
		if err != nil {
			continue
		}
		switch {
		case n == 0:
			st = tcell.StyleDefault
		case n == 1:
			st = st.Bold(true)
		case n >= 30 && n <= 37:
			st = st.Foreground(tcell.PaletteColor(n - 30))
		case n >= 90 && n <= 97:
			st = st.Foreground(tcell.PaletteColor(n - 90 + 8))
		case n == 39:
			st = st.Foreground(tcell.ColorDefault)
		}
	}
	return st
}
