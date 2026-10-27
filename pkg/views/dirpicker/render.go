package dirpicker

import (
	"fmt"
	"path/filepath"

	"github.com/gdamore/tcell/v2"
	"github.com/yourusername/claude-manager/pkg/i18n"
)

// renderTitle 渲染标题栏
func (v *DirPickerView) renderTitle() {
	title := i18n.DirPickerTitle.Get()
	cancelHint := "ESC: " + i18n.ConfirmNo.Get()[3:] // 取 "取消" 部分

	y := 1

	// 居中标题
	titleX := (v.width - len(title)) / 2
	if titleX < 0 {
		titleX = 0
	}
	v.drawText(titleX, y, title, tcell.StyleDefault.
		Foreground(tcell.ColorLightGreen).Bold(true))

	// 右对齐取消提示
	cancelX := v.width - len(cancelHint) - 2
	if cancelX > titleX+len(title)+3 {
		v.drawText(cancelX, y, cancelHint, tcell.StyleDefault.
			Foreground(tcell.ColorGray))
	}

	// 分隔线
	v.drawHorizontalLine(y + 2)
}

// renderColumns 渲染多列视图
func (v *DirPickerView) renderColumns() {
	columnStartY := 5
	columnHeight := v.height - 12 // 为标题、路径和提示留出空间

	if len(v.columns) == 0 {
		return
	}

	// 计算列的数量和宽度
	numVisibleCols := len(v.columns)
	if numVisibleCols > 5 {
		numVisibleCols = 5 // 最多显示 5 列
	}

	colWidth := (v.width - 6) / numVisibleCols

	// 确定显示哪些列（如果超过 5 列，滚动显示最右侧 5 列）
	startCol := 0
	if len(v.columns) > 5 {
		startCol = len(v.columns) - 5
	}

	// 渲染每个可见列
	for i := 0; i < numVisibleCols && startCol+i < len(v.columns); i++ {
		colIdx := startCol + i
		col := v.columns[colIdx]
		xPos := 3 + (i * colWidth)
		isActive := (colIdx == v.activeColumn)

		v.renderColumn(col, xPos, columnStartY, colWidth, columnHeight, isActive)
	}
}

// renderColumn 渲染单个列
func (v *DirPickerView) renderColumn(col Column, x, y, width, height int, isActive bool) {
	// 列边框样式
	borderStyle := tcell.StyleDefault.Foreground(tcell.ColorGray)
	if isActive {
		borderStyle = tcell.StyleDefault.Foreground(tcell.ColorLightCyan)
	}

	// 绘制列框
	v.drawBox(x, y, width, height, borderStyle)

	// 列标题（截断路径）
	title := col.Path
	if title == "Drives" {
		title = i18n.DirPickerDrives.Get()
	} else {
		title = filepath.Base(title)
	}

	if len(title) > width-4 {
		title = title[:width-7] + "..."
	}
	v.drawText(x+2, y, title, borderStyle.Bold(true))

	// 标题下方的分隔线
	v.drawHorizontalLineRange(y+1, x+1, x+width-1)

	// 渲染条目
	maxVisible := height - 3 // 扣除标题和边框

	// 调整滚动偏移量
	if col.Selected < col.ScrollOff {
		col.ScrollOff = col.Selected
	}
	if col.Selected >= col.ScrollOff+maxVisible {
		col.ScrollOff = col.Selected - maxVisible + 1
	}

	// 绘制条目
	for i := col.ScrollOff; i < len(col.Entries) && i < col.ScrollOff+maxVisible; i++ {
		entry := col.Entries[i]
		entryY := y + 2 + (i - col.ScrollOff)

		// 确定样式
		style := tcell.StyleDefault.Foreground(tcell.ColorWhite)
		if entry.Hidden {
			style = style.Foreground(tcell.ColorGray)
		}
		if entry.Error != nil {
			style = style.Foreground(tcell.ColorRed)
		}

		// 高亮选中项
		if i == col.Selected && isActive {
			style = style.Background(tcell.ColorLightGreen).Foreground(tcell.ColorBlack)
		}

		// 格式化条目名称
		displayName := entry.Name
		if entry.IsDir && entry.Error == nil {
			displayName = "> " + displayName + "/"
		} else {
			displayName = "  " + displayName
		}

		// 截断过长的名称
		maxNameLen := width - 4
		if len(displayName) > maxNameLen {
			displayName = displayName[:maxNameLen-3] + "..."
		}

		v.drawText(x+2, entryY, displayName, style)
	}
}

