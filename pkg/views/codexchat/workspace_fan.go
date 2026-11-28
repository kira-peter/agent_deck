package codexchat

import (
	"math"
	"slices"
	"time"

	"github.com/kira-peter/agent_deck/pkg/ui/textwidth"
	visual "github.com/will-to-meaning/terminal_visuals"
)

type fanParent struct {
	menu workspaceMenu
	rows []menuRow
}

func (v *workspaceView) rootMenuRows() []menuRow {
	open, running := v.workspace.exitStatus()
	empty := open == 0
	return []menuRow{
		{label: local("Theme", "主题"), detail: themeName(currentTheme)},
		{label: local("Window model", "窗口模型"), disabled: empty},
		{label: local("Open session", "打开会话")},
		{label: local("Restart window", "重启窗口"), disabled: empty},
		{label: local("Close window", "关闭窗口"), disabled: empty},
		{label: local("Close all windows and quit", "关闭所有窗口并退出"), disabled: running > 0},
	}
}

func (v *workspaceView) fanClosing() bool {
	return !v.workspace.motionOff && !v.fanClosedAt.IsZero() && time.Since(v.fanClosedAt) < visual.FanCloseDuration
}

func (v *workspaceView) renderRootFan(width, height int) {
	if v.menu.kind == menuNone && !v.fanClosing() {
		return
	}
	area := v.fanArea(width, height)
	if area.Width < 22 || area.Height < 5 {
		return
	}
	_, rows := v.menuRows()
	closing := v.menu.kind == menuNone
	selected, elapsed := v.menu.list.Selected, time.Since(v.menu.openedAt)
	if closing {
		rows = v.rootMenuRows()
		selected, elapsed = v.fanClosedSelection, time.Since(v.fanClosedAt)
	}
	var bendX, bendFrom, bendTo int
	if n := len(v.fanTrail); n > 0 && !closing {
		parent := v.fanTrail[n-1]
		parentRows := slices.Clone(parent.rows)
		if selected := parent.menu.list.Selected; selected >= 0 && selected < len(parentRows) {
			parentRows[selected].detail = ""
		}
		anchor := v.drawFan(area, parentRows, parent.menu.list.Selected, visual.FanOpenDuration(len(parentRows)), false, true)
		if anchor.Width > 0 {
			end := area.X + area.Width
			required := 22
			for _, row := range rows {
				required = max(required, textwidth.StringWidth(fanLabel(row))+11)
			}
			if next := anchor.X + anchor.Width + 1; end-next >= required {
				area.X, area.Width = next, end-next
			} else {
				bottom := area.Y + area.Height
				bendX, bendFrom = anchor.X-2, anchor.Y
				area.X, area.Width = bendX, end-bendX
				if below := bottom - anchor.Y - 1; below >= 3 {
					area.Y, area.Height = anchor.Y+1, below
				} else {
					area.Height = anchor.Y - area.Y
				}
				bendTo = area.Y + area.Height/2
			}
		}
	}
	if v.workspace.motionOff {
		elapsed = visual.FanOpenDuration(len(rows))
	}
	anchor := v.drawFan(area, rows, selected, elapsed, closing, false)
	if bendFrom != bendTo {
		accent := baseStyle.Foreground(inkColor(ink.Accent))
		for y := min(bendFrom, bendTo) + 1; y < max(bendFrom, bendTo); y++ {
			v.screen.SetContent(bendX, y, '│', nil, accent)
		}
		corner, junction := '└', '┬'
		if bendTo < bendFrom {
			corner, junction = '┌', '┴'
		}
		v.screen.SetContent(bendX, bendFrom, junction, nil, accent)
		v.screen.SetContent(bendX, bendTo, corner, nil, accent)
	}
	if !closing {
		v.screen.HideCursor()
		if v.menu.kind == menuModelText && anchor.Width > 0 {
			v.screen.ShowCursor(min(anchor.X+anchor.Width, area.X+area.Width-1), anchor.Y)
		}
		if v.menu.notice != "" {
			y := min(area.Y+area.Height-1, area.Y+area.Height/2+2)
			textwidth.Draw(v.screen, area.X+2, y, textwidth.Truncate(v.menu.notice, area.Width-4), chartDim)
		}
	}
}

func fanLabel(row menuRow) string {
	if row.detail == "" {
		return row.label
	}
	return row.label + "  ·  " + row.detail
}

