package codexchat

import (
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/kira-peter/agent_deck/pkg/ui/textwidth"
	visual "github.com/will-to-meaning/terminal_visuals"
)

const (
	minPaneWidth  = 36
	minPaneHeight = 12
	workspaceTop  = 3
)

type sessionStatus struct {
	text  string
	mark  rune
	color tcell.Color
}

// status is read while holding the session lock.
func (c *chat) status() sessionStatus {
	switch {
	case len(c.prompts) > 0:
		return sessionStatus{local("Waiting for input", "等待选择"), '!', inkColor(ink.Warning)}
	case c.working:
		return sessionStatus{local("Running", "运行中"), '●', inkColor(ink.Accent)}
	case c.ready:
		switch c.lastTurnStatus {
		case "completed":
			return sessionStatus{local("Done", "已完成"), '✓', inkColor(ink.Success)}
		case "failed":
			return sessionStatus{local("Failed", "失败"), '×', inkColor(ink.Danger)}
		case "interrupted":
			return sessionStatus{local("Interrupted", "已中断"), '–', inkColor(ink.Muted)}
		default:
			return sessionStatus{local("Ready", "就绪"), '○', inkColor(ink.Muted)}
		}
	case c.notice != "":
		return sessionStatus{local("Unavailable", "不可用"), '×', inkColor(ink.Danger)}
	default:
		return sessionStatus{local("Connecting", "正在连接"), '…', inkColor(ink.Muted)}
	}
}

func (v *workspaceView) Render() error {
	screen := v.screen
	w := v.workspace
	width, height := screen.Size()
	screen.Fill(' ', baseStyle)
	screen.HideCursor()
	defer screen.Show()
	defer v.renderRootFan(width, height)
	if width < minPaneWidth || height < minPaneHeight+workspaceTop+1 {
		if v.closingAll {
			v.startPaneMotion()
			v.scheduleFrame(v.elapsed())
		} else {
			v.stopFrame()
		}
		textwidth.Draw(screen, 0, 0, textwidth.Truncate(local("Resize terminal · Esc Back", "请放大终端 · Esc 返回"), width), chartDim)
		return nil
	}
	elapsed := v.elapsed()
	v.startPaneMotion()
	defer v.scheduleFrame(elapsed)
	if len(w.tabs) == 0 && v.motion == nil {
		v.renderEmpty(width, height, elapsed)
		return nil
	}
	area := paneBounds{width: width, height: height - workspaceTop - 1, y: workspaceTop}
	focus := w.active
	preview := v.previewWindow()
	if preview >= 0 {
		focus = preview
	}
	panes, paneCount := v.paneFrames(area, focus)
	mode := local("Single", "单窗")
	if w.split {
		mode = local("Split", "分屏")
	}
	var waiting, running int
	statuses := make([]sessionStatus, len(w.tabs))
	for i, c := range w.tabs {
		c.mu.Lock()
		statuses[i] = c.status()
		c.mu.Unlock()
		switch statuses[i].mark {
		case '!':
			waiting++
		case '●':
			running++
		}
	}
	header := fmt.Sprintf(local("Agent Workspace  ·  %s %d/%d  ·  %d running  ·  %d waiting", "Agent 工作区  ·  %s %d/%d  ·  %d 运行中  ·  %d 等待选择"),
		mode, paneCount, len(w.tabs), running, waiting)
	chrome := focusStyle.Background(inkColor(ink.Surface))
	paintRow(screen, 0, 0, width, chrome)
	textwidth.Draw(screen, 2, 0, textwidth.Truncate(header, width-4), chrome)
	selecting := !w.overview || preview >= 0
	tabFocus := -1
	if selecting {
		tabFocus = focus
	}
	if len(w.tabs) > 0 {
		v.renderTabs(width, statuses, elapsed, tabFocus)
	}
	for _, entry := range panes {
		bounds := entry.bounds
		c := entry.chat
		focused := selecting && focus < len(w.tabs) && c == w.tabs[focus]
		flashing := preview >= 0 && c == w.tabs[preview] && !w.motionOff && visual.Blink(time.Since(v.menu.flashAt))
		content := bounds
		framed := entry.framed
		if framed {
			content = paneBounds{x: bounds.x + 1, y: bounds.y + 1, width: bounds.width - 2, height: bounds.height - 2}
		}
		pane := &paneScreen{Screen: screen, paneBounds: content, focused: focused && !entry.reveal, dimmed: selecting && !focused && !entry.reveal, flashing: flashing}
		var top, bottom int
		if entry.reveal {
			top, bottom = visual.SplitLines(bounds.height, entry.openness)
			top, bottom = bounds.y+top, bounds.y+bottom
			pane.clipped, pane.clipTop, pane.clipBottom = true, top, bottom
		}
		c.mu.Lock()
		c.screen = pane
		c.render()
		c.screen = nil
		c.mu.Unlock()
		if entry.reveal {
			v.drawSplitLines(bounds, top, bottom)
		} else if framed {
			v.drawPaneFrame(bounds, focused, flashing, elapsed)
		} else {
			v.drawPaneRule(pane, elapsed)
		}
	}
	footer := local("F1 Options · F4/F5 Switch · F6 Split · F7 Close · F8 Overview · Esc Back · Alt+M Motion", "F1 选项 · F4/F5 切换 · F6 分屏 · F7 关闭 · F8 总览 · Esc 返回 · Alt+M 动效")
	if w.overview {
		footer = local("Overview · F4/F5 Focus a window · F1 Options · Esc Back", "总览 · F4/F5 选择窗口 · F1 选项 · Esc 返回")
	}
	if waiting > 0 {
		footer = local("Tab Focus a waiting form · ", "Tab 跳到待填写的表单 · ") + footer
	}
	if v.message != "" {
		footer = v.message
	}
	if v.closingAll {
		footer = local("Closing windows…", "正在逐个关闭窗口…")
	}
	paintRow(screen, 0, height-1, width, chrome)
	textwidth.Draw(screen, 2, height-1, textwidth.Truncate(footer, width-4), chrome)
	return nil
}

