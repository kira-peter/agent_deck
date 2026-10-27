# Claude Manager

一个用 Go 编写的终端 TUI 工具，用来浏览和管理本机 Claude Code 的数据：项目、会话记录、记忆文件（MEMORY.md），并且可以直接在指定目录中启动 Claude Code。

```
┌──────────────────────────────────────────────┐
│              Claude Manager                  │
│              2026-10-27 17:20                │
├──────────────────────────────────────────────┤
│           ⠙ Overview                         │
│             Projects                         │
│             Sessions                         │
│             Memory                           │
│             Guide                            │
│             Settings                         │
├──────────────────────────────────────────────┤
│   View Claude Code session, memory and ...   │
├──────────────────────────────────────────────┤
│  ↑↓/jk: Navigate  Enter: Select  T: Toggle … │
└──────────────────────────────────────────────┘
```

## 功能

- **总览（Overview）** — 项目数、会话数、记忆文件数、总占用空间，以及当前活跃的 Claude Code 会话和最近会话
- **项目（Projects）** — 按最后活动时间列出所有项目，支持打开、删除、刷新
- **开启新项目** — 内置 Miller Columns 风格目录选择器，选好目录后直接在新终端窗口运行 `claude <路径>`
- **会话（Sessions）** — 浏览所有会话，支持关键字搜索、按时间过滤、查看对话详情、删除或归档
- **记忆（Memory）** — 汇总所有项目的 `MEMORY.md`，显示条目数、大小和修改时间
- **说明（Guide）** — 内置 Claude 记忆系统文档（记忆类型、CLAUDE.md 位置、目录结构、路径编码）
- **设置（Settings）** — 查看配置，切换中英文
- **九种菜单动画** — 每次上下切换选项都会随机换一种 spinner 动画（保证与上一次不同）
- **中英双语** — 按 `T` 随时切换，语言会持久化到配置文件

## 环境要求

- Go 1.25.5 或更高（`go.mod` 中声明）
- 已安装并至少运行过一次 Claude Code（存在 `~/.claude` 或 `%USERPROFILE%\.claude`）
- 支持 UTF-8 与 Unicode 的终端；Windows 推荐 Windows Terminal
- 目录选择器要求终端宽度至少 60 列，其余视图建议 80x24 以上

## 构建与运行

```bash
# 使用 Makefile（推荐）
make build          # 产物：build/claude-manager(.exe)
make run            # 构建并运行
make test           # 运行全部测试
make clean          # 清理构建产物

# 或者直接使用 go
go build -o build/claude-manager.exe ./cmd/claude-manager
./build/claude-manager.exe
```

Windows 下也可以直接双击 `run.bat`。

构建时可以注入构建时间，菜单标题下方会显示这个版本标签：

```bash
make build BUILD_TIME=2026-10-27T17:20
```

不传时回退为可执行文件的修改时间。

## 快捷键

### 主菜单

| 按键 | 作用 |
|---|---|
| `↑` / `↓` / `k` / `j` | 移动选择（同时切换 spinner 动画） |
| `Enter` | 进入选中的视图 |
| `T` | 切换中英文 |
| `Q` / `ESC` / `Ctrl+C` | 退出程序 |

### 各视图

| 视图 | 按键 |
|---|---|
| 总览 | `R` 刷新，`ESC` 返回，`Q` 退出 |
| 项目 | `↑↓/jk` 移动，`Enter` 打开，`D` 删除，`R` 刷新，`ESC` 返回，`Q` 退出 |
| 会话列表 | `↑↓/jk` 移动，`Enter` 查看详情，`D` 删除，`A` 归档，`/` 搜索，`F` 切换过滤（全部/今天/本周/本月），`R` 刷新，`ESC` 返回 |
| 会话详情 | `↑↓/jk` 滚动，`PgUp`/`PgDn` 翻页，`ESC` 返回列表 |
| 记忆 | `↑↓/jk` 移动，`R` 刷新，`ESC` 返回 |
| 说明 | `↑↓/jk` 滚动，`ESC` 返回 |
| 设置 | `T` 切换语言，`ESC` 返回，`Q` 退出 |

### 目录选择器

| 按键 | 作用 |
|---|---|
| `←` / `h` | 返回上一级 |
| `→` / `l` | 进入选中的目录 |
| `↑` / `↓` / `k` / `j` | 在当前列中移动 |
| `Enter` | 确认当前目录 |
| `ESC` / `Q` | 取消并返回项目列表 |

确认对话框只有两个选项（`是，打开` / `否，重新选择`），用 `↑↓/jk` 选择后按 `Enter` 确认；`ESC` 直接放弃整个流程返回项目列表。这里刻意不提供 `Y`/`N` 快捷键，避免误按。详见 [docs/confirmation-design.md](docs/confirmation-design.md)。

## 配置

配置文件位于 `~/.claude-manager/config.json`（Windows 为 `C:\Users\<用户名>\.claude-manager\config.json`），首次运行使用默认值：

```json
{
  "version": "1.0",
  "language": "en",
  "claudeDir": "C:\\Users\\<用户名>\\.claude",
  "display": {
    "dateFormat": "relative",
    "sizeUnit": "auto",
    "itemsPerPage": 20
  }
}
```

| 字段 | 说明 |
|---|---|
| `language` | `en` 或 `zh`，可通过设置页或按 `T` 修改并自动保存 |
| `claudeDir` | `.claude` 目录位置，目前需手动编辑配置 |
| `display.*` | 预留字段（日期格式、大小单位、每页条数），当前版本视图未读取 |

