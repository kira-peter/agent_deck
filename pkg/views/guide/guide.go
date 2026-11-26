package guide

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/kira-peter/agent_deck/pkg/i18n"
	"github.com/kira-peter/agent_deck/pkg/views"
	"github.com/kira-peter/agent_deck/pkg/ui/textwidth"
)

// GuideView displays documentation about Claude's memory system
type GuideView struct {
	screen       tcell.Screen
	width        int
	height       int
	scrollOffset int
	content      []string
}

func NewGuideView() views.View {
	return &GuideView{
		scrollOffset: 0,
	}
}

func (v *GuideView) Metadata() views.Metadata {
	return views.Metadata{
		ID:     "guide",
		NameEn: "Guide",
		NameZh: "指南",
		DescEn: "Learn about Claude's memory system",
		DescZh: "了解 Claude 的记忆系统",
		Icon:   "📖",
	}
}

func (v *GuideView) Init(screen tcell.Screen) error {
	v.screen = screen
	v.width, v.height = screen.Size()
	v.loadContent()
	return nil
}

func (v *GuideView) loadContent() {
	mgr := i18n.GetManager()

	if mgr.IsEnglish() {
		v.content = []string{
			"# Claude Memory System Guide",
			"",
			"## Auto-Memory Feature",
			"Claude Code automatically saves useful context to auto-memory.",
			"Manage with the /memory command.",
			"",
			"## Memory Types",
			"",
			"1. USER - Information about your role, preferences, and knowledge",
			"2. FEEDBACK - Guidance on how to approach work (corrections & confirmations)",
			"3. PROJECT - Ongoing work, goals, initiatives within the project",
			"4. REFERENCE - Pointers to external systems and resources",
			"",
			"## MEMORY.md Index File",
			"- Acts as an index for all memory entries",
			"- Limited to 25KB or 200 lines",
			"- Always loaded into conversation context",
			"",
			"## CLAUDE.md Files",
			"",
			"Five types of CLAUDE.md locations:",
			"1. Project root (./CLAUDE.md) - Primary project context",
			"2. Local overrides (./.claude.local.md) - Personal settings",
			"3. Global defaults (~/.claude/CLAUDE.md) - User-wide defaults",
			"4. Package-specific (./packages/*/CLAUDE.md) - Module-level",
			"5. Subdirectory (any nested location) - Feature-specific",
			"",
			"## Data Storage Structure",
			"",
			"~/.claude/",
			"  projects/",
			"    [encoded-path]/         # e.g., D--jarvis",
			"      [session-id].jsonl    # Conversation transcript",
			"      [session-id]/          # Session artifacts",
			"        subagents/",
			"        tool-results/",
			"      sessions-index.json   # Fast session lookup",
			"      memory/",
			"        MEMORY.md           # Project memory",
			"  sessions/",
			"    [pid].json              # Active session metadata",
			"",
			"## Path Encoding",
			"File paths are encoded for directory names:",
			"  D:\\jarvis → D--jarvis",
			"  C:\\Users\\TU → C--Users-TU",
			"  /home/user → home-user",
			"",
			"Press ESC to go back",
		}
	} else {
		v.content = []string{
			"# Claude 记忆系统指南",
			"",
			"## 自动记忆功能",
			"Claude Code 会自动保存有用的上下文到记忆系统。",
			"使用 /memory 命令进行管理。",
			"",
			"## 记忆类型",
			"",
			"1. USER - 关于用户角色、偏好和知识的信息",
			"2. FEEDBACK - 工作方法指导（纠正和确认）",
			"3. PROJECT - 项目内的进行中工作、目标和计划",
			"4. REFERENCE - 指向外部系统和资源的指针",
			"",
			"## MEMORY.md 索引文件",
			"- 充当所有记忆条目的索引",
			"- 限制为 25KB 或 200 行",
			"- 总是加载到对话上下文中",
			"",
			"## CLAUDE.md 文件",
			"",
			"五种 CLAUDE.md 文件位置：",
			"1. 项目根目录 (./CLAUDE.md) - 主项目上下文",
			"2. 本地覆盖 (./.claude.local.md) - 个人设置",
			"3. 全局默认 (~/.claude/CLAUDE.md) - 用户全局默认",
			"4. 包特定 (./packages/*/CLAUDE.md) - 模块级别",
			"5. 子目录（任何嵌套位置）- 功能特定",
			"",
			"## 数据存储结构",
			"",
			"~/.claude/",
			"  projects/",
			"    [编码路径]/              # 例如 D--jarvis",
			"      [会话ID].jsonl         # 对话记录",
			"      [会话ID]/              # 会话工件",
			"        subagents/",
			"        tool-results/",
			"      sessions-index.json   # 快速会话查找",
			"      memory/",
			"        MEMORY.md           # 项目记忆",
			"  sessions/",
			"    [进程ID].json           # 活跃会话元数据",
			"",
			"## 路径编码",
			"文件路径被编码为目录名：",
			"  D:\\jarvis → D--jarvis",
			"  C:\\Users\\TU → C--Users-TU",
			"  /home/user → home-user",
			"",
			"按 ESC 返回",
		}
	}
}