func (v *workspaceView) renderTabs(width int, statuses []sessionStatus, elapsed time.Duration, focus int) {
	w := v.workspace
	tabWidth := min(24, max(12, (width-4)/min(len(w.tabs), 4)))
	capacity := max((width-4)/tabWidth, 1)
	anchor := focus
	if anchor < 0 {
		anchor = w.active
	}
	start := min(max(anchor-capacity/2, 0), max(len(w.tabs)-capacity, 0))
	for i := start; i < min(start+capacity, len(w.tabs)); i++ {
		x := 2 + (i-start)*tabWidth
		label := strconv.Itoa(i+1) + " " + filepath.Base(w.tabs[i].project)
		if i == focus {
			label = "▸ " + label
		}
		v.drawTabText(x, tabWidth-2, label, i == focus, elapsed)
		style := baseStyle
		if i == focus {
			style = selectedTab
		}
		v.screen.SetContent(x+tabWidth-2, 1, statuses[i].mark, nil, style.Foreground(statuses[i].color).Bold(true))
	}
}

type paneBounds struct{ x, y, width, height int }

func arrangePanes(area paneBounds, count, active int, split bool) ([]paneBounds, int) {
	if count <= 1 || !split {
		return []paneBounds{area}, active
	}
	maxCols := max(1, (area.width+1)/(minPaneWidth+3))
	maxRows := max(1, (area.height+1)/(minPaneHeight+3))
	cols := min(maxCols, max(2, (count+maxRows-1)/maxRows))
	capacity := cols * maxRows
	start := active / capacity * capacity
	visible := min(count-start, capacity)
	cols = min(cols, visible)
	rows := (visible + cols - 1) / cols
	panes := make([]paneBounds, visible)
	for i := range panes {
		row, col := i/cols, i%cols
		columnsInRow := min(cols, visible-row*cols)
		left, right := col*(area.width+1)/columnsInRow, (col+1)*(area.width+1)/columnsInRow-1
		top, bottom := row*(area.height+1)/rows, (row+1)*(area.height+1)/rows-1
		panes[i] = paneBounds{x: area.x + left, y: area.y + top, width: right - left, height: bottom - top}
	}
	return panes, start
}

// paneScreen clips local drawing to a rectangle and gives cursor control to
// the focused pane. The workspace owns clearing, event polling, and flushing.
type paneScreen struct {
	tcell.Screen
	paneBounds
	focused             bool
	dimmed, flashing    bool
	dimStyles           map[tcell.Style]tcell.Style
	clipped             bool
	clipTop, clipBottom int
}

func (p *paneScreen) Size() (int, int) { return p.width, p.height }
func (p *paneScreen) Clear()           {}
func (p *paneScreen) Show()            {}
func (p *paneScreen) HideCursor()      {}

func (p *paneScreen) contains(x, y int) bool {
	return x >= 0 && y >= 0 && x < p.width && y < p.height && (!p.clipped || y+p.y > p.clipTop && y+p.y < p.clipBottom)
}

func (p *paneScreen) SetContent(x, y int, mainc rune, combc []rune, style tcell.Style) {
	if p.contains(x, y) && (x < p.width-1 || textwidth.StringWidth(string(mainc)) <= 1) {
		if p.dimmed {
			dim, ok := p.dimStyles[style]
			if !ok {
				if p.dimStyles == nil {
					p.dimStyles = make(map[tcell.Style]tcell.Style)
				}
				fg, bg, _ := style.Decompose()
				dim = style.Foreground(dimColor(fg, ink.Text, .32)).Background(dimColor(bg, ink.Background, .30)).Bold(false)
				if ul := style.GetUnderlineColor(); ul != tcell.ColorDefault {
					dim = dim.Underline(dimColor(ul, ink.Text, .32))
				}
				p.dimStyles[style] = dim
			}
			style = dim
		}
		p.Screen.SetContent(x+p.x, y+p.y, mainc, combc, style)
	}
}

func dimColor(color tcell.Color, fallback visual.RGB, strength float64) tcell.Color {
	rgb := fallback
	if hex := color.Hex(); hex >= 0 {
		rgb = visual.RGB(hex)
	}
	return inkColor(visual.Mix(ink.Background, rgb, strength))
}

func (p *paneScreen) GetContent(x, y int) (rune, []rune, tcell.Style, int) {
	if p.contains(x, y) {
		return p.Screen.GetContent(x+p.x, y+p.y)
	}
	return ' ', nil, tcell.StyleDefault, 1
}

func (p *paneScreen) ShowCursor(x, y int) {
	if p.focused && p.contains(x, y) {
		p.Screen.ShowCursor(x+p.x, y+p.y)
	}
}
