package transition

import "github.com/gdamore/tcell/v2"

// Offscreen wraps a screen so that Show and Sync do nothing while Hold is
// set. Views call Show at the end of Render; holding it lets the caller have
// a view draw its first frame into the buffer without it appearing, then
// reveal it with an animation.
type Offscreen struct {
	tcell.Screen
	Hold bool
}

func (o *Offscreen) Show() {
	if !o.Hold {
		o.Screen.Show()
	}
}

func (o *Offscreen) Sync() {
	if !o.Hold {
		o.Screen.Sync()
	}
}
