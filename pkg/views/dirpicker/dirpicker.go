package dirpicker

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/kira-peter/agentdeck/pkg/views"
)

// NewDirPickerView 创建新的目录选择器视图
func NewDirPickerView() *DirPickerView {
	return &DirPickerView{
		activeColumn:       0,
		confirmSelectedIdx: 0, // 默认选中"是"
	}
}

// Metadata 返回视图元数据
func (v *DirPickerView) Metadata() views.Metadata {
	return views.Metadata{
		ID:     "dirpicker",
		NameEn: "Directory Picker",
		NameZh: "目录选择器",
		DescEn: "Select a directory to open a new Claude project",
		DescZh: "选择目录以开启新的 Claude 项目",
		Icon:   "📁",
	}
}

// Init 初始化视图
func (v *DirPickerView) Init(screen tcell.Screen) error {
	v.screen = screen
	v.width, v.height = screen.Size()

	// 检查终端最小宽度
	if v.width < 60 {
		return fmt.Errorf("terminal too narrow (minimum 60 columns required, got %d)", v.width)
	}

	// 初始化列：从驱动器选择开始
	drivesCol := createDrivesColumn()
	v.columns = []Column{drivesCol}
	v.activeColumn = 0

	return nil
}

// Render 渲染视图
func (v *DirPickerView) Render() error {
	v.screen.Clear()

	// 标题
	v.renderTitle()

	// 多列视图
	v.renderColumns()

	// 当前路径显示
	v.renderCurrentPath()

	// 操作提示
	v.renderHints()

	v.screen.Show()
	return nil
}

// HandleEvent 处理键盘事件
func (v *DirPickerView) HandleEvent(event tcell.Event) error {
	switch ev := event.(type) {
	case *tcell.EventKey:
		switch ev.Key() {
		case tcell.KeyEscape:
			v.cancelled = true
			return fmt.Errorf("back")

		case tcell.KeyEnter:
			return v.handleEnter()

		case tcell.KeyLeft:
			v.navigateLeft()

		case tcell.KeyRight:
			v.navigateRight()

		case tcell.KeyUp:
			v.moveUp()

		case tcell.KeyDown:
			v.moveDown()

		case tcell.KeyRune:
			switch ev.Rune() {
			case 'q', 'Q':
				v.cancelled = true
				return fmt.Errorf("back")
			case 'h': // Vim 风格左
				v.navigateLeft()
			case 'l': // Vim 风格右
				v.navigateRight()
			case 'k': // Vim 风格上
				v.moveUp()
			case 'j': // Vim 风格下
				v.moveDown()
			}
		}

	case *tcell.EventResize:
		v.width, v.height = v.screen.Size()
		v.screen.Sync()
	}

	return nil
}

// handleEnter 处理回车键 - 选择目录
func (v *DirPickerView) handleEnter() error {
	// 获取当前选择的路径
	currentPath := v.getCurrentPath()
	if currentPath == "" {
		return nil // 还在驱动器列或无效路径
	}

	v.selectedPath = currentPath

	// 显示确认对话框
	return v.showConfirmation()
}

// showConfirmation 显示确认对话框（选项模式，仅上下键导航）
func (v *DirPickerView) showConfirmation() error {
	// 重置选中索引为第一个选项（是）
	v.confirmSelectedIdx = 0

	// 渲染确认界面
	v.renderConfirmation()

	// 等待用户输入
	for {
		ev := v.screen.PollEvent()
		if keyEv, ok := ev.(*tcell.EventKey); ok {
			switch keyEv.Key() {
			case tcell.KeyUp:
				// 向上移动
				if v.confirmSelectedIdx > 0 {
					v.confirmSelectedIdx--
					v.renderConfirmation()
				}

			case tcell.KeyDown:
				// 向下移动
				if v.confirmSelectedIdx < 1 { // 只有 0=是, 1=否 两个选项
					v.confirmSelectedIdx++
					v.renderConfirmation()
				}

			case tcell.KeyEnter:
				// 确认当前选择
				switch v.confirmSelectedIdx {
				case 0: // 是 - 打开项目
					v.confirmed = true
					return fmt.Errorf("confirmed")
				case 1: // 否 - 返回继续选择
					return nil
				}

			case tcell.KeyEscape:
				// ESC 键 - 返回 Projects 页面（不是继续选择目录）
				v.cancelled = true
				return fmt.Errorf("back")

			case tcell.KeyRune:
				// 只支持 Vim 风格上下键
				switch keyEv.Rune() {
				case 'k': // Vim 风格上
					if v.confirmSelectedIdx > 0 {
						v.confirmSelectedIdx--
						v.renderConfirmation()
					}
				case 'j': // Vim 风格下
					if v.confirmSelectedIdx < 1 {
						v.confirmSelectedIdx++
						v.renderConfirmation()
					}
				}
			}
		}
	}
}

// GetSelectedPath 获取用户确认的路径
func (v *DirPickerView) GetSelectedPath() string {
	return v.selectedPath
}

// IsConfirmed 返回用户是否确认了选择
func (v *DirPickerView) IsConfirmed() bool {
	return v.confirmed
}

// IsCancelled 返回用户是否取消了选择
func (v *DirPickerView) IsCancelled() bool {
	return v.cancelled
}

// Cleanup 清理资源
func (v *DirPickerView) Cleanup() error {
	// 无需清理
	return nil
}
