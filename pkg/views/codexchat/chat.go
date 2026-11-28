// Package codexchat renders a local Codex app-server session inside AgentDeck.
package codexchat

import (
	"context"
	"encoding/json"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/kira-peter/agent_deck/pkg/core"
	"github.com/kira-peter/agent_deck/pkg/i18n"
)

type item struct {
	ID        string               `json:"id"`
	Type      string               `json:"type"`
	Text      string               `json:"text"`
	Command   string               `json:"command"`
	Output    string               `json:"aggregatedOutput"`
	Status    string               `json:"status"`
	Delivery  string               `json:"delivery"`
	Questions []codexAsyncQuestion `json:"questions"`
	Content   []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Changes []struct {
		Path string `json:"path"`
		Diff string `json:"diff"`
	} `json:"changes"`
}

type turn struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Items  []item `json:"items"`
	Error  *struct {
		Message string `json:"message"`
	} `json:"error"`
}

type chat struct {
	mu                 sync.Mutex
	screen             tcell.Screen
	server             *appServer
	kind               core.SourceKind
	claudePath         string
	stopClaude         context.CancelFunc
	interruptClaude    func()
	wakeClaude         func()
	home               string
	project, sessionID string
	threadID, model    string
	modelProvider      string
	desiredModel       string
	models             []string
	turnID, notice     string
	ready, working     bool
	interruptPending   bool
	connecting, used   bool
	keepItems          bool
	lastTurnStatus     string
	closed             bool
	showTools          bool
	revealTool         bool
	chartCollapsed     bool
	chart              tokenHistory
	items              []item
	itemIndex          map[string]int
	requests           map[string]string
	nextID             int
	prompts            []sessionPrompt
	asyncSeen          map[string]bool
	asyncReplies       []codexAsyncReply
	asyncReplyRequest  string
	asyncRetryAfter    string
	draft              []rune
	cursor, scroll     int
}

func (c *chat) agentKind() core.SourceKind {
	if c.kind == core.SourceClaude {
		return core.SourceClaude
	}
	return core.SourceCodex
}

func (c *chat) toggleToolDetails() string {
	c.showTools = !c.showTools
	c.scroll, c.revealTool = 0, false
	if !c.showTools {
		return local("Tool details collapsed for this window.", "此窗口的工具详情已收起。")
	}
	for _, item := range c.items {
		if item.Type != "userMessage" && item.Type != "agentMessage" {
			c.revealTool = true
			return local("Tool details expanded for this window.", "此窗口的工具详情已展开。")
		}
	}
	return local("Tool details expanded. This window has no tool calls yet.", "工具详情已展开，此窗口暂无工具调用。")
}

func local(en, zh string) string {
	return (i18n.Text{En: en, Zh: zh}).Get()
}

func (c *chat) send(message map[string]any) bool {
	if err := c.server.writer.Encode(message); err != nil {
		c.notice, c.ready, c.working = err.Error(), false, false
		c.connecting = false
		c.interruptPending = false
		c.prompts = nil
		return false
	}
	return true
}

func (c *chat) request(method string, params any) bool {
	c.nextID++
	if !c.send(map[string]any{"id": c.nextID, "method": method, "params": params}) {
		return false
	}
	c.requests[strconv.Itoa(c.nextID)] = method
	return true
}