func (v *GuideView) Render() error {
	v.screen.Clear()

	title := i18n.ViewGuide.Get()
	v.drawCenteredText(1, title, tcell.StyleDefault.
		Foreground(tcell.ColorLightGreen).Bold(true))
	v.drawHorizontalLine(3)

	// Render content with scroll
	startY := 5
	maxVisible := v.height - 8

	for i := v.scrollOffset; i < len(v.content) && i < v.scrollOffset+maxVisible; i++ {
		line := v.content[i]
		y := startY + (i - v.scrollOffset)

		// Style based on content
		style := tcell.StyleDefault.Foreground(tcell.ColorWhite)
		if strings.HasPrefix(line, "#") {
			style = tcell.StyleDefault.Foreground(tcell.ColorLightCyan).Bold(true)
		} else if strings.HasPrefix(line, "  ") {
			style = tcell.StyleDefault.Foreground(tcell.ColorGray)
		}

		v.drawText(4, y, line, style)
	}

	// Scroll indicator
	if len(v.content) > maxVisible {
		scrollInfo := fmt.Sprintf("[%d/%d]", v.scrollOffset+1, len(v.content))
		v.drawText(v.width-15, v.height-3, scrollInfo,
			tcell.StyleDefault.Foreground(tcell.ColorGray))
	}

	// Hints
	v.drawHorizontalLine(v.height - 3)
	v.drawCenteredText(v.height-2, "↑↓: Scroll  ESC: Back  T: Toggle Language",
		tcell.StyleDefault.Foreground(tcell.ColorGray))

	v.screen.Show()
	return nil
}

func (v *GuideView) HandleEvent(event tcell.Event) error {
	maxVisible := v.height - 8
	maxScroll := len(v.content) - maxVisible
	if maxScroll < 0 {
		maxScroll = 0
	}

	switch ev := event.(type) {
	case *tcell.EventKey:
		switch ev.Key() {
		case tcell.KeyEscape:
			return fmt.Errorf("back")
		case tcell.KeyUp:
			if v.scrollOffset > 0 {
				v.scrollOffset--
			}
		case tcell.KeyDown:
			if v.scrollOffset < maxScroll {
				v.scrollOffset++
			}
		case tcell.KeyRune:
			switch ev.Rune() {
			case 'q', 'Q':
				return fmt.Errorf("quit")
			case 'k':
				if v.scrollOffset > 0 {
					v.scrollOffset--
				}
			case 'j':
				if v.scrollOffset < maxScroll {
					v.scrollOffset++
				}
			case 't', 'T':
				v.loadContent()
				v.scrollOffset = 0
			}
		}
	case *tcell.EventResize:
		v.width, v.height = v.screen.Size()
		v.screen.Sync()
	}
	return nil
}

func (v *GuideView) Cleanup() error {
	return nil
}

func (v *GuideView) drawText(x, y int, text string, style tcell.Style) {
	textwidth.Draw(v.screen, x, y, text, style)
}

func (v *GuideView) drawCenteredText(y int, text string, style tcell.Style) {
	textwidth.DrawCentered(v.screen, v.width, y, text, style)
}

func (v *GuideView) drawHorizontalLine(y int) {
	for x := 2; x < v.width-2; x++ {
		v.screen.SetContent(x, y, '─', nil, tcell.StyleDefault.
			Foreground(tcell.ColorGray))
	}
}
