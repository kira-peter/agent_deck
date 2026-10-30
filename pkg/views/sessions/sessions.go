package sessions

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/yourusername/claude-manager/pkg/core"
	"github.com/yourusername/claude-manager/pkg/i18n"
	"github.com/yourusername/claude-manager/pkg/views"
	"github.com/yourusername/claude-manager/pkg/ui/textwidth"
)

// SessionsView displays all sessions
type SessionsView struct {
	screen       tcell.Screen
	sessions     []core.Session
	claudeDir    *core.ClaudeDir
	width        int
	height       int
	selectedIdx  int
	scrollOffset int

	// Detail mode state
	detailMode         bool
	detailLines        []string
	detailScrollOffset int
	detailSession      *core.Session

	// Search / filter state
	searchMode       bool
	searchQuery      string
	filterIndex      int // 0=All 1=Today 2=Week 3=Month
	displayedSessions []core.Session
}

// NewSessionsView creates a new sessions view
func NewSessionsView() views.View {
	config := core.GetConfig()
	return &SessionsView{
		claudeDir:    core.NewClaudeDir(config.GetClaudeDir()),
		selectedIdx:  0,
		scrollOffset: 0,
	}
}

func (v *SessionsView) Metadata() views.Metadata {
	return views.Metadata{
		ID:     "sessions",
		NameEn: "Sessions",
		NameZh: "会话",
		DescEn: "Browse all conversation sessions",
		DescZh: "浏览所有会话记录",
		Icon:   "💬",
	}
}

func (v *SessionsView) Init(screen tcell.Screen) error {
	v.screen = screen
	v.width, v.height = screen.Size()

	sessions, err := v.claudeDir.GetSessions()
	if err != nil {
		return err
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Modified.After(sessions[j].Modified)
	})

	v.sessions = sessions
	v.rebuildDisplay()
	return nil
}

func (v *SessionsView) Render() error {
	v.screen.Clear()

	if v.detailMode {
		v.renderDetail()
	} else {
		v.renderTitle()
		v.renderTableHeader()
		v.renderSessionList()
		v.renderHints()
	}

	v.screen.Show()
	return nil
}

// ── List view ──────────────────────────────────────────────────────────────

func (v *SessionsView) renderTitle() {
	filterLabels := []i18n.Text{i18n.FilterAll, i18n.FilterToday, i18n.FilterWeek, i18n.FilterMonth}
	title := fmt.Sprintf("%s (%d/%d)", i18n.ViewSessions.Get(), len(v.displayedSessions), len(v.sessions))
	if v.filterIndex > 0 {
		title += fmt.Sprintf(" [%s]", filterLabels[v.filterIndex].Get())
	}
	if v.searchQuery != "" {
		title += fmt.Sprintf(" [%s%s]", i18n.HintSearch.Get(), v.searchQuery)
	}
	v.drawCenteredText(1, title, tcell.StyleDefault.
		Foreground(tcell.ColorLightGreen).Bold(true))
	v.drawHorizontalLine(3)
}

func (v *SessionsView) renderTableHeader() {
	y := 4
	startX := 4

	headers := []struct {
		text  string
		width int
	}{
		{i18n.HeaderID.Get(), 10},
		{i18n.HeaderFirstPrompt.Get(), 45},
		{i18n.HeaderMessages.Get(), 10},
		{i18n.HeaderTime.Get(), 15},
		{i18n.HeaderSize.Get(), 10},
	}

	x := startX
	for _, h := range headers {
		v.drawText(x, y, h.text, tcell.StyleDefault.
			Foreground(tcell.ColorLightCyan).Bold(true))
		x += h.width
	}

	v.drawHorizontalLine(y + 1)
}

