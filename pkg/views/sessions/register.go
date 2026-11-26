package sessions

import "github.com/kira-peter/agent_deck/pkg/views"

func init() {
	views.Register(NewSessionsView)
}
