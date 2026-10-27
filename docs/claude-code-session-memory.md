# Claude Code 会话管理与记忆机制详解

> 基于 Claude Code 实际目录结构分析，面向开发者。

---

## 一、目录结构总览

```
C:\Users\<用户名>\.claude\
├── history.jsonl          ← 所有对话的原始数据（核心）
├── CLAUDE.md              ← 全局个人指令
├── config.json            ← 配置
├── settings.json          ← 设置
├── stats-cache.json       ← 统计缓存
├── sessions/              ← 会话元数据
├── projects/              ← 项目级持久上下文
├── plans/                 ← 任务计划
├── tasks/                 ← 子任务
├── plugins/               ← 插件
├── skills/                ← 自定义技能
├── backups/               ← 备份
├── cache/                 ← 缓存
├── debug/                 ← 调试日志
├── downloads/             ← 下载文件
├── file-history/          ← 文件修改历史
├── paste-cache/           ← 粘贴板缓存
├── shell-snapshots/       ← shell 快照
├── session-env/           ← 会话环境变量
├── ide/                   ← IDE 集成状态
└── .last-cleanup          ← 最后清理时间戳
```

---

## 二、核心文件：`history.jsonl`

### 2.1 格式

每行一条 JSON 记录，结构大致如下：

```json
{
  "id": "msg-uuid",
  "sessionId": "session-uuid",
  "project": "/path/to/project",
  "parentId": "parent-msg-uuid",
  "role": "user|assistant|system",
  "content": [{ "type": "text", "text": "..." }],
  "timestamp": 1783667528470,
  "toolCalls": [],
  "toolResults": [],
  "compactedFrom": ["msg-id-1", "msg-id-2"],
  "compacted": true
}
```

### 2.2 关键字段

| 字段 | 说明 |
|---|---|
| `sessionId` | 会话唯一标识，关联 `sessions/` 目录 |
| `project` | 项目路径，用于按项目筛选 |
| `parentId` | 对话树结构，支持分支 |
| `compacted` | 是否是 compact 后的摘要消息 |
| `compactedFrom` | 摘要来源消息 ID 列表 |

### 2.3 读写特性

- **追加写入**：新消息 append 到文件末尾
- **全量读取**：启动时读取整个文件，按 `sessionId` + `project` 过滤
- **单文件多会话**：所有项目的所有会话混在一起，靠字段区分

---

## 三、`sessions/` 目录

每个文件对应一个会话，存储元数据：

```json
{
  "id": "session-uuid",
  "project": "D:\\fireWar\\icefire-game",
  "createdAt": 1783667528470,
  "updatedAt": 1783667528470,
  "title": "修复武器系统bug",
  "status": "active|archived",
  "messageCount": 42,
  "compacted": true,
  "summary": "本次对话主要讨论..."
}
```

**作用**：快速查找"某个项目有哪些会话"，避免每次扫描整个 `history.jsonl`。

---

## 四、`projects/` 目录

存储项目级持久信息，通常包含：

```json
{
  "projectPath": "D:\\fireWar\\icefire-game",
  "activeSessionId": "session-uuid",
  "CLAUDE.md": "...",
  "customInstructions": "...",
  "lastAccessed": 1783667528470
}
```

**关键**：`activeSessionId` 就是"当前项目对应哪个会话"的指针。下次在同目录启动 Claude，先读这个文件找到 sessionId，再从 `history.jsonl` 加载对应消息。

---

## 五、记忆的三层架构

### 第一层：`CLAUDE.md`（永久记忆）

| 位置 | 作用域 | 优先级 |
|---|---|---|
| `~/.claude/CLAUDE.md` | 全局，所有项目 | 低 |
| `<项目根目录>/CLAUDE.md` | 仅当前项目 | 中 |
| `<项目根目录>/.claude/CLAUDE.md` | 仅当前项目 | 高 |

启动时自动加载，**不占对话窗口**，注入到 system prompt。

**写入方式**：

- 手动编辑
- `/init` 命令自动生成
- Claude 在对话中主动写入

**适合放的内容**：

- 技术栈、架构约定
- 命名规范、代码风格
- 常用命令、环境变量
- 项目结构说明

### 第二层：`/compact` 摘要（半持久记忆）

**机制**：

1. 调用 `/compact`
2. Claude 读取当前对话全文
3. 生成一条**压缩摘要消息**，替代原始消息
4. 原始消息标记 `compacted: true`，不再加载
5. 摘要消息进入上下文窗口

**compact 前**：

```
[msg1] [msg2] [msg3] ... [msg100]  ← 占 100K tokens
```

**compact 后**：

```
[摘要: 本次对话讨论了武器系统的设计...]  ← 占 500 tokens
```

**关键特性**：

- 摘要本身也是调用模型生成的，所以对话太长导致模型拒绝时会 compact 失败
- 摘要丢失细节（具体代码片段、中间讨论），保留决策和结论
- 摘要消息在 `history.jsonl` 中也有 `compactedFrom` 字段指向原始消息

