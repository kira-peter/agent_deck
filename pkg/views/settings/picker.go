package settings

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/yourusername/claude-manager/pkg/core"
	"github.com/yourusername/claude-manager/pkg/i18n"
	"github.com/yourusername/claude-manager/pkg/ui/textwidth"
)

// picker is the two-column model chooser: vendors on the left, that vendor's
// models on the right. Exactly one column has focus; ↑↓ moves inside it,
// → enters the model list and ← goes back to the vendors. The last "vendor"
// is a free-text entry.
type picker struct {
	current string
	apply   func(model string) error

	family      int   // selected vendor; len(catalog) = custom entry
	index       []int // selected model per vendor
	scroll      []int // first visible model per vendor
	vendorScrol int   // first visible vendor
	inModels    bool  // true: focus is on the model column
	custom      []rune
	err         string
}

func newPicker(current string, apply func(string) error) *picker {
	n := len(core.ModelCatalog)
	p := &picker{
		current: current,
		apply:   apply,
		index:   make([]int, n),
		scroll:  make([]int, n),
	}
	if f, i, ok := core.FindModel(current); ok {
		p.family, p.index[f], p.inModels = f, i, true // land on the model in use
	} else if current != "" {
		p.family, p.custom = n, []rune(current)
	}
	return p
}

func (p *picker) isCustom() bool { return p.family == len(core.ModelCatalog) }

func (p *picker) moveFamily(delta int) {
	p.family = min(max(p.family+delta, 0), len(core.ModelCatalog))
	p.err = ""
}

func (p *picker) moveModel(delta int) {
	if p.isCustom() {
		return
	}
	n := len(core.ModelCatalog[p.family].Models)
	p.index[p.family] = min(max(p.index[p.family]+delta, 0), n-1)
}

// move moves inside whichever column has focus.
func (p *picker) move(delta int) {
	if p.inModels {
		p.moveModel(delta)
	} else {
		p.moveFamily(delta)
	}
}

// enterModels gives focus to the model column (not possible on the custom row,
// which has nothing to list).
func (p *picker) enterModels() {
	if !p.isCustom() {
		p.inModels = true
	}
}

// choice returns the model id the picker would apply right now.
func (p *picker) choice() string {
	if p.isCustom() {
		return strings.TrimSpace(string(p.custom))
	}
	return core.ModelCatalog[p.family].Models[p.index[p.family]]
}

func (v *SettingsView) openPicker(current string, apply func(string) error) {
	v.pick = newPicker(current, apply)
}

func (v *SettingsView) handlePickKey(ev *tcell.EventKey) {
	p := v.pick
	switch ev.Key() {
	case tcell.KeyEscape:
		v.pick = nil
	case tcell.KeyUp:
		p.move(-1)
	case tcell.KeyDown:
		p.move(1)
	case tcell.KeyPgUp:
		p.move(-5)
	case tcell.KeyPgDn:
		p.move(5)
	case tcell.KeyHome:
		p.move(-1 << 20)
	case tcell.KeyEnd:
		p.move(1 << 20)
	case tcell.KeyRight:
		p.enterModels()
	case tcell.KeyLeft:
		p.inModels = false
	case tcell.KeyEnter:
		if !p.inModels && !p.isCustom() {
			p.enterModels() // on a vendor, Enter opens its list
			return
		}
		model := p.choice()
		if model == "" {
			p.err = i18n.PickEmptyCustom.Get()
			return
		}
		v.pick = nil
		if model != p.current {
			v.finish(p.apply(model))
		}
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if p.isCustom() && len(p.custom) > 0 {
			p.custom = p.custom[:len(p.custom)-1]
			p.err = ""
		}
	case tcell.KeyCtrlU:
		if p.isCustom() {
			p.custom = nil
		}
	case tcell.KeyRune:
		if p.isCustom() {
			p.custom = append(p.custom, ev.Rune())
			p.err = ""
			return
		}
		switch ev.Rune() {
		case 'k':
			p.move(-1)
		case 'j':
			p.move(1)
		case 'l':
			p.enterModels()
		case 'h':
			p.inModels = false
		case 'g':
			p.move(-1 << 20)
		case 'G':
			p.move(1 << 20)
		}
	}
}

