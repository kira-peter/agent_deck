package visual

import "time"

// OptionList holds selection and scrolling for a keyboard-operated picker.
type OptionList struct {
	Selected int
	start    int
}

// Move wraps the selection when the user steps past an end of the list.
func (l *OptionList) Move(count, delta int) {
	if count > 0 {
		l.Selected = (l.Selected + delta) % count
		if l.Selected < 0 {
			l.Selected += count
		}
	}
}

// Visible returns a half-open range that keeps the selected row in view.
func (l *OptionList) Visible(count, height int) (start, end int) {
	if count <= 0 || height <= 0 {
		return 0, 0
	}
	l.Selected = min(max(l.Selected, 0), count-1)
	l.start = min(max(l.start, 0), max(count-height, 0))
	l.start = min(max(l.start, l.Selected-height+1), l.Selected)
	return l.start, min(count, l.start+height)
}

// OptionColors gives the foreground and background for a menu row.
func OptionColors(p Palette, selected, disabled bool) (fg, bg RGB) {
	fg, bg = p.Text, p.Surface
	if selected {
		fg, bg = p.Highlight, p.Selection
	}
	if disabled {
		fg = p.Muted
	}
	return fg, bg
}

// Blink returns a brief two-flash highlight, then settles to a static state.
func Blink(elapsed time.Duration) bool {
	return elapsed >= 0 && elapsed < 640*time.Millisecond && elapsed/(160*time.Millisecond)%2 == 0
}
