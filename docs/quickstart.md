# 快速上手

## 安装运行

1. **编译程序**

   ```bash
   make build
   ```

   或者直接使用 go：

   ```bash
   go build -o build/claude-manager.exe ./cmd/claude-manager
   ```

2. **运行程序**

   ```bash
   ./build/claude-manager.exe
   ```

   Windows 下也可以双击 `run.bat`。

## 界面说明

程序启动后是主菜单，6 个视图纵向排列，当前选中项前有一个 spinner 动画（每次切换选项都会换一种动画）：

```
┌────────────────────────────────────────────────────────┐
│                  Claude Manager                        │
│                  2026-10-27 17:20                      │
├────────────────────────────────────────────────────────┤
│                 ⠙ Overview                             │
│                   Projects                             │
│                   Sessions                             │
│                   Memory                               │
│                   Guide                                │
│                   Settings                             │
├────────────────────────────────────────────────────────┤
│   View Claude Code session, memory and project status  │
├────────────────────────────────────────────────────────┤
│   ↑↓/jk: 导航  Enter: 选择  T: 切换语言  Q: 退出       │
└────────────────────────────────────────────────────────┘
```

标题下方的日期是构建时间（`make build BUILD_TIME=...` 可注入，缺省为可执行文件的修改时间）。

## 快捷键

### 主菜单

- **↑↓** 或 **k/j** — 上下移动（同时切换 spinner 动画）
- **Enter** — 进入选中的视图
- **T** — 切换中英文
- **Q** / **ESC** / **Ctrl+C** — 退出程序

> 主菜单不支持数字键跳转：Bubble Tea 版选择器只处理上下移动、Enter、T 和退出键。

### 视图内

**通用**

- **ESC** — 返回主菜单
- **Q** — 退出程序
- **R** — 刷新当前视图（总览、项目、会话、记忆）

**总览（Overview）**：只支持 ESC / R / Q。

**项目（Projects）**

- **Enter** — 在选中的项目目录中打开 Claude Code；选中列表顶部的 `[+] 开启新项目` 时启动目录选择器
- **D** — 删除项目（需按 `Y` 确认，会删除该项目的全部会话与记忆）

**会话（Sessions）**

