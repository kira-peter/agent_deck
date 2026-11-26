package projects

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"github.com/kira-peter/agent_deck/pkg/core"
	"github.com/kira-peter/agent_deck/pkg/i18n"
	"github.com/kira-peter/agent_deck/pkg/ui/rowfx"
	"github.com/kira-peter/agent_deck/pkg/views"
	"github.com/kira-peter/agent_deck/pkg/views/dirpicker"
)

// cw measures display width the way terminals draw it: CJK is two cells, but
// "ambiguous" glyphs such as ─ and ↑ stay one cell even on a Chinese locale.
var cw = func() *runewidth.Condition {
	c := runewidth.NewCondition()
	c.EastAsianWidth = false
	return c
}()

// ProjectsView displays all projects
type ProjectsView struct {
	screen       tcell.Screen
	projects     []core.Project
	src          core.Source
	width        int
	height       int
	selectedIdx  int
	scrollOffset int
	totalUsage   core.Usage

	// gapAt is the visible project row left blank while rows slide up after
	// a delete; rows below it are drawn one row lower. -1 when not animating.
	gapAt int

	// dim is set while the list is drawn as the backdrop of a popup.
	dim bool
}

// NewProjectsView creates a new projects view
func NewProjectsView() views.View {
	return &ProjectsView{
		src:          core.CurrentSource(),
		selectedIdx:  -1, // 默认选中 "新项目" 选项
		scrollOffset: 0,
		gapAt:        -1,
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
	projects, err := v.src.GetProjects()
	if err != nil {
		return err
	}
	sortProjects(projects)

	v.projects = projects
	v.refreshUsage()

	return nil
}

// sortProjects orders projects by last activity (most recent first); projects
// with no activity come last, sorted by path.
func sortProjects(projects []core.Project) {
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
}

// refreshUsage recomputes per-project usage and the all-projects total.
func (v *ProjectsView) refreshUsage() {
	v.src.FillUsage(v.projects)
	v.totalUsage = core.Usage{}
	for _, p := range v.projects {
		v.totalUsage.Add(p.Usage)
	}
}

func (v *ProjectsView) Render() error {
	v.renderPage(false)
	v.screen.Show()
	return nil
}

// renderPage draws the whole page without showing it. dim draws it in gray,
// as the backdrop of a popup.
func (v *ProjectsView) renderPage(dim bool) {
	v.screen.Clear()
	v.dim = dim
	v.renderTitle()
	v.renderTableHeader()
	v.renderProjectList()
	v.renderUsagePanel()
	v.renderHints()
	v.dim = false
}

// dimStyle is what everything on the page is drawn with behind a popup.
var dimStyle = tcell.StyleDefault.Foreground(tcell.NewRGBColor(90, 90, 90))

// st returns style, or the dimmed style while a popup is open.
func (v *ProjectsView) st(style tcell.Style) tcell.Style {
	if v.dim {
		return dimStyle
	}
	return style
}

func (v *ProjectsView) renderTitle() {
	title := fmt.Sprintf("%s (%d)", i18n.ViewProjects.Get(), len(v.projects))
	y := 1
	v.drawCenteredText(y, title, v.st(tcell.StyleDefault.
		Foreground(tcell.ColorLightGreen).Bold(true)))
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
		v.drawText(x, y, h.text, v.st(tcell.StyleDefault.
			Foreground(tcell.ColorLightCyan).Bold(true)))
		x += h.width
	}

	v.drawHorizontalLine(y + 1)
}

// projectListY is the screen row of the first project (below the "new
// project" row and its separator).
const projectListY = 10

// maxProjectRows is how many project rows the list shows at the current size.
func (v *ProjectsView) maxProjectRows() int {
	maxVisible := v.height - 12 // Leave space for title and hints
	if v.usagePanelVisible() {
		maxVisible = v.height - 19 // ...and for the usage panel
	}
	return core.VisibleRows(maxVisible) - 1 // -1 for the new project row
}

