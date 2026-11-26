# AgentDeck

一个用 Go 编写的终端 TUI 工具，集中浏览和管理本机编程助手的数据：项目、会话记录、用量和记忆文件。目前支持 **Claude Code** 和 **Codex**，在主菜单按 `F1` 切换。可以直接在指定目录中启动对应的助手，或续接已有会话。

> 本项目原名 Claude Manager。

```
┌──────────────────────────────────────────────┐
│   AgentDeck - Coding Agent Session Manager   │
│              2026-11-26 11:05                │
├──────────────────────────────────────────────┤
│                                              │
│             ⠙ Overview                       │
│               Projects                       │
│               Sessions                       │
│               Memory                         │
│               Guide                          │
│               Settings                       │
│                                              │
├──────────────────────────────────────────────┤
│      View session, memory and project status │
├──────────────────────────────────────────────┤
│ ↑↓/jk: Navigate  Enter: Select  F1: Claude/… │
└──────────────────────────────────────────────┘
```

## 功能

- **两种数据来源** — Claude Code（`~/.claude`）与 Codex（`~/.codex`），按 `F1` 切换，选择会保存
- **总览** — 项目数、会话数、总占用空间、活跃会话与最近会话
- **项目** — 按最后活动时间列出项目，显示每个项目的 token 用量；打开时可以新建会话或续接已有会话
- **开启新项目** — 内置 Miller Columns 风格目录选择器，选好目录后在新终端窗口启动助手
- **会话** — 浏览所有会话，支持关键字搜索、按时间过滤、查看对话详情、删除或归档
- **记忆**（仅 Claude Code）— 汇总全局和各项目的 `CLAUDE.md` 与 `MEMORY.md`
- **指南**（仅 Claude Code）— 内置 Claude 记忆系统说明
- **设置** — AgentDeck 自身的显示选项；Claude Code 的 `settings.json` 常用项可直接修改（多选项用轮盘选择）；Codex 提供打开 `config.toml` 的入口
- **动画** — 菜单 spinner、进出页面的横线转场、删除时的射线清除
- **中英双语** — 按 `T` 随时切换，语言会保存到配置文件

## 环境要求

- Go 1.25.5 或更高（`go.mod` 中声明）
- 至少运行过一次 Claude Code 或 Codex（存在 `~/.claude` 或 `~/.codex`，二者有其一即可）
- 支持 UTF-8 与 Unicode 的终端；Windows 推荐 Windows Terminal
- 目录选择器要求终端宽度至少 60 列，其余视图建议 100x24 以上

## 构建与运行

