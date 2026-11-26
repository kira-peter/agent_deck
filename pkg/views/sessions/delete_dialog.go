package sessions

import (
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"github.com/kira-peter/agent_deck/pkg/core"
	"github.com/kira-peter/agent_deck/pkg/i18n"
)

// cw measures display width the way terminals draw it: CJK is two cells, but
// "ambiguous" glyphs such as ─ and ↑ stay one cell even on a Chinese locale.
var cw = func() *runewidth.Condition {
	c := runewidth.NewCondition()
	c.EastAsianWidth = false
	return c
}()

const (
	deleteOptCancel = 0
	deleteOptDelete = 1
)

// sessionAction describes a confirm popup for an action on one session.
type sessionAction struct {
	title   i18n.Text
	warning i18n.Text
	confirm i18n.Text
	color   tcell.Color // border and confirm button
	// extra is one more detail row (label, value); empty label means none
	extra func(s core.Session) (string, string)
}

var (
	deleteAction = sessionAction{
		title:   i18n.MsgDeleteConfirm,
		warning: i18n.DeleteDlgSessionWarning,
		confirm: i18n.DeleteDlgConfirm,
		color:   tcell.ColorRed,
	}
	archiveAction = sessionAction{
		title:   i18n.MsgArchiveConfirm,
		warning: i18n.ArchiveDlgWarning,
		confirm: i18n.ArchiveDlgConfirm,
		color:   tcell.ColorYellow,
	}
)

// confirmDeleteSession shows the delete popup and reports whether the user
// chose to delete. The default selection is Cancel.
func (v *SessionsView) confirmDeleteSession(session core.Session) bool {
	return v.confirmAction(deleteAction, session)
}

// confirmArchiveSession shows the archive popup, naming the folder the
// session will be moved to.
func (v *SessionsView) confirmArchiveSession(session core.Session) bool {
	a := archiveAction
	a.extra = func(s core.Session) (string, string) {
		return i18n.ArchiveDlgTarget.Get(), v.src.ArchiveDir(s)
	}
	return v.confirmAction(a, session)
}

func (v *SessionsView) confirmAction(a sessionAction, session core.Session) bool {
	sel := deleteOptCancel
	for {
		v.renderActionDialog(a, session, sel)

		switch ev := v.screen.PollEvent().(type) {
		case *tcell.EventResize:
			v.width, v.height = v.screen.Size()
			v.screen.Sync()
		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyEscape:
				return false
			case tcell.KeyUp:
				sel = deleteOptCancel
			case tcell.KeyDown:
				sel = deleteOptDelete
			case tcell.KeyEnter:
				return sel == deleteOptDelete
			case tcell.KeyRune:
				switch ev.Rune() {
				case 'k':
					sel = deleteOptCancel
				case 'j':
					sel = deleteOptDelete
				}
			}
		}
	}
}

// confirmStyle is the selected confirm button: white on red for delete,
// black on light backgrounds such as yellow so the text stays readable.
func confirmStyle(bg tcell.Color) tcell.Style {
	fg := tcell.ColorWhite
	if bg == tcell.ColorYellow {
		fg = tcell.ColorBlack
	}
	return tcell.StyleDefault.Foreground(fg).Background(bg).Bold(true)
}

// renderDeleteDialog draws the delete popup (kept for callers and tests).
func (v *SessionsView) renderDeleteDialog(session core.Session, sel int) {
	v.renderActionDialog(deleteAction, session, sel)
}

// renderActionDialog draws the confirmation as a popup over the session list,
// which stays visible (dimmed) behind it.
func (v *SessionsView) renderActionDialog(a sessionAction, session core.Session, sel int) {
	v.renderList(true)

	details := []struct{ label, value string }{
		{i18n.HeaderPath.Get(), sessionPath(session)},
		{i18n.HeaderMessages.Get(), fmt.Sprintf("%d", session.MessageCount)},
		{i18n.HeaderSize.Get(), core.FormatSize(session.Size)},
		{i18n.HeaderTime.Get(), core.FormatRelativeTime(session.Modified)},
	}
	if a.extra != nil {
		if label, value := a.extra(session); label != "" {
			details = append(details, struct{ label, value string }{label, value})
		}
	}
	labelW := 0
	for _, d := range details {
		labelW = max(labelW, cw.StringWidth(d.label))
	}

	title := a.title.Get()
	warning := a.warning.Get()
	hint := i18n.DeleteDlgHint.Get()
	prompt := strings.Join(strings.Fields(session.FirstPrompt), " ")
	if prompt == "" {
		prompt = session.ID
	}

	const pad = 3 // inner horizontal padding
	content := max(
		min(cw.StringWidth(prompt), 60),
		cw.StringWidth(warning),
		cw.StringWidth(hint),
		cw.StringWidth(title)+2,
	)
	boxW := min(max(content+2*pad+2, 48), v.width-4)
	innerW := boxW - 2*pad - 2
	boxH := 12 + len(details)
	x0 := max((v.width-boxW)/2, 0)
	y0 := max((v.height-boxH)/2, 0)

	borderStyle := tcell.StyleDefault.Foreground(a.color)
	v.fillRect(x0, y0, boxW, boxH)
	v.drawBox(x0, y0, boxW, boxH, borderStyle)

	// Title embedded in the top border
	v.drawTextW(x0+2, y0, " "+title+" ", borderStyle.Bold(true))

	tx := x0 + 1 + pad
	y := y0 + 2

	v.drawTextW(tx, y, cw.Truncate(prompt, innerW, "..."),
		tcell.StyleDefault.Foreground(tcell.ColorWhite).Bold(true))

	y += 2
	labelStyle := tcell.StyleDefault.Foreground(tcell.ColorGray)
	valueStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite)
	for _, d := range details {
		v.drawTextW(tx, y, d.label, labelStyle)
		v.drawTextW(tx+labelW+3, y, cw.Truncate(d.value, innerW-labelW-3, "..."), valueStyle)
		y++
	}

	y++
	v.drawTextW(tx, y, cw.Truncate(warning, innerW, "..."),
		tcell.StyleDefault.Foreground(tcell.ColorYellow))

	y += 2
	options := []struct {
		text     string
		selected tcell.Style
	}{
		{i18n.DeleteDlgCancel.Get(), tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorLightGreen)},
		{a.confirm.Get(), confirmStyle(a.color)},
	}
	for i, o := range options {
		style := tcell.StyleDefault.Foreground(tcell.ColorWhite)
		prefix := "   "
		if i == sel {
			style = o.selected
			prefix = " > "
		}
		line := prefix + o.text
		line += strings.Repeat(" ", max(innerW-cw.StringWidth(line), 0))
		v.drawTextW(tx, y, line, style)
		y++
	}

	v.drawTextW(x0+1, y0+boxH-3, strings.Repeat("─", boxW-2), labelStyle)
	v.screen.SetContent(x0, y0+boxH-3, '├', nil, borderStyle)
	v.screen.SetContent(x0+boxW-1, y0+boxH-3, '┤', nil, borderStyle)
	v.drawCenteredTextIn(x0, boxW, y0+boxH-2, hint, labelStyle)

	v.screen.Show()
}

