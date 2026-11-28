package sessions

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/kira-peter/agent_deck/pkg/core"
)

func TestWorkspaceMarksSurviveFilteringBySessionID(t *testing.T) {
	v := &SessionsView{src: core.NewClaudeDir(""), sessions: []core.Session{
		{ID: "first", FirstPrompt: "frontend"}, {ID: "second", FirstPrompt: "backend"},
	}}
	space := tcell.NewEventKey(tcell.KeyRune, ' ', 0)
	v.rebuildDisplay()
	_ = v.handleListEvent(space)
	v.searchQuery = "back"
	v.rebuildDisplay()
	_ = v.handleListEvent(space)
	if len(v.marked) != 2 || !v.marked["first"] || !v.marked["second"] {
		t.Fatal("filtering removed a mark or marked the wrong session")
	}
	_ = v.handleListEvent(space)
	if len(v.marked) != 1 || !v.marked["first"] {
		t.Fatal("Space should toggle only the session at the cursor")
	}
	selected := v.workspaceSelection()
	if len(selected) != 1 || selected[0].ID != "first" {
		t.Fatal("opening Claude sessions should include selections filtered out of view")
	}
	selected = v.workspaceSelection()
	if len(selected) != 1 || selected[0].ID != "second" {
		t.Fatal("F4 with no marks should use the active Claude session")
	}
}
