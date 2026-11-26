package settings

import (
	"strconv"

	"github.com/gdamore/tcell/v2"
	"github.com/kira-peter/agentdeck/pkg/i18n"
	"github.com/kira-peter/agentdeck/pkg/ui/textwidth"
)

const (
	// wheelRadius is how many choices are shown above and below the selected one.
	wheelRadius = 2
	// wheelClearance is how many rows above and below the selected row are
	// hidden while the wheel is open. It is one more than wheelRadius so the
	// wheel always has a blank line between it and the rest of the page.
	wheelClearance = wheelRadius + 1
	// numberSteps is how many multiples are offered on each side of a number.
	numberSteps = 4
)

// wheel is the inline chooser opened on a row with fixed choices. The choices
// scroll vertically through the row's value column: the selected one sits on
// the row itself in yellow and its neighbours fade out with distance.
type wheel struct {
	choices []string
	index   int
	current string // value stored before the wheel opened
	numeric bool   // number row: typing a digit switches to free input
}

func newWheel(choices []string, current string) *wheel {
	return &wheel{choices: choices, index: indexOf(choices, current), current: current}
}

// bigFactors are the large jumps offered above the fine steps.
var bigFactors = []int{30, 20, 10}

// numberChoices puts cur in the middle and offers steps of cur/5 above and
// below it, plus 10×/20×/30× jumps on top, largest first:
// 100 → 3000 2000 1000 180 160 140 120 100 80 60 40 20.
// The step is at least 1 and values below 1 are left out.
func numberChoices(cur int) []string {
	step := max(cur/5, 1)
	var out []string
	if cur > 0 {
		for _, f := range bigFactors {
			out = append(out, strconv.Itoa(cur*f))
		}
	}
	for i := numberSteps; i >= -numberSteps; i-- {
		if n := cur + i*step; n > 0 || n == cur {
			out = append(out, strconv.Itoa(n))
		}
	}
	return out
}

func (w *wheel) move(delta int) {
	w.index = min(max(w.index+delta, 0), len(w.choices)-1)
}

// fade is the colour of a choice at the given distance from the selection.
var fade = []tcell.Color{
	tcell.ColorYellow,
	tcell.NewRGBColor(150, 150, 150),
	tcell.NewRGBColor(85, 85, 85),
}

// dimmed is used for the rest of the page while the wheel is open.
var dimmed = tcell.StyleDefault.Foreground(tcell.NewRGBColor(80, 80, 80))

// openWheel opens the wheel on r. Number rows only get one when they hold a
// number to build the choices from; otherwise they fall back to typing.
func (v *SettingsView) openWheel(r row) bool {
	if r.kind == kindCycle {
		v.wheel = newWheel(r.choices, r.value)
		return true
	}
	n, err := strconv.Atoi(r.seed)
	if !r.isInt || err != nil {
		return false
	}
	v.wheel = newWheel(numberChoices(n), r.seed)
	v.wheel.numeric = true
	return true
}

func (v *SettingsView) handleWheelKey(ev *tcell.EventKey) {
	w := v.wheel
	switch ev.Key() {
	case tcell.KeyEscape:
		v.wheel = nil
	case tcell.KeyUp:
		w.move(-1)
	case tcell.KeyDown:
		w.move(1)
	case tcell.KeyHome:
		w.move(-len(w.choices))
	case tcell.KeyEnd:
		w.move(len(w.choices))
	case tcell.KeyEnter:
		v.wheel = nil
		if choice := w.choices[w.index]; choice != w.current {
			v.finish(v.rows[v.selected].apply(choice))
		}
	case tcell.KeyRune:
		r := ev.Rune()
		switch {
		case r == 'k':
			w.move(-1)
		case r == 'j':
			w.move(1)
		case r == 'q' || r == 'Q':
			v.wheel = nil
		case w.numeric && r >= '0' && r <= '9':
			// a digit starts typing an exact number instead
			v.wheel = nil
			v.editing = true
			v.input = []rune{r}
			v.inputErr = ""
		}
	}
}

// drawWheel draws the open wheel centred on line y, starting at column x and
// limited to width w.
func (v *SettingsView) drawWheel(x, y, width int) {
	wh := v.wheel
	for i, c := range wh.choices {
		d := i - wh.index
		dist := max(d, -d)
		if dist > wheelRadius {
			continue
		}
		style := tcell.StyleDefault.Foreground(fade[dist])
		text := "  " + c
		if dist == 0 {
			style = style.Bold(true)
			text = "▸ " + c
			if c == wh.current {
				text += "  " + i18n.PickInUse.Get()
			}
		}
		v.drawText(x, y+d, textwidth.Truncate(text, width), style)
	}
}
