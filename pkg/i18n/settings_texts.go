package i18n

// Texts for the settings view.
var (
	SetSecManager = Text{En: "Claude Manager", Zh: "Claude Manager 设置"}
	SetSecClaude  = Text{En: "Claude Code (settings.json)", Zh: "Claude Code 设置 (settings.json)"}
	SetSecEnv     = Text{En: "Environment variables (env)", Zh: "环境变量 (env)"}
	SetSecFiles   = Text{En: "Files & folders", Zh: "文件与目录"}
	SetSecCodex   = Text{En: "Codex (config.toml)", Zh: "Codex 设置 (config.toml)"}

	SetOpenCodexConfig = Text{En: "Open config.toml", Zh: "打开 config.toml"}
	SetOpenCodexDir    = Text{En: "Open Codex folder", Zh: "打开 Codex 目录"}

	SetDateFormat   = Text{En: "Date format", Zh: "日期格式"}
	SetSizeUnit     = Text{En: "Size unit", Zh: "大小单位"}
	SetItemsPerPage = Text{En: "Items per page", Zh: "每页显示数"}

	SetModel        = Text{En: "Model", Zh: "模型"}
	SetClaudeLang   = Text{En: "Response language", Zh: "回复语言"}
	SetTheme        = Text{En: "Theme", Zh: "主题"}
	SetAutoCompact  = Text{En: "Auto compact", Zh: "自动压缩上下文"}
	SetAutoMemory   = Text{En: "Auto memory", Zh: "自动记忆"}
	SetAutoContinue = Text{En: "Continue at usage limit", Zh: "额度用尽后自动继续"}
	SetDefaultMode  = Text{En: "Default permission mode", Zh: "默认权限模式"}
	SetBashOutput   = Text{En: "Bash output max chars", Zh: "Bash 输出上限(字符)"}
	SetAllowRules   = Text{En: "Allowed tool rules", Zh: "已允许的工具规则"}

	SetOpenClaudeDir = Text{En: "Open Claude folder", Zh: "打开 Claude 目录"}
	SetOpenSettings  = Text{En: "Open settings.json", Zh: "打开 settings.json"}
	SetOpenManager   = Text{En: "Open manager config", Zh: "打开管理器配置"}

	SetNotSet     = Text{En: "(not set)", Zh: "(未设置)"}
	SetEntries    = Text{En: "%d entries", Zh: "%d 条"}
	SetNoEnv      = Text{En: "(no env variables)", Zh: "(没有环境变量)"}
	SetOpened     = Text{En: "Opened: %s", Zh: "已打开：%s"}
	SetOpenFailed = Text{En: "Open failed: %v", Zh: "打开失败：%v"}
	SetSaved      = Text{En: "Saved", Zh: "已保存"}
	SetSaveFailed = Text{En: "Save failed: %v", Zh: "保存失败：%v"}
	SetCleared    = Text{En: "Cleared", Zh: "已清除"}
	SetReloaded   = Text{En: "Reloaded", Zh: "已重新加载"}
	SetInvalidInt = Text{En: "Please enter a whole number", Zh: "请输入整数"}
	SetLoadFailed = Text{En: "Cannot parse settings.json (editing disabled): %v", Zh: "settings.json 解析失败，已禁用编辑：%v"}
	SetExternal   = Text{En: "Enter: open file to edit", Zh: "Enter: 打开文件编辑"}

	SetHintBrowse = Text{
		En: "↑↓/jk: Select  Enter: Change/Open  x: Clear  R: Reload  T: Language  ESC: Back  Q: Quit",
		Zh: "↑↓/jk: 选择  Enter: 修改/打开  x: 清除  R: 重新加载  T: 切换语言  ESC: 返回  Q: 退出",
	}
	SetHintWheel = Text{
		En: "↑↓/jk: Scroll  Enter: Apply  ESC: Cancel",
		Zh: "↑↓/jk: 滚动选择  Enter: 确定  ESC: 取消",
	}
	SetHintWheelNum = Text{
		En: "↑↓/jk: Scroll  0-9: Type a number  Enter: Apply  ESC: Cancel",
		Zh: "↑↓/jk: 滚动选择  0-9: 手动输入  Enter: 确定  ESC: 取消",
	}
	SetHintEdit = Text{
		En: "Type to edit  Enter: Save  ESC: Cancel",
		Zh: "输入内容  Enter: 保存  ESC: 取消",
	}
)