func (v *SettingsView) renderPicker() {
	p := v.pick
	catalog := core.ModelCatalog
	gray := tcell.StyleDefault.Foreground(tcell.ColorGray)

	v.drawCenteredText(1, i18n.PickTitle.Get(), tcell.StyleDefault.
		Foreground(tcell.ColorLightGreen).Bold(true))

	// current model, with a note when it is not in the catalog
	cur := p.current
	if cur == "" {
		cur = i18n.PickNotSet.Get()
	} else if _, _, ok := core.FindModel(cur); !ok {
		cur += "  " + i18n.PickNotInList.Get()
	}
	label := i18n.PickCurrent.Get() + ": "
	x := max((v.width-textwidth.StringWidth(label+cur))/2, 0)
	v.drawText(x, 2, label, gray)
	v.drawText(x+textwidth.StringWidth(label), 2, cur,
		tcell.StyleDefault.Foreground(tcell.ColorYellow).Bold(true))
	v.drawHorizontalLine(3)

	const top, leftX, leftW = 6, 4, 18
	rightX := leftX + leftW + 4
	rightW := max(v.width-rightX-4, 12)
	rows := max(v.height-top-5, 3)

	// styles: the focused column gets a bright bar, the other one a dim marker
	bright := tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorLightGreen)
	dim := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorDarkGreen)
	inUse := tcell.StyleDefault.Foreground(tcell.ColorYellow)

	// column titles; the focused one is highlighted
	titleStyle := func(focused bool) tcell.Style {
		if focused {
			return tcell.StyleDefault.Foreground(tcell.ColorLightGreen).Bold(true).Underline(true)
		}
		return gray
	}
	v.drawText(leftX, top-2, i18n.PickVendor.Get(), titleStyle(!p.inModels))
	v.drawText(rightX, top-2, i18n.PickModels.Get(), titleStyle(p.inModels))
	for y := top - 2; y < top+rows; y++ {
		v.screen.SetContent(leftX+leftW+1, y, '│', nil, gray)
	}

	// left: vendors (+ custom). The vendor holding the current model is marked.
	curFamily, _, curKnown := core.FindModel(p.current)
	nVendors := len(catalog) + 1
	if p.family < p.vendorScrol {
		p.vendorScrol = p.family
	}
	if p.family >= p.vendorScrol+rows {
		p.vendorScrol = p.family - rows + 1
	}
	for i := p.vendorScrol; i < nVendors && i < p.vendorScrol+rows; i++ {
		y := top + i - p.vendorScrol
		name, count := i18n.PickCustom.Get(), ""
		if i < len(catalog) {
			name, count = catalog[i].Name, fmt.Sprint(len(catalog[i].Models))
		}

		style := tcell.StyleDefault.Foreground(tcell.ColorWhite)
		mark := "  "
		if curKnown && i == curFamily || !curKnown && p.current != "" && i == len(catalog) {
			mark = "● "
			style = inUse
		}
		if i == p.family {
			style = dim
			if !p.inModels {
				style = bright
			}
			v.fillLine(leftX, y, leftW, style)
		}
		v.drawText(leftX, y, textwidth.Truncate(mark+name, leftW-4), style)
		if count != "" {
			v.drawText(leftX+leftW-textwidth.StringWidth(count)-1, y, count, style)
		}
	}

	// right: models of the chosen vendor, or the free-text box
	if p.isCustom() {
		v.drawText(rightX, top, textwidth.Truncate(i18n.PickCustomTip.Get(), rightW), gray)
		v.drawText(rightX, top+2, "> ", tcell.StyleDefault.Foreground(tcell.ColorLightGreen))
		box := textwidth.PadRight(string(p.custom)+"█", rightW-2)
		v.drawText(rightX+2, top+2, textwidth.Truncate(box, rightW-2), bright)
	} else {
		models := catalog[p.family].Models
		sel := p.index[p.family]
		sc := &p.scroll[p.family]
		if sel < *sc {
			*sc = sel
		}
		if sel >= *sc+rows {
			*sc = sel - rows + 1
		}
		for i := *sc; i < len(models) && i < *sc+rows; i++ {
			y := top + i - *sc
			style := tcell.StyleDefault.Foreground(tcell.ColorWhite)
			mark := "  "
			if models[i] == p.current {
				mark = "● "
				style = inUse
			}
			if i == sel {
				style = dim
				if p.inModels {
					style = bright
				}
				v.fillLine(rightX, y, rightW, style)
			}
			v.drawText(rightX, y, textwidth.Truncate(mark+models[i], rightW-2), style)
			if models[i] == p.current && i == sel {
				tag := i18n.PickInUse.Get()
				v.drawText(rightX+rightW-textwidth.StringWidth(tag)-1, y, tag, style)
			}
		}
		if len(models) > rows {
			pos := fmt.Sprintf("%d/%d", sel+1, len(models))
			v.drawText(rightX+rightW-textwidth.StringWidth(pos), top-2, pos, gray)
		}
	}

	if p.err != "" {
		v.drawCenteredText(v.height-4, p.err, tcell.StyleDefault.Foreground(tcell.ColorRed))
	}

	v.drawHorizontalLine(v.height - 3)
	hint := i18n.PickHintVendor.Get()
	switch {
	case p.isCustom():
		hint = i18n.PickHintCustom.Get()
	case p.inModels:
		hint = i18n.PickHintModels.Get()
	}
	v.drawCenteredText(v.height-2, hint, gray)
}