func (c *chat) response(msg rpcMessage) {
	id := string(msg.ID)
	method := c.requests[id]
	delete(c.requests, id)
	if id == c.asyncReplyRequest && id != "" {
		c.finishCodexAsyncReply(method, msg)
		return
	}
	if msg.Error != nil {
		if method == "model/list" {
			return // the picker still accepts a manually entered model ID
		}
		c.notice = msg.Error.Message
		switch method {
		case "initialize", "thread/start", "thread/resume":
			c.ready, c.connecting = false, false
		case "thread/turns/list":
			c.ready, c.connecting = true, false
		case "turn/start":
			c.working, c.interruptPending = false, false
			c.lastTurnStatus = "failed"
		}
		return
	}
	switch method {
	case "initialize":
		if !c.send(map[string]any{"method": "initialized"}) {
			return
		}
		params := map[string]any{"cwd": c.project}
		if c.desiredModel != "" {
			params["model"] = c.desiredModel
		}
		if c.sessionID == "" {
			c.request("thread/start", params)
		} else {
			params["threadId"], params["excludeTurns"] = c.sessionID, true
			c.request("thread/resume", params)
		}
	case "thread/start", "thread/resume":
		var result struct {
			Thread        struct{ ID string } `json:"thread"`
			Model         string              `json:"model"`
			ModelProvider string              `json:"modelProvider"`
		}
		if json.Unmarshal(msg.Result, &result) != nil || result.Thread.ID == "" {
			c.notice = local("Codex did not return a session ID", "Codex 未返回会话 ID")
			c.connecting = false
			return
		}
		c.threadID, c.model = result.Thread.ID, result.Model
		c.modelProvider = result.ModelProvider
		c.used = method == "thread/resume"
		if c.used && !c.keepItems {
			c.request("thread/turns/list", map[string]any{"threadId": c.threadID, "limit": 10, "sortDirection": "desc", "itemsView": "full"})
		} else {
			c.ready, c.connecting = true, false
		}
	case "thread/turns/list":
		var result struct{ Data []turn }
		if json.Unmarshal(msg.Result, &result) == nil {
			for i := len(result.Data) - 1; i >= 0; i-- {
				for _, entry := range result.Data[i].Items {
					c.saveItem(entry)
				}
			}
		}
		c.ready, c.connecting = true, false
	case "model/list":
		var result struct{ Data []struct{ ID, Model string } }
		if json.Unmarshal(msg.Result, &result) == nil {
			c.models = nil
			for _, option := range result.Data {
				id := option.Model
				if id == "" {
					id = option.ID
				}
				if id != "" && !slices.Contains(c.models, id) {
					c.models = append(c.models, id)
				}
			}
		}
	case "turn/start":
		var result struct{ Turn turn }
		if json.Unmarshal(msg.Result, &result) == nil && c.working {
			c.turnID = result.Turn.ID
			c.flushInterrupt()
			c.sendCodexAsyncReply()
		}
	}
}

func (c *chat) receive(msg rpcMessage) {
	if msg.Method == "" {
		c.response(msg)
		return
	}
	if len(msg.ID) > 0 {
		if msg.Method == "item/tool/requestUserInput" {
			c.queueCodexQuestions(msg)
		} else {
			c.queueApproval(msg)
		}
		return
	}
	var params struct {
		ThreadID   string          `json:"threadId"`
		ItemID     string          `json:"itemId"`
		Delta      string          `json:"delta"`
		Item       item            `json:"item"`
		Turn       turn            `json:"turn"`
		RequestID  json.RawMessage `json:"requestId"`
		TokenUsage struct {
			Total struct {
				Input  int64 `json:"inputTokens"`
				Output int64 `json:"outputTokens"`
			} `json:"total"`
		} `json:"tokenUsage"`
	}
	if json.Unmarshal(msg.Params, &params) != nil || (params.ThreadID != "" && params.ThreadID != c.threadID && params.ThreadID != c.sessionID) {
		return
	}
	switch msg.Method {
	case "thread/tokenUsage/updated":
		total := params.TokenUsage.Total
		c.chart.record(core.CodexTokenSample{At: time.Now(), Input: total.Input, Output: total.Output})
	case "item/started", "item/completed":
		c.saveItem(params.Item)
		if msg.Method == "item/completed" && params.Item.Delivery == "async" {
			c.queueCodexAsyncQuestions(params.Item)
		}
	case "item/agentMessage/delta", "item/commandExecution/outputDelta":
		if index, ok := c.itemIndex[params.ItemID]; ok {
			if msg.Method == "item/agentMessage/delta" {
				c.items[index].Text += params.Delta
			} else {
				c.items[index].Output += params.Delta
			}
		}
	case "turn/started":
		c.turnID, c.working, c.used = params.Turn.ID, true, true
		c.flushInterrupt()
		c.sendCodexAsyncReply()
	case "turn/completed":
		c.turnID, c.working = "", false
		c.interruptPending = false
		c.lastTurnStatus = params.Turn.Status
		c.prompts = slices.DeleteFunc(c.prompts, func(p sessionPrompt) bool { return !p.async })
		if params.Turn.Error != nil {
			c.notice = params.Turn.Error.Message
		}
		c.sendCodexAsyncReply()
	case "serverRequest/resolved":
		c.removePrompt(string(params.RequestID))
	}
}

func (c *chat) saveItem(entry item) {
	switch entry.Type {
	case "userMessage", "agentMessage", "commandExecution", "fileChange", "mcpToolCall", "webSearch":
	default:
		return
	}
	if index, ok := c.itemIndex[entry.ID]; ok {
		if entry.Output == "" {
			entry.Output = c.items[index].Output
		}
		c.items[index] = entry
	} else if entry.ID != "" {
		c.itemIndex[entry.ID] = len(c.items)
		c.items = append(c.items, entry)
	}
}

