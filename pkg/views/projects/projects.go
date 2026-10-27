package projects

import (
	"fmt"
	"os"
	"os/exec"
	"sort"

	"github.com/gdamore/tcell/v2"
	"github.com/yourusername/claude-manager/pkg/core"
	"github.com/yourusername/claude-manager/pkg/i18n"
	"github.com/yourusername/claude-manager/pkg/views"
	"github.com/yourusername/claude-manager/pkg/views/dirpicker"
)

// ProjectsView displays all projects
type ProjectsView struct {
	screen    tcell.Screen
	projects  []core.Project
	claudeDir *core.ClaudeDir
	width     int
	height    int
	selectedIdx int
	scrollOffset int
}

// NewProjectsView creates a new projects view
func NewProjectsView() views.View {
	config := core.GetConfig()
	return &ProjectsView{
		claudeDir:    core.NewClaudeDir(config.GetClaudeDir()),
		selectedIdx:  -1, // 默认选中 "新项目" 选项
		scrollOffset: 0,
	}
}

func (v *ProjectsView) Metadata() views.Metadata {
	return views.Metadata{
		ID:     "projects",
		NameEn: "Projects",
		NameZh: "项目",
		DescEn: "Browse and manage all projects",
		DescZh: "浏览和管理所有项目",
		Icon:   "📁",
	}
}

func (v *ProjectsView) Init(screen tcell.Screen) error {
	v.screen = screen
	v.width, v.height = screen.Size()

	// Load projects
	projects, err := v.claudeDir.GetProjects()
	if err != nil {
		return err
	}

	// Sort by last activity (most recent first)
	// For projects with no activity (0001-01-01), sort by path
	sort.Slice(projects, func(i, j int) bool {
		// If both have valid activity times, compare them
		if !projects[i].LastActivity.IsZero() && !projects[j].LastActivity.IsZero() {
			return projects[i].LastActivity.After(projects[j].LastActivity)
		}
		// If only one has valid activity, it comes first
		if !projects[i].LastActivity.IsZero() {
			return true
		}
		if !projects[j].LastActivity.IsZero() {
			return false
		}
		// Both have no activity, sort by path alphabetically
		return projects[i].Path < projects[j].Path
	})

	v.projects = projects

	return nil
}

func (v *ProjectsView) Render() error {
	v.screen.Clear()

	// Title
	v.renderTitle()

	// Table header
	v.renderTableHeader()

	// Project list
	v.renderProjectList()

	// Hints
	v.renderHints()

	v.screen.Show()
	return nil
}

func (v *ProjectsView) renderTitle() {
	title := fmt.Sprintf("%s (%d)", i18n.ViewProjects.Get(), len(v.projects))
	y := 1
	v.drawCenteredText(y, title, tcell.StyleDefault.
		Foreground(tcell.ColorLightGreen).Bold(true))
	v.drawHorizontalLine(y + 2)
}

func (v *ProjectsView) renderTableHeader() {
	y := 4
	startX := 4

	headers := []struct {
		text  string
		width int
	}{
		{i18n.HeaderPath.Get(), 50},
		{i18n.HeaderSessions.Get(), 10},
		{i18n.HeaderSize.Get(), 10},
		{i18n.HeaderLastActivity.Get(), 15},
	}

	x := startX
	for _, h := range headers {
		v.drawText(x, y, h.text, tcell.StyleDefault.
			Foreground(tcell.ColorLightCyan).Bold(true))
		x += h.width
	}

	v.drawHorizontalLine(y + 1)
}

func (v *ProjectsView) renderProjectList() {
	startY := 7
	startX := 4
	maxVisible := v.height - 12 // Leave space for title and hints

	// 渲染 "新项目" 选项
	newProjectStyle := tcell.StyleDefault.Foreground(tcell.ColorLightGreen).Bold(true)
	if v.selectedIdx == -1 {
		newProjectStyle = newProjectStyle.
			Background(tcell.ColorDarkGreen).
			Foreground(tcell.ColorWhite)
	}
	v.drawText(startX, startY, i18n.NewProjectOption.Get(), newProjectStyle)

	// 分隔线
	v.drawHorizontalLine(startY + 1)

	// 如果没有项目
	if len(v.projects) == 0 {
		v.drawCenteredText(startY+3, i18n.StatusNoData.Get(),
			tcell.StyleDefault.Foreground(tcell.ColorGray))
		return
	}

	// 调整滚动偏移量（考虑 -1 的新项目选项）
	adjustedIdx := v.selectedIdx
	if adjustedIdx == -1 {
		adjustedIdx = 0
		v.scrollOffset = 0
	} else {
		if adjustedIdx < v.scrollOffset {
			v.scrollOffset = adjustedIdx
		}
		if adjustedIdx >= v.scrollOffset+maxVisible-1 { // -1 for new project row
			v.scrollOffset = adjustedIdx - maxVisible + 2
		}
	}

	// 渲染可见的项目（从 startY + 3 开始，跳过新项目和分隔线）
	for i := v.scrollOffset; i < len(v.projects) && i < v.scrollOffset+maxVisible-1; i++ {
		project := v.projects[i]
		y := startY + 3 + (i - v.scrollOffset)

		// Determine style
		style := tcell.StyleDefault.Foreground(tcell.ColorWhite)
		if i == v.selectedIdx {
			style = tcell.StyleDefault.
				Foreground(tcell.ColorBlack).
				Background(tcell.ColorLightGreen)
		}

		// Path (truncate if too long)
		path := project.Path
		if len(path) > 48 {
			path = "..." + path[len(path)-45:]
		}
		v.drawText(startX, y, fmt.Sprintf("%-48s", path), style)

		// Sessions count
		v.drawText(startX+50, y, fmt.Sprintf("%-8d", project.SessionCount), style)

		// Size
		v.drawText(startX+60, y, fmt.Sprintf("%-8s", core.FormatSize(project.TotalSize)), style)

		// Last activity
		timeStr := core.FormatRelativeTime(project.LastActivity)
		v.drawText(startX+70, y, timeStr, style)
	}
}

