package sessions

import (
	"time"

	"github.com/gdamore/tcell/v2"
)

// Delete animation: a short red line flies from the left edge across the
// deleted row, erasing every character it passes; once it has left the right
// edge, the rows below slide up one by one to close the gap.
const (
	shotLength   = 4
	shotDuration = 300 * time.Millisecond
	shotFrame    = 12 * time.Millisecond
	slideStep    = 45 * time.Millisecond // time between two rows moving up
)

// red, because the row is being deleted
var shotStyle = tcell.StyleDefault.Foreground(tcell.ColorRed).Bold(true)

const listStartY = 7

// selectedRowY is the screen row of the selected session after a render.
func (v *SessionsView) selectedRowY() int {
	return listStartY + v.selectedIdx - v.scrollOffset
}

// animateShot flies a fixed-length line across screen row y. Everything to
// the left of the line's tail is already erased; the line itself never grows.
// The screen must already show the list.
func (v *SessionsView) animateShot(y int) {
	blank := tcell.StyleDefault
	start := time.Now()
	for {
		p := float64(time.Since(start)) / float64(shotDuration)
		if p > 1 {
			p = 1
		}
		head := int(p * float64(v.width+shotLength)) // column just past the tip
		tail := head - shotLength
		// behind the tail: erased (this also wipes the line's previous frame)
		for x := 0; x < min(max(tail, 0), v.width); x++ {
			v.screen.SetContent(x, y, ' ', nil, blank)
		}
		for x := max(tail, 0); x < min(head, v.width); x++ {
			v.screen.SetContent(x, y, '━', nil, shotStyle)
		}
		v.screen.Show()
		if p >= 1 {
			return
		}
		sleepUntil(start, time.Since(start)+shotFrame)
	}
}

// animateSlide closes the gap left at visible row gap: the rows below it move
// up one at a time, top first, so the blank row travels down and out of the
// list. Each step is scheduled from the start time rather than by sleeping a
// fixed amount, so timer jitter does not add up into a stutter. Call it after
// the session has been removed from the list.
func (v *SessionsView) animateSlide(gap int) {
	visible := v.listRows()
	start := time.Now()
	for step, at := 0, gap; at < visible; step, at = step+1, at+1 {
		sleepUntil(start, time.Duration(step)*slideStep)
		v.gapAt = at
		v.Render()
	}
	sleepUntil(start, time.Duration(visible-gap)*slideStep)
	v.gapAt = -1
	v.Render()
}

// sleepUntil waits until offset has passed since start.
func sleepUntil(start time.Time, offset time.Duration) {
	if d := time.Until(start.Add(offset)); d > 0 {
		time.Sleep(d)
	}
}

// listRows is how many sessions the list shows at the current size.
func (v *SessionsView) listRows() int {
	return min(maxListRows(v.height), len(v.displayedSessions)-v.scrollOffset)
}
