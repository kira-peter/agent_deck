package codexchat

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/kira-peter/agent_deck/pkg/core"
)

type claudeContent struct {
	Type, ID, Name, Text string
	ToolUseID            string `json:"tool_use_id"`
	Input, Content       json.RawMessage
	IsError              bool `json:"is_error"`
}

type claudeEvent struct {
	Type, Subtype, Model, Result string
	SessionID                    string `json:"session_id"`
	ParentToolID                 string `json:"parent_tool_use_id"`
	IsError                      bool   `json:"is_error"`
	Errors                       []string
	Denials                      []json.RawMessage `json:"permission_denials"`
	RequestID                    string            `json:"request_id"`
	Request                      struct {
		Subtype, Title, Description string
		ToolName                    string `json:"tool_name"`
		BlockedPath                 string `json:"blocked_path"`
		Input                       map[string]json.RawMessage
	}
	Message struct {
		Model   string
		Content []claudeContent
	}
	Event struct {
		Index int
		Delta struct{ Type, Text string }
	}
	Usage struct {
		Input      int64 `json:"input_tokens"`
		Output     int64 `json:"output_tokens"`
		CacheRead  int64 `json:"cache_read_input_tokens"`
		CacheWrite int64 `json:"cache_creation_input_tokens"`
	}
}

func (c *chat) connectClaude(historyPath string, wake func()) {
	path, lookupErr := exec.LookPath("claude")
	var history []core.SessionMessage
	var usage core.Usage
	var modified time.Time
	if historyPath != "" && !c.keepItems {
		history, _ = core.NewClaudeDir(c.home).GetSessionMessagesFromFile(historyPath, 0)
		usage, _ = core.ScanSessionUsage(historyPath)
		if info, err := os.Stat(historyPath); err == nil {
			modified = info.ModTime()
		}
	}
	c.mu.Lock()
	defer wake()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	c.connecting = false
	if lookupErr != nil {
		c.notice = lookupErr.Error()
		return
	}
	c.claudePath, c.wakeClaude = path, wake
	c.threadID, c.used = c.sessionID, c.sessionID != ""
	if c.desiredModel != "" {
		c.model = c.desiredModel
	}
	for i, message := range history {
		c.appendClaudeMessage(fmt.Sprintf("claude-history-%d", i), message.Role, message.Content)
	}
	if !modified.IsZero() && usage.InputTokens+usage.OutputTokens+usage.CacheReadTokens+usage.CacheWriteTokens > 0 {
		c.chart.record(core.CodexTokenSample{At: modified, Input: usage.InputTokens + usage.CacheReadTokens + usage.CacheWriteTokens, Output: usage.OutputTokens})
	}
	c.ready = true
}

func newClaudeID() string {
	id := make([]byte, 16)
	rand.Read(id)
	id[6] = (id[6] & 15) | 64
	id[8] = (id[8] & 63) | 128
	return fmt.Sprintf("%x-%x-%x-%x-%x", id[:4], id[4:6], id[6:8], id[8:10], id[10:])
}

func (c *chat) appendClaudeMessage(id, role, text string) {
	if role == "assistant" {
		c.saveItem(item{ID: id, Type: "agentMessage", Text: text})
		return
	}
	entry := item{ID: id, Type: "userMessage"}
	entry.Content = append(entry.Content, struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}{Type: "text", Text: text})
	c.saveItem(entry)
}

