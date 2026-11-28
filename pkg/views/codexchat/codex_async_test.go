package codexchat

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestCodexAsyncFormSubmitsFromSplitOverview(t *testing.T) {
	for _, scenario := range []string{"active-turn", "completed-turn", "turn-ends-during-submit"} {
		t.Run(scenario, func(t *testing.T) {
			var sent bytes.Buffer
			c := &chat{
				project: "codex", threadID: "thread", turnID: "turn", ready: true, working: true,
				server: &appServer{writer: json.NewEncoder(&sent)}, requests: map[string]string{}, itemIndex: map[string]int{},
			}
			other := &chat{project: "other", ready: true}
			w := &workspace{tabs: []*chat{other, c}, split: true, overview: true, motionOff: true}
			v := &workspaceView{workspace: w}
			screen := tcell.NewSimulationScreen("UTF-8")
			if err := screen.Init(); err != nil {
				t.Fatal(err)
			}
			defer screen.Fini()
			screen.SetSize(120, 30)
			if err := v.Init(screen); err != nil {
				t.Fatal(err)
			}
			defer v.Cleanup()
			key := func(code tcell.Key) {
				t.Helper()
				if err := v.HandleEvent(tcell.NewEventKey(code, 0, 0)); err != nil {
					t.Fatal(err)
				}
			}
			question := rpcMessage{Method: "item/completed", Params: json.RawMessage(`{
				"threadId":"thread","item":{
					"id":"call_color","type":"agentMessage","delivery":"async","text":"",
					"questions":[{"title":"你喜欢哪个颜色？","options":["蓝色","绿色","橙色"]}]
				}
			}`)}
			complete := rpcMessage{Method: "turn/completed", Params: json.RawMessage(`{"threadId":"thread","turn":{"id":"turn","status":"completed"}}`)}
			c.receive(question)
			c.receive(question)
			if scenario == "completed-turn" {
				c.receive(complete)
			}
			if len(c.prompts) != 1 {
				t.Fatalf("expected one pending question, got %d", len(c.prompts))
			}
			if err := v.Render(); err != nil {
				t.Fatal(err)
			}
			key(tcell.KeyTab)
			if w.active != 1 || w.overview || sent.Len() != 0 {
				t.Fatal("Tab should focus the pending form without submitting")
			}
			key(tcell.KeyDown)
			key(tcell.KeyEnter)
			check := func(method, expectedTurnID string) {
				t.Helper()
				var request struct {
					Method string
					Params struct {
						ThreadID, ExpectedTurnID string
						Input                    []struct{ Text string }
					}
				}
				err := json.NewDecoder(&sent).Decode(&request)
				if err != nil || request.Method != method || request.Params.ThreadID != "thread" || request.Params.ExpectedTurnID != expectedTurnID {
					t.Fatalf("unexpected request %s: %+v (%v)", method, request, err)
				}
				want := codexQuestionReplyStart + `[{"answer":"绿色","question":"你喜欢哪个颜色？","questionItemId":"[\"request_user_input_async\",\"call_color\",0]"}]` + codexQuestionReplyEnd
				if len(request.Params.Input) != 1 || request.Params.Input[0].Text != want {
					t.Fatalf("wrong answer payload: %+v", request.Params.Input)
				}
				if got := formatCodexQuestionReply(request.Params.Input[0].Text); !strings.HasSuffix(got, " → 绿色") {
					t.Fatalf("wrong display text: %s", got)
				}
			}
			if scenario == "completed-turn" {
				check("turn/start", "")
			} else {
				check("turn/steer", "turn")
			}
			if scenario == "turn-ends-during-submit" {
				c.receive(complete)
				errorReply := rpcMessage{ID: json.RawMessage(`1`)}
				errorReply.Error = &struct {
					Message string `json:"message"`
				}{Message: "no active turn"}
				c.receive(errorReply)
				check("turn/start", "")
			}
			if len(c.prompts) != 0 || len(other.draft) != 0 {
				t.Fatal("the answer should close only its own form")
			}
		})
	}
}