func (v *ProjectsView) renderProjectList() {
	startY := 7
	startX := 4
	rows := v.maxProjectRows()

	// 渲染 "新项目" 选项
	newProjectStyle := tcell.StyleDefault.Foreground(tcell.ColorLightGreen).Bold(true)
	if v.selectedIdx == -1 {
		newProjectStyle = newProjectStyle.
			Background(tcell.ColorDarkGreen).
			Foreground(tcell.ColorWhite)
	}
	v.drawText(startX, startY, i18n.NewProjectOption.Get(), v.st(newProjectStyle))

	// 分隔线
	v.drawHorizontalLine(startY + 1)

	// 如果没有项目
	if len(v.projects) == 0 {
		v.drawCenteredText(startY+3, i18n.StatusNoData.Get(),
			v.st(tcell.StyleDefault.Foreground(tcell.ColorGray)))
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
		if adjustedIdx >= v.scrollOffset+rows {
			v.scrollOffset = adjustedIdx - rows + 1
		}
	}

	// 渲染可见的项目（跳过新项目和分隔线）
	for row := 0; row < rows; row++ {
		// while closing a gap, rows below it still show the project above
		i := v.scrollOffset + row
		if v.gapAt >= 0 {
			if row == v.gapAt {
				continue
			}
			if row > v.gapAt {
				i--
			}
		}
		if i >= len(v.projects) {
			break
		}
		project := v.projects[i]
		y := projectListY + row

		// Determine style
		style := tcell.StyleDefault.Foreground(tcell.ColorWhite)
		if i == v.selectedIdx {
			style = tcell.StyleDefault.
				Foreground(tcell.ColorBlack).
				Background(tcell.ColorLightGreen)
		}
		style = v.st(style)

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

const usagePanelMinHeight = 28

func (v *ProjectsView) usagePanelVisible() bool {
	return v.height >= usagePanelMinHeight
}

func (v *ProjectsView) renderUsagePanel() {
	if !v.usagePanelVisible() {
		return
	}

	scope := i18n.UsageAllProjects.Get()
	usage := v.totalUsage
	if v.selectedIdx >= 0 && v.selectedIdx < len(v.projects) {
		scope = v.projects[v.selectedIdx].Path
		usage = v.projects[v.selectedIdx].Usage
	}
	if r := []rune(scope); len(r) > v.width-20 && v.width > 24 {
		scope = "..." + string(r[len(r)-(v.width-23):])
	}

	y := v.height - 10
	v.drawHorizontalLine(y)

	headerStyle := v.st(tcell.StyleDefault.Foreground(tcell.ColorLightCyan))
	valueStyle := v.st(tcell.StyleDefault.Foreground(tcell.ColorWhite))
	x := 4

	titleStyle := v.st(tcell.StyleDefault.Foreground(tcell.ColorLightGreen).Bold(true))
	v.drawText(x, y+1, i18n.UsageTitle.Get()+": ", titleStyle)
	v.drawText(x+cw.StringWidth(i18n.UsageTitle.Get())+2, y+1, scope, valueStyle)

	columns := []struct {
		header string
		value  string
	}{
		{i18n.UsageInput.Get(), core.FormatTokens(usage.InputTokens)},
		{i18n.UsageOutput.Get(), core.FormatTokens(usage.OutputTokens)},
		{i18n.UsageCacheRead.Get(), core.FormatTokens(usage.CacheReadTokens)},
		{i18n.UsageCacheWrite.Get(), core.FormatTokens(usage.CacheWriteTokens)},
		{i18n.UsageLinesAdded.Get(), v.lineStat(usage.LinesAdded)},
		{i18n.UsageLinesRemoved.Get(), v.lineStat(usage.LinesRemoved)},
		{i18n.UsageWorkTime.Get(), core.FormatHours(usage.Duration)},
	}
	for i, c := range columns {
		cx := x + i*usageColWidth
		v.drawText(cx, y+2, padLeft(c.header, usageColWidth-1), headerStyle)
		v.drawText(cx, y+3, padLeft(c.value, usageColWidth-1), valueStyle)
	}
}

const usageColWidth = 12

// lineStat formats a lines-added/removed count, or "-" when the agent does
// not record it (Codex).
func (v *ProjectsView) lineStat(n int) string {
	if !v.src.HasLineStats() {
		return "-"
	}
	return fmt.Sprintf("%d", n)
}

// padLeft right-aligns s within width display cells (CJK-aware).
func padLeft(s string, width int) string {
	if gap := width - cw.StringWidth(s); gap > 0 {
		return strings.Repeat(" ", gap) + s
	}
	return s
}

func (v *ProjectsView) renderHints() {
	y := v.height - 2
	v.drawHorizontalLine(y - 1)

	hints := i18n.HintProjectActions.Get()
	v.drawCenteredText(y, hints, v.st(tcell.StyleDefault.Foreground(tcell.ColorGray)))
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
				return v.openProject(v.projects[v.selectedIdx])
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
			case 'd': // Delete project (with confirmation)
				if v.selectedIdx >= 0 && v.selectedIdx < len(v.projects) {
					v.deleteProject(v.projects[v.selectedIdx])
				}
			case 'D': // Shift+D: delete immediately, no confirmation
				if v.selectedIdx >= 0 && v.selectedIdx < len(v.projects) {
					v.quickDeleteProject(v.projects[v.selectedIdx])
				}
			case 'r', 'R': // Refresh
				projects, err := v.src.GetProjects()
				if err == nil {
					sortProjects(projects)
					v.projects = projects
					v.refreshUsage()
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

// openProject opens the project in the current agent. With existing sessions
// the user first picks between a new session and resuming one of them.
func (v *ProjectsView) openProject(project core.Project) error {
	sessions := v.loadProjectSessions(project)
	if len(sessions) == 0 {
		return v.launchAgent(project.Path, "")
	}

	switch choice := v.chooseSession(project, sessions); choice {
	case resumeCancel:
		return nil
	case resumeNew:
		return v.launchAgent(project.Path, "")
	default:
		return v.launchAgent(project.Path, sessions[choice].ID)
	}
}

// launchAgent starts the current agent (claude or codex) in projectPath in a
// new window. A non-empty sessionID resumes that session instead of starting
// a new one.
func (v *ProjectsView) launchAgent(projectPath, sessionID string) error {
	// Check if the directory exists
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		v.showResultBox(fmt.Sprintf(i18n.MsgProjectDirMissing.Get(), projectPath), tcell.ColorRed, 0)
		return nil
	}

	// The tab title is the project folder (Claude only; Codex ignores it).
	exe, args := v.src.LaunchArgs(projectPath, sessionID, filepath.Base(projectPath))
	quoted := make([]string, len(args))
	for i, a := range args {
		quoted[i] = "'" + psQuote(a) + "'"
	}
	argList := ""
	if len(quoted) > 0 {
		argList = " -ArgumentList " + strings.Join(quoted, ",")
	}
	// Start-Process opens a new terminal window running the agent
	cmd := exec.Command("powershell", "-Command",
		fmt.Sprintf("Start-Process -FilePath '%s'%s -WorkingDirectory '%s'",
			psQuote(exe), argList, psQuote(projectPath)))

	if err := cmd.Run(); err != nil {
		v.showResultBox(fmt.Sprintf(i18n.MsgLaunchFailed.Get(), v.src.Kind().Label(), err), tcell.ColorRed, 0)
		return nil
	}

	// Success - exit the manager
	return fmt.Errorf("quit")
}

// deleteProject deletes a project after the user confirms in a popup. The
// delete animation is the feedback, so success shows no message box.
func (v *ProjectsView) deleteProject(project core.Project) {
	if !v.src.CanDelete() {
		v.showResultBox(i18n.MsgDeleteUnsupported.Get(), tcell.ColorYellow, 0)
		return
	}
	if v.confirmDeleteProject(project) {
		v.removeProject(project)
	}
}

// quickDeleteProject deletes a project immediately, without any confirmation
// or success message. Failures are still reported.
func (v *ProjectsView) quickDeleteProject(project core.Project) {
	if !v.src.CanDelete() {
		v.showResultBox(i18n.MsgDeleteUnsupported.Get(), tcell.ColorYellow, 0)
		return
	}
	v.removeProject(project)
}

// removeProject deletes the project's data and then animates its row away:
// the shot erases it, the list is reloaded and the rows below slide up.
// Failures are reported in a popup.
func (v *ProjectsView) removeProject(project core.Project) {
	v.renderPage(true) // keep the dimmed backdrop while the agent CLI runs
	v.screen.Show()
	if err := v.src.DeleteProject(project); err != nil {
		v.showResultBox(fmt.Sprintf("%s: %v", i18n.DeleteDlgFailed.Get(), err), tcell.ColorRed, 0)
		v.reloadProjects()
		return
	}

	v.Render() // the list as it was, without any popup on top
	gap := v.selectedIdx - v.scrollOffset
	rowfx.Shot(v.screen, projectListY+gap)

	offset := v.scrollOffset
	v.reloadProjects()
	v.scrollOffset = min(offset, max(v.selectedIdx, 0))
	if v.scrollOffset == offset && v.selectedIdx >= 0 {
		visible := min(v.maxProjectRows(), len(v.projects)-v.scrollOffset)
		rowfx.Slide(gap, visible, func(at int) { v.gapAt = at }, func() { v.Render() })
	}
}

// reloadProjects re-reads the project list and keeps the selection in range.
func (v *ProjectsView) reloadProjects() {
	projects, err := v.src.GetProjects()
	if err != nil {
		return
	}
	sortProjects(projects)
	v.projects = projects
	v.refreshUsage()

	if v.selectedIdx >= len(v.projects) {
		v.selectedIdx = len(v.projects) - 1 // -1 selects the "new project" row
	}
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
				return v.launchAgent(selectedPath, "")
			}
			// 取消或返回 - 返回项目列表
			return nil
		}
	}
}

// Helper drawing functions

func (v *ProjectsView) drawText(x, y int, text string, style tcell.Style) {
	for _, r := range text {
		v.screen.SetContent(x, y, r, nil, style)
		if w := cw.RuneWidth(r); w > 1 {
			x += w
		} else {
			x++
		}
	}
}

func (v *ProjectsView) drawCenteredText(y int, text string, style tcell.Style) {
	v.drawCenteredTextIn(0, v.width, y, text, style)
}

func (v *ProjectsView) drawHorizontalLine(y int) {
	for x := 2; x < v.width-2; x++ {
		v.screen.SetContent(x, y, '─', nil, tcell.StyleDefault.
			Foreground(tcell.ColorGray))
	}
}

// psQuote escapes a value for use inside a single-quoted PowerShell string.
func psQuote(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
