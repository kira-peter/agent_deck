package codexchat

import (
	"encoding/json"
	"testing"

	"github.com/kira-peter/agent_deck/pkg/core"
)

func TestClaudeStreamUpdatesExistingReplyAndTracksToolResult(t *testing.T) {
	c := &chat{kind: core.SourceClaude, itemIndex: map[string]int{}}
	turn := claudeTurn{prefix: "test-"}
	for _, line := range []string{
		`{"type":"stream_event","event":{"index":0,"delta":{"type":"text_delta","text":"hel"}}}`,
		`{"type":"assistant","session_id":"session","message":{"model":"claude-sonnet","content":[{"type":"text","text":"hello"},{"type":"tool_use","id":"tool-1","name":"Read","input":{"file_path":"README.md"}}]}}`,
		`{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"tool-1","content":"contents"}]}}`,
		`{"type":"result","subtype":"success","session_id":"session","result":"hello","usage":{"input_tokens":100,"cache_read_input_tokens":20,"output_tokens":12}}`,
	} {
		var event claudeEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatal(err)
		}
		turn.apply(c, event)
	}
	if len(c.items) != 2 || c.items[0].Text != "hello" || c.items[1].Command != "Read README.md" || c.items[1].Output != "contents" || c.items[1].Status != "completed" {
		t.Fatalf("unexpected transcript: %+v", c.items)
	}
	if !c.used || c.threadID != "session" || c.lastTurnStatus != "completed" || len(c.chart.samples) != 1 || c.chart.samples[0].Input != 120 || c.chart.samples[0].Output != 12 {
		t.Fatalf("unexpected Claude session or usage: %s %s %+v", c.threadID, c.lastTurnStatus, c.chart.samples)
	}
}