- **Enter** — 查看会话详情（详情页用 ↑↓/jk 滚动，PgUp/PgDn 翻页）
- **/** — 搜索（对首次提示内容做关键字匹配）
- **F** — 循环切换过滤条件：全部 → 今天 → 本周 → 本月
- **D** — 删除会话文件（需按 `Y` 确认）
- **A** — 归档会话（把 `.jsonl` 移到项目的 `archived/` 子目录，需按 `Y` 确认）

**记忆（Memory）**：↑↓/jk 移动，R 刷新。

**说明（Guide）**：↑↓/jk 滚动阅读内置文档。

**设置（Settings）**：`T` 切换语言（自动保存），其余配置项只读。

### 目录选择器（Miller Columns）

- **←** / **h** — 返回上一级
- **→** / **l** — 进入选中目录
- **↑↓** / **k/j** — 在当前列内移动
- **Enter** — 选中当前目录并弹出确认对话框
- **ESC** / **Q** — 取消，返回项目列表

确认对话框只有两个选项（`是，打开` / `否，重新选择`），用 ↑↓/jk 选择后 Enter 确认；ESC 直接返回项目列表。这里没有 `Y`/`N` 快捷键，是刻意的设计，详见 [confirmation-design.md](confirmation-design.md)。

## 视图功能详解

### 1. Overview（总览）
- 项目总数、会话总数、记忆文件数、总占用空间
- 当前活跃的 Claude Code 会话（最多显示 3 条，来自 `sessions/<pid>.json`）
- 最近会话（最多 5 条）

### 2. Projects（项目）
- 列表顶部固定一项 `[+] 开启新项目`
- 每个项目显示路径、会话数、占用空间、最后活动时间
- 按最后活动时间倒序；没有活动时间的项目按路径排序

### 3. Sessions（会话）
- 会话 ID（前 8 位）、首次提示、消息数、修改时间、文件大小
- 标题栏显示 `已过滤/总数`，以及当前过滤条件和搜索词
- 数据优先来自各项目的 `sessions-index.json`

### 4. Memory（记忆）
- 每个项目的 `MEMORY.md`：路径、条目数、文件大小、修改时间

### 5. Guide（说明）
内置 Claude 记忆系统文档：自动记忆、4 种记忆类型、`MEMORY.md` 索引、`CLAUDE.md` 位置、数据目录结构、路径编码规则。支持滚动查看。

### 6. Settings（设置）
- 显示 Claude 目录、语言、日期格式、大小单位、每页条数，以及配置文件路径
- `T` 可切换语言并保存

## 配置文件

位置：`C:\Users\<用户名>\.claude-manager\config.json`（其他平台为 `~/.claude-manager/config.json`）

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

`display` 下的三项目前只是占位，视图尚未读取；要改 Claude 目录需要手动编辑 `claudeDir` 后重启。

## Claude 数据目录结构

```
.claude/
├── projects/
│   ├── D--jarvis/                 # 项目目录（路径编码）
│   │   ├── <session-id>.jsonl     # 会话对话记录
│   │   ├── <session-id>/
│   │   │   ├── subagents/
│   │   │   └── tool-results/
│   │   ├── sessions-index.json    # 会话索引
│   │   ├── archived/              # 归档的会话文件
│   │   └── memory/
│   │       └── MEMORY.md          # 项目记忆
│   └── ...
├── sessions/
│   └── <pid>.json                 # 活跃会话元数据
├── settings.json
└── history.jsonl
```

### 路径编码规则

文件系统路径被编码为目录名：

- `D:\jarvis` → `D--jarvis`
- `C:\Users\TU` → `C--Users-TU`
- `/home/user/projects` → `home-user-projects`

## 常见问题

**Q: 启动后显示 "Claude directory not found"**
A: 确保 Claude Code 已安装并至少运行过一次，生成了 `.claude` 目录；如果目录不在默认位置，在配置文件中修改 `claudeDir`。

**Q: 看不到任何项目/会话**
A: 确认配置文件里的 `claudeDir` 指向正确；可以在设置页查看当前路径。

**Q: 如何切换语言？**
A: 按 `T`，主菜单和设置页都支持，切换后自动保存。

**Q: 终端显示乱码**
A: 确保终端支持 UTF-8 和 Unicode；Windows 推荐 Windows Terminal。

**Q: 如何自定义 Claude 目录位置？**
A: 手动编辑 `~/.claude-manager/config.json` 中的 `claudeDir`，然后重启程序。

**Q: 打开项目没反应 / 报错？**
A: "打开项目"通过 PowerShell 的 `Start-Process` 启动新窗口，目前仅支持 Windows，且要求 `claude` 在 PATH 中。

**Q: 删除项目会删掉什么？**
A: 整个 `.claude/projects/<编码名>/` 目录，包括该项目所有会话与记忆，不可恢复。删除前必须按 `Y` 确认。

## 系统要求

- Windows 10/11、macOS 或 Linux（"打开项目"功能仅 Windows）
- Go 1.25.5+（自行编译时需要）
- Claude Code 已安装
- 支持 Unicode 的终端；目录选择器需要至少 60 列宽度，其余视图建议 80x24

## 技术架构

- **语言**：Go
- **主菜单**：Bubble Tea（动画 + 定时刷新）
- **视图**：tcell v2（全屏渲染，每个视图独立事件循环）
- **视图组织**：注册表 + `init()` 自动注册，新增视图不需要改选择器
- **国际化**：i18n 包集中管理中英文文案，语言写入配置持久化

## 待实现

- 在文件管理器中打开项目目录
- 记忆文件的内联编辑
- 自动清理旧数据
- 会话导出（Markdown/HTML）
- 统计图表
- 设置页可编辑（当前只能改语言）

## 相关文档

- [confirmation-design.md](confirmation-design.md) — 确认对话框设计
- [dirpicker-test-guide.md](dirpicker-test-guide.md) — 目录选择器测试清单
- [animation-design.md](animation-design.md) — 菜单动画设计
- [claude-code-session-memory.md](claude-code-session-memory.md) — Claude Code 会话与记忆机制

---

## English Quick Start

1. Build: `make build` (or `go build -o build/claude-manager.exe ./cmd/claude-manager`)
2. Run: `./build/claude-manager.exe`

Main menu: `↑↓/jk` navigate, `Enter` select, `T` toggle language, `Q`/`ESC` quit.

Views: `ESC` back, `Q` quit, `R` refresh. Projects additionally use `Enter` to open, `D` to delete. Sessions add `Enter` for details, `/` search, `F` filter, `D` delete, `A` archive — deletion and archiving both ask for `Y` confirmation.

Directory picker: `←→/hl` move between levels, `↑↓/jk` move within a column, `Enter` select, `ESC` cancel.

See [README.md](../README.md) for the full overview.
