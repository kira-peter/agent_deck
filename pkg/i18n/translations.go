package i18n

// Text represents a bilingual text pair
type Text struct {
	En string // English text
	Zh string // Chinese text
}

// Get returns the text in the current language
func (t Text) Get() string {
	if GetManager().IsEnglish() {
		return t.En
	}
	return t.Zh
}

// Common UI strings
var (
	// Main title
	AppTitle = Text{
		En: "AgentDeck - Coding Agent Session Manager",
		Zh: "AgentDeck - 编程助手会话管理工具",
	}
	AppTitleCodex = Text{
		En: "AgentDeck · Codex",
		Zh: "AgentDeck · Codex",
	}

	// View names
	ViewOverview = Text{
		En: "Overview",
		Zh: "总览",
	}
	ViewProjects = Text{
		En: "Projects",
		Zh: "项目",
	}
	ViewSessions = Text{
		En: "Sessions",
		Zh: "会话",
	}
	ViewMemory = Text{
		En: "Memory",
		Zh: "记忆",
	}
	ViewGuide = Text{
		En: "Guide",
		Zh: "指南",
	}
	ViewSettings = Text{
		En: "Settings",
		Zh: "设置",
	}

	// View descriptions
	DescOverview = Text{
		En: "View Claude Code session, memory and project status",
		Zh: "查看 Claude Code 的会话、记忆和项目状态",
	}
	DescProjects = Text{
		En: "Browse and manage all projects",
		Zh: "浏览和管理所有项目",
	}
	DescSessions = Text{
		En: "Browse all conversation sessions",
		Zh: "浏览所有会话记录",
	}
	DescMemory = Text{
		En: "View and manage memory files",
		Zh: "查看和管理记忆文件",
	}
	DescGuide = Text{
		En: "Learn about Claude's memory system",
		Zh: "了解 Claude 的记忆系统",
	}
	DescSettings = Text{
		En: "Configure application settings",
		Zh: "配置应用设置",
	}

	// Shown when the terminal cannot hold the main menu
	WindowTooSmall     = Text{En: "Window too small", Zh: "窗口太小，放不下菜单"}
	WindowSizeNow      = Text{En: "Current: %d × %d", Zh: "当前：%d × %d"}
	WindowSizeNeed     = Text{En: "Needed:  %d × %d", Zh: "至少：%d × %d"}
	WindowTooSmallHint = Text{En: "Enlarge the terminal or press Q to quit", Zh: "请放大终端窗口，或按 Q 退出"}

	// Navigation hints
	HintNavigation = Text{
		En: "↑↓/jk: Navigate  Enter: Select  F1: Claude/Codex  T: Language  Q: Quit",
		Zh: "↑↓/jk: 导航  Enter: 选择  F1: 切换 Claude/Codex  T: 切换语言  Q: 退出",
	}
	HintListActions = Text{
		En: "Enter: Details  d: Delete  D: Delete now  A: Archive  /: Search  F: Filter  ESC: Back",
		Zh: "Enter: 查看详情  d: 删除  D: 直接删除  A: 归档  /: 搜索  F: 过滤  ESC: 返回",
	}
	HintProjectActions = Text{
		En: "↑↓/jk: Navigate  Enter: Open  d: Delete  D: Delete now  R: Refresh  ESC: Back  Q: Quit",
		Zh: "↑↓/jk: 导航  Enter: 打开  d: 删除  D: 直接删除  R: 刷新  ESC: 返回  Q: 退出",
	}

	// Overview labels
	LabelTotalProjects = Text{
		En: "Total Projects",
		Zh: "项目总数",
	}
	LabelTotalSessions = Text{
		En: "Total Sessions",
		Zh: "会话总数",
	}
	LabelTotalMemories = Text{
		En: "Memory Files",
		Zh: "记忆文件数",
	}
	LabelTotalSize = Text{
		En: "Total Size",
		Zh: "总占用空间",
	}
	LabelRecentSessions = Text{
		En: "Recent Sessions",
		Zh: "最近会话",
	}
	LabelActiveSessions = Text{
		En: "Active Sessions",
		Zh: "活跃会话",
	}

	// Project list headers
	HeaderPath = Text{
		En: "Path",
		Zh: "路径",
	}
	HeaderSessions = Text{
		En: "Sessions",
		Zh: "会话数",
	}
	HeaderSize = Text{
		En: "Size",
		Zh: "大小",
	}
	HeaderLastActivity = Text{
		En: "Last Activity",
		Zh: "最后活动",
	}

	// Project usage panel
	UsageTitle = Text{
		En: "Usage",
		Zh: "用量",
	}
	UsageAllProjects = Text{
		En: "All projects",
		Zh: "全部项目",
	}
	UsageInput = Text{
		En: "input",
		Zh: "输入",
	}
	UsageOutput = Text{
		En: "output",
		Zh: "输出",
	}
	UsageCacheRead = Text{
		En: "cache read",
		Zh: "缓存读",
	}
	UsageCacheWrite = Text{
		En: "cache write",
		Zh: "缓存写",
	}
	UsageLinesAdded = Text{
		En: "added",
		Zh: "新增行",
	}
	UsageLinesRemoved = Text{
		En: "removed",
		Zh: "删除行",
	}
	UsageWorkTime = Text{
		En: "work time",
		Zh: "工作耗时",
	}

	// Session list headers
	HeaderID = Text{
		En: "ID",
		Zh: "ID",
	}
	HeaderFirstPrompt = Text{
		En: "First Prompt",
		Zh: "首次提示",
	}
	HeaderMessages = Text{
		En: "Messages",
		Zh: "消息数",
	}
	HeaderTime = Text{
		En: "Time",
		Zh: "时间",
	}

	// Confirmation dialogs
	ConfirmDelete = Text{
		En: "Confirm Delete",
		Zh: "确认删除",
	}
	ConfirmDeleteSession = Text{
		En: "Delete this session?",
		Zh: "是否删除会话?",
	}
	ConfirmDeleteProject = Text{
		En: "Delete this project?",
		Zh: "是否删除项目?",
	}
	WarningIrreversible = Text{
		En: "This operation is irreversible!",
		Zh: "此操作不可恢复!",
	}
	ConfirmYes = Text{
		En: "Y: Yes",
		Zh: "Y: 确认",
	}
	ConfirmNo = Text{
		En: "N: No",
		Zh: "N: 取消",
	}

	// Delete project dialog
	DeleteDlgTitle = Text{
		En: "Delete Project",
		Zh: "删除项目",
	}
	DeleteDlgWarning = Text{
		En: "All sessions and memory will be permanently deleted.",
		Zh: "将永久删除该项目的全部会话与记忆，无法恢复",
	}
	DeleteDlgSessionWarning = Text{
		En: "This session will be permanently deleted.",
		Zh: "将永久删除该会话，无法恢复",
	}
	DeleteDlgCancel = Text{
		En: "Cancel",
		Zh: "取消",
	}
	DeleteDlgConfirm = Text{
		En: "Delete permanently",
		Zh: "永久删除",
	}
	DeleteDlgHint = Text{
		En: "↑↓/jk: Select   Enter: Confirm   ESC: Cancel",
		Zh: "↑↓/jk: 选择   Enter: 确认   ESC: 取消",
	}
	DeleteDlgFailed = Text{
		En: "Delete failed",
		Zh: "删除失败",
	}
	DeleteDlgAnyKey = Text{
		En: "Press any key to continue",
		Zh: "按任意键继续",
	}
	DeleteDlgAutoClose = Text{
		En: "Closing in %ds · press any key to close now",
		Zh: "%d 秒后自动关闭 · 按任意键立即关闭",
	}

	// Status messages
	StatusLoading = Text{
		En: "Loading...",
		Zh: "加载中...",
	}
	StatusNoData = Text{
		En: "No data found",
		Zh: "未找到数据",
	}
	StatusError = Text{
		En: "Error",
		Zh: "错误",
	}

	// Settings
	SettingsClaudeDir = Text{
		En: "Claude Directory",
		Zh: "Claude 目录",
	}
	SettingsLanguage = Text{
		En: "Language",
		Zh: "语言",
	}
	SettingsDisplay = Text{
		En: "Display Options",
		Zh: "显示选项",
	}
	SettingsDateFormat = Text{
		En: "Date Format",
		Zh: "日期格式",
	}
	SettingsSizeUnit = Text{
		En: "Size Unit",
		Zh: "大小单位",
	}
	SettingsItemsPerPage = Text{
		En: "Items Per Page",
		Zh: "每页显示数",
	}

	// Time formats
	TimeJustNow = Text{
		En: "just now",
		Zh: "刚刚",
	}
	TimeMinuteAgo = Text{
		En: "1 minute ago",
		Zh: "1分钟前",
	}
	TimeMinutesAgo = Text{
		En: "%d minutes ago",
		Zh: "%d分钟前",
	}
	TimeHourAgo = Text{
		En: "1 hour ago",
		Zh: "1小时前",
	}
	TimeHoursAgo = Text{
		En: "%d hours ago",
		Zh: "%d小时前",
	}
	TimeDayAgo = Text{
		En: "1 day ago",
		Zh: "1天前",
	}
	TimeDaysAgo = Text{
		En: "%d days ago",
		Zh: "%d天前",
	}

	// Directory Picker
	DirPickerTitle = Text{
		En: "Select Project Directory",
		Zh: "选择项目目录",
	}
	DirPickerCurrent = Text{
		En: "Current",
		Zh: "当前",
	}
	DirPickerConfirm = Text{
		En: "Open Claude Code in this directory?",
		Zh: "在此目录中打开 Claude Code?",
	}
	DirPickerHints = Text{
		En: "←→: Navigate  ↑↓: Move  Enter: Select  ESC: Cancel",
		Zh: "←→: 切换层级  ↑↓: 移动  Enter: 选择  ESC: 取消",
	}
	DirPickerPermissionDenied = Text{
		En: "<Permission Denied>",
		Zh: "<权限不足>",
	}
	DirPickerEmpty = Text{
		En: "<Empty Directory>",
		Zh: "<空目录>",
	}
	NewProjectOption = Text{
		En: "[+] Open New Project",
		Zh: "[+] 开启新项目",
	}
	DirPickerConfirmYes = Text{
		En: "Yes, Open",
		Zh: "是，打开",
	}
	DirPickerConfirmNo = Text{
		En: "No, Choose Another",
		Zh: "否，重新选择",
	}
	DirPickerDrives = Text{
		En: "Drives",
		Zh: "驱动器",
	}

	// Session actions
	MsgDeleteConfirm  = Text{En: "Delete Session", Zh: "删除会话"}
	MsgArchiveConfirm = Text{En: "Archive Session", Zh: "归档会话"}
	MsgArchiveSuccess = Text{En: "Session archived", Zh: "会话已归档"}
	FilterAll         = Text{En: "All", Zh: "全部"}
	FilterToday       = Text{En: "Today", Zh: "今天"}
	FilterWeek        = Text{En: "Week", Zh: "本周"}
	FilterMonth       = Text{En: "Month", Zh: "本月"}
	HintSearch        = Text{En: "Search: ", Zh: "搜索: "}

	// Data source (Claude Code / Codex)
	MsgDeleteUnsupported = Text{En: "Deleting and archiving Codex data will come in a later version", Zh: "Codex 的删除和归档将在后续版本支持"}
	MsgProjectDirMissing = Text{En: "Project folder not found: %s", Zh: "项目目录不存在：%s"}
	MsgLaunchFailed      = Text{En: "Cannot start %s: %v", Zh: "无法启动 %s：%v"}
	MsgSourceMissing     = Text{En: "%s data folder not found: %s", Zh: "未找到 %s 数据目录：%s"}
)
