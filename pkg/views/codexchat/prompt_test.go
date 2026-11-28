package codexchat

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestCtrlCInterruptsOnlyFocusedWindow(t *testing.T) {
	var sent, other bytes.Buffer
	front := &chat{threadID: "focused", working: true, server: &appServer{writer: json.NewEncoder(&sent)}, requests: map[string]string{}}
	back := &chat{threadID: "other", turnID: "other-turn", working: true, server: &appServer{writer: json.NewEncoder(&other)}}
	w := &workspace{tabs: []*chat{front, back}, split: true}
	v := &workspaceView{workspace: w}
	if err := v.HandleEvent(tcell.NewEventKey(tcell.KeyCtrlC, 0, 0)); err != nil || !front.interruptPending || sent.Len() != 0 {
		t.Fatalf("early interrupt was not deferred: %v %s", err, &sent)
	}
	front.receive(rpcMessage{Method: "turn/started", Params: json.RawMessage(`{"threadId":"focused","turn":{"id":"first-turn"}}`)})
	if !strings.Contains(sent.String(), `"method":"turn/interrupt"`) || !strings.Contains(sent.String(), `"turnId":"first-turn"`) || other.Len() != 0 {
		t.Fatalf("incorrect interrupt routing: %s %s", &sent, &other)
	}
	w.overview = true
	if err := v.HandleEvent(tcell.NewEventKey(tcell.KeyCtrlC, 0, 0)); err != errQuit {
		t.Fatalf("overview Ctrl+C should request Deck exit: %v", err)
	}
}

func TestCodexQuestionKeyboardSendsAnswers(t *testing.T) {
	var sent bytes.Buffer
	c := &chat{threadID: "thread", server: &appServer{writer: json.NewEncoder(&sent)}}
	c.receive(rpcMessage{ID: json.RawMessage(`8`), Method: "item/tool/requestUserInput", Params: json.RawMessage(`{
		"threadId":"thread","questions":[
			{"id":"language","question":"Language?","options":[{"label":"Go"},{"label":"Rust"}]},
			{"id":"notes","question":"Notes?"}
		]
	}`)})
	for _, key := range []tcell.Key{tcell.KeyDown, tcell.KeyEnter} {
		c.handleKey(tcell.NewEventKey(key, 0, 0))
	}
	c.handleKey(tcell.NewEventKey(tcell.KeyRune, '✓', 0))
	c.handleKey(tcell.NewEventKey(tcell.KeyEnter, 0, 0))
	if len(c.prompts) != 0 || strings.TrimSpace(sent.String()) != `{"id":8,"result":{"answers":{"language":{"answers":["Rust"]},"notes":{"answers":["✓"]}}}}` {
		t.Fatalf("unexpected answer: %s", &sent)
	}
}

func TestClaudeQuestionMultipleSelectionsAndCancellation(t *testing.T) {
	var request claudeEvent
	err := json.Unmarshal([]byte(`{"type":"control_request","request_id":"request","request":{"subtype":"can_use_tool","tool_name":"AskUserQuestion","input":{"questions":[{"question":"Sections?","multiSelect":true,"options":[{"label":"Intro"},{"label":"Summary"}]}]}}}`), &request)
	if err != nil {
		t.Fatal(err)
	}
	var values []string
	send := func(message map[string]any) {
		raw, err := json.Marshal(message)
		if err != nil {
			t.Fatal(err)
		}
		values = append(values, string(raw))
	}
	c := &chat{}
	c.queueClaudePrompt(request, send)
	c.handleKey(tcell.NewEventKey(tcell.KeyRune, ' ', 0))
	c.handleKey(tcell.NewEventKey(tcell.KeyDown, 0, 0))
	c.handleKey(tcell.NewEventKey(tcell.KeyRune, ' ', 0))
	c.handleKey(tcell.NewEventKey(tcell.KeyEnter, 0, 0))
	c.queueClaudePrompt(request, send)
	c.handleKey(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	if len(c.prompts) != 0 || len(values) != 2 || !strings.Contains(values[0], `"Sections?":"Intro, Summary"`) || !strings.Contains(values[1], `"behavior":"deny"`) {
		t.Fatalf("unexpected Claude replies: %v", values)
	}
}