func (v *SessionsView) renderSessionList() {
	startY := 7
	startX := 4
	maxVisible := v.height - 12

	if len(v.displayedSessions) == 0 {
		v.drawCenteredText(startY+2, i18n.StatusNoData.Get(),
			tcell.StyleDefault.Foreground(tcell.ColorGray))
		return
	}

	if v.selectedIdx < v.scrollOffset {
		v.scrollOffset = v.selectedIdx
	}
	if v.selectedIdx >= v.scrollOffset+maxVisible {
		v.scrollOffset = v.selectedIdx - maxVisible + 1
	}

	for i := v.scrollOffset; i < len(v.displayedSessions) && i < v.scrollOffset+maxVisible; i++ {
		session := v.displayedSessions[i]
		y := startY + (i - v.scrollOffset)

		style := tcell.StyleDefault.Foreground(tcell.ColorWhite)
		if i == v.selectedIdx {
			style = tcell.StyleDefault.
				Foreground(tcell.ColorBlack).
				Background(tcell.ColorLightGreen)
		}

		sessionID := session.ID
		if len(sessionID) > 8 {
			sessionID = sessionID[:8]
		}
		v.drawText(startX, y, fmt.Sprintf("%-8s", sessionID), style)

		prompt := session.FirstPrompt
		if len(prompt) > 43 {
			prompt = prompt[:40] + "..."
		}
		v.drawText(startX+10, y, fmt.Sprintf("%-43s", prompt), style)

		v.drawText(startX+55, y, fmt.Sprintf("%-8d", session.MessageCount), style)

		timeStr := core.FormatRelativeTime(session.Modified)
		v.drawText(startX+65, y, fmt.Sprintf("%-13s", timeStr), style)

		v.drawText(startX+80, y, core.FormatSize(session.Size), style)
	}
}

func (v *SessionsView) renderHints() {
	y := v.height - 2
	v.drawHorizontalLine(y - 1)
	if v.searchMode {
		query := i18n.HintSearch.Get() + v.searchQuery + "█"
		v.drawCenteredText(y, query, tcell.StyleDefault.Foreground(tcell.ColorYellow))
	} else {
		hints := i18n.HintListActions.Get()
		v.drawCenteredText(y, hints, tcell.StyleDefault.Foreground(tcell.ColorGray))
	}
}

// ── Detail view ────────────────────────────────────────────────────────────

// loadSessionDetail fetches messages for the given session and pre-renders lines.
func (v *SessionsView) loadSessionDetail(session core.Session) {
	v.detailSession = &session
	v.detailScrollOffset = 0
	v.detailLines = nil

	// Metadata header lines
	id := session.ID
	if len(id) > 36 {
		id = id[:36]
	}
	v.detailLines = append(v.detailLines, fmt.Sprintf("ID: %s", id))

	project := session.ProjectPath
	if len(project) > v.width-12 {
		project = "..." + project[len(project)-(v.width-15):]
	}
	v.detailLines = append(v.detailLines, fmt.Sprintf("Project: %s", project))

	meta := fmt.Sprintf("Messages: %d   Size: %s", session.MessageCount, core.FormatSize(session.Size))
	if session.GitBranch != "" {
		meta += fmt.Sprintf("   Branch: %s", session.GitBranch)
	}
	if session.HasSubagents {
		meta += "   [has subagents]"
	}
	v.detailLines = append(v.detailLines, meta)
	v.detailLines = append(v.detailLines, strings.Repeat("─", v.width-4))
	v.detailLines = append(v.detailLines, "")

	// Load messages
	messages, err := v.claudeDir.GetSessionMessages(session.FilePath, 0)
	if err != nil {
		v.detailLines = append(v.detailLines, fmt.Sprintf("Error loading messages: %v", err))
		return
	}

	if len(messages) == 0 {
		v.detailLines = append(v.detailLines, "(no messages)")
		return
	}

	wrapWidth := v.width - 8
	if wrapWidth < 20 {
		wrapWidth = 20
	}

	for _, msg := range messages {
		// Role header
		if msg.Role == "user" {
			header := "[ USER ]"
			if !msg.Time.IsZero() {
				header = fmt.Sprintf("[ USER  %s ]", msg.Time.Format("15:04:05"))
			}
			v.detailLines = append(v.detailLines, header)
		} else {
			header := "[ ASSISTANT ]"
			if !msg.Time.IsZero() {
				header = fmt.Sprintf("[ ASSISTANT  %s ]", msg.Time.Format("15:04:05"))
			}
			v.detailLines = append(v.detailLines, header)
		}

		// Word-wrapped content
		for _, line := range strings.Split(msg.Content, "\n") {
			wrapped := wordWrap(line, wrapWidth)
			v.detailLines = append(v.detailLines, wrapped...)
		}
		v.detailLines = append(v.detailLines, "")
	}
}