// startClaudeTurn is called with the chat locked. Claude Code's print process is
// scoped to one turn; later turns resume the same local session ID.
func (c *chat) startClaudeTurn(input string) {
	if c.threadID == "" {
		c.threadID = newClaudeID()
	}
	args := []string{"--print", "--input-format", "stream-json", "--output-format", "stream-json", "--verbose",
		"--include-partial-messages", "--permission-prompt-tool", "stdio"}
	if c.used {
		args = append(args, "--resume="+c.threadID)
	} else {
		args = append(args, "--session-id="+c.threadID)
	}
	if c.desiredModel != "" {
		args = append(args, "--model="+c.desiredModel)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cmd, err := claudeCommand(ctx, c.claudePath, args)
	if err != nil {
		cancel()
		c.notice = err.Error()
		return
	}
	cmd.Dir = c.project
	cmd.Env = append(cmd.Environ(), "CLAUDE_CONFIG_DIR="+c.home)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdout, err := cmd.StdoutPipe()
	var stdin io.WriteCloser
	if err == nil {
		stdin, err = cmd.StdinPipe()
	}
	if err == nil {
		err = cmd.Start()
	}
	if err != nil {
		cancel()
		if stdout != nil {
			stdout.Close()
		}
		if stdin != nil {
			stdin.Close()
		}
		c.notice = err.Error()
		return
	}
	c.nextID++
	prefix := fmt.Sprintf("claude-%d-", c.nextID)
	c.appendClaudeMessage(prefix+"user", "user", input)
	c.draft, c.cursor, c.scroll, c.working = nil, 0, 0, true
	c.notice, c.stopClaude = "", cancel
	writer := json.NewEncoder(stdin)
	interrupted := false
	var stopTimer *time.Timer
	send := func(event map[string]any) {
		if err := writer.Encode(event); err != nil {
			c.notice = err.Error()
			_ = stdin.Close()
		}
	}
	control := func(id, subtype string) {
		send(map[string]any{"type": "control_request", "request_id": id, "request": map[string]string{"subtype": subtype}})
	}
	c.interruptClaude = func() {
		if interrupted {
			return
		}
		interrupted = true
		c.prompts = nil
		c.notice, c.lastTurnStatus = "", "interrupted"
		stopTimer = time.AfterFunc(2*time.Second, cancel)
		control("deck-interrupt", "interrupt")
	}
	sessionID := c.threadID
	wake := c.wakeClaude
	go func() {
		defer cancel()
		c.mu.Lock()
		if !c.closed && !interrupted {
			control("deck-init", "initialize")
			send(map[string]any{"type": "user", "session_id": sessionID, "parent_tool_use_id": nil,
				"message": map[string]string{"role": "user", "content": input}})
		}
		c.mu.Unlock()
		state := claudeTurn{prefix: prefix}
		scan := bufio.NewScanner(stdout)
		scan.Buffer(make([]byte, 64*1024), 16*1024*1024)
		for scan.Scan() {
			var event claudeEvent
			if json.Unmarshal(scan.Bytes(), &event) != nil {
				continue
			}
			c.mu.Lock()
			if !c.closed {
				switch event.Type {
				case "control_request":
					if !interrupted {
						c.queueClaudePrompt(event, send)
					}
				case "control_cancel_request":
					c.removePrompt(event.RequestID)
				case "result":
					if interrupted {
						state.finished = true
					} else {
						state.apply(c, event)
					}
				default:
					state.apply(c, event)
				}
				if state.finished {
					_ = stdin.Close()
				}
			}
			c.mu.Unlock()
			wake()
		}
		if scan.Err() != nil {
			_ = cmd.Process.Kill()
		}
		waitErr := cmd.Wait()
		c.mu.Lock()
		if stopTimer != nil {
			stopTimer.Stop()
		}
		if !c.closed {
			c.working, c.stopClaude, c.interruptClaude = false, nil, nil
			c.prompts = nil
			switch {
			case interrupted || ctx.Err() != nil:
				c.lastTurnStatus = "interrupted"
				c.notice = ""
			case scan.Err() != nil:
				c.notice, c.lastTurnStatus = scan.Err().Error(), "failed"
			case waitErr != nil || !state.finished:
				c.lastTurnStatus = "failed"
				if c.notice == "" {
					c.notice = strings.TrimSpace(stderr.String())
					if c.notice == "" && waitErr != nil {
						c.notice = waitErr.Error()
					}
					if c.notice == "" {
						c.notice = local("Claude Code closed before sending a result.", "Claude Code 在返回结果前退出。")
					}
				}
			}
			if !c.used {
				path := filepath.Join(c.home, "projects", core.EncodePath(c.project), c.threadID+".jsonl")
				if _, err := os.Stat(path); err == nil {
					c.used = true
				}
			}
		}
		c.mu.Unlock()
		wake()
	}()
}

func (c *chat) queueClaudePrompt(event claudeEvent, send func(map[string]any)) {
	id, request := event.RequestID, event.Request
	if id == "" {
		return
	}
	reply := func(subtype string, payload map[string]any) {
		payload["subtype"], payload["request_id"] = subtype, id
		send(map[string]any{"type": "control_response", "response": payload})
	}
	if request.Subtype != "can_use_tool" {
		reply("error", map[string]any{"error": "Unsupported AgentDeck control request"})
		return
	}
	decide := func(allow bool) {
		response := map[string]any{"behavior": "deny", "message": "User declined this request"}
		if allow {
			response = map[string]any{"behavior": "allow", "updatedInput": request.Input}
		}
		reply("success", map[string]any{"response": response})
	}
	if request.ToolName == "AskUserQuestion" {
		var questions []promptQuestion
		if json.Unmarshal(request.Input["questions"], &questions) != nil || len(questions) == 0 {
			decide(false)
			return
		}
		for i := range questions {
			questions[i].ID = questions[i].Question
			questions[i].IsOther = true
		}
		c.queueQuestions(id, questions, func(answers map[string][]string) {
			if len(answers) == 0 {
				decide(false)
				return
			}
			values := make(map[string]string, len(answers))
			for question, selections := range answers {
				values[question] = strings.Join(selections, ", ")
			}
			request.Input["answers"], _ = json.Marshal(values)
			decide(true)
		})
		return
	}
	input, _ := json.MarshalIndent(request.Input, "", "  ")
	detail := strings.TrimSpace(strings.Join([]string{request.ToolName, request.Title, request.Description, request.BlockedPath, string(input)}, "\n"))
	c.queuePermission(id, detail, request.ToolName != "" && request.Input != nil, decide)
}

type claudeTurn struct {
	prefix            string
	message           int
	finished, replied bool
}

func (t *claudeTurn) textID(index int) string {
	return fmt.Sprintf("%s%d-%d", t.prefix, t.message, index)
}

func (t *claudeTurn) apply(c *chat, event claudeEvent) {
	if event.ParentToolID != "" && event.Type != "user" {
		return
	}
	if event.SessionID != "" && event.ParentToolID == "" {
		c.threadID = event.SessionID
	}
	switch event.Type {
	case "system":
		if event.Subtype == "init" && event.Model != "" {
			c.model = event.Model
		}
	case "stream_event":
		if event.Event.Delta.Type == "text_delta" {
			id := t.textID(event.Event.Index)
			if i, ok := c.itemIndex[id]; ok {
				c.items[i].Text += event.Event.Delta.Text
			} else {
				c.appendClaudeMessage(id, "assistant", event.Event.Delta.Text)
			}
		}
	case "assistant":
		c.used = true
		if event.Message.Model != "" && event.Message.Model != "<synthetic>" {
			c.model = event.Message.Model
		}
		for i, part := range event.Message.Content {
			switch part.Type {
			case "text":
				c.appendClaudeMessage(t.textID(i), "assistant", part.Text)
				t.replied = true
			case "tool_use":
				command := part.Name
				var args struct {
					Command  string
					FilePath string `json:"file_path"`
				}
				if json.Unmarshal(part.Input, &args) == nil {
					if args.Command != "" {
						command = args.Command
					} else if args.FilePath != "" {
						command += " " + args.FilePath
					}
				}
				c.saveItem(item{ID: part.ID, Type: "commandExecution", Command: command, Status: "inProgress"})
			}
		}
		t.message++
	case "user":
		for _, part := range event.Message.Content {
			if part.Type != "tool_result" {
				continue
			}
			if i, ok := c.itemIndex[part.ToolUseID]; ok {
				var output string
				if json.Unmarshal(part.Content, &output) != nil {
					var blocks []claudeContent
					if json.Unmarshal(part.Content, &blocks) == nil {
						var lines []string
						for _, b := range blocks {
							if b.Type == "text" {
								lines = append(lines, b.Text)
							}
						}
						output = strings.Join(lines, "\n")
					}
				}
				c.items[i].Output, c.items[i].Status = output, "completed"
				if part.IsError {
					c.items[i].Status = "failed"
				}
			}
		}
	case "result":
		t.finished = true
		c.lastTurnStatus = "completed"
		if event.IsError || strings.HasPrefix(event.Subtype, "error") {
			c.lastTurnStatus = "failed"
			c.notice = event.Result
			if c.notice == "" {
				c.notice = strings.Join(event.Errors, "\n")
			}
			if c.notice == "" {
				c.notice = event.Subtype
			}
		} else {
			c.used = true
			if !t.replied && event.Result != "" {
				c.appendClaudeMessage(t.textID(0), "assistant", event.Result)
			}
			if len(event.Denials) > 0 {
				c.notice = local("A Claude Code tool call was declined.", "Claude Code 工具调用被拒绝。")
			}
		}
		input := event.Usage.Input + event.Usage.CacheRead + event.Usage.CacheWrite
		output := event.Usage.Output
		if n := len(c.chart.samples); n > 0 {
			input += c.chart.samples[n-1].Input
			output += c.chart.samples[n-1].Output
		}
		c.chart.record(core.CodexTokenSample{At: time.Now(), Input: input, Output: output})
	}
}
