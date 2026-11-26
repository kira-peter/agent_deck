package settings

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/kira-peter/agentdeck/pkg/core"
	"github.com/kira-peter/agentdeck/pkg/i18n"
	"github.com/kira-peter/agentdeck/pkg/ui/textwidth"
	"github.com/kira-peter/agentdeck/pkg/views"
)

// SettingsView displays and edits AgentDeck and agent settings.
type SettingsView struct {
	screen tcell.Screen
	config *core.Config
	width  int
	height int

	settings *core.ClaudeSettings // nil when settings.json cannot be parsed
	loadErr  error

	rows      []row
	selected  int // index into rows; always a selectable row
	scroll    int
	editing   bool
	pick      *picker
	wheel     *wheel // open choice wheel on the selected row, nil when closed
	input     []rune
	inputErr  string
	statusMsg string
	statusErr bool
}

func NewSettingsView() views.View {
	return &SettingsView{
		config: core.GetConfig(),
	}
}

func (v *SettingsView) Metadata() views.Metadata {
	return views.Metadata{
		ID:     "settings",
		NameEn: "Settings",
		NameZh: "设置",
		DescEn: "Configure application settings",
		DescZh: "配置应用设置",
		Icon:   "⚙️",
	}
}

func (v *SettingsView) Init(screen tcell.Screen) error {
	v.screen = screen
	v.width, v.height = screen.Size()
	v.reload()
	v.selected = v.firstSelectable()
	return nil
}

// reload re-reads settings.json and rebuilds the rows, keeping the cursor on
// the same line number where possible.
func (v *SettingsView) reload() {
	v.settings, v.loadErr = nil, nil
	if v.config.GetSource() != core.SourceCodex {
		path := filepath.Join(v.config.GetClaudeDir(), "settings.json")
		v.settings, v.loadErr = core.LoadClaudeSettings(path)
	}
	v.rows = v.buildRows()
	if v.selected >= len(v.rows) || !v.rows[v.selected].selectable() {
		v.selected = v.nearestSelectable(min(v.selected, len(v.rows)-1))
	}
}

func (v *SettingsView) firstSelectable() int {
	for i, r := range v.rows {
		if r.selectable() {
			return i
		}
	}
	return 0
}

func (v *SettingsView) nearestSelectable(from int) int {
	for i := from; i >= 0; i-- {
		if v.rows[i].selectable() {
			return i
		}
	}
	return v.firstSelectable()
}

func (v *SettingsView) move(delta int) {
	for i := v.selected + delta; i >= 0 && i < len(v.rows); i += delta {
		if v.rows[i].selectable() {
			v.selected = i
			return
		}
	}
}

func (v *SettingsView) setStatus(msg string, isErr bool) {
	v.statusMsg, v.statusErr = msg, isErr
}

