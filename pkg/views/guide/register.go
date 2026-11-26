package guide

import "github.com/kira-peter/agentdeck/pkg/views"

func init() {
	views.Register(NewGuideView)
}
