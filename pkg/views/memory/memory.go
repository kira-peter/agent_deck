package memory

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/yourusername/claude-manager/pkg/core"
	"github.com/yourusername/claude-manager/pkg/i18n"
	"github.com/yourusername/claude-manager/pkg/views"
	"github.com/yourusername/claude-manager/pkg/ui/textwidth"
)

// MemoryView displays memory files
type MemoryView struct {
	screen       tcell.Screen
	memories     []core.Memory
	claudeDir    *core.ClaudeDir
	width        int
	height       int
	selectedIdx  int
	scrollOffset int
}

func NewMemoryView() views.View {
	config := core.GetConfig()
	return &MemoryView{
		claudeDir:    core.NewClaudeDir(config.GetClaudeDir()),
		selectedIdx:  0,
		scrollOffset: 0,
	}
}

func (v *MemoryView) Metadata() views.Metadata {
	return views.Metadata{
		ID:     "memory",
		NameEn: "Memory",
		NameZh: "记忆",
		DescEn: "View and manage memory files",
		DescZh: "查看和管理记忆文件",
		Icon:   "🧠",
	}
}

func (v *MemoryView) Init(screen tcell.Screen) error {
	v.screen = screen
	v.width, v.height = screen.Size()

	memories, err := v.claudeDir.GetMemories()
	if err != nil {
		return err
	}
	v.memories = memories

	return nil
}

func (v *MemoryView) Render() error {
	v.screen.Clear()

	title := fmt.Sprintf("%s (%d)", i18n.ViewMemory.Get(), len(v.memories))
	v.drawCenteredText(1, title, tcell.StyleDefault.
		Foreground(tcell.ColorLightGreen).Bold(true))
	v.drawHorizontalLine(3)

	// Render memory list
	startY := 5
	if len(v.memories) == 0 {
		v.drawCenteredText(startY+2, i18n.StatusNoData.Get(),
			tcell.StyleDefault.Foreground(tcell.ColorGray))
	} else {
		for i, mem := range v.memories {
			y := startY + i*3
			if y >= v.height-5 {
				break
			}

			style := tcell.StyleDefault.Foreground(tcell.ColorWhite)
			if i == v.selectedIdx {
				style = tcell.StyleDefault.
					Foreground(tcell.ColorBlack).
					Background(tcell.ColorLightGreen)
			}

			v.drawText(4, y, mem.ProjectPath, style)
			v.drawText(4, y+1,
				fmt.Sprintf("  Entries: %d | Size: %s | Modified: %s",
					mem.EntryCount,
					core.FormatSize(mem.Size),
					core.FormatRelativeTime(mem.LastModified)),
				tcell.StyleDefault.Foreground(tcell.ColorGray))
		}
	}

	// Hints
	v.drawHorizontalLine(v.height - 3)
	v.drawCenteredText(v.height-2, "ESC: Back  R: Refresh  Q: Quit",
		tcell.StyleDefault.Foreground(tcell.ColorGray))

	v.screen.Show()
	return nil
}

func (v *MemoryView) HandleEvent(event tcell.Event) error {
	switch ev := event.(type) {
	case *tcell.EventKey:
		switch ev.Key() {
		case tcell.KeyEscape:
			return fmt.Errorf("back")
		case tcell.KeyUp:
			if v.selectedIdx > 0 {
				v.selectedIdx--
			}
		case tcell.KeyDown:
			if v.selectedIdx < len(v.memories)-1 {
				v.selectedIdx++
			}
		case tcell.KeyRune:
			switch ev.Rune() {
			case 'q', 'Q':
				return fmt.Errorf("quit")
			case 'k':
				if v.selectedIdx > 0 {
					v.selectedIdx--
				}
			case 'j':
				if v.selectedIdx < len(v.memories)-1 {
					v.selectedIdx++
				}
			case 'r', 'R':
				memories, err := v.claudeDir.GetMemories()
				if err == nil {
					v.memories = memories
				}
			}
		}
	case *tcell.EventResize:
		v.width, v.height = v.screen.Size()
		v.screen.Sync()
	}
	return nil
}

func (v *MemoryView) Cleanup() error {
	return nil
}

func (v *MemoryView) drawText(x, y int, text string, style tcell.Style) {
	textwidth.Draw(v.screen, x, y, text, style)
}

func (v *MemoryView) drawCenteredText(y int, text string, style tcell.Style) {
	textwidth.DrawCentered(v.screen, v.width, y, text, style)
}

func (v *MemoryView) drawHorizontalLine(y int) {
	for x := 2; x < v.width-2; x++ {
		v.screen.SetContent(x, y, '─', nil, tcell.StyleDefault.
			Foreground(tcell.ColorGray))
	}
}
