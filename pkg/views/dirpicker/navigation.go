package dirpicker

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kira-peter/agentdeck/pkg/i18n"
)

// navigateLeft 返回上级目录
func (v *DirPickerView) navigateLeft() {
	if v.activeColumn > 0 {
		v.activeColumn--
		// 移除当前列右侧的所有列
		v.columns = v.columns[:v.activeColumn+1]
	}
}

// navigateRight 进入选中的目录
func (v *DirPickerView) navigateRight() error {
	if v.activeColumn >= len(v.columns) {
		return nil
	}

	activeCol := &v.columns[v.activeColumn]
	if activeCol.Selected >= len(activeCol.Entries) {
		return nil
	}

	selectedEntry := activeCol.Entries[activeCol.Selected]

	// 只能进入目录，不能进入文件
	if !selectedEntry.IsDir {
		return nil
	}

	// 如果条目有错误（如权限不足），不能进入
	if selectedEntry.Error != nil {
		return nil
	}

	// 加载新列
	newCol, err := v.loadColumn(selectedEntry.FullPath)
	if err != nil {
		// 即使加载失败，也创建一个显示错误的列
		// err 已经在 loadColumn 中处理了
	}

	// 移除右侧的所有列（用户可能回退后重新选择）
	v.columns = v.columns[:v.activeColumn+1]
	// 添加新列
	v.columns = append(v.columns, newCol)
	v.activeColumn++

	return nil
}

// moveUp 在当前列中向上移动
func (v *DirPickerView) moveUp() {
	if v.activeColumn >= len(v.columns) {
		return
	}

	activeCol := &v.columns[v.activeColumn]
	if activeCol.Selected > 0 {
		activeCol.Selected--
	}
}

// moveDown 在当前列中向下移动
func (v *DirPickerView) moveDown() {
	if v.activeColumn >= len(v.columns) {
		return
	}

	activeCol := &v.columns[v.activeColumn]
	if activeCol.Selected < len(activeCol.Entries)-1 {
		activeCol.Selected++
	}
}

// loadColumn 加载指定路径的目录内容
func (v *DirPickerView) loadColumn(path string) (Column, error) {
	col := Column{
		Path:     path,
		Selected: 0,
	}

	// 规范化路径
	path = normalizePath(path)

	// 尝试读取目录
	entries, err := os.ReadDir(path)
	if err != nil {
		// 权限不足或其他错误
		col.Entries = []DirEntry{{
			Name:  i18n.DirPickerPermissionDenied.Get(),
			Error: err,
			IsDir: false,
		}}
		return col, nil
	}

	// 分离目录和文件
	var dirs []DirEntry
	var files []DirEntry

	for _, entry := range entries {
		// 跳过系统目录
		if shouldSkipDir(entry.Name()) {
			continue
		}

		de := DirEntry{
			Name:     entry.Name(),
			IsDir:    entry.IsDir(),
			FullPath: filepath.Join(path, entry.Name()),
			Hidden:   strings.HasPrefix(entry.Name(), "."),
		}

		if de.IsDir {
			dirs = append(dirs, de)
		} else {
			files = append(files, de)
		}
	}

	// 如果目录为空
	if len(dirs) == 0 && len(files) == 0 {
		col.Entries = []DirEntry{{
			Name:  i18n.DirPickerEmpty.Get(),
			IsDir: false,
		}}
		return col, nil
	}

	// 排序：目录优先（字母序），然后是文件
	sort.Slice(dirs, func(i, j int) bool {
		return strings.ToLower(dirs[i].Name) < strings.ToLower(dirs[j].Name)
	})
	sort.Slice(files, func(i, j int) bool {
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})

	// 合并目录和文件
	col.Entries = append(dirs, files...)

	return col, nil
}

// getCurrentPath 获取当前选择的完整路径
func (v *DirPickerView) getCurrentPath() string {
	if v.activeColumn >= len(v.columns) {
		return ""
	}

	activeCol := &v.columns[v.activeColumn]
	if activeCol.Selected >= len(activeCol.Entries) {
		return ""
	}

	selectedEntry := activeCol.Entries[activeCol.Selected]
	if selectedEntry.IsDir && selectedEntry.Error == nil {
		return selectedEntry.FullPath
	}

	// 如果选中的不是有效目录，返回当前列的路径
	if activeCol.Path != "Drives" {
		return activeCol.Path
	}

	return ""
}
