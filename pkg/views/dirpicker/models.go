package dirpicker

import (
	"github.com/gdamore/tcell/v2"
)

// Column 表示一个层级的目录
type Column struct {
	Path      string     // 该列代表的绝对路径
	Entries   []DirEntry // 目录和文件条目
	Selected  int        // 当前选中的索引
	ScrollOff int        // 滚动偏移量
}

// DirEntry 表示一个文件或目录
type DirEntry struct {
	Name     string // 文件/目录名
	IsDir    bool   // 是否为目录
	FullPath string // 完整路径
	Error    error  // 权限错误等
	Hidden   bool   // 以 . 开头或系统文件
}

// DirPickerView 目录选择器主视图结构
type DirPickerView struct {
	screen            tcell.Screen
	width             int
	height            int
	columns           []Column // 列堆栈（面包屑路径）
	activeColumn      int      // 当前焦点列（0-based）
	selectedPath      string   // 最终选择的路径
	confirmed         bool     // 用户确认选择
	cancelled         bool     // 用户取消
	confirmSelectedIdx int     // 确认对话框选中的选项索引（0=是，1=否）
}
