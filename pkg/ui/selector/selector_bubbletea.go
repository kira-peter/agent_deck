package selector

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kira-peter/agentdeck/pkg/core"
	"github.com/kira-peter/agentdeck/pkg/i18n"
	"github.com/kira-peter/agentdeck/pkg/ui/animation"
	"github.com/kira-peter/agentdeck/pkg/ui/textwidth"
	"github.com/kira-peter/agentdeck/pkg/views"
)

// BubbleTeaSelector is the Bubble Tea version of selector
type BubbleTeaSelector struct {
	viewList    []views.Metadata
	selectedIdx int
	width       int
	height      int
	spinner     *animation.Spinner
	trans       *transition // page transition in progress, nil when idle
	notice      string      // one-off message shown under the title
}

// codexOnlyHidden lists the views that have no Codex counterpart yet and are
// left out of the menu in Codex mode.
var codexOnlyHidden = map[string]bool{"memory": true, "guide": true}

// NewBubbleTeaSelector creates a new Bubble Tea selector
func NewBubbleTeaSelector() *BubbleTeaSelector {
	s := &BubbleTeaSelector{
		selectedIdx: 0,
		spinner:     animation.NewSpinner(),
	}
	s.loadViews()
	s.spinner.Start()
	return s
}

// loadViews builds the menu for the current data source.
func (s *BubbleTeaSelector) loadViews() {
	all := views.GetRegistry().GetAll()
	codex := core.GetConfig().GetSource() == core.SourceCodex
	s.viewList = s.viewList[:0]
	for _, v := range all {
		if codex && codexOnlyHidden[v.ID] {
			continue
		}
		s.viewList = append(s.viewList, v)
	}
	s.selectedIdx = min(s.selectedIdx, len(s.viewList)-1)
}

// toggleSource switches between Claude Code and Codex, keeping the cursor on
// the same view when it exists in both menus.
func (s *BubbleTeaSelector) toggleSource() {
	cfg := core.GetConfig()
	next := core.SourceCodex
	if cfg.GetSource() == core.SourceCodex {
		next = core.SourceClaude
	}
	dir := cfg.GetClaudeDir()
	if next == core.SourceCodex {
		dir = cfg.GetCodexDir()
	}
	if !core.SourceAvailable(next) {
		s.notice = fmt.Sprintf(i18n.MsgSourceMissing.Get(), next.Label(), dir)
		return
	}
	s.notice = ""
	current := s.GetSelectedViewID()
	_ = cfg.SetSource(next)
	s.loadViews()
	s.SelectByID(current)
}

// SelectByID moves the selection to the view with the given ID. Unknown IDs
// leave the selection unchanged.
func (s *BubbleTeaSelector) SelectByID(id string) {
	for i, v := range s.viewList {
		if v.ID == id {
			s.selectedIdx = i
			return
		}
	}
}

// SetSize updates the dimensions
func (s *BubbleTeaSelector) SetSize(width, height int) {
	s.width = width
	s.height = height
}

// HandleKey handles keyboard input
// Returns: -1 (continue), -2 (quit), -3 (refresh), >=0 (view index selected)
func (s *BubbleTeaSelector) HandleKey(msg tea.KeyMsg) int {
	switch msg.String() {
	case "ctrl+c", "esc", "q", "Q":
		return -2 // Quit

	case "up", "k":
		if s.selectedIdx > 0 {
			s.selectedIdx--
			// 切换选项时随机选择动画（排除当前动画）
			currentAnim := s.spinner.GetAnimationType()
			s.spinner.SetAnimationType(animation.RandomAnimationTypeExcept(currentAnim))
		}
		return -1

	case "down", "j":
		if s.selectedIdx < len(s.viewList)-1 {
			s.selectedIdx++
			// 切换选项时随机选择动画（排除当前动画）
			currentAnim := s.spinner.GetAnimationType()
			s.spinner.SetAnimationType(animation.RandomAnimationTypeExcept(currentAnim))
		}
		return -1

	case "enter":
		return s.selectedIdx // Select

	case "f1":
		s.toggleSource()
		return -3 // Refresh

	case "t", "T":
		// Toggle language
		mgr := i18n.GetManager()
		mgr.Toggle()
		return -3 // Refresh
	}

	return -1
}

const (
	ansiReset = "\033[0m"
	ansiGray  = "\033[90m"
	ansiGreen = "\033[1;32m"
	ansiWarn  = "\033[1;33m"
)

// menuLine is one rendered line of the menu plus the width it needs to be
// shown uncut (0 when it may be truncated).
type menuLine struct {
	text     string
	minWidth int
}

// View renders the selector centered on the screen. When the terminal is too
// small to hold the whole menu it shows a short notice instead.
func (s *BubbleTeaSelector) View() string {
	if s.width == 0 {
		return "Loading..."
	}

	lines, selRow, ok := s.compose()
	if !ok {
		s.trans = nil
		return strings.Join(lines, "\n")
	}
	if s.trans != nil {
		return s.transitionView(lines, selRow)
	}
	return strings.Join(lines, "\n")
}