### 第三层：上下文窗口（临时记忆）

当前对话中模型能"看到"的内容，受 token 上限限制。

**工具调用也在窗口里**：每次 tool call 和 tool result 都占用 token。

---

## 六、会话生命周期

```
启动 claude
  │
  ├─→ 检查 projects/ 是否有当前项目的 activeSessionId
  │     ├─ 有 → 从 history.jsonl 加载该会话
  │     │       ├─ 过滤出未被 compact 的消息
  │     │       └─ 注入 CLAUDE.md
  │     └─ 无 → 创建新 session
  │
  ├─→ 对话中每条消息写入 history.jsonl
  │
  ├─→ /compact 触发
  │     ├─ 生成摘要消息
  │     ├─ 标记原始消息 compacted
  │     └─ 更新 sessions/ 元数据
  │
  ├─→ /clear
  │     ├─ 创建新 session
  │     └─ 旧 session 标记 archived
  │
  └─→ 退出
        ├─ 更新 sessions/ 的 updatedAt、messageCount
        └─ 更新 projects/ 的 lastAccessed
```

---

## 七、跨重启恢复流程

1. 启动 `claude`
2. 读取 `projects/<project-hash>.json` → 获取 `activeSessionId`
3. 读取 `sessions/<sessionId>.json` → 获取 `status`、`compacted` 等
4. 扫描 `history.jsonl` → 过滤 `sessionId == activeSessionId && compacted != true`
5. 按 `parentId` 构建对话树
6. 组装上下文窗口 + CLAUDE.md + 系统提示词

---

## 八、开发工具时的关键点

### 8.1 解析/导出会话

```python
import json

def load_session(history_path, session_id):
    messages = []
    with open(history_path, 'r', encoding='utf-8') as f:
        for line in f:
            msg = json.loads(line)
            if msg.get('sessionId') == session_id and not msg.get('compacted'):
                messages.append(msg)
    return messages
```

### 8.2 会话备份/迁移

打包以下内容即可完整迁移：

```
~/.claude/history.jsonl
~/.claude/sessions/<session-id>.json
~/.claude/projects/<project-hash>.json
```

### 8.3 会话分析

- 统计 token 消耗：从 `toolResults` 中提取 `usage` 字段
- 统计工具调用频率：解析 `toolCalls` 数组
- 统计对话长度：`messageCount` 或 `history.jsonl` 中相关行数

### 8.4 自动 compact

在工具中监控 `history.jsonl` 中当前会话的未 compact 消息数，超过阈值时：

```python
# 通过 stdin 发送 /compact 命令
# 或模拟 claude 的内部 compact 流程
```

### 8.5 注意事项

- **不要在高频写入时并发读**：`history.jsonl` 是追加写，但并发读可能读到不完整行
- **compact 是不可逆的**：原始消息标记后不会再被加载，虽然没删但恢复非常麻烦
- **sessionId 是 UUID**：不要自己生成，用现有的
- **跨平台路径问题**：`project` 字段在 Windows 上是 `D:\xxx`，Linux 上是 `/home/xxx`，迁移时需注意

---

## 九、死锁问题：对话过长导致 compact 失败

### 问题机制

```
对话长度 → 超过模型 token 上限
  → 任何请求都被拒绝（包括 /compact）
  → /compact 无法执行
  → 对话无法缩短
  → 永远卡死
```

### 工具层面解决方案

在 `history.jsonl` 层面手动截断，绕过模型调用：

```python
def manual_compact(history_path, session_id, keep_last_n=50):
    # 1. 备份
    # 2. 读取所有消息
    # 3. 找到该 session 的非 compacted 消息
    # 4. 保留最后 N 条，其余标记 compacted
    # 5. 生成一条人工摘要消息，compactedFrom 指向被截断的消息
    # 6. 写回
```

### 临时脱困方法

```powershell
# 方案 A：删除当前会话文件，白板重开
ren C:\Users\<用户名>\.claude\sessions\<session-id>.json <session-id>.json.bak

# 方案 B：手动截断 history.jsonl（保留最后 500 行）
$lines = Get-Content C:\Users\<用户名>\.claude\history.jsonl
$lines[-500..-1] | Set-Content C:\Users\<用户名>\.claude\history.jsonl
```

---

## 十、总结

Claude Code 的会话体系核心：

| 组件 | 作用 |
|---|---|
| `history.jsonl` | 全量对话数据，所有项目所有会话混存 |
| `sessions/` | 会话索引，快速查找 |
| `projects/` | 项目级指针，记录 activeSessionId |
| `CLAUDE.md` | 永久记忆，不占对话窗口 |
| `/compact` 摘要 | 半持久记忆，压缩长对话 |
| 上下文窗口 | 临时记忆，受 token 上限限制 |

三层记忆各司其职：**CLAUDE.md 永久**、**compact 半持久**、**上下文窗口临时**。开发工具时主要操作 `history.jsonl` 和 `sessions/` 即可。
