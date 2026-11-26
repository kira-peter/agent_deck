package core

import (
	"os"
	"strings"
)

// Build metadata. Override at link time, e.g.
//
//	go build -ldflags "-X github.com/kira-peter/agentdeck/pkg/core.BuildTime=2026-10-27T16:40"
var (
	BuildTime = "" // set by build scripts / link flags
)

// VersionLabel returns a short version string for display, e.g. "2026-10-27 16:40".
// When BuildTime was not injected at link time it falls back to the modification
// time of the running executable, which still tells you which build you have.
func VersionLabel() string {
	if BuildTime != "" {
		// Link flags cannot carry spaces; build scripts use "2006-01-02T15:04".
		return strings.Replace(BuildTime, "T", " ", 1)
	}

	exe, err := os.Executable()
	if err != nil {
		return "dev"
	}
	info, err := os.Stat(exe)
	if err != nil {
		return "dev"
	}
	return info.ModTime().Format("2006-01-02 15:04")
}