// showResultBox displays a one-line message in a small box. With countdown > 0
// it closes itself after that many seconds; with 0 it waits for a key. Any key
// closes it immediately.
func (v *SessionsView) showResultBox(text string, color tcell.Color, countdown int) {
	if countdown > 0 {
		done := make(chan struct{})
		defer close(done)
		go func() {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-done:
					return
				case <-ticker.C:
					_ = v.screen.PostEvent(tcell.NewEventInterrupt(nil))
				}
			}
		}()
	}

	for remaining := countdown; ; {
		v.renderResultBox(text, color, remaining)

		switch v.screen.PollEvent().(type) {
		case *tcell.EventKey:
			return
		case *tcell.EventInterrupt:
			if remaining--; remaining <= 0 {
				return
			}
		case *tcell.EventResize:
			v.width, v.height = v.screen.Size()
			v.screen.Sync()
		}
	}
}

// renderResultBox draws a message popup over the session list.
func (v *SessionsView) renderResultBox(text string, color tcell.Color, remaining int) {
	v.renderList(true)

	hint := i18n.DeleteDlgAnyKey.Get()
	if remaining > 0 {
		hint = fmt.Sprintf(i18n.DeleteDlgAutoClose.Get(), remaining)
	}
	boxW := min(max(cw.StringWidth(text), cw.StringWidth(hint))+10, v.width-4)
	const boxH = 6
	x0 := max((v.width-boxW)/2, 0)
	y0 := max((v.height-boxH)/2, 0)

	style := tcell.StyleDefault.Foreground(color)
	v.fillRect(x0, y0, boxW, boxH)
	v.drawBox(x0, y0, boxW, boxH, style)
	v.drawCenteredTextIn(x0, boxW, y0+2, cw.Truncate(text, boxW-4, "..."), style.Bold(true))
	v.drawCenteredTextIn(x0, boxW, y0+4, hint, tcell.StyleDefault.Foreground(tcell.ColorGray))

	v.screen.Show()
}

// fillRect blanks a rectangle so a popup hides what is under it.
func (v *SessionsView) fillRect(x, y, w, h int) {
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			v.screen.SetContent(x+i, y+j, ' ', nil, tcell.StyleDefault)
		}
	}
}

func (v *SessionsView) drawBox(x, y, w, h int, style tcell.Style) {
	for i := 1; i < w-1; i++ {
		v.screen.SetContent(x+i, y, '─', nil, style)
		v.screen.SetContent(x+i, y+h-1, '─', nil, style)
	}
	for i := 1; i < h-1; i++ {
		v.screen.SetContent(x, y+i, '│', nil, style)
		v.screen.SetContent(x+w-1, y+i, '│', nil, style)
	}
	v.screen.SetContent(x, y, '┌', nil, style)
	v.screen.SetContent(x+w-1, y, '┐', nil, style)
	v.screen.SetContent(x, y+h-1, '└', nil, style)
	v.screen.SetContent(x+w-1, y+h-1, '┘', nil, style)
}

// drawTextW draws text advancing by display width, so CJK stays aligned.
func (v *SessionsView) drawTextW(x, y int, text string, style tcell.Style) {
	for _, r := range text {
		v.screen.SetContent(x, y, r, nil, style)
		if w := cw.RuneWidth(r); w > 1 {
			x += w
		} else {
			x++
		}
	}
}

// drawCenteredTextIn centers text within the span [x, x+w).
func (v *SessionsView) drawCenteredTextIn(x, w, y int, text string, style tcell.Style) {
	v.drawTextW(x+max((w-cw.StringWidth(text))/2, 0), y, text, style)
}
