package sessions

import (
	"github.com/yourusername/claude-manager/pkg/ui/rowfx"
)

const listStartY = 7

// selectedRowY is the screen row of the selected session after a render.
func (v *SessionsView) selectedRowY() int {
	return listStartY + v.selectedIdx - v.scrollOffset
}

// animateShot plays the delete shot across screen row y.
func (v *SessionsView) animateShot(y int) { rowfx.Shot(v.screen, y) }

// animateSlide closes the gap left at visible row gap. Call it after the
// session has been removed from the list.
func (v *SessionsView) animateSlide(gap int) {
	rowfx.Slide(gap, v.listRows(), func(at int) { v.gapAt = at }, func() { v.Render() })
}

// listRows is how many sessions the list shows at the current size.
func (v *SessionsView) listRows() int {
	return min(maxListRows(v.height), len(v.displayedSessions)-v.scrollOffset)
}
