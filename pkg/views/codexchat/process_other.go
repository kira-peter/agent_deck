//go:build !windows

package codexchat

import (
	"context"
	"os/exec"
)

func claudeCommand(ctx context.Context, path string, args []string) (*exec.Cmd, error) {
	return exec.CommandContext(ctx, path, args...), nil
}

func serverCommand(path string) *exec.Cmd {
	return exec.Command(path, "app-server", "--stdio")
}