## 数据来源

程序只读取、不解析 Claude Code 的私有协议；所有数据来自 `.claude` 目录：

```
~/.claude/
├── projects/
│   ├── D--jarvis/                 # 项目目录名 = 路径编码
│   │   ├── <session-id>.jsonl     # 会话对话记录
│   │   ├── <session-id>/          # 会话附加文件
│   │   │   ├── subagents/
│   │   │   └── tool-results/
│   │   ├── sessions-index.json    # 会话索引（优先使用）
│   │   ├── archived/              # 归档后的会话文件
│   │   └── memory/
│   │       └── MEMORY.md          # 项目记忆
│   └── ...
├── sessions/
│   └── <pid>.json                 # 活跃会话元数据
├── settings.json
└── history.jsonl
```

- **项目与会话**：优先解析 `sessions-index.json`；该文件缺失时回退为扫描目录下的 `.jsonl` 文件
- **路径解码**：目录名是路径编码的结果（`D:\jarvis` → `D--jarvis`、`C:\Users\TU` → `C--Users-TU`、`/home/user` → `home-user`），无法从索引拿到原始路径时会尝试按文件系统逐级还原
- **删除项目**：删除整个 `projects/<编码名>/` 目录，包括该项目的所有会话与记忆，不可恢复
- **归档会话**：把 `.jsonl` 移到同项目的 `archived/` 子目录
- **删除会话**：直接删除对应的 `.jsonl` 文件

## 项目结构

```
claude-manager/
├── cmd/claude-manager/main.go     # 入口：主菜单(Bubble Tea) → 视图(tcell) 循环
├── pkg/
│   ├── core/                      # 配置、.claude 目录解析、数据模型、版本标签
│   ├── i18n/                      # 中英文文案与语言管理
│   ├── ui/
│   │   ├── selector/              # 主菜单选择器
│   │   └── animation/             # 九种 spinner 动画
│   └── views/
│       ├── registry.go            # 视图注册表
│       ├── view.go                # View 接口
│       ├── overview/  projects/  sessions/  memory/  guide/  settings/
│       └── dirpicker/             # 目录选择器（模态，由 Projects 调起）
├── docs/                          # 设计文档与测试指南
├── Makefile
└── run.bat
```

### 架构要点

主菜单用 [Bubble Tea](https://github.com/charmbracelet/bubbletea) 渲染（需要动画与定时刷新），用户选定视图后退出 Bubble Tea，切换到 [tcell](https://github.com/gdamore/tcell) 全屏运行该视图；视图返回 `back` 时回到主菜单，返回 `quit` 时退出程序。视图通过 `init()` 注册到全局注册表，`main.go` 用空导入触发注册。

### 新增一个视图

1. 在 `pkg/views/` 下新建包，实现 `views.View` 接口（`Metadata` / `Init` / `Render` / `HandleEvent` / `Cleanup`）
2. 在同一包内写 `register.go`，在 `init()` 中调用 `views.Register(factory)`
3. 在 `cmd/claude-manager/main.go` 里用空导入引入该包

`HandleEvent` 返回 `fmt.Errorf("back")` 表示返回主菜单，`fmt.Errorf("quit")` 表示退出程序。

## 文档

| 文档 | 内容 |
|---|---|
| [docs/quickstart.md](docs/quickstart.md) | 快速上手：界面、快捷键、常见问题 |
| [docs/claude-code-session-memory.md](docs/claude-code-session-memory.md) | Claude Code 的会话与三层记忆机制（针对开发者） |
| [docs/dirpicker-test-guide.md](docs/dirpicker-test-guide.md) | 目录选择器与新项目流程的手工测试清单 |
| [docs/confirmation-design.md](docs/confirmation-design.md) | 确认对话框的最终设计（为什么去掉 Y/N 快捷键） |
| [docs/confirmation-improvement.md](docs/confirmation-improvement.md) | 确认对话框从"按键提示"到"选项列表"的演进过程 |
| [docs/animation-design.md](docs/animation-design.md) | 菜单 spinner 动画的设计与切换机制 |
| [docs/animation-options.md](docs/animation-options.md) | 候选动画清单与取舍（含已否决的 emoji 方案） |

## 已知限制

- "打开项目"通过 PowerShell `Start-Process` 启动新窗口，目前仅适用于 Windows
- 设置页只能切换语言，`claudeDir` 等字段需要手动编辑配置文件
- `display` 下的显示选项尚未接入视图
- 目录选择器最多同时显示 5 列，更深的路径靠横向滚动

## License

MIT

## 致谢

- 参考了 [symbol_move](https://github.com/symbolmove/symbol_move) 的 TUI 架构设计
- 终端处理：[tcell](https://github.com/gdamore/tcell)
- 主菜单与动画：[Bubble Tea](https://github.com/charmbracelet/bubbletea)

---

## English summary

Claude Manager is a TUI for browsing Claude Code data — projects, sessions, memory files — with a built-in directory picker that launches `claude` in any folder. Built with Bubble Tea (main menu) and tcell (views). Requires Go 1.25.5+, an existing `~/.claude`, and a Unicode-capable terminal.

```bash
make build && ./build/claude-manager.exe
```

Keys: `↑↓/jk` navigate, `Enter` select, `T` toggle language, `ESC` back, `Q` quit.
