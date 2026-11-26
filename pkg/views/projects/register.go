package projects

import "github.com/kira-peter/agentdeck/pkg/views"

func init() {
	views.Register(NewProjectsView)
}