func (v *SettingsView) Render() error {
	v.screen.Clear()

	if v.pick != nil {
		v.renderPicker()
		v.screen.Show()
		return nil
	}

	v.drawCenteredText(1, i18n.ViewSettings.Get(), tcell.StyleDefault.
		Foreground(tcell.ColorLightGreen).Bold(true))
	v.drawHorizontalLine(3)

	const startY, startX = 5, 6
	listH := max(v.height-startY-5, 1)

	// the wheel may have scrolled past the ends of the list; undo that
	if v.wheel == nil {
		v.scroll = min(max(v.scroll, 0), max(len(v.rows)-listH, 0))
	}

	// keep the cursor visible (the section header above it counts as context)
	if v.selected < v.scroll {
		v.scroll = v.selected
		if v.scroll > 0 && v.rows[v.scroll-1].kind == kindHeader {
			v.scroll--
		}
	}
	if v.selected >= v.scroll+listH {
		v.scroll = v.selected - listH + 1
	}
	// an open wheel needs a fixed band of rows above and below its row; the
	// band may run past the ends of the list, those lines are just blank
	if v.wheel != nil {
		top := v.selected - wheelClearance
		if v.selected+wheelClearance >= v.scroll+listH {
			v.scroll = v.selected + wheelClearance - listH + 1
		}
		if top < v.scroll {
			v.scroll = top
		}
	}

	labelW := 0
	for _, r := range v.rows {
		if r.kind != kindHeader {
			labelW = max(labelW, textwidth.StringWidth(r.label))
		}
	}
	labelW = min(labelW, 34)
	valueX := startX + 2 + labelW + 4
	valueW := max(v.width-valueX-4, 10)

	for i := max(v.scroll, 0); i < len(v.rows) && i < v.scroll+listH; i++ {
		r := v.rows[i]
		y := startY + (i - v.scroll)

		// while the wheel is open the rows next to it are hidden and the rest
		// of the page is dimmed
		open := v.wheel != nil
		if open && i != v.selected && i >= v.selected-wheelClearance && i <= v.selected+wheelClearance {
			continue
		}

		switch r.kind {
		case kindHeader:
			st := tcell.StyleDefault.Foreground(tcell.ColorLightCyan).Bold(true)
			if open {
				st = dimmed
			}
			v.drawText(startX, y, r.label, st)
			continue
		case kindInfo:
			st := tcell.StyleDefault.Foreground(tcell.ColorGray)
			if open {
				st = dimmed
			}
			v.drawText(startX+2, y, textwidth.Truncate(r.value, v.width-startX-8), st)
			continue
		}

		if open && i != v.selected {
			v.drawText(startX+2, y, textwidth.Truncate(r.label, labelW), dimmed)
			v.drawText(valueX, y, textwidth.Truncate(r.value, valueW), dimmed)
			continue
		}

		style := tcell.StyleDefault.Foreground(tcell.ColorWhite)
		if i == v.selected {
			style = tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorLightGreen)
			v.fillLine(startX, y, v.width-startX-2, style)
		}

		if i == v.selected && v.wheel != nil {
			labelStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow).Bold(true)
			v.drawText(startX, y, "▌", labelStyle)
			v.drawText(startX+2, y, textwidth.Truncate(r.label, labelW), labelStyle)
			v.drawWheel(valueX-2, y, valueW+2)
			continue
		}

		v.drawText(startX+2, y, textwidth.Truncate(r.label, labelW), style)

		value := r.value
		if i == v.selected && v.editing {
			value = string(v.input) + "█"
		}
		valStyle := style
		if i != v.selected && (r.kind == kindOpen || r.value == i18n.SetNotSet.Get()) {
			valStyle = tcell.StyleDefault.Foreground(tcell.ColorGray)
		}
		v.drawText(valueX, y, textwidth.Truncate(value, valueW), valStyle)
	}

	if len(v.rows) > listH {
		v.drawText(v.width-8, startY-1, fmt.Sprintf("%d/%d", v.selected+1, len(v.rows)),
			tcell.StyleDefault.Foreground(tcell.ColorGray))
	}

	// status line
	statusY := v.height - 4
	switch {
	case v.editing && v.inputErr != "":
		v.drawCenteredText(statusY, v.inputErr, tcell.StyleDefault.Foreground(tcell.ColorRed))
	case v.statusMsg != "":
		color := tcell.ColorLightGreen
		if v.statusErr {
			color = tcell.ColorRed
		}
		v.drawCenteredText(statusY, textwidth.Truncate(v.statusMsg, v.width-6),
			tcell.StyleDefault.Foreground(color))
	}

	// hints
	v.drawHorizontalLine(v.height - 3)
	hint := i18n.SetHintBrowse.Get()
	switch {
	case v.editing:
		hint = i18n.SetHintEdit.Get()
	case v.wheel != nil && v.wheel.numeric:
		hint = i18n.SetHintWheelNum.Get()
	case v.wheel != nil:
		hint = i18n.SetHintWheel.Get()
	}
	v.drawCenteredText(v.height-2, hint, tcell.StyleDefault.Foreground(tcell.ColorGray))

	v.screen.Show()
	return nil
}

func (v *SettingsView) HandleEvent(event tcell.Event) error {
	switch ev := event.(type) {
	case *tcell.EventKey:
		if v.pick != nil {
			v.handlePickKey(ev)
			return nil
		}
		if v.editing {
			v.handleEditKey(ev)
			return nil
		}
		if v.wheel != nil {
			v.handleWheelKey(ev)
			return nil
		}
		return v.handleBrowseKey(ev)
	case *tcell.EventResize:
		v.width, v.height = v.screen.Size()
		v.screen.Sync()
	}
	return nil
}

