package selector

import (
	"strings"
	"time"

	"github.com/kira-peter/agent_deck/pkg/ui/textwidth"
)

// The page transition between the menu and a view has two halves:
//
//  1. stretch (played here): the selected entry turns into a line, growing
//     from its centre to both edges of the window;
//  2. split (played by the view, see package ui/transition): the line splits
//     in two that move to the top and bottom edges, uncovering the view.
//
// The view loads while the menu still shows the finished line, so the split
// starts on real content. Coming back, the view plays the split backwards and
// the menu then shrinks the line back into the entry.
const (
	stretchDuration = 220 * time.Millisecond
	transitionTotal = stretchDuration

	lineRune  = '━'
	lineColor = "\033[1;32m"
)

type transition struct {
	start   time.Time
	reverse bool
}

// StartTransition begins the stretch. reverse plays it backwards (used when
// returning to the menu). It reports false, starting nothing, when the menu
// cannot be drawn at the current size.
func (s *BubbleTeaSelector) StartTransition(reverse bool) bool {
	if _, _, ok := s.compose(); !ok {
		return false
	}
	s.trans = &transition{start: time.Now(), reverse: reverse}
	return true
}

// Animating reports whether a transition is playing.
func (s *BubbleTeaSelector) Animating() bool { return s.trans != nil }

// TransitionDone reports whether the running transition has finished. A
// finished reverse transition is cleared so the menu shows again; a finished
// forward one is kept, so every later render still shows the full line until
// the program exits and the view takes over. Clearing it would let Bubble Tea
// draw one full menu frame right before quitting.
func (s *BubbleTeaSelector) TransitionDone() bool {
	if s.trans == nil {
		return true
	}
	if time.Since(s.trans.start) < transitionTotal {
		return false
	}
	if s.trans.reverse {
		s.trans = nil
	}
	return true
}

// LineFrame is the menu with the selected entry fully turned into a line, as
// terminal rows with ANSI colours, and the row of that line. It is the frame
// the stretch ends on and the split starts from.
func (s *BubbleTeaSelector) LineFrame() (lines []string, lineRow int, ok bool) {
	lines, selRow, ok := s.compose()
	if !ok {
		return nil, 0, false
	}
	lines[selRow] = s.stretchRow(1)
	return lines, selRow, true
}

// transitionView draws the current frame of the stretch over the menu.
func (s *BubbleTeaSelector) transitionView(lines []string, selRow int) string {
	elapsed := time.Since(s.trans.start)
	if elapsed > transitionTotal {
		elapsed = transitionTotal
	}
	if s.trans.reverse {
		elapsed = transitionTotal - elapsed
	}
	lines[selRow] = s.stretchRow(ease(float64(elapsed) / float64(stretchDuration)))
	return strings.Join(lines, "\n")
}

// stretchRow draws the selected entry with a line covering it from the
// entry's centre outwards; p = 0 shows the entry, p = 1 a full-width line.
func (s *BubbleTeaSelector) stretchRow(p float64) string {
	name := entryName(s.viewList[s.selectedIdx])
	text := "  " + name
	startX := s.entryStartX()
	cx := startX + 2 + textwidth.StringWidth(name)/2 // centre of the label itself
	half := int(p*float64(max(cx, s.width-cx)) + 0.5)
	left, right := max(cx-half, 0), min(cx+half, s.width)

	// lay the entry out on a row of cells
	cells := make([]string, s.width)
	for i := range cells {
		cells[i] = " "
	}
	x := startX
	for _, r := range text {
		w := textwidth.StringWidth(string(r))
		if x+w > s.width {
			break
		}
		cells[x] = string(r)
		for k := 1; k < w; k++ {
			cells[x+k] = "" // covered by the wide rune on its left
		}
		x += w
	}

	lineOn := right > left
	var b strings.Builder
	for x := 0; x < s.width; x++ {
		if lineOn && x >= left && x < right {
			if x == left {
				b.WriteString(lineColor)
			}
			b.WriteRune(lineRune)
			if x == right-1 {
				b.WriteString(ansiReset)
			}
			continue
		}
		c := cells[x]
		if c == "" {
			// the wide rune this cell belongs to was cut by the line
			if lineOn && x == right {
				b.WriteString(" ")
			}
			continue
		}
		if lineOn && textwidth.StringWidth(c) > 1 && x+1 == left {
			b.WriteString(" ") // the line would cover its right half
			continue
		}
		b.WriteString(ansiGreen + c + ansiReset)
	}
	return b.String()
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
