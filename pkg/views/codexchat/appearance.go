package codexchat

import (
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/kira-peter/agent_deck/pkg/ui/textwidth"
	visual "github.com/will-to-meaning/terminal_visuals"
)

var (
	currentTheme = "ink"
	ink          = visual.Ink()
	baseStyle    = tcell.StyleDefault.Foreground(inkColor(ink.Text)).Background(inkColor(ink.Background))
	borderStyle  = baseStyle.Foreground(inkColor(ink.Border))
	focusStyle   = baseStyle.Foreground(inkColor(ink.Accent)).Bold(true)
	selectedTab  = focusStyle.Background(inkColor(ink.Selection))
	focusSweep   = visual.Sweep{Duration: 720 * time.Millisecond, Radius: 9}
)

func applyTheme(id string) {
	currentTheme = "ink"
	for _, theme := range visual.Themes() {
		if id == theme.ID {
			currentTheme = id
			break
		}
	}
	ink = visual.PaletteFor(currentTheme)
	baseStyle = tcell.StyleDefault.Foreground(inkColor(ink.Text)).Background(inkColor(ink.Background))
	borderStyle = baseStyle.Foreground(inkColor(ink.Border))
	focusStyle = baseStyle.Foreground(inkColor(ink.Accent)).Bold(true)
	selectedTab = focusStyle.Background(inkColor(ink.Selection))
	chartDim = baseStyle.Foreground(inkColor(ink.Muted))
}

func themeName(id string) string {
	switch id {
	case "black":
		return local("Classic black", "原版黑色")
	case "violet":
		return local("Violet", "暮紫")
	default:
		return local("Ink", "墨色")
	}
}

func inkColor(color visual.RGB) tcell.Color { return tcell.NewHexColor(int32(color)) }

func (v *workspaceView) scheduleFrame(elapsed time.Duration) {
	v.stopFrame()
	if v.workspace.motionOff {
		return
	}
	var delay time.Duration
	switch {
	case v.motion != nil:
		delay = 16 * time.Millisecond
	case v.closingAll:
		delay = 30 * time.Millisecond
	case v.fanClosing() || v.menu.kind != menuNone:
		delay = 30 * time.Millisecond
	case len(v.workspace.tabs) == 0:
		delay = 180 * time.Millisecond
	case v.previewWindow() >= 0 && time.Since(v.menu.flashAt) < 640*time.Millisecond:
		delay = 40 * time.Millisecond
	case !v.workspace.overview && elapsed < focusSweep.Duration:
		delay = 40 * time.Millisecond
	}
	if delay > 0 {
		v.frameTimer = time.AfterFunc(delay, v.workspace.wake)
	}
}

func (v *workspaceView) drawPaneFrame(bounds paneBounds, focused, flashing bool, elapsed time.Duration) {
	w, h := bounds.width, bounds.height
	length := 2*(w+h) - 4
	band := focusSweep.At(elapsed, length)
	if focused {
		band = (visual.Sweep{Duration: focusSweep.Duration, Radius: float64(length) / 11}).At(elapsed, length)
	}
	light := visual.RGB(0xb8ffc4)
	style := func(i int) tcell.Style {
		if !focused {
			return borderStyle
		}
		strength := band.Strength(i)
		if flashing {
			strength = 1
		}
		return baseStyle.Foreground(inkColor(visual.Mix(0x468d68, light, strength))).Bold(true)
	}
	set := func(x, y, i int, r rune) { v.screen.SetContent(bounds.x+x, bounds.y+y, r, nil, style(i)) }
	topLeft, topRight, bottomLeft, bottomRight, horizontal, vertical := '┌', '┐', '└', '┘', '─', '│'
	if focused {
		topLeft, topRight, bottomLeft, bottomRight, horizontal, vertical = '┏', '┓', '┗', '┛', '━', '┃'
	}
	for x := 1; x < w-1; x++ {
		set(x, 0, x, horizontal)
		set(x, h-1, 2*(w-1)+h-1-x, horizontal)
	}
	for y := 1; y < h-1; y++ {
		set(w-1, y, w-1+y, vertical)
		set(0, y, length-y, vertical)
	}
	set(0, 0, 0, topLeft)
	set(w-1, 0, w-1, topRight)
	set(w-1, h-1, w+h-2, bottomRight)
	set(0, h-1, 2*w+h-3, bottomLeft)
}

