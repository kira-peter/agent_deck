package projects

import (
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/kira-peter/agent_deck/pkg/core"
	"github.com/kira-peter/agent_deck/pkg/i18n"
)

const (
	deleteOptCancel = 0
	deleteOptDelete = 1
)

// confirmDeleteProject shows a modal confirmation box and reports whether the
// user chose to delete. The default selection is Cancel.
func (v *ProjectsView) confirmDeleteProject(project core.Project) bool {
	sel := deleteOptCancel
	for {
		v.renderDeleteDialog(project, sel)

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

// renderDeleteDialog draws the confirmation as a popup over the project list,
// which stays visible (dimmed) behind it.
func (v *ProjectsView) renderDeleteDialog(project core.Project, sel int) {
	v.renderPage(true)

	lastActivity := core.FormatRelativeTime(project.LastActivity)
	details := []struct{ label, value string }{
		{i18n.HeaderSessions.Get(), fmt.Sprintf("%d", project.SessionCount)},
		{i18n.HeaderSize.Get(), core.FormatSize(project.TotalSize)},
		{i18n.HeaderLastActivity.Get(), lastActivity},
	}
	labelW := 0
	for _, d := range details {
		labelW = max(labelW, cw.StringWidth(d.label))
	}

	title := i18n.DeleteDlgTitle.Get()
	warning := i18n.DeleteDlgWarning.Get()
	hint := i18n.DeleteDlgHint.Get()

	const pad = 3 // inner horizontal padding
	content := max(
		cw.StringWidth(project.Path),
		cw.StringWidth(warning),
		cw.StringWidth(hint),
		cw.StringWidth(title)+2,
	)
	boxW := min(max(content+2*pad+2, 48), v.width-4)
	innerW := boxW - 2*pad - 2
	const boxH = 15
	x0 := max((v.width-boxW)/2, 0)
	y0 := max((v.height-boxH)/2, 0)

	borderStyle := tcell.StyleDefault.Foreground(tcell.ColorRed)
	v.fillRect(x0, y0, boxW, boxH)
	v.drawBox(x0, y0, boxW, boxH, borderStyle)

	// Title embedded in the top border
	v.drawText(x0+2, y0, " "+title+" ", borderStyle.Bold(true))

	tx := x0 + 1 + pad
	y := y0 + 2

	path := project.Path
	if cw.StringWidth(path) > innerW {
		path = cw.TruncateLeft(path, cw.StringWidth(path)-innerW+3, "...")
	}
	v.drawText(tx, y, path, tcell.StyleDefault.Foreground(tcell.ColorWhite).Bold(true))

	y += 2
	labelStyle := tcell.StyleDefault.Foreground(tcell.ColorGray)
	valueStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite)
	for _, d := range details {
		v.drawText(tx, y, d.label, labelStyle)
		v.drawText(tx+labelW+3, y, d.value, valueStyle)
		y++
	}

	y++
	v.drawText(tx, y, cw.Truncate(warning, innerW, "..."),
		tcell.StyleDefault.Foreground(tcell.ColorYellow))

	y += 2
	options := []struct {
		text     string
		selected tcell.Style
	}{
		{i18n.DeleteDlgCancel.Get(), tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorLightGreen)},
		{i18n.DeleteDlgConfirm.Get(), tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorRed).Bold(true)},
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
		v.drawText(tx, y, line, style)
		y++
	}

	v.drawText(x0+1, y0+boxH-3, strings.Repeat("─", boxW-2), labelStyle)
	v.screen.SetContent(x0, y0+boxH-3, '├', nil, borderStyle)
	v.screen.SetContent(x0+boxW-1, y0+boxH-3, '┤', nil, borderStyle)
	v.drawCenteredTextIn(x0, boxW, y0+boxH-2, hint, labelStyle)

	v.screen.Show()
}

// showResultBox displays a one-line message in a small box. With countdown > 0
// it closes itself after that many seconds; with 0 it waits for a key. Any key
// closes it immediately.
func (v *ProjectsView) showResultBox(text string, color tcell.Color, countdown int) {
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

// renderResultBox draws a message popup over the project list.
func (v *ProjectsView) renderResultBox(text string, color tcell.Color, remaining int) {
	v.renderPage(true)

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
func (v *ProjectsView) fillRect(x, y, w, h int) {
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			v.screen.SetContent(x+i, y+j, ' ', nil, tcell.StyleDefault)
		}
	}
}

func (v *ProjectsView) drawBox(x, y, w, h int, style tcell.Style) {
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

// drawCenteredTextIn centers text within the span [x, x+w).
func (v *ProjectsView) drawCenteredTextIn(x, w, y int, text string, style tcell.Style) {
	v.drawText(x+max((w-cw.StringWidth(text))/2, 0), y, text, style)
}
