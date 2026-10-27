# 文档索引

| 文档 | 内容 |
|---|---|
| [quickstart.md](quickstart.md) | 快速上手：构建、界面、全部快捷键、常见问题 |
| [claude-code-session-memory.md](claude-code-session-memory.md) | Claude Code 的会话管理与会话/记忆三层架构（针对开发者，含 `history.jsonl` 字段说明） |
| [dirpicker-test-guide.md](dirpicker-test-guide.md) | 目录选择器与"开启新项目"流程的手工测试清单 |
| [confirmation-design.md](confirmation-design.md) | 确认对话框的最终设计（为什么去掉 Y/N 快捷键，ESC 返回哪里） |
| [confirmation-improvement.md](confirmation-improvement.md) | 确认对话框的演进记录：从按键提示到选项列表（已被上一份文档取代） |
| [animation-design.md](animation-design.md) | 主菜单 spinner 动画：九种动画、切换机制、技术实现 |
| [animation-options.md](animation-options.md) | 动画候选清单与取舍记录（含被否决的 emoji 方案） |

根目录的 [README.md](../README.md) 是项目总览，包含功能、快捷键、配置和项目结构。

## 文档与代码的对应关系

| 文档 | 对应源码 |
|---|---|
| animation-design.md | `pkg/ui/animation/spinner.go` |
| confirmation-design.md | `pkg/views/dirpicker/dirpicker.go`、`render.go`、`models.go` |
| dirpicker-test-guide.md | `pkg/views/dirpicker/*`、`pkg/views/projects/projects.go` |
| claude-code-session-memory.md | `pkg/core/claude_dir.go`、`pkg/core/models.go` |

> 写文档时以代码为准：上述文档描述的是 Claude Code 的目录格式与工具的设计意图，当实际实现变化时请同步更新。