func (v *SettingsView) handleBrowseKey(ev *tcell.EventKey) error {
	switch ev.Key() {
	case tcell.KeyEscape:
		return fmt.Errorf("back")
	case tcell.KeyUp:
		v.move(-1)
	case tcell.KeyDown:
		v.move(1)
	case tcell.KeyEnter:
		v.activate()
	case tcell.KeyRune:
		switch ev.Rune() {
		case 'q', 'Q':
			return fmt.Errorf("quit")
		case 'k':
			v.move(-1)
		case 'j':
			v.move(1)
		case 'r', 'R':
			v.reload()
			v.setStatus(i18n.SetReloaded.Get(), false)
		case 'x', 'X':
			v.clearCurrent()
		case 't', 'T':
			lang := "en"
			if i18n.GetManager().Toggle() == i18n.LanguageChinese {
				lang = "zh"
			}
			v.config.SetLanguage(lang)
			v.reload()
		}
	}
	return nil
}

// activate handles Enter on the selected row.
func (v *SettingsView) activate() {
	if v.selected >= len(v.rows) {
		return
	}
	r := v.rows[v.selected]
	v.setStatus("", false)

	switch r.kind {
	case kindCycle:
		v.openWheel(r)
	case kindPick:
		v.openPicker(r.seed, r.apply)
	case kindText:
		if v.openWheel(r) {
			return
		}
		v.editing = true
		v.input = []rune(r.seed)
		v.inputErr = ""
	case kindOpen:
		if err := core.OpenPath(r.path); err != nil {
			v.setStatus(fmt.Sprintf(i18n.SetOpenFailed.Get(), err), true)
		} else {
			v.setStatus(fmt.Sprintf(i18n.SetOpened.Get(), r.path), false)
		}
	}
}

func (v *SettingsView) clearCurrent() {
	if v.selected >= len(v.rows) || v.rows[v.selected].clear == nil {
		return
	}
	if err := v.rows[v.selected].clear(); err != nil {
		v.setStatus(fmt.Sprintf(i18n.SetSaveFailed.Get(), err), true)
		return
	}
	v.reload()
	v.setStatus(i18n.SetCleared.Get(), false)
}

// finish reports the outcome of a save and refreshes the page.
func (v *SettingsView) finish(err error) {
	if err != nil {
		v.setStatus(fmt.Sprintf(i18n.SetSaveFailed.Get(), err), true)
		return
	}
	v.reload()
	v.setStatus(i18n.SetSaved.Get(), false)
}

func (v *SettingsView) handleEditKey(ev *tcell.EventKey) {
	switch ev.Key() {
	case tcell.KeyEscape:
		v.editing = false
	case tcell.KeyEnter:
		text := strings.TrimSpace(string(v.input))
		r := v.rows[v.selected]
		if r.isInt {
			if _, err := strconv.Atoi(text); err != nil {
				v.inputErr = i18n.SetInvalidInt.Get()
				return
			}
		}
		v.editing = false
		v.finish(r.apply(text))
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if n := len(v.input); n > 0 {
			v.input = v.input[:n-1]
		}
		v.inputErr = ""
	case tcell.KeyCtrlU:
		v.input = nil
	case tcell.KeyRune:
		v.input = append(v.input, ev.Rune())
		v.inputErr = ""
	}
}

func (v *SettingsView) Cleanup() error {
	return nil
}

func (v *SettingsView) fillLine(x, y, w int, style tcell.Style) {
	for i := 0; i < w; i++ {
		v.screen.SetContent(x+i, y, ' ', nil, style)
	}
}

func (v *SettingsView) drawText(x, y int, text string, style tcell.Style) {
	textwidth.Draw(v.screen, x, y, text, style)
}

func (v *SettingsView) drawCenteredText(y int, text string, style tcell.Style) {
	textwidth.DrawCentered(v.screen, v.width, y, text, style)
}

func (v *SettingsView) drawHorizontalLine(y int) {
	for x := 2; x < v.width-2; x++ {
		v.screen.SetContent(x, y, '─', nil, tcell.StyleDefault.
			Foreground(tcell.ColorGray))
	}
}