// compose lays out the full menu screen, one string per terminal row, and
// returns the row of the selected entry. ok is false when the window is too
// small, in which case lines hold the notice instead.
func (s *BubbleTeaSelector) compose() (lines []string, selRow int, ok bool) {
	header, body, footer := s.headerLines(), s.bodyLines(), s.footerLines()
	needW := 0
	for _, part := range [][]menuLine{header, body, footer} {
		for _, l := range part {
			needW = max(needW, l.minWidth)
		}
	}
	// at least one blank row between the body and the header / footer
	needH := len(header) + len(body) + len(footer) + 2
	if s.width < needW || s.height < needH {
		return strings.Split(s.tooSmallView(needW, needH), "\n"), 0, false
	}

	// header sticks to the top, footer to the bottom, the menu sits in the
	// middle of the space between them
	free := s.height - len(header) - len(body) - len(footer)
	above := free / 2
	var texts []string
	for _, l := range header {
		texts = append(texts, l.text)
	}
	for i := 0; i < above; i++ {
		texts = append(texts, "")
	}
	selRow = len(texts) + s.selectedIdx
	for _, l := range body {
		texts = append(texts, l.text)
	}
	for i := 0; i < free-above; i++ {
		texts = append(texts, "")
	}
	for _, l := range footer {
		texts = append(texts, l.text)
	}
	return texts, selRow, true
}

func (s *BubbleTeaSelector) headerLines() []menuLine {
	title := i18n.AppTitle.Get()
	if core.GetConfig().GetSource() == core.SourceCodex {
		title = i18n.AppTitleCodex.Get()
	}
	sub := s.centerStyledText(core.VersionLabel(), ansiGray)
	if s.notice != "" {
		sub = s.centerStyledText(s.notice, ansiWarn)
	}
	return []menuLine{
		{text: ""},
		{text: s.centerText(title), minWidth: calculateDisplayWidth(title) + 4},
		{text: sub},
		{text: s.horizontalLine()},
	}
}

func (s *BubbleTeaSelector) footerLines() []menuLine {
	desc := ""
	if s.selectedIdx >= 0 && s.selectedIdx < len(s.viewList) {
		view := s.viewList[s.selectedIdx]
		desc = view.DescZh
		if i18n.GetManager().IsEnglish() {
			desc = view.DescEn
		}
	}
	return []menuLine{
		{text: s.horizontalLine()},
		{text: s.centerText(desc)},
		{text: s.horizontalLine()},
		{text: s.centerText(i18n.HintNavigation.Get())},
	}
}

// name returns the entry's label in the current language.
func entryName(v views.Metadata) string {
	if i18n.GetManager().IsEnglish() {
		return v.NameEn
	}
	return v.NameZh
}

// entryWidth is the width of the widest entry including its spinner prefix.
func (s *BubbleTeaSelector) entryWidth() int {
	w := 0
	for _, view := range s.viewList {
		w = max(w, calculateDisplayWidth("⠋ "+entryName(view)))
	}
	return w
}

// entryStartX is the common left edge of the menu entries.
func (s *BubbleTeaSelector) entryStartX() int {
	return max((s.width-s.entryWidth())/2, 2)
}

// bodyLines are the menu entries, aligned on a common left edge.
func (s *BubbleTeaSelector) bodyLines() []menuLine {
	var lines []menuLine
	maxWidth := s.entryWidth()
	startX := s.entryStartX()

	for idx, view := range s.viewList {
		line := strings.Repeat(" ", startX)
		if idx == s.selectedIdx {
			line += fmt.Sprintf("%s%c %s%s", ansiGreen, s.spinner.GetFrame(), entryName(view), ansiReset)
		} else {
			line += "  " + entryName(view)
		}
		lines = append(lines, menuLine{line, maxWidth + 4})
	}
	return lines
}

// tooSmallView asks the user to enlarge the terminal, showing the current and
// required size.
func (s *BubbleTeaSelector) tooSmallView(needW, needH int) string {
	msg := []string{
		i18n.WindowTooSmall.Get(),
		"",
		fmt.Sprintf(i18n.WindowSizeNow.Get(), s.width, s.height),
		fmt.Sprintf(i18n.WindowSizeNeed.Get(), needW, needH),
		"",
		i18n.WindowTooSmallHint.Get(),
	}
	lines := make([]menuLine, len(msg))
	for i, m := range msg {
		color := ansiGray
		if i == 0 {
			color = ansiWarn
		}
		lines[i] = menuLine{text: s.centerStyledText(m, color)}
	}
	return s.vcenter(lines)
}

// vcenter pads the lines with blank rows above so they sit in the middle of
// the screen. Lines with a 0 minWidth are left to be cut by the terminal, so
// keep them on one row by never emitting more lines than the screen holds.
func (s *BubbleTeaSelector) vcenter(lines []menuLine) string {
	if len(lines) > s.height {
		lines = lines[:max(s.height, 0)]
	}
	texts := make([]string, len(lines))
	for i, l := range lines {
		texts[i] = l.text
	}
	top := (s.height - len(lines)) / 2
	return strings.Repeat("\n", top) + strings.Join(texts, "\n")
}

// centerText centers plain text, cutting it when wider than the screen so it
// never wraps onto a second row.
func (s *BubbleTeaSelector) centerText(text string) string {
	text = textwidth.Truncate(text, s.width-2)
	padding := max((s.width-calculateDisplayWidth(text))/2, 0)
	return strings.Repeat(" ", padding) + text
}

// centerStyledText centers text and wraps it in an ANSI color, padding by the
// visible width only.
func (s *BubbleTeaSelector) centerStyledText(text, color string) string {
	text = textwidth.Truncate(text, s.width-2)
	padding := max((s.width-calculateDisplayWidth(text))/2, 0)
	return strings.Repeat(" ", padding) + color + text + ansiReset
}

func (s *BubbleTeaSelector) horizontalLine() string {
	if s.width < 4 {
		return ""
	}
	return "  " + strings.Repeat("─", s.width-4)
}

// GetSelectedViewID 返回当前选中视图的 ID
func (s *BubbleTeaSelector) GetSelectedViewID() string {
	if s.selectedIdx >= 0 && s.selectedIdx < len(s.viewList) {
		return s.viewList[s.selectedIdx].ID
	}
	return ""
}