func (v *SessionsView) renderDetail() {
	if v.detailSession == nil {
		return
	}

	// Title bar
	shortID := v.detailSession.ID
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}
	title := fmt.Sprintf("Session Detail: %s", shortID)
	v.drawCenteredText(1, title, tcell.StyleDefault.
		Foreground(tcell.ColorLightGreen).Bold(true))
	v.drawHorizontalLine(3)

	// Content area
	startY := 4
	maxVisible := v.height - 7

	maxScroll := len(v.detailLines) - maxVisible
	if maxScroll < 0 {
		maxScroll = 0
	}
	if v.detailScrollOffset > maxScroll {
		v.detailScrollOffset = maxScroll
	}

	for i := v.detailScrollOffset; i < len(v.detailLines) && i < v.detailScrollOffset+maxVisible; i++ {
		line := v.detailLines[i]
		y := startY + (i - v.detailScrollOffset)

		style := tcell.StyleDefault.Foreground(tcell.ColorWhite)
		if strings.HasPrefix(line, "[ USER") {
			style = tcell.StyleDefault.Foreground(tcell.ColorYellow).Bold(true)
		} else if strings.HasPrefix(line, "[ ASSISTANT") {
			style = tcell.StyleDefault.Foreground(tcell.ColorLightCyan).Bold(true)
		} else if strings.HasPrefix(line, "ID:") || strings.HasPrefix(line, "Project:") || strings.HasPrefix(line, "Messages:") {
			style = tcell.StyleDefault.Foreground(tcell.ColorLightBlue)
		} else if strings.HasPrefix(line, "─") {
			style = tcell.StyleDefault.Foreground(tcell.ColorGray)
		}

		v.drawText(4, y, line, style)
	}

	// Scroll indicator + hints
	v.drawHorizontalLine(v.height - 3)
	scrollInfo := fmt.Sprintf("[%d/%d]", v.detailScrollOffset+1, len(v.detailLines))
	v.drawText(v.width-15, v.height-2, scrollInfo,
		tcell.StyleDefault.Foreground(tcell.ColorGray))

	hints := "↑↓/jk: Scroll  PgUp/PgDn: Fast Scroll  ESC: Back"
	if !i18n.GetManager().IsEnglish() {
		hints = "↑↓/jk: 滚动  PgUp/PgDn: 快速滚动  ESC: 返回"
	}
	v.drawCenteredText(v.height-2, hints, tcell.StyleDefault.Foreground(tcell.ColorGray))
}

// ── Event handling ─────────────────────────────────────────────────────────

func (v *SessionsView) HandleEvent(event tcell.Event) error {
	switch ev := event.(type) {
	case *tcell.EventKey:
		if v.detailMode {
			return v.handleDetailEvent(ev)
		}
		return v.handleListEvent(ev)
	case *tcell.EventResize:
		v.width, v.height = v.screen.Size()
		v.screen.Sync()
	}
	return nil
}

func (v *SessionsView) handleListEvent(ev *tcell.EventKey) error {
	if v.searchMode {
		return v.handleSearchInput(ev)
	}
	switch ev.Key() {
	case tcell.KeyEscape:
		return fmt.Errorf("back")
	case tcell.KeyUp:
		v.movePrev()
	case tcell.KeyDown:
		v.moveNext()
	case tcell.KeyEnter:
		if len(v.displayedSessions) > 0 && v.selectedIdx < len(v.displayedSessions) {
			v.loadSessionDetail(v.displayedSessions[v.selectedIdx])
			v.detailMode = true
		}
	case tcell.KeyRune:
		switch ev.Rune() {
		case 'q', 'Q':
			return fmt.Errorf("quit")
		case 'k':
			v.movePrev()
		case 'j':
			v.moveNext()
		case 'r', 'R':
			sessions, err := v.claudeDir.GetSessions()
			if err == nil {
				sort.Slice(sessions, func(i, j int) bool {
					return sessions[i].Modified.After(sessions[j].Modified)
				})
				v.sessions = sessions
				v.rebuildDisplay()
			}
		case 'd': // delete (with confirmation)
			if len(v.displayedSessions) > 0 && v.selectedIdx < len(v.displayedSessions) {
				v.deleteSession(v.displayedSessions[v.selectedIdx])
			}
		case 'D': // Shift+D: delete immediately, no confirmation
			if len(v.displayedSessions) > 0 && v.selectedIdx < len(v.displayedSessions) {
				v.quickDeleteSession(v.displayedSessions[v.selectedIdx])
			}
		case 'a', 'A':
			if len(v.displayedSessions) > 0 && v.selectedIdx < len(v.displayedSessions) {
				v.archiveSession(v.displayedSessions[v.selectedIdx])
			}
		case '/':
			v.searchMode = true
		case 'f', 'F':
			v.filterIndex = (v.filterIndex + 1) % 4
			v.rebuildDisplay()
		}
	}
	return nil
}

