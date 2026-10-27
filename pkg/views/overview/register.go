package overview

import "github.com/yourusername/claude-manager/pkg/views"

func init() {
	views.Register(NewOverviewView)
}