// renderCurrentPath 渲染当前路径显示
func (v *DirPickerView) renderCurrentPath() {
	y := v.height - 6

	currentPath := v.getCurrentPath()
	if currentPath == "" && len(v.columns) > 0 {
		// 如果还在驱动器列，显示提示
		currentPath = i18n.DirPickerDrives.Get()
	}

	label := i18n.DirPickerCurrent.Get() + ": "
	text := label + currentPath

	// 截断过长的路径
	maxLen := v.width - 6
	if len(text) > maxLen {
		// 保留开头和结尾
		pathOnly := currentPath
		if len(pathOnly) > maxLen-len(label)-6 {
			start := pathOnly[:20]
			end := pathOnly[len(pathOnly)-maxLen+len(label)+26:]
			pathOnly = start + "..." + end
		}
		text = label + pathOnly
	}

	v.drawCenteredText(y, text, tcell.StyleDefault.Foreground(tcell.ColorLightBlue))
}

// renderHints 渲染操作提示
func (v *DirPickerView) renderHints() {
	y := v.height - 2
	v.drawHorizontalLine(y - 1)

	hints := i18n.DirPickerHints.Get()
	v.drawCenteredText(y, hints, tcell.StyleDefault.Foreground(tcell.ColorGray))
}

// 辅助绘图函数

func (v *DirPickerView) drawText(x, y int, text string, style tcell.Style) {
	for i, r := range text {
		v.screen.SetContent(x+i, y, r, nil, style)
	}
}

func (v *DirPickerView) drawCenteredText(y int, text string, style tcell.Style) {
	x := (v.width - len(text)) / 2
	if x < 0 {
		x = 0
	}
	v.drawText(x, y, text, style)
}

func (v *DirPickerView) drawHorizontalLine(y int) {
	for x := 2; x < v.width-2; x++ {
		v.screen.SetContent(x, y, '─', nil, tcell.StyleDefault.
			Foreground(tcell.ColorGray))
	}
}

func (v *DirPickerView) drawHorizontalLineRange(y, x1, x2 int) {
	for x := x1; x < x2; x++ {
		v.screen.SetContent(x, y, '─', nil, tcell.StyleDefault.
			Foreground(tcell.ColorGray))
	}
}

func (v *DirPickerView) drawBox(x, y, width, height int, style tcell.Style) {
	// 顶部和底部
	for i := 1; i < width-1; i++ {
		v.screen.SetContent(x+i, y, '─', nil, style)
		v.screen.SetContent(x+i, y+height-1, '─', nil, style)
	}

	// 左侧和右侧
	for i := 1; i < height-1; i++ {
		v.screen.SetContent(x, y+i, '│', nil, style)
		v.screen.SetContent(x+width-1, y+i, '│', nil, style)
	}

	// 四角
	v.screen.SetContent(x, y, '┌', nil, style)
	v.screen.SetContent(x+width-1, y, '┐', nil, style)
	v.screen.SetContent(x, y+height-1, '└', nil, style)
	v.screen.SetContent(x+width-1, y+height-1, '┘', nil, style)
}

// renderConfirmation 渲染确认对话框（选项模式）
func (v *DirPickerView) renderConfirmation() {
	v.screen.Clear()

	y := v.height/2 - 5

	// 标题
	v.drawCenteredText(y, i18n.DirPickerConfirm.Get(),
		tcell.StyleDefault.Foreground(tcell.ColorYellow).Bold(true))

	y += 2

	// 显示选择的路径
	path := v.selectedPath
	maxPathLen := v.width - 10
	if len(path) > maxPathLen {
		// 截断中间部分
		start := path[:20]
		end := path[len(path)-maxPathLen+23:]
		path = start + "..." + end
	}
	v.drawCenteredText(y, path, tcell.StyleDefault.Foreground(tcell.ColorWhite))

	y += 3

	// 绘制选项列表（只有两个选项）
	options := []string{
		i18n.DirPickerConfirmYes.Get(), // "是，打开"
		i18n.DirPickerConfirmNo.Get(),  // "否，重新选择"
	}

	// 计算选项最大宽度以居中显示
	maxWidth := 0
	for _, opt := range options {
		displayText := fmt.Sprintf(">  %s", opt)
		if len(displayText) > maxWidth {
			maxWidth = len(displayText)
		}
	}

	startX := (v.width - maxWidth) / 2

	// 渲染每个选项
	for i, opt := range options {
		optY := y + i*2

		style := tcell.StyleDefault.Foreground(tcell.ColorWhite)
		prefix := "   "

		// 高亮选中的选项
		if i == v.confirmSelectedIdx {
			style = style.Background(tcell.ColorLightGreen).Foreground(tcell.ColorBlack)
			prefix = ">  "
		}

		displayText := prefix + opt
		v.drawText(startX, optY, displayText, style)
	}

	y += len(options)*2 + 1

	// 操作提示
	hints := "↑↓/jk: 移动  Enter: 确认  ESC: 返回"
	v.drawCenteredText(y, hints, tcell.StyleDefault.Foreground(tcell.ColorGray))

	v.screen.Show()
}
