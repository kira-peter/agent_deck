// Package rowfx holds the list-row delete animation shared by the list views:
// a short red line flies from the left edge across the deleted row, erasing
// every character it passes; once it has left the right edge, the rows below
// slide up one by one to close the gap.
package rowfx

import (
	"time"

	"github.com/gdamore/tcell/v2"
)

const (
	shotLength   = 4
	shotDuration = 300 * time.Millisecond
	shotFrame    = 12 * time.Millisecond
	// SlideStep is the time between two rows moving up.
	SlideStep = 45 * time.Millisecond
)

// red, because the row is being deleted
var shotStyle = tcell.StyleDefault.Foreground(tcell.ColorRed).Bold(true)

// Shot flies a fixed-length line across screen row y. Everything to the left
// of the line's tail is erased; the line itself never grows. The screen must
// already show the list.
func Shot(screen tcell.Screen, y int) {
	width, _ := screen.Size()
	blank := tcell.StyleDefault
	start := time.Now()
	for {
		p := float64(time.Since(start)) / float64(shotDuration)
		if p > 1 {
			p = 1
		}
		head := int(p * float64(width+shotLength)) // column just past the tip
		tail := head - shotLength
		// behind the tail: erased (this also wipes the line's previous frame)
		for x := 0; x < min(max(tail, 0), width); x++ {
			screen.SetContent(x, y, ' ', nil, blank)
		}
		for x := max(tail, 0); x < min(head, width); x++ {
			screen.SetContent(x, y, '━', nil, shotStyle)
		}
		screen.Show()
		if p >= 1 {
			return
		}
		SleepUntil(start, time.Since(start)+shotFrame)
	}
}

// Slide closes the gap at visible row gap of a list showing visible rows: the
// rows below it move up one at a time, top first, so the blank row travels
// down and out of the list. setGap(at) must make the next render leave row at
// blank and draw the rows below it one row lower; setGap(-1) ends that.
// Steps are scheduled from the start time rather than by sleeping a fixed
// amount, so timer jitter does not add up into a stutter.
func Slide(gap, visible int, setGap func(int), render func()) {
	start := time.Now()
	for step, at := 0, gap; at < visible; step, at = step+1, at+1 {
		SleepUntil(start, time.Duration(step)*SlideStep)
		setGap(at)
		render()
	}
	SleepUntil(start, time.Duration(max(visible-gap, 0))*SlideStep)
	setGap(-1)
	render()
}

// SleepUntil waits until offset has passed since start.
func SleepUntil(start time.Time, offset time.Duration) {
	if d := time.Until(start.Add(offset)); d > 0 {
		time.Sleep(d)
	}
}