func (v *workspaceView) drawFan(area visual.FanRect, rows []menuRow, selected int, elapsed time.Duration, closing, muted bool) visual.FanRect {
	frame := visual.FanAt(len(rows), elapsed, closing)
	positions := visual.SideFan(area, len(rows), selected)
	if len(positions) == 0 {
		return visual.FanRect{}
	}
	labels := make([]string, len(rows))
	for i, row := range rows {
		labels[i] = textwidth.Truncate(fanLabel(row), area.Width-11)
	}
	v.shadeFan(area, labels, frame.Shade)
	middle := area.Y + area.Height/2
	accent := baseStyle.Foreground(inkColor(ink.Accent))
	if !closing || frame.Connector > 0 {
		v.screen.SetContent(area.X, middle, '│', nil, accent)
	}
	var anchor visual.FanRect
	for _, pos := range positions {
		if muted && pos.Distance != 0 {
			continue
		}
		strength := frame.Ring(pos.Distance)
		if strength < .02 {
			continue
		}
		label := labels[pos.Index]
		labelWidth := textwidth.StringWidth(label)
		if pos.Distance == 0 {
			anchor = visual.FanRect{X: pos.X, Y: pos.Y, Width: labelWidth, Height: 1}
		}
		for x := pos.X - 1; x <= pos.X+labelWidth; x++ {
			v.clearFanCell(area, x, pos.Y)
		}
		fg := ink.Text
		if pos.Distance == 0 {
			fg = ink.Warning
		}
		if rows[pos.Index].disabled {
			fg = ink.Muted
		}
		fg = visual.Mix(ink.Background, fg, strength)
		if pos.Distance == 0 && !closing && !muted && !v.workspace.motionOff && elapsed >= visual.FanOpenDuration(len(rows)) && !rows[pos.Index].disabled {
			band := (visual.Sweep{Duration: 900 * time.Millisecond, Radius: 2.2}).Loop(elapsed-visual.FanOpenDuration(len(rows)), 750*time.Millisecond, labelWidth)
			x := pos.X
			for _, r := range label {
				color := visual.Mix(fg, ink.Highlight, band.Strength(x-pos.X))
				v.screen.SetContent(x, pos.Y, r, nil, baseStyle.Foreground(inkColor(color)).Bold(true))
				x += max(1, textwidth.StringWidth(string(r)))
			}
		} else {
			textwidth.Draw(v.screen, pos.X, pos.Y, label, baseStyle.Foreground(inkColor(fg)).Bold(pos.Distance == 0))
		}
	}
	for i := 1; i <= int(math.Ceil(frame.Connector*7)); i++ {
		x := area.X + i
		v.clearFanCell(area, x, middle)
		v.screen.SetContent(x, middle, '─', nil, accent)
	}
	return anchor
}

func (v *workspaceView) fanArea(width, height int) visual.FanRect {
	area := paneBounds{y: workspaceTop, width: width, height: height - workspaceTop - 1}
	w := v.workspace
	if v.fanPane >= 0 && v.fanPane < len(w.tabs) {
		panes, start := arrangePanes(area, len(w.tabs), v.fanPane, w.split)
		if index := v.fanPane - start; index >= 0 && index < len(panes) {
			area = panes[index]
		}
	}
	return visual.FanRect{X: area.x, Y: area.y, Width: area.width, Height: area.height}
}

func (v *workspaceView) shadeFan(area visual.FanRect, labels []string, strength float64) {
	if strength <= 0 {
		return
	}
	const feather = 3.0
	middle := area.Y + area.Height/2
	left, right, top, bottom := area.X+9, area.X+9, middle, middle
	for selected := range labels {
		for _, pos := range visual.SideFan(area, len(labels), selected) {
			left = min(left, pos.X-1)
			right = max(right, pos.X+textwidth.StringWidth(labels[pos.Index])+1)
			top = min(top, pos.Y)
			bottom = max(bottom, pos.Y)
		}
	}
	regions := []visual.FanRect{
		{X: left, Y: top, Width: right - left, Height: bottom - top + 1},
		{X: area.X + 1, Y: middle, Width: 7, Height: 1},
	}
	for y := max(area.Y, top-3); y < min(area.Y+area.Height, bottom+4); y++ {
		for x := area.X + 1; x < min(area.X+area.Width, right+6); {
			r, comb, style, cells := v.screen.GetContent(x, y)
			amount := visual.Feather(x, y, regions, feather) * strength
			if amount > 0 && r != 0 && x+max(cells, 1) <= area.X+area.Width {
				fg, bg, _ := style.Decompose()
				style = style.Foreground(dimColor(fg, ink.Text, 1-amount)).Background(dimColor(bg, ink.Background, 1-amount))
				v.screen.SetContent(x, y, r, comb, style)
			}
			x += max(cells, 1)
		}
	}
}

func (v *workspaceView) clearFanCell(area visual.FanRect, x, y int) {
	if x <= area.X || x >= area.X+area.Width {
		return
	}
	if _, _, _, cells := v.screen.GetContent(x-1, y); cells == 2 && x > area.X+1 {
		v.screen.SetContent(x-1, y, ' ', nil, baseStyle)
	}
	if _, _, _, cells := v.screen.GetContent(x, y); cells == 2 && x+1 < area.X+area.Width {
		v.screen.SetContent(x+1, y, ' ', nil, baseStyle)
	}
	v.screen.SetContent(x, y, ' ', nil, baseStyle)
}
