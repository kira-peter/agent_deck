package projects

import "github.com/kira-peter/agent_deck/pkg/views"

func init() {
	views.Register(NewProjectsView)
}