func (v *SessionsView) handleSearchInput(ev *tcell.EventKey) error {
	switch ev.Key() {
	case tcell.KeyEscape:
		v.searchMode = false
		v.searchQuery = ""
		v.rebuildDisplay()
	case tcell.KeyEnter:
		v.searchMode = false
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		runes := []rune(v.searchQuery)
		if len(runes) > 0 {
			v.searchQuery = string(runes[:len(runes)-1])
			v.rebuildDisplay()
		}
	case tcell.KeyRune:
		v.searchQuery += string(ev.Rune())
		v.rebuildDisplay()
	}
	return nil
}

func (v *SessionsView) handleDetailEvent(ev *tcell.EventKey) error {
	maxVisible := v.height - 7
	maxScroll := len(v.detailLines) - maxVisible
	if maxScroll < 0 {
		maxScroll = 0
	}

	switch ev.Key() {
	case tcell.KeyEscape:
		v.detailMode = false
	case tcell.KeyUp:
		if v.detailScrollOffset > 0 {
			v.detailScrollOffset--
		}
	case tcell.KeyDown:
		if v.detailScrollOffset < maxScroll {
			v.detailScrollOffset++
		}
	case tcell.KeyPgUp:
		v.detailScrollOffset -= maxVisible / 2
		if v.detailScrollOffset < 0 {
			v.detailScrollOffset = 0
		}
	case tcell.KeyPgDn:
		v.detailScrollOffset += maxVisible / 2
		if v.detailScrollOffset > maxScroll {
			v.detailScrollOffset = maxScroll
		}
	case tcell.KeyRune:
		switch ev.Rune() {
		case 'q', 'Q':
			return fmt.Errorf("quit")
		case 'k':
			if v.detailScrollOffset > 0 {
				v.detailScrollOffset--
			}
		case 'j':
			if v.detailScrollOffset < maxScroll {
				v.detailScrollOffset++
			}
		}
	}
	return nil
}

func (v *SessionsView) Cleanup() error {
	return nil
}

// ── Helpers ────────────────────────────────────────────────────────────────

func (v *SessionsView) movePrev() {
	if v.selectedIdx > 0 {
		v.selectedIdx--
	}
}

func (v *SessionsView) moveNext() {
	if v.selectedIdx < len(v.displayedSessions)-1 {
		v.selectedIdx++
	}
}

// rebuildDisplay filters v.sessions by searchQuery and filterIndex into displayedSessions.
func (v *SessionsView) rebuildDisplay() {
	now := time.Now()
	var cutoff time.Time
	switch v.filterIndex {
	case 1:
		y, m, d := now.Date()
		cutoff = time.Date(y, m, d, 0, 0, 0, 0, now.Location())
	case 2:
		cutoff = now.AddDate(0, 0, -7)
	case 3:
		cutoff = now.AddDate(0, 0, -30)
	}

	q := strings.ToLower(v.searchQuery)
	var result []core.Session
	for _, s := range v.sessions {
		if v.filterIndex > 0 && s.Modified.Before(cutoff) {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(s.FirstPrompt), q) {
			continue
		}
		result = append(result, s)
	}
	v.displayedSessions = result
	v.selectedIdx = 0
	v.scrollOffset = 0
}

// deleteSession deletes a session after the user confirms in a dialog.
func (v *SessionsView) deleteSession(session core.Session) {
	if !v.confirmDeleteSession(session) {
		return
	}
	if err := v.removeSession(session); err != nil {
		v.showResultBox(fmt.Sprintf("%s: %v", i18n.DeleteDlgFailed.Get(), err), tcell.ColorRed, 0)
		return
	}
	v.showResultBox(i18n.MsgDeleteSuccess.Get(), tcell.ColorGreen, resultBoxSeconds)
}

