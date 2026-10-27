package selector

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yourusername/claude-manager/pkg/core"
	"github.com/yourusername/claude-manager/pkg/i18n"
	"github.com/yourusername/claude-manager/pkg/ui/animation"
	"github.com/yourusername/claude-manager/pkg/views"
)

// BubbleTeaSelector is the Bubble Tea version of selector
type BubbleTeaSelector struct {
	viewList    []views.Metadata
	selectedIdx int
	width       int
	height      int
	spinner     *animation.Spinner
}

// NewBubbleTeaSelector creates a new Bubble Tea selector
func NewBubbleTeaSelector() *BubbleTeaSelector {
	registry := views.GetRegistry()
	s := &BubbleTeaSelector{
		viewList:    registry.GetAll(),
		selectedIdx: 0,
		spinner:     animation.NewSpinner(),
	}
	s.spinner.Start()
	return s
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

	case "t", "T":
		// Toggle language
		mgr := i18n.GetManager()
		mgr.Toggle()
		return -3 // Refresh
	}

	return -1
}

// View renders the selector
func (s *BubbleTeaSelector) View() string {
	if s.width == 0 {
		return "Loading..."
	}

	var b strings.Builder

	// Title
	title := i18n.AppTitle.Get()
	titleLine := s.centerText(title)
	b.WriteString("\n\n")
	b.WriteString(titleLine)
	b.WriteString("\n")
	b.WriteString(s.centerStyledText(core.VersionLabel(), "\033[90m"))
	b.WriteString("\n")
	b.WriteString(s.horizontalLine())
	b.WriteString("\n\n")

	// View list
	mgr := i18n.GetManager()
	isEnglish := mgr.IsEnglish()

	// Calculate max width for alignment
	maxWidth := 0
	for _, view := range s.viewList {
		name := view.NameZh
		if isEnglish {
			name = view.NameEn
		}
		text := fmt.Sprintf("⠋ %s", name)
		width := calculateDisplayWidth(text)
		if width > maxWidth {
			maxWidth = width
		}
	}

	// Calculate start position for centering
	startX := (s.width - maxWidth) / 2
	if startX < 2 {
		startX = 2
	}

	// Render view list
	for idx, view := range s.viewList {
		name := view.NameZh
		if isEnglish {
			name = view.NameEn
		}

		// Format text with animation for selected item
		var text string
		if idx == s.selectedIdx {
			frame := s.spinner.GetFrame()
			text = fmt.Sprintf("%c %s", frame, name)
		} else {
			text = fmt.Sprintf("  %s", name)
		}

		// Color and style
		line := strings.Repeat(" ", startX)
		if idx == s.selectedIdx {
			// Green color for selected
			line += fmt.Sprintf("\033[1;32m%s\033[0m", text)
		} else {
			// White for unselected
			line += text
		}

		b.WriteString(line)
		b.WriteString("\n")
	}

	// Description
	b.WriteString("\n")
	b.WriteString(s.horizontalLine())
	b.WriteString("\n")

	if s.selectedIdx >= 0 && s.selectedIdx < len(s.viewList) {
		view := s.viewList[s.selectedIdx]
		desc := view.DescZh
		if isEnglish {
			desc = view.DescEn
		}
		b.WriteString(s.centerText(desc))
	}
	b.WriteString("\n")

	// Hints
	b.WriteString("\n")
	b.WriteString(s.horizontalLine())
	b.WriteString("\n")
	hints := i18n.HintNavigation.Get()
	b.WriteString(s.centerText(hints))
	b.WriteString("\n")

	return b.String()
}

func (s *BubbleTeaSelector) centerText(text string) string {
	width := calculateDisplayWidth(text)
	padding := (s.width - width) / 2
	if padding < 0 {
		padding = 0
	}
	return strings.Repeat(" ", padding) + text
}

// centerStyledText centers text and wraps it in an ANSI color, padding by the
// visible width only.
func (s *BubbleTeaSelector) centerStyledText(text, color string) string {
	width := calculateDisplayWidth(text)
	padding := (s.width - width) / 2
	if padding < 0 {
		padding = 0
	}
	return strings.Repeat(" ", padding) + color + text + "\033[0m"
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
