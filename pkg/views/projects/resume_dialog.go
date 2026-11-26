package projects

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/kira-peter/agent_deck/pkg/core"
	"github.com/kira-peter/agent_deck/pkg/i18n"
)

// resumeNew is the choice returned for "start a new session".
const resumeNew = -1

// resumeCancel is returned when the user backs out.
const resumeCancel = -2

// loadProjectSessions returns the project's sessions, newest first.
func (v *ProjectsView) loadProjectSessions(project core.Project) []core.Session {
	sessions, err := v.src.GetProjectSessions(project)
	if err != nil {
		return nil
	}
	sort.SliceStable(sessions, func(i, j int) bool {
		return sessions[i].Modified.After(sessions[j].Modified)
	})
	return sessions
}

// promptOneLine flattens a prompt into a single trimmed line.
func promptOneLine(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if s == "" {
		return i18n.ResumeNoPrompt.Get()
	}
	return s
}

// chooseSession lets the user pick between a new session and one of the
// existing sessions. It returns resumeNew, resumeCancel or the session index.
func (v *ProjectsView) chooseSession(project core.Project, sessions []core.Session) int {
	// row 0 is "new session"; row i+1 is sessions[i]
	sel, scroll := 1, 0 // default to the latest session: the usual intent
	rows := len(sessions) + 1

	for {
		v.renderResumeDialog(project, sessions, sel, &scroll)

		switch ev := v.screen.PollEvent().(type) {
		case *tcell.EventResize:
			v.width, v.height = v.screen.Size()
			v.screen.Sync()
		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyEscape:
				return resumeCancel
			case tcell.KeyUp:
				sel = max(sel-1, 0)
			case tcell.KeyDown:
				sel = min(sel+1, rows-1)
			case tcell.KeyPgUp:
				sel = max(sel-5, 0)
			case tcell.KeyPgDn:
				sel = min(sel+5, rows-1)
			case tcell.KeyHome:
				sel = 0
			case tcell.KeyEnd:
				sel = rows - 1
			case tcell.KeyEnter:
				return sel - 1 // 0 -> resumeNew, n -> session n-1
			case tcell.KeyRune:
				switch ev.Rune() {
				case 'k':
					sel = max(sel-1, 0)
				case 'j':
					sel = min(sel+1, rows-1)
				case 'n', 'N':
					return resumeNew
				}
			}
		}
	}
}

func (v *ProjectsView) renderResumeDialog(project core.Project, sessions []core.Session, sel int, scroll *int) {
	v.screen.Clear()

	hint := i18n.ResumeHint.Get()
	title := i18n.ResumeTitle.Get()

	boxW := min(max(cw.StringWidth(project.Path)+10, 76), v.width-4)
	boxW = max(boxW, min(40, v.width))
	innerW := boxW - 8
	// fixed rows: border, path, blank, "new", blank, section title, blank, ..., rule, hint, border
	const fixed = 11
	listH := max(min(len(sessions), v.height-fixed-2), 1)
	boxH := fixed + listH
	x0 := max((v.width-boxW)/2, 0)
	y0 := max((v.height-boxH)/2, 0)

	borderStyle := tcell.StyleDefault.Foreground(tcell.ColorLightGreen)
	gray := tcell.StyleDefault.Foreground(tcell.ColorGray)
	v.drawBox(x0, y0, boxW, boxH, borderStyle)
	v.drawText(x0+2, y0, " "+title+" ", borderStyle.Bold(true))

	tx := x0 + 4
	y := y0 + 2
	v.drawText(tx, y, truncateMiddle(project.Path, innerW),
		tcell.StyleDefault.Foreground(tcell.ColorWhite).Bold(true))

	// row 0: new session
	y += 2
	v.drawResumeRow(tx, y, innerW, sel == 0,
		"+ "+i18n.ResumeNewSession.Get(), "", tcell.StyleDefault.Foreground(tcell.ColorLightGreen))

	// section title
	y += 2
	v.drawText(tx, y, fmt.Sprintf(i18n.ResumeSessions.Get(), len(sessions)), gray)

	// session rows, scrolled so the selection stays visible
	if sel >= 1 {
		if sel-1 < *scroll {
			*scroll = sel - 1
		}
		if sel-1 >= *scroll+listH {
			*scroll = sel - listH
		}
	}
	y += 2
	for i := *scroll; i < len(sessions) && i < *scroll+listH; i++ {
		s := sessions[i]
		right := fmt.Sprintf("%s · %s",
			fmt.Sprintf(i18n.ResumeMessages.Get(), s.MessageCount),
			core.FormatRelativeTime(s.Modified))
		left := promptOneLine(s.FirstPrompt)
		if i == 0 {
			left = "[" + i18n.ResumeLatest.Get() + "] " + left
		}
		v.drawResumeRow(tx, y+i-*scroll, innerW, sel == i+1, left, right,
			tcell.StyleDefault.Foreground(tcell.ColorWhite))
	}
	if len(sessions) > listH {
		pos := fmt.Sprintf("%d/%d", max(sel, 1), len(sessions))
		v.drawText(x0+boxW-cw.StringWidth(pos)-3, y0+6, pos, gray)
	}

	v.drawText(x0+1, y0+boxH-3, strings.Repeat("─", boxW-2), gray)
	v.screen.SetContent(x0, y0+boxH-3, '├', nil, borderStyle)
	v.screen.SetContent(x0+boxW-1, y0+boxH-3, '┤', nil, borderStyle)
	v.drawCenteredTextIn(x0, boxW, y0+boxH-2, hint, gray)

	v.screen.Show()
}

// drawResumeRow draws "prefix left ........ right" on one line; the left part
// is cut to make room for the right part.
func (v *ProjectsView) drawResumeRow(x, y, w int, selected bool, left, right string, normal tcell.Style) {
	style := normal
	prefix := "   "
	if selected {
		style = tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorLightGreen)
		prefix = " > "
	}
	v.drawText(x, y, strings.Repeat(" ", w), style)

	rightW := cw.StringWidth(right)
	leftMax := w - cw.StringWidth(prefix) - 1
	if rightW > 0 {
		leftMax -= rightW + 2
	}
	v.drawText(x, y, prefix+cw.Truncate(left, max(leftMax, 4), "..."), style)
	if rightW > 0 {
		v.drawText(x+w-rightW-1, y, right, style)
	}
}

// truncateMiddle shortens s to w cells, keeping the tail (the folder name).
func truncateMiddle(s string, w int) string {
	if cw.StringWidth(s) <= w {
		return s
	}
	return cw.TruncateLeft(s, cw.StringWidth(s)-w+3, "...")
}
