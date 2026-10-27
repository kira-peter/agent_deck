package dirpicker

import (
	"os"
	"strings"
)

// Windows 系统目录黑名单 - 这些目录通常有权限问题或不应被访问
var skipDirs = map[string]bool{
	"$Recycle.Bin":              true,
	"System Volume Information": true,
	"Recovery":                  true,
	"ProgramData":               true,
	"RECYCLER":                  true,
	"$RECYCLE.BIN":              true,
}

// getDrives 检测并返回所有可用的 Windows 驱动器
func getDrives() []string {
	var drives []string

	// 检测常见的驱动器字母（C-Z, 然后 A-B）
	for _, letter := range "CDEFGHIJKLMNOPQRSTUVWXYZAB" {
		drive := string(letter) + ":\\"
		// 尝试访问驱动器根目录以确认其存在
		if _, err := os.Stat(drive); err == nil {
			drives = append(drives, drive)
		}
	}

	// 如果没有找到任何驱动器，返回默认的 C:\
	if len(drives) == 0 {
		drives = []string{"C:\\"}
	}

	return drives
}

// normalizePath 规范化 Windows 路径
func normalizePath(path string) string {
	// 将正斜杠替换为反斜杠
	path = strings.ReplaceAll(path, "/", "\\")

	// 确保驱动器路径有尾部反斜杠（如 C:\ 而不是 C:）
	if len(path) == 2 && path[1] == ':' {
		path += "\\"
	}

	// 移除多余的尾部反斜杠（除了驱动器根目录）
	if len(path) > 3 && strings.HasSuffix(path, "\\") {
		path = strings.TrimSuffix(path, "\\")
	}

	return path
}

// shouldSkipDir 判断是否应该跳过某个目录
func shouldSkipDir(name string) bool {
	return skipDirs[name]
}

// createDrivesColumn 创建初始的驱动器选择列
func createDrivesColumn() Column {
	drives := getDrives()

	entries := make([]DirEntry, len(drives))
	for i, drive := range drives {
		entries[i] = DirEntry{
			Name:     drive,
			IsDir:    true,
			FullPath: drive,
		}
	}

	return Column{
		Path:    "Drives",
		Entries: entries,
	}
}
