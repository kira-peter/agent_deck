package core

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DeleteSessionFiles removes a session: its .jsonl transcript and the sibling
// directory named after the session id (tool results, subagent logs). The
// project directory and its memory/ folder are left alone.
func DeleteSessionFiles(session Session) error {
	if err := os.Remove(session.FilePath); err != nil {
		return err
	}

	id := strings.TrimSuffix(filepath.Base(session.FilePath), ".jsonl")
	// Only delete a directory that is exactly <id> next to the transcript.
	if id == "" || id == "." || id != filepath.Base(id) || strings.ContainsAny(id, `/\`) {
		return nil
	}
	dir := filepath.Join(filepath.Dir(session.FilePath), id)
	if info, err := os.Lstat(dir); err == nil && info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("transcript deleted, but removing %s failed: %w", dir, err)
		}
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