```bash
# 使用 Makefile（推荐）
make build          # 产物：build/deck(.exe)
make run            # 构建并运行
make test           # 运行全部测试
make clean          # 清理构建产物

# 或者直接使用 go
go build -o build/deck.exe ./cmd/deck
./build/deck.exe
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
| `F1` | 切换 Claude Code / Codex |
| `T` | 切换中英文 |
| `Q` / `ESC` / `Ctrl+C` | 退出程序 |

### 各视图

| 视图 | 按键 |
|---|---|
| 总览 | `R` 刷新，`ESC` 返回，`Q` 退出 |
| 项目 | `↑↓/jk` 移动，`Enter` 打开，`d` 删除（确认），`D` 直接删除，`R` 刷新，`ESC` 返回，`Q` 退出 |
| 会话列表 | `↑↓/jk` 移动，`Enter` 查看详情，`d` 删除（确认），`D` 直接删除，`A` 归档，`/` 搜索，`F` 切换过滤（全部/今天/本周/本月），`ESC` 返回 |
| 会话详情 | `↑↓/jk` 滚动，`PgUp`/`PgDn` 翻页，`ESC` 返回列表 |
| 记忆 | `↑↓/jk` 移动，`R` 刷新，`ESC` 返回 |
| 指南 | `↑↓/jk` 滚动，`ESC` 返回 |
| 设置 | `↑↓/jk` 选择，`Enter` 修改或打开，`x` 清除，`R` 重新加载，`T` 切换语言，`ESC` 返回 |

Codex 模式下的删除和归档通过 `codex delete` / `codex archive` 完成，Codex 自己的索引会同步更新。

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

配置文件位于 `~/.agentdeck/config.json`（Windows 为 `C:\Users\<用户名>\.agentdeck\config.json`）。首次运行时，如果存在旧版的 `~/.claude-manager/config.json`，会自动复制过来；旧文件保留不动。

```json
{
  "version": "1.0",
  "language": "en",
  "claudeDir": "C:\\Users\\<用户名>\\.claude",
  "codexDir": "C:\\Users\\<用户名>\\.codex",
  "source": "claude",
  "display": {
    "dateFormat": "relative",
    "sizeUnit": "auto",
    "itemsPerPage": 20
  }
}
```

| 字段 | 说明 |
|---|---|
| `language` | `en` 或 `zh`，可在设置页或按 `T` 修改 |
| `claudeDir` | Claude Code 数据目录 |
| `codexDir` | Codex 数据目录，默认取 `CODEX_HOME`，没有则为 `~/.codex` |
| `source` | 当前数据来源，`claude` 或 `codex`，按 `F1` 修改 |
| `display.*` | 日期格式、大小单位、每页条数，可在设置页修改 |

## 数据来源

程序只读取各助手写在本地的文件，不依赖私有接口。

### Claude Code（`~/.claude`）

```
~/.claude/
├── projects/
│   ├── D--jarvis/                 # 项目目录名 = 路径编码
│   │   ├── <session-id>.jsonl     # 会话对话记录
│   │   ├── <session-id>/          # 会话附加文件（subagents/ 等）
│   │   ├── sessions-index.json    # 会话索引（优先使用）
│   │   ├── archived/              # 归档后的会话文件
│   │   └── memory/MEMORY.md       # 项目记忆
│   └── ...
├── sessions/<pid>.json            # 活跃会话元数据
├── settings.json
└── CLAUDE.md
```

- **项目与会话**：优先解析 `sessions-index.json`；缺失时扫描目录下的 `.jsonl` 文件
- **路径解码**：目录名是路径编码的结果（`D:\jarvis` → `D--jarvis`），拿不到原始路径时按文件系统逐级还原
- **删除项目**：删除整个 `projects/<编码名>/` 目录，包括该项目的所有会话与记忆，不可恢复
- **删除会话**：删除 `.jsonl` 文件和同名附加目录；**归档**：把 `.jsonl` 移到 `archived/`

### Codex（`~/.codex`）

```
~/.codex/
├── sessions/YYYY/MM/DD/rollout-<时间>-<id>.jsonl   # 按日期存放的会话
├── session_index.jsonl                             # 会话标题
├── config.toml
└── *.sqlite                                        # Codex 内部索引（不读取）
```

- **项目**：Codex 没有项目目录，按每个会话 `session_meta` 里的工作目录 `cwd` 分组
- **会话标题**：优先用 `session_index.jsonl` 中的标题，否则取第一条用户输入（跳过 Codex 自动注入的环境信息）
- **用量**：取会话中最后一次 `token_count` 的累计值；Codex 不记录代码增删行数，显示为 `-`
- **删除与归档**：不直接改文件，调用 `codex delete` / `codex archive`。Codex 在 sqlite 中记录了会话文件的绝对路径，直接删文件会让索引失效。归档后的文件在 `archived_sessions/`
- **删除项目**：逐个删除该工作目录下的全部 Codex 会话

## 项目结构

```
agentdeck/
├── cmd/deck/main.go               # 入口：主菜单(Bubble Tea) → 视图(tcell) 循环
├── pkg/
│   ├── core/                      # 配置、数据来源（source.go）、Claude 与 Codex 解析、数据模型
│   ├── i18n/                      # 中英文文案与语言管理
│   ├── ui/
│   │   ├── selector/              # 主菜单选择器
│   │   ├── transition/            # 主菜单 ⇄ 视图的转场动画
│   │   ├── rowfx/                 # 列表行删除动画
│   │   ├── animation/             # spinner 动画
│   │   └── textwidth/             # 按显示宽度绘制和截断文本
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

视图不直接读目录，而是通过 `core.Source` 接口取数据（`core.CurrentSource()` 按配置返回 Claude 或 Codex 的实现）。接入新的助手时，实现这个接口即可。

### 新增一个视图

1. 在 `pkg/views/` 下新建包，实现 `views.View` 接口（`Metadata` / `Init` / `Render` / `HandleEvent` / `Cleanup`）
2. 在同一包内写 `register.go`，在 `init()` 中调用 `views.Register(factory)`
3. 在 `cmd/deck/main.go` 里用空导入引入该包

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
- Codex 模式暂不支持记忆页和设置项编辑
- Codex 没有设置终端标题的参数，打开后的标签名由 Codex 决定
- 在 VS Code 内置终端中，`F1` 可能被 VS Code 拦截为命令面板
- 目录选择器最多同时显示 5 列，更深的路径靠横向滚动

## License

MIT

## 致谢

- 参考了 [symbol_move](https://github.com/symbolmove/symbol_move) 的 TUI 架构设计
- 终端处理：[tcell](https://github.com/gdamore/tcell)
- 主菜单与动画：[Bubble Tea](https://github.com/charmbracelet/bubbletea)

---

## English summary

AgentDeck (formerly Claude Manager) is a TUI for browsing the local data of coding agents — projects, sessions, token usage and memory files. It supports **Claude Code** and **Codex**; press `F1` on the main menu to switch. A built-in directory picker launches the agent in any folder, and existing sessions can be resumed. Built with Bubble Tea (main menu) and tcell (views). Requires Go 1.25.5+, an existing `~/.claude` or `~/.codex`, and a Unicode-capable terminal.

```bash
make build && ./build/deck.exe
```

Keys: `↑↓/jk` navigate, `Enter` select, `F1` Claude/Codex, `T` toggle language, `ESC` back, `Q` quit.
