package core

import (
	"os"
	"os/exec"
	"runtime"
)

// OpenPath opens a file with its default application, or a directory in the
// system file manager, without waiting for it to exit.
func OpenPath(path string) error {
	if _, err := os.Stat(path); err != nil {
		return err
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", path)
	case "darwin":
		cmd = exec.Command("open", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait()
	return nil
}