func (c *chat) queueApproval(msg rpcMessage) {
	if msg.Method != "item/commandExecution/requestApproval" && msg.Method != "item/fileChange/requestApproval" {
		c.notice = local("Open the original CLI for this request: ", "此请求请使用原版 CLI：") + msg.Method
		c.send(map[string]any{"id": msg.ID, "error": map[string]any{"code": -32601, "message": "Unsupported AgentDeck request"}})
		return
	}
	var params struct {
		ThreadID  string                           `json:"threadId"`
		ItemID    string                           `json:"itemId"`
		Kind      string                           `json:"kind"`
		Command   string                           `json:"command"`
		CWD       string                           `json:"cwd"`
		Reason    string                           `json:"reason"`
		Root      string                           `json:"grantRoot"`
		Network   *struct{ Host, Protocol string } `json:"networkApprovalContext"`
		Decisions []json.RawMessage                `json:"availableDecisions"`
	}
	if json.Unmarshal(msg.Params, &params) != nil || params.ThreadID != c.threadID || (params.Kind != "" && params.Kind != "command") {
		c.send(map[string]any{"id": msg.ID, "result": map[string]string{"decision": "cancel"}})
		return
	}
	accept, decline := len(params.Decisions) == 0, len(params.Decisions) == 0
	for _, decision := range params.Decisions {
		accept = accept || string(decision) == `"accept"`
		decline = decline || string(decision) == `"decline"`
	}
	details := []string{params.Reason}
	hasTarget := false
	switch {
	case params.Network != nil:
		details = append(details, params.Network.Protocol+": "+params.Network.Host)
		hasTarget = params.Network.Host != ""
	case msg.Method == "item/commandExecution/requestApproval":
		details = append(details, params.Command, params.CWD)
		hasTarget = params.Command != ""
	default:
		if params.Root != "" {
			details = append(details, params.Root)
			hasTarget = true
		}
		if index, ok := c.itemIndex[params.ItemID]; ok {
			for _, change := range c.items[index].Changes {
				details = append(details, change.Path, change.Diff)
				hasTarget = hasTarget || change.Path != ""
			}
		}
	}
	c.queuePermission(string(msg.ID), strings.TrimSpace(strings.Join(details, "\n")), accept && hasTarget, func(allowed bool) {
		decision := "cancel"
		if allowed {
			decision = "accept"
		} else if decline {
			decision = "decline"
		}
		c.send(map[string]any{"id": msg.ID, "result": map[string]string{"decision": decision}})
	})
}

func (c *chat) flushInterrupt() {
	if c.interruptPending && c.turnID != "" {
		c.interruptPending = false
		c.request("turn/interrupt", map[string]string{"threadId": c.threadID, "turnId": c.turnID})
	}
}

func (c *chat) interrupt() bool {
	switch {
	case c.working && c.interruptClaude != nil:
		c.interruptClaude()
	case c.working && c.server != nil:
		c.interruptPending = true
		c.flushInterrupt()
	case len(c.prompts) > 0:
		c.finishPrompt(nil)
	default:
		return false
	}
	return true
}

func (c *chat) handleKey(event *tcell.EventKey) {
	if event.Key() == tcell.KeyF9 || event.Key() == tcell.KeyCtrlC {
		c.interrupt()
		return
	}
	if len(c.prompts) > 0 {
		c.handlePrompt(event)
		return
	}
	switch event.Key() {
	case tcell.KeyF3:
		c.chartCollapsed = !c.chartCollapsed
	case tcell.KeyPgUp:
		c.scroll += 5
	case tcell.KeyPgDn:
		c.scroll = max(c.scroll-5, 0)
	case tcell.KeyLeft:
		c.cursor = max(c.cursor-1, 0)
	case tcell.KeyRight:
		c.cursor = min(c.cursor+1, len(c.draft))
	case tcell.KeyHome:
		c.cursor = 0
	case tcell.KeyEnd:
		c.cursor = len(c.draft)
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if c.cursor > 0 {
			c.draft = slices.Delete(c.draft, c.cursor-1, c.cursor)
			c.cursor--
		}
	case tcell.KeyDelete:
		if c.cursor < len(c.draft) {
			c.draft = slices.Delete(c.draft, c.cursor, c.cursor+1)
		}
	case tcell.KeyRune:
		c.draft = slices.Insert(c.draft, c.cursor, event.Rune())
		c.cursor++
	case tcell.KeyEnter:
		if input := strings.TrimSpace(string(c.draft)); c.ready && !c.working && input != "" {
			if c.agentKind() == core.SourceClaude {
				c.startClaudeTurn(input)
				return
			}
			c.notice = ""
			if c.request("turn/start", c.codexInput(input)) {
				c.draft, c.cursor, c.scroll, c.working = nil, 0, 0, true
			}
		}
	}
}