// quickDeleteSession deletes a session immediately, without confirmation or a
// success message. Failures are still reported.
func (v *SessionsView) quickDeleteSession(session core.Session) {
	if err := v.removeSession(session); err != nil {
		v.showResultBox(fmt.Sprintf("%s: %v", i18n.DeleteDlgFailed.Get(), err), tcell.ColorRed, 0)
	}
}

// removeSession deletes the session file, drops it from the list and keeps
// the selection in range.
func (v *SessionsView) removeSession(session core.Session) error {
	if err := os.Remove(session.FilePath); err != nil {
		return err
	}
	for i, s := range v.sessions {
		if s.ID == session.ID {
			v.sessions = append(v.sessions[:i], v.sessions[i+1:]...)
			break
		}
	}
	idx, offset := v.selectedIdx, v.scrollOffset
	v.rebuildDisplay()
	if n := len(v.displayedSessions); n > 0 {
		v.selectedIdx = min(idx, n-1)
		v.scrollOffset = min(offset, v.selectedIdx)
	}
	return nil
}

func (v *SessionsView) archiveSession(session core.Session) {
	v.screen.Clear()
	y := v.height/2 - 3
	v.drawCenteredText(y, i18n.MsgArchiveConfirm.Get(),
		tcell.StyleDefault.Foreground(tcell.ColorYellow).Bold(true))
	y += 2
	prompt := session.FirstPrompt
	if len(prompt) > v.width-10 {
		prompt = prompt[:v.width-13] + "..."
	}
	v.drawCenteredText(y, prompt, tcell.StyleDefault.Foreground(tcell.ColorWhite))
	y += 2
	archiveDir := filepath.Join(filepath.Dir(session.FilePath), "archived")
	dest := filepath.Join(archiveDir, filepath.Base(session.FilePath))
	v.drawCenteredText(y, fmt.Sprintf("→ %s", archiveDir),
		tcell.StyleDefault.Foreground(tcell.ColorGray))
	y += 2
	v.drawCenteredText(y, fmt.Sprintf("%s  %s", i18n.ConfirmYes.Get(), i18n.ConfirmNo.Get()),
		tcell.StyleDefault.Foreground(tcell.ColorLightCyan))
	v.screen.Show()

	ev := v.screen.PollEvent()
	if keyEv, ok := ev.(*tcell.EventKey); ok {
		if keyEv.Rune() == 'y' || keyEv.Rune() == 'Y' {
			var err error
			if err = os.MkdirAll(archiveDir, 0o755); err == nil {
				err = os.Rename(session.FilePath, dest)
			}
			v.screen.Clear()
			mid := v.height / 2
			if err != nil {
				v.drawCenteredText(mid, fmt.Sprintf("Error: %v", err),
					tcell.StyleDefault.Foreground(tcell.ColorRed))
			} else {
				v.drawCenteredText(mid, i18n.MsgArchiveSuccess.Get(),
					tcell.StyleDefault.Foreground(tcell.ColorGreen))
				for i, s := range v.sessions {
					if s.ID == session.ID {
						v.sessions = append(v.sessions[:i], v.sessions[i+1:]...)
						break
					}
				}
				v.rebuildDisplay()
			}
			v.drawCenteredText(mid+2, "按任意键继续...",
				tcell.StyleDefault.Foreground(tcell.ColorGray))
			v.screen.Show()
			v.screen.PollEvent()
		}
	}
}

// wordWrap splits a string into lines of at most maxWidth runes.
func wordWrap(text string, maxWidth int) []string {
	if len(text) == 0 {
		return []string{""}
	}
	if len([]rune(text)) <= maxWidth {
		return []string{text}
	}

	var lines []string
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}

	current := ""
	for _, word := range words {
		if current == "" {
			current = word
		} else if len([]rune(current))+1+len([]rune(word)) <= maxWidth {
			current += " " + word
		} else {
			lines = append(lines, current)
			current = word
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

func (v *SessionsView) drawText(x, y int, text string, style tcell.Style) {
	textwidth.Draw(v.screen, x, y, text, style)
}

func (v *SessionsView) drawCenteredText(y int, text string, style tcell.Style) {
	textwidth.DrawCentered(v.screen, v.width, y, text, style)
}

func (v *SessionsView) drawHorizontalLine(y int) {
	for x := 2; x < v.width-2; x++ {
		v.screen.SetContent(x, y, '─', nil, tcell.StyleDefault.
			Foreground(tcell.ColorGray))
	}
}