func (v *ProjectsView) renderHints() {
	y := v.height - 2
	v.drawHorizontalLine(y - 1)

	hints := i18n.HintProjectActions.Get()
	v.drawCenteredText(y, hints, tcell.StyleDefault.Foreground(tcell.ColorGray))
}

func (v *ProjectsView) HandleEvent(event tcell.Event) error {
	switch ev := event.(type) {
	case *tcell.EventKey:
		switch ev.Key() {
		case tcell.KeyEscape:
			return fmt.Errorf("back")
		case tcell.KeyEnter:
			// Open Claude in the selected project directory
			if v.selectedIdx == -1 {
				// 打开新项目 - 启动目录选择器
				return v.openNewProject()
			} else if v.selectedIdx >= 0 && v.selectedIdx < len(v.projects) {
				project := v.projects[v.selectedIdx]
				return v.openProjectInClaude(project.Path)
			}
		case tcell.KeyUp:
			v.movePrev()
		case tcell.KeyDown:
			v.moveNext()
		case tcell.KeyRune:
			switch ev.Rune() {
			case 'q', 'Q':
				return fmt.Errorf("quit")
			case 'k': // Vim-style up
				v.movePrev()
			case 'j': // Vim-style down
				v.moveNext()
			case 'd', 'D': // Delete project
				if v.selectedIdx >= 0 && v.selectedIdx < len(v.projects) {
					project := v.projects[v.selectedIdx]
					v.deleteProject(project)
				}
			case 'r', 'R': // Refresh
				projects, err := v.claudeDir.GetProjects()
				if err == nil {
					sort.Slice(projects, func(i, j int) bool {
						return projects[i].LastActivity.After(projects[j].LastActivity)
					})
					v.projects = projects
				}
			}
		}
	case *tcell.EventResize:
		v.width, v.height = v.screen.Size()
		v.screen.Sync()
	}
	return nil
}

func (v *ProjectsView) Cleanup() error {
	return nil
}

// Movement functions
func (v *ProjectsView) movePrev() {
	if v.selectedIdx > -1 { // 允许移动到 -1（新项目选项）
		v.selectedIdx--
	}
}

func (v *ProjectsView) moveNext() {
	if v.selectedIdx < len(v.projects)-1 {
		v.selectedIdx++
	}
}

// openProjectInClaude opens Claude Code in the specified project directory
func (v *ProjectsView) openProjectInClaude(projectPath string) error {
	// Check if the directory exists
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		// Show error message on screen
		v.screen.Clear()
		v.drawCenteredText(v.height/2, fmt.Sprintf("项目目录不存在: %s", projectPath),
			tcell.StyleDefault.Foreground(tcell.ColorRed))
		v.drawCenteredText(v.height/2+2, "按任意键继续...",
			tcell.StyleDefault.Foreground(tcell.ColorGray))
		v.screen.Show()
		v.screen.PollEvent() // Wait for any key
		return nil
	}

	// On Windows, use PowerShell to start a new terminal with Claude
	// Use 'start' to open a new window
	cmd := exec.Command("powershell", "-Command",
		fmt.Sprintf("Start-Process -FilePath 'claude' -ArgumentList '%s' -WorkingDirectory '%s'",
			projectPath, projectPath))

	// Try to start the command
	if err := cmd.Run(); err != nil {
		// Show error message on screen
		v.screen.Clear()
		v.drawCenteredText(v.height/2, fmt.Sprintf("无法启动 Claude: %v", err),
			tcell.StyleDefault.Foreground(tcell.ColorRed))
		v.drawCenteredText(v.height/2+2, "按任意键继续...",
			tcell.StyleDefault.Foreground(tcell.ColorGray))
		v.screen.Show()
		v.screen.PollEvent() // Wait for any key
		return nil
	}

	// Success - exit the manager
	return fmt.Errorf("quit")
}

