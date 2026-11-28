package codexchat

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

func claudeCommand(ctx context.Context, path string, args []string) (*exec.Cmd, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".cmd" && ext != ".bat" {
		cmd := exec.CommandContext(ctx, path, args...)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		return cmd, nil
	}
	cmd := exec.CommandContext(ctx, "cmd.exe")
	cmd.Env = append(cmd.Environ(), "AGENTDECK_CLAUDE_SHIM="+path)
	var line strings.Builder
	line.WriteString(`cmd.exe /d /v:off /s /c ""%AGENTDECK_CLAUDE_SHIM%"`)
	for i, arg := range args {
		if strings.ContainsAny(arg, "%\"\r\n") {
			return nil, fmt.Errorf("invalid Claude Code argument for Windows launcher: %q", arg)
		}
		key := fmt.Sprintf("AGENTDECK_CLAUDE_ARG_%d", i)
		cmd.Env = append(cmd.Env, key+"="+arg)
		fmt.Fprintf(&line, ` "%%%s%%"`, key)
	}
	line.WriteByte('"')
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CmdLine: line.String()}
	return cmd, nil
}

func serverCommand(path string) *exec.Cmd {
	cmd := exec.Command(path, "app-server", "--stdio")
	if ext := strings.ToLower(filepath.Ext(path)); ext == ".cmd" || ext == ".bat" {
		cmd = exec.Command("cmd.exe")
		cmd.Env = append(cmd.Environ(), "AGENTDECK_CODEX_SHIM="+path)
		// Keep the executable path in the environment and launch the shim with
		// a fixed command line so project paths never need shell quoting.
		cmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow: true,
			CmdLine:    `cmd.exe /d /s /c ""%AGENTDECK_CODEX_SHIM%" app-server --stdio"`,
		}
	} else {
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	}
	return cmd
}