func (v *workspaceView) stopFrame() {
	if v.frameTimer != nil {
		v.frameTimer.Stop()
		v.frameTimer = nil
	}
}

func (v *workspaceView) elapsed() time.Duration {
	if v.workspace.motionOff {
		return 0
	}
	return time.Since(v.focusSince)
}

func (v *workspaceView) drawPaneRule(p *paneScreen, elapsed time.Duration) {
	if !p.focused {
		textwidth.Draw(p, 0, 0, strings.Repeat("─", p.width), borderStyle)
		return
	}
	band := focusSweep.At(elapsed, p.width)
	rest := visual.Mix(ink.Border, ink.Accent, 0.57)
	for x := range p.width {
		color := visual.Mix(rest, ink.Highlight, band.Strength(x))
		if p.flashing {
			color = ink.Highlight
		}
		p.SetContent(x, 0, '━', nil, baseStyle.Foreground(inkColor(color)))
	}
}

func (v *workspaceView) drawTabText(x, width int, label string, active bool, elapsed time.Duration) {
	style := chartDim
	if active {
		style = selectedTab
	}
	textwidth.Draw(v.screen, x, 1, textwidth.PadRight(textwidth.Truncate(label, width), width), style)
	if !active {
		return
	}
	band := (visual.Sweep{Duration: focusSweep.Duration, Radius: 4}).At(elapsed, width)
	for col := 0; col < width; {
		r, comb, _, cells := v.screen.GetContent(x+col, 1)
		if strength := band.Strength(col); strength > 0 {
			fg := inkColor(visual.Mix(ink.Accent, ink.Highlight, strength))
			bg := inkColor(visual.Mix(ink.Selection, ink.Raised, strength*0.32))
			v.screen.SetContent(x+col, 1, r, comb, style.Foreground(fg).Background(bg))
		}
		col += max(cells, 1)
	}
}

func (v *workspaceView) renderEmpty(width, height int, elapsed time.Duration) {
	screen := v.screen
	textwidth.Draw(screen, 2, 0, local("Agent Workspace", "Agent 工作区"), focusStyle)
	w := min(54, width-6)
	x, y := (width-w)/2, (height-5)/2
	visual.Starfield(width-4, height-5, min(80, width*height/35), 37, elapsed, func(point visual.Speck) {
		px, py := point.X+2, point.Y+3
		if px >= x-1 && px <= x+w && py >= y-1 && py <= y+5 {
			return
		}
		color := visual.Mix(ink.Background, ink.Muted, point.Brightness)
		screen.SetContent(px, py, '·', nil, baseStyle.Foreground(inkColor(color)))
	})
	style := baseStyle.Background(inkColor(ink.Surface))
	edge := style.Foreground(inkColor(ink.Border))
	textwidth.Draw(screen, x, y, "╭"+strings.Repeat("─", w-2)+"╮", edge)
	for row := 1; row <= 3; row++ {
		textwidth.Draw(screen, x, y+row, "│"+strings.Repeat(" ", w-2)+"│", edge)
	}
	textwidth.Draw(screen, x, y+4, "╰"+strings.Repeat("─", w-2)+"╯", edge)
	catalog := []string{local("Open an agent session from Projects or Sessions", "在「项目」或「会话」中打开编程助手"), local("It will appear here as a tab.", "打开后，会话会显示在这里。")}
	for row, line := range catalog {
		textwidth.Draw(screen, x+3, y+1+row, textwidth.Truncate(line, w-6), style)
	}
	textwidth.Draw(screen, 2, height-1, local("F1 Options · Alt+M Motion · Esc Back", "F1 选项 · Alt+M 动效 · Esc 返回"), chartDim)
}

func paintRow(screen tcell.Screen, x, y, width int, style tcell.Style) {
	for col := range width {
		screen.SetContent(x+col, y, ' ', nil, style)
	}
}
