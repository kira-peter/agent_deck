package main

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
)

func TestExitPopupOnlyOffersWorkspaceWhileRunning(t *testing.T) {
	popup := exitPopup{}
	options := func(running int) int {
		return len(popup.dialog(time.Time{}, 3, running).Buttons)
	}
	if options(0) != options(1)+1 {
		t.Fatal("running tasks should remove the direct exit option")
	}
	if popup.handle("right", 0) != exitPending || popup.handle("enter", 0) != exitConfirm {
		t.Fatal("idle windows should allow direct exit")
	}
	if popup.handle("enter", 1) != exitWorkspace || popup.handle("down", 1) != exitPending || popup.handle("enter", 1) != exitWorkspace {
		t.Fatal("an active task should force the Workspace choice")
	}
	if popup.handle("down", 0) != exitPending || popup.handle("enter", 0) != exitConfirm {
		t.Fatal("direct exit should become available after the task stops")
	}
	if popup.handle("esc", 0) != exitCancel {
		t.Fatal("Escape should leave the current view open")
	}
}

func TestExitPopupPreservesSelectorAroundIt(t *testing.T) {
	const width, height = 80, 15
	line := "\033[32m" + strings.Repeat("a", width) + "\033[0m"
	background := strings.TrimSuffix(strings.Repeat(line+"\n", height), "\n")
	lines := strings.Split(ansi.Strip((exitPopup{}).overlay(background, width, height, 2, 0)), "\n")
	if len(lines) != height || lines[0] != strings.Repeat("a", width) || !strings.HasPrefix(lines[7], strings.Repeat("a", 14)) || !strings.HasSuffix(lines[7], strings.Repeat("a", 14)) {
		t.Fatal("the confirmation should overlay the existing selector")
	}
}
