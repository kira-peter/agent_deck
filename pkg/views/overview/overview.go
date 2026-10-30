package overview

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/yourusername/claude-manager/pkg/core"
	"github.com/yourusername/claude-manager/pkg/i18n"
	"github.com/yourusername/claude-manager/pkg/views"
)

// OverviewView displays summary statistics
type OverviewView struct {
	screen   tcell.Screen
	overview core.Overview
	claudeDir *core.ClaudeDir
	width    int
	height   int
}

// NewOverviewView creates a new overview view
func NewOverviewView() views.View {
	config := core.GetConfig()
	return &OverviewView{
		claudeDir: core.NewClaudeDir(config.GetClaudeDir()),
	}
}

func (v *OverviewView) Metadata() views.Metadata {
	return views.Metadata{
		ID:     "overview",
		NameEn: "Overview",
		NameZh: "总览",
		DescEn: "View Claude Code session, memory and project status",
		DescZh: "查看 Claude Code 的会话、记忆和项目状态",
		Icon:   "🏠",
	}
}

func (v *OverviewView) Init(screen tcell.Screen) error {
	v.screen = screen
	v.width, v.height = screen.Size()

	// Load overview data
	overview, err := v.claudeDir.GetOverview()
	if err != nil {
		return err
	}
	v.overview = overview

	return nil
}

func (v *OverviewView) Render() error {
	v.screen.Clear()

	// Title
	v.renderTitle()

	// Statistics
	v.renderStatistics()

	// Active sessions
	v.renderActiveSessions()

	// Recent sessions
	v.renderRecentSessions()

	// Hints
	v.renderHints()

	v.screen.Show()
	return nil
}

func (v *OverviewView) renderTitle() {
	title := i18n.ViewOverview.Get()
	y := 1
	v.drawCenteredText(y, title, tcell.StyleDefault.
		Foreground(tcell.ColorLightGreen).Bold(true))
	v.drawHorizontalLine(y + 2)
}

func (v *OverviewView) renderStatistics() {
	startY := 4
	startX := 10

	stats := []struct {
		label string
		value string
	}{
		{i18n.LabelTotalProjects.Get(), fmt.Sprintf("%d", v.overview.ProjectCount)},
		{i18n.LabelTotalSessions.Get(), fmt.Sprintf("%d", v.overview.SessionCount)},
		{i18n.LabelTotalMemories.Get(), fmt.Sprintf("%d", v.overview.MemoryCount)},
		{i18n.LabelTotalSize.Get(), core.FormatSize(v.overview.TotalSize)},
	}

	for i, stat := range stats {
		y := startY + i
		text := fmt.Sprintf("%-20s: %s", stat.label, stat.value)
		v.drawText(startX, y, text, tcell.StyleDefault.Foreground(tcell.ColorWhite))
	}
}

func (v *OverviewView) renderActiveSessions() {
	if len(v.overview.ActiveSessions) == 0 {
		return
	}

	startY := 9
	startX := 10

	// Header
	header := i18n.LabelActiveSessions.Get()
	v.drawText(startX, startY, header, tcell.StyleDefault.
		Foreground(tcell.ColorLightCyan).Bold(true))

	// List active sessions
	for i, session := range v.overview.ActiveSessions {
		if i >= 3 {
			break // Show max 3
		}
		y := startY + 1 + i
		text := fmt.Sprintf("  PID %d - %s (%s)",
			session.PID,
			session.CWD,
			session.Status)
		v.drawText(startX, y, text, tcell.StyleDefault.Foreground(tcell.ColorGray))
	}
}

func (v *OverviewView) renderRecentSessions() {
	if len(v.overview.RecentSessions) == 0 {
		return
	}

	startY := 14
	startX := 10

	// Header
	header := i18n.LabelRecentSessions.Get()
	v.drawText(startX, startY, header, tcell.StyleDefault.
		Foreground(tcell.ColorLightCyan).Bold(true))

	// List recent sessions
	for i, session := range v.overview.RecentSessions {
		if i >= 5 {
			break // Show max 5
		}
		y := startY + 1 + i

		// Truncate first prompt if too long
		prompt := session.FirstPrompt
		if len(prompt) > 50 {
			prompt = prompt[:47] + "..."
		}

		text := fmt.Sprintf("  %s - %s",
			core.FormatRelativeTime(session.Modified),
			prompt)
		v.drawText(startX, y, text, tcell.StyleDefault.Foreground(tcell.ColorGray))
	}
}

func (v *OverviewView) renderHints() {
	y := v.height - 2
	v.drawHorizontalLine(y - 1)

	hints := "ESC: Back  R: Refresh  Q: Quit"
	v.drawCenteredText(y, hints, tcell.StyleDefault.Foreground(tcell.ColorGray))
}

func (v *OverviewView) HandleEvent(event tcell.Event) error {
	switch ev := event.(type) {
	case *tcell.EventKey:
		switch ev.Key() {
		case tcell.KeyEscape:
			return fmt.Errorf("back") // Signal to go back
		case tcell.KeyRune:
			switch ev.Rune() {
			case 'q', 'Q':
				return fmt.Errorf("quit")
			case 'r', 'R':
				// Refresh data
				overview, err := v.claudeDir.GetOverview()
				if err == nil {
					v.overview = overview
				}
				return nil
			}
		}
	case *tcell.EventResize:
		v.width, v.height = v.screen.Size()
		v.screen.Sync()
	}
	return nil
}

func (v *OverviewView) Cleanup() error {
	return nil
}

// Helper drawing functions

func (v *OverviewView) drawText(x, y int, text string, style tcell.Style) {
	for i, r := range text {
		v.screen.SetContent(x+i, y, r, nil, style)
	}
}

func (v *OverviewView) drawCenteredText(y int, text string, style tcell.Style) {
	x := (v.width - len(text)) / 2
	if x < 0 {
		x = 0
	}
	v.drawText(x, y, text, style)
}

func (v *OverviewView) drawHorizontalLine(y int) {
	for x := 2; x < v.width-2; x++ {
		v.screen.SetContent(x, y, '─', nil, tcell.StyleDefault.
			Foreground(tcell.ColorGray))
	}
}