// deleteProject deletes a project from .claude/projects/
func (v *ProjectsView) deleteProject(project core.Project) {
	// Show confirmation dialog
	v.screen.Clear()

	y := v.height/2 - 3
	v.drawCenteredText(y, "确认删除项目",
		tcell.StyleDefault.Foreground(tcell.ColorYellow).Bold(true))

	y += 2
	v.drawCenteredText(y, fmt.Sprintf("项目: %s", project.Path),
		tcell.StyleDefault.Foreground(tcell.ColorWhite))

	y += 1
	v.drawCenteredText(y, fmt.Sprintf("会话数: %d  |  大小: %s",
		project.SessionCount, core.FormatSize(project.TotalSize)),
		tcell.StyleDefault.Foreground(tcell.ColorGray))

	y += 2
	v.drawCenteredText(y, "这将删除该项目的所有会话和记忆数据",
		tcell.StyleDefault.Foreground(tcell.ColorRed))

	y += 2
	v.drawCenteredText(y, "按 Y 确认删除，按其他键取消",
		tcell.StyleDefault.Foreground(tcell.ColorLightCyan))

	v.screen.Show()

	// Wait for user input
	ev := v.screen.PollEvent()
	if keyEv, ok := ev.(*tcell.EventKey); ok {
		if keyEv.Rune() == 'y' || keyEv.Rune() == 'Y' {
			// Perform deletion
			v.performDelete(project)
		}
	}

	// Refresh the list and re-render
	projects, err := v.claudeDir.GetProjects()
	if err == nil {
		sort.Slice(projects, func(i, j int) bool {
			return projects[i].LastActivity.After(projects[j].LastActivity)
		})
		v.projects = projects

		// Adjust selectedIdx if needed
		if v.selectedIdx >= len(v.projects) && len(v.projects) > 0 {
			v.selectedIdx = len(v.projects) - 1
		}
	}
}

// performDelete actually deletes the project directory from .claude/projects/
func (v *ProjectsView) performDelete(project core.Project) {
	config := core.GetConfig()
	claudeDir := config.GetClaudeDir()

	// The project directory in .claude/projects/ is named using the encoded name
	projectDir := fmt.Sprintf("%s/projects/%s", claudeDir, project.EncodedName)

	// Delete the directory
	err := os.RemoveAll(projectDir)

	// Show result
	v.screen.Clear()
	y := v.height / 2

	if err != nil {
		v.drawCenteredText(y, fmt.Sprintf("删除失败: %v", err),
			tcell.StyleDefault.Foreground(tcell.ColorRed))
	} else {
		v.drawCenteredText(y, "项目已成功删除",
			tcell.StyleDefault.Foreground(tcell.ColorGreen))
	}

	v.drawCenteredText(y+2, "按任意键继续...",
		tcell.StyleDefault.Foreground(tcell.ColorGray))
	v.screen.Show()
	v.screen.PollEvent()
}

// openNewProject 启动目录选择器以打开新项目
func (v *ProjectsView) openNewProject() error {
	// 创建目录选择器
	picker := dirpicker.NewDirPickerView()
	if err := picker.Init(v.screen); err != nil {
		// 初始化失败（如终端太窄），显示错误并返回
		v.screen.Clear()
		v.drawCenteredText(v.height/2, fmt.Sprintf("无法启动目录选择器: %v", err),
			tcell.StyleDefault.Foreground(tcell.ColorRed))
		v.drawCenteredText(v.height/2+2, "按任意键继续...",
			tcell.StyleDefault.Foreground(tcell.ColorGray))
		v.screen.Show()
		v.screen.PollEvent()
		return nil
	}

	// 模态运行选择器
	for {
		picker.Render()
		ev := v.screen.PollEvent()
		err := picker.HandleEvent(ev)

		if err != nil {
			if err.Error() == "confirmed" {
				// 用户确认了选择
				selectedPath := picker.GetSelectedPath()
				return v.openProjectInClaude(selectedPath)
			}
			// 取消或返回 - 返回项目列表
			return nil
		}
	}
}

// Helper drawing functions

func (v *ProjectsView) drawText(x, y int, text string, style tcell.Style) {
	for i, r := range text {
		v.screen.SetContent(x+i, y, r, nil, style)
	}
}

func (v *ProjectsView) drawCenteredText(y int, text string, style tcell.Style) {
	x := (v.width - len(text)) / 2
	if x < 0 {
		x = 0
	}
	v.drawText(x, y, text, style)
}

func (v *ProjectsView) drawHorizontalLine(y int) {
	for x := 2; x < v.width-2; x++ {
		v.screen.SetContent(x, y, '─', nil, tcell.StyleDefault.
			Foreground(tcell.ColorGray))
	}
}
