package i18n

// Texts for the memory view.
var (
	MemKindUser    = Text{En: "User", Zh: "用户"}
	MemKindProject = Text{En: "Project", Zh: "项目"}
	MemKindAuto    = Text{En: "Auto index", Zh: "自动记忆索引"}
	MemKindTopic   = Text{En: "Auto topic", Zh: "自动记忆主题"}

	MemUserGlobal = Text{En: "(all projects)", Zh: "(全部项目)"}
	MemEmptyHelp  = Text{
		En: "No memory files found. Claude writes them to ~/.claude/projects/<project>/memory, or you can create CLAUDE.md.",
		Zh: "没有找到记忆文件。Claude 会把记忆写到 ~/.claude/projects/<项目>/memory，你也可以自己创建 CLAUDE.md。",
	}
	MemEntries   = Text{En: "%d entries", Zh: "%d 条"}
	MemEmptyFile = Text{En: "(empty file)", Zh: "(空文件)"}
	MemReadFail  = Text{En: "Cannot read file: %v", Zh: "无法读取文件：%v"}
	MemOpened    = Text{En: "Opened: %s", Zh: "已打开：%s"}
	MemOpenFail  = Text{En: "Open failed: %v", Zh: "打开失败：%v"}

	MemHintList = Text{
		En: "↑↓/jk: Select  Enter: View  o: Open in editor  R: Refresh  ESC: Back  Q: Quit",
		Zh: "↑↓/jk: 选择  Enter: 查看  o: 用编辑器打开  R: 刷新  ESC: 返回  Q: 退出",
	}
	MemHintDetail = Text{
		En: "↑↓/jk: Scroll  PgUp/PgDn: Page  g/G: Top/Bottom  o: Open in editor  ESC: Back",
		Zh: "↑↓/jk: 滚动  PgUp/PgDn: 翻页  g/G: 顶部/底部  o: 用编辑器打开  ESC: 返回",
	}
)
