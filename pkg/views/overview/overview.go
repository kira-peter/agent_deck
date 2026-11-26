package overview

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/kira-peter/agent_deck/pkg/core"
	"github.com/kira-peter/agent_deck/pkg/i18n"
	"github.com/kira-peter/agent_deck/pkg/views"
	"github.com/kira-peter/agent_deck/pkg/ui/textwidth"
)

// OverviewView displays summary statistics
type OverviewView struct {
	screen   tcell.Screen
	overview core.Overview
	src      core.Source
	width    int
	height   int
}

// NewOverviewView creates a new overview view
func NewOverviewView() views.View {
	return &OverviewView{
		src: core.CurrentSource(),
	}
}

func (v *OverviewView) Metadata() views.Metadata {
	return views.Metadata{
		ID:     "overview",
		NameEn: "Overview",
		NameZh: "总览",
		DescEn: "View session, memory and project status",
		DescZh: "查看会话、记忆和项目状态",
		Icon:   "🏠",
	}
}

func (v *OverviewView) Init(screen tcell.Screen) error {
	v.screen = screen
	v.width, v.height = screen.Size()

	// Load overview data
	overview, err := v.src.GetOverview()
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
	title := i18n.ViewOverview.Get() + " · " + v.src.Kind().Label()
	y := 1
	v.drawCenteredText(y, title, tcell.StyleDefault.
		Foreground(tcell.ColorLightGreen).Bold(true))
	v.drawHorizontalLine(y + 2)
}

func (v *OverviewView) renderStatistics() {
	startY := 4
	startX := 10

	type stat struct {
		label string
		value string
	}
	stats := []stat{
		{i18n.LabelTotalProjects.Get(), fmt.Sprintf("%d", v.overview.ProjectCount)},
		{i18n.LabelTotalSessions.Get(), fmt.Sprintf("%d", v.overview.SessionCount)},
	}
	stats = append(stats, stat{i18n.LabelTotalMemories.Get(), fmt.Sprintf("%d", v.overview.MemoryCount)})
	stats = append(stats, stat{i18n.LabelTotalSize.Get(), core.FormatSize(v.overview.TotalSize)})

	for i, stat := range stats {
		y := startY + i
		text := fmt.Sprintf("%s: %s", textwidth.PadRight(stat.label, 20), stat.value)
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

	// List recent sessions, stopping above the footer line
	for i, session := range v.overview.RecentSessions {
		if i >= 5 || startY+1+i >= v.height-3 {
			break // Show max 5
		}
		y := startY + 1 + i

		// Truncate first prompt by display width so CJK is not cut mid-rune
		prompt := textwidth.Truncate(strings.Join(strings.Fields(session.FirstPrompt), " "), 50)

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
				overview, err := v.src.GetOverview()
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
	textwidth.Draw(v.screen, x, y, text, style)
}

func (v *OverviewView) drawCenteredText(y int, text string, style tcell.Style) {
	textwidth.DrawCentered(v.screen, v.width, y, text, style)
}

func (v *OverviewView) drawHorizontalLine(y int) {
	for x := 2; x < v.width-2; x++ {
		v.screen.SetContent(x, y, '─', nil, tcell.StyleDefault.
			Foreground(tcell.ColorGray))
	}
}
