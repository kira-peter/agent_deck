package sessions

import "github.com/kira-peter/agentdeck/pkg/views"

func init() {
	views.Register(NewSessionsView)
}
