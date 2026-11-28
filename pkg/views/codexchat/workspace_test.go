package codexchat

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/kira-peter/agent_deck/pkg/core"
)

func TestF1OpenSessionRoutesToFocusedHomeWithoutStoppingWork(t *testing.T) {
	left := &chat{home: "first-home", working: true}
	right := &chat{home: "second-home", kind: core.SourceClaude, working: true}
	w := &workspace{tabs: []*chat{left, right}, active: 1, split: true}
	v := &workspaceView{workspace: w}
	for _, key := range []tcell.Key{tcell.KeyF1, tcell.KeyDown, tcell.KeyDown} {
		if err := v.HandleEvent(tcell.NewEventKey(key, 0, 0)); err != nil {
			t.Fatal(err)
		}
	}
	var request SessionPageRequest
	err := v.HandleEvent(tcell.NewEventKey(tcell.KeyEnter, 0, 0))
	if !errors.As(err, &request) || request.Home != right.home || request.Kind != core.SourceClaude {
		t.Fatalf("expected Claude session page for the focused home, got %v (%+v)", err, request)
	}
	if v.menu.kind != menuNone || len(w.tabs) != 2 || !left.working || !right.working {
		t.Fatal("navigation should close the menu and leave both running panes alive")
	}
}

func TestWorkspaceRoutesInputAndKeepsDetachedUpdates(t *testing.T) {
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	defer screen.Fini()
	screen.SetSize(120, 30)
	first := &chat{project: "frontend", home: "codex-home", threadID: "front", ready: true, working: true, itemIndex: make(map[string]int)}
	second := &chat{project: "backend", threadID: "back", ready: true, working: true, itemIndex: make(map[string]int)}
	w := &workspace{tabs: []*chat{first, second}}
	v := &workspaceView{workspace: w}
	if err := v.Init(screen); err != nil {
		t.Fatal(err)
	}
	defer v.Cleanup()
	key := func(code tcell.Key, char rune) {
		t.Helper()
		if err := v.HandleEvent(tcell.NewEventKey(code, char, 0)); err != nil {
			t.Fatal(err)
		}
	}
	key(tcell.KeyRune, 'a')
	key(tcell.KeyF5, 0)
	key(tcell.KeyRune, 'b')
	if string(first.draft) != "a" || string(second.draft) != "b" {
		t.Fatalf("drafts leaked across tabs: %q %q", first.draft, second.draft)
	}
	if err := v.HandleEvent(tcell.NewEventKey(tcell.KeyEscape, 0, 0)); err != errBack {
		t.Fatalf("Escape should leave the workspace: %v", err)
	}
	v.Cleanup()
	first.applyProcessEvent(processEvent{message: rpcMessage{Method: "item/completed", Params: json.RawMessage(`{
		"threadId":"front","item":{"id":"1","type":"agentMessage","text":"Frontend finished while hidden"}
	}`)}})
	first.applyProcessEvent(processEvent{message: rpcMessage{Method: "turn/completed", Params: json.RawMessage(`{
		"threadId":"front","turn":{"status":"completed"}
	}`)}})
	second.applyProcessEvent(processEvent{message: rpcMessage{Method: "item/started", Params: json.RawMessage(`{
		"threadId":"back","item":{"id":"2","type":"commandExecution","command":"go test ./...","status":"inProgress"}
	}`)}})
	w.openFor(core.SourceCodex, "frontend", "codex-home", &core.Session{ID: "front"})
	if len(w.tabs) != 2 || w.active != 0 {
		t.Fatal("resuming an open thread should focus its existing tab")
	}
	if err := v.Init(screen); err != nil {
		t.Fatal(err)
	}
	key(tcell.KeyF6, 0)
	if err := v.Render(); err != nil {
		t.Fatal(err)
	}
	cells, _, _ := screen.GetContents()
	var rendered strings.Builder
	for _, cell := range cells {
		rendered.WriteString(string(cell.Runes))
	}
	for _, want := range []string{"Split 2/2", "Frontend finished while hidden", "go test ./...", "Done", "Running"} {
		if !strings.Contains(rendered.String(), want) {
			t.Errorf("split is missing %q", want)
		}
	}
	key(tcell.KeyF5, 0)
	key(tcell.KeyF7, 0)
	key(tcell.KeyF7, 0)
	if len(w.tabs) != 2 || v.message == "" || second.closed {
		t.Fatal("a running session should stay open")
	}
	second.working = false
	key(tcell.KeyF7, 0)
	if len(w.tabs) != 1 || w.tabs[0] != first || !second.closed || first.closed {
		t.Fatal("closing a pane should affect only the selected session")
	}
}

func TestWorkspaceMenuStagesModelAndExitFocusesActiveWork(t *testing.T) {
	front := &chat{project: "frontend", model: "model-a", ready: true, models: []string{"model-a", "model-b"}}
	back := &chat{project: "backend", model: "model-a", ready: true, working: true}
	w := &workspace{tabs: []*chat{front, back}}
	v := &workspaceView{workspace: w}
	key := func(code tcell.Key) error { return v.HandleEvent(tcell.NewEventKey(code, 0, 0)) }
	for _, code := range []tcell.Key{tcell.KeyF1, tcell.KeyDown, tcell.KeyEnter, tcell.KeyEnter, tcell.KeyDown, tcell.KeyDown, tcell.KeyEnter} {
		if err := key(code); err != nil {
			t.Fatal(err)
		}
	}
	if front.desiredModel != "model-b" || front.model != "model-a" || back.desiredModel != "" || v.menu.kind != menuRestart || v.previewWindow() != 0 {
		t.Fatal("model should be staged for only the chosen tab, and preview its restart")
	}
	_ = key(tcell.KeyDown)
	if v.previewWindow() != 1 || w.active != 0 || v.menu.flashAt.IsZero() {
		t.Fatal("hover should preview the second tab without switching focus")
	}
	_ = key(tcell.KeyEnter)
	if w.tabs[1] != back || back.closed || v.menu.notice == "" {
		t.Fatal("active windows must block restart")
	}
	if open, running := w.exitStatus(); open != 2 || running != 1 {
		t.Fatalf("exit status = %d open, %d running", open, running)
	}
	w.overview = true
	w.focusForExit()
	if w.active != 1 || w.overview || back.closed || !back.working {
		t.Fatal("jumping from the exit popup should focus the running window without affecting its work")
	}
	v.closeMenu()
	back.working = false
	if open, running := w.exitStatus(); open != 2 || running != 0 {
		t.Fatalf("exit status = %d open, %d running", open, running)
	}
}

func TestCloseAllAndQuitWaitsForEveryWindow(t *testing.T) {
	first := &chat{project: "first", ready: true}
	second := &chat{project: "second", ready: true, working: true}
	w := &workspace{tabs: []*chat{first, second}, split: true}
	v := &workspaceView{workspace: w}
	key := func(code tcell.Key) error { return v.HandleEvent(tcell.NewEventKey(code, 0, 0)) }
	advance := func() error { return v.HandleEvent(tcell.NewEventInterrupt(workspaceUpdate{owner: w})) }
	_ = key(tcell.KeyF1)
	_ = key(tcell.KeyUp)
	if !v.rootMenuRows()[rootCloseAll].disabled || key(tcell.KeyEnter) != nil || v.menu.notice != closeAllBlocked() || first.closed || second.closed {
		t.Fatal("a running window should block the action before any window is closed")
	}
	second.working = false
	if err := key(tcell.KeyEnter); err != nil || !first.closed || second.closed || len(w.tabs) != 1 {
		t.Fatalf("only the focused window should close first: %v", err)
	}
	if err := advance(); err != nil || v.motion == nil || !v.motion.closing || second.closed {
		t.Fatalf("closing should wait for the first window's animation: %v", err)
	}
	v.motion.started = time.Now().Add(-paneReveal - paneReflow)
	backendDone := make(chan struct{})
	v.closingWindow = backendDone
	if err := advance(); err != nil || second.closed {
		t.Fatalf("closing should wait for the first backend to stop: %v", err)
	}
	close(backendDone)
	if err := advance(); err != nil || !second.closed || len(w.tabs) != 0 {
		t.Fatalf("the remaining window should close next: %v", err)
	}
	if err := advance(); err != nil || v.motion == nil || !v.motion.closing {
		t.Fatalf("the final window should still animate before quit: %v", err)
	}
	v.motion.started = time.Now().Add(-paneReveal)
	if err := advance(); err != errQuit {
		t.Fatalf("expected to quit after the final close: %v", err)
	}
}

func TestSplitFitsPanesAndFollowsFocus(t *testing.T) {
	for _, tc := range []struct {
		area                      paneBounds
		count, focus, want, start int
	}{
		{paneBounds{width: 100, height: 21}, 5, 3, 2, 2},
		{paneBounds{width: 60, height: 29}, 2, 1, 2, 0},
		{paneBounds{width: 120, height: 30}, 4, 2, 4, 0},
		{paneBounds{width: 36, height: 12}, 3, 2, 1, 2},
	} {
		panes, start := arrangePanes(tc.area, tc.count, tc.focus, true)
		if len(panes) != tc.want || start != tc.start {
			t.Fatalf("wrong page for %+v: panes=%+v, start=%d", tc, panes, start)
		}
		for _, pane := range panes {
			if pane.width < minPaneWidth || pane.height < minPaneHeight || pane.x+pane.width > tc.area.width || pane.y+pane.height > tc.area.height {
				t.Errorf("pane outside layout bounds: %+v in %+v", pane, tc.area)
			}
		}
	}
}

func TestOverviewShowsEveryPaneAtFullStrength(t *testing.T) {
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	defer screen.Fini()
	screen.SetSize(120, 30)
	a := &chat{project: "first", ready: true}
	b := &chat{project: "second", ready: true}
	w := &workspace{tabs: []*chat{a, b}, split: true, motionOff: true}
	v := &workspaceView{workspace: w}
	if err := v.Init(screen); err != nil {
		t.Fatal(err)
	}
	defer v.Cleanup()
	read := func() (left, right rune, leftFG, rightFG tcell.Color) {
		t.Helper()
		if err := v.Render(); err != nil {
			t.Fatal(err)
		}
		panes, _ := arrangePanes(paneBounds{y: workspaceTop, width: 120, height: 26}, 2, 0, true)
		for i, p := range panes {
			r, _, _, _ := screen.GetContent(p.x, p.y+p.height-1)
			_, _, s, _ := screen.GetContent(p.x+3, p.y+2)
			fg, _, _ := s.Decompose()
			if i == 0 {
				left, leftFG = r, fg
			} else {
				right, rightFG = r, fg
			}
		}
		return
	}
	if l, r, lf, rf := read(); l != '┗' || r != '└' || lf == rf {
		t.Fatal("focus should frame one pane and dim the other")
	}
	_ = v.HandleEvent(tcell.NewEventKey(tcell.KeyF8, 0, 0))
	_ = v.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'x', 0))
	if l, r, lf, rf := read(); !w.overview || l != '└' || r != '└' || lf != rf || len(a.draft) != 0 {
		t.Fatal("overview should show both panes at full intensity without routing input")
	}
	_ = v.HandleEvent(tcell.NewEventKey(tcell.KeyF5, 0, 0))
	if l, r, lf, rf := read(); w.overview || l != '└' || r != '┗' || lf == rf {
		t.Fatal("switching should focus the next pane")
	}
}

func TestOpenSessionsReusesTabsAndFocusesFirstSelection(t *testing.T) {
	first := &chat{project: "front", home: "codex", threadID: "one"}
	second := &chat{project: "back", home: "codex", threadID: "two"}
	w := &workspace{tabs: []*chat{first, second}, overview: true}
	w.openSessions(core.SourceCodex, "codex", []core.Session{{ID: "two"}, {ID: "one"}, {ID: "two"}})
	if len(w.tabs) != 2 || w.tabs[0] != first || w.tabs[1] != second || w.active != 1 || !w.split || w.overview {
		t.Fatal("batch open should reuse tabs, focus the first selected session and open split view")
	}
}

func TestThemePreviewUsesFanAndCancelsWithoutSaving(t *testing.T) {
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	defer screen.Fini()
	screen.SetSize(112, 26)
	v := &workspaceView{workspace: &workspace{tabs: []*chat{{project: "a"}, {project: "b"}}, active: 1, split: true, motionOff: true}}
	if err := v.Init(screen); err != nil {
		t.Fatal(err)
	}
	defer v.Cleanup()
	configured := core.GetConfig().GetDisplay().WorkspaceTheme
	original := currentTheme
	key := func(k tcell.Key) {
		t.Helper()
		if err := v.HandleEvent(tcell.NewEventKey(k, 0, 0)); err != nil {
			t.Fatal(err)
		}
	}
	key(tcell.KeyF1)
	key(tcell.KeyEnter)
	key(tcell.KeyDown)
	if v.menu.kind != menuTheme || len(v.fanTrail) != 1 || currentTheme == original || core.GetConfig().GetDisplay().WorkspaceTheme != configured {
		t.Fatal("theme should only preview within the same menu until Enter")
	}
	if err := v.Render(); err != nil {
		t.Fatal(err)
	}
	cells, _, _ := screen.GetContents()
	var joined strings.Builder
	for _, cell := range cells {
		joined.WriteString(string(cell.Runes))
	}
	if !strings.Contains(joined.String(), "Theme") || !strings.Contains(joined.String(), themeName(currentTheme)) {
		t.Fatal("parent and child options did not render together")
	}
	key(tcell.KeyRight)
	if v.menu.kind != menuTheme {
		t.Fatal("only Enter should confirm a theme")
	}
	key(tcell.KeyEscape)
	if v.menu.kind != menuRoot || currentTheme != original || core.GetConfig().GetDisplay().WorkspaceTheme != configured {
		t.Fatal("return should restore the original theme without saving")
	}
}

func TestPaneMotionReflowsBeforeOpeningAndAfterClosing(t *testing.T) {
	a := &chat{project: "first"}
	b := &chat{project: "second"}
	w := &workspace{tabs: []*chat{a}, split: true}
	v := &workspaceView{workspace: w}
	area := paneBounds{width: 124, height: 28, y: 3}
	w.rememberLayout()
	w.tabs = append(w.tabs, b)
	w.active = 1
	v.startPaneMotion()
	v.motion.started = time.Now().Add(-paneReflow / 2)
	frames, _ := v.paneFrames(area, w.active)
	if len(frames) != 1 || frames[0].chat != a || frames[0].bounds.width >= area.width || frames[0].bounds.width <= 61 {
		t.Fatalf("existing pane should shrink first: %+v", frames)
	}
	v.motion.started = time.Now().Add(-paneReflow - paneReveal/2)
	frames, _ = v.paneFrames(area, w.active)
	if len(frames) != 2 || frames[0].reveal || !frames[1].reveal || frames[1].openness <= 0 || frames[1].openness >= 1 {
		t.Fatalf("only the added pane should open: %+v", frames)
	}
	v.closeSelectedWindow(1)
	v.startPaneMotion()
	v.motion.started = time.Now().Add(-paneReveal / 2)
	frames, _ = v.paneFrames(area, w.active)
	if len(w.tabs) != 1 || len(frames) != 2 || frames[0].reveal || !frames[1].reveal || !b.closed {
		t.Fatalf("the closing pane should collapse before reflow: %+v", frames)
	}
	v.motion.started = time.Now().Add(-paneReveal - paneReflow/2)
	frames, _ = v.paneFrames(area, w.active)
	if len(frames) != 1 || frames[0].bounds.width <= 61 || frames[0].bounds.width >= area.width {
		t.Fatalf("the surviving pane should expand after closing: %+v", frames)
	}
}

func TestWindowFanUsesFocusedPaneAndScheduledFrames(t *testing.T) {
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	defer screen.Fini()
	screen.SetSize(124, 30)
	for screen.HasPendingEvent() {
		screen.PollEvent()
	}
	w := &workspace{tabs: []*chat{{project: "first"}, {project: "second"}}, active: 1, split: true}
	v := &workspaceView{workspace: w}
	if err := v.Init(screen); err != nil {
		t.Fatal(err)
	}
	defer v.Cleanup()
	_ = v.HandleEvent(tcell.NewEventKey(tcell.KeyF1, 0, 0))
	area := v.fanArea(124, 30)
	if v.menu.kind != menuRoot || area.X < 60 || area.X+area.Width > 124 {
		t.Fatalf("F1 is anchored to the wrong pane: %+v", area)
	}
	if err := v.Render(); err != nil {
		t.Fatal(err)
	}
	if v.frameTimer == nil {
		t.Fatal("opening F1 did not schedule a redraw")
	}
	deadline := time.Now().Add(time.Second)
	for !screen.HasPendingEvent() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if !screen.HasPendingEvent() {
		t.Fatal("the animation timer did not post a redraw event")
	}
	if _, ok := screen.PollEvent().(*tcell.EventInterrupt); !ok {
		t.Fatal("expected a redraw interrupt")
	}
	v.menu.openedAt = time.Now().Add(-400 * time.Millisecond)
	if err := v.Render(); err != nil {
		t.Fatal(err)
	}
	y := area.Y + area.Height/2
	if got, _, _, _ := screen.GetContent(area.X+7, y); got != '─' {
		t.Fatalf("no connector at the right pane: %q", got)
	}
	if got, _, _, _ := screen.GetContent(7, y); got == '─' {
		t.Fatal("F1 should not draw a second connector from the screen edge")
	}
}

func TestF2RevealsEarlierToolOutputInOnlyTheFocusedWindow(t *testing.T) {
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	defer screen.Fini()
	screen.SetSize(124, 30)
	left := &chat{project: "first", ready: true}
	right := &chat{project: "second", ready: true, items: []item{
		{Type: "commandExecution", Command: "go test ./...", Output: "tool output visible here"},
		{Type: "agentMessage", Text: strings.Repeat("A newer response line\n", 40)},
	}}
	v := &workspaceView{workspace: &workspace{tabs: []*chat{left, right}, active: 1, split: true, motionOff: true}}
	if err := v.Init(screen); err != nil {
		t.Fatal(err)
	}
	defer v.Cleanup()
	if err := v.HandleEvent(tcell.NewEventKey(tcell.KeyF2, 0, 0)); err != nil {
		t.Fatal(err)
	}
	if err := v.Render(); err != nil {
		t.Fatal(err)
	}
	cells, _, _ := screen.GetContents()
	var output strings.Builder
	for _, cell := range cells {
		output.WriteString(string(cell.Runes))
	}
	if left.showTools || !right.showTools || right.scroll == 0 || !strings.Contains(output.String(), "tool output visible here") {
		t.Fatal("F2 did not show the earlier output in the focused pane")
	}
	_ = v.HandleEvent(tcell.NewEventKey(tcell.KeyF2, 0, 0))
	if right.showTools || right.scroll != 0 {
		t.Fatal("F2 should close tool details and return to the newest message")
	}
}

func TestWorkspaceMotionSettlesAndCanBeDisabled(t *testing.T) {
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	defer screen.Fini()
	screen.SetSize(80, 26)
	c := &chat{project: "frontend", ready: true}
	w := &workspace{tabs: []*chat{c}}
	v := &workspaceView{workspace: w}
	if err := v.Init(screen); err != nil {
		t.Fatal(err)
	}
	defer v.Cleanup()
	pane := &paneScreen{Screen: screen, paneBounds: paneBounds{width: 80, height: 22, y: workspaceTop}, focused: true}
	v.drawPaneRule(pane, focusSweep.Duration)
	_, _, settled, _ := pane.GetContent(40, 0)
	v.drawPaneRule(pane, focusSweep.Duration/2)
	_, _, moving, _ := pane.GetContent(40, 0)
	if moving == settled {
		t.Fatal("the active rule should brighten as the band passes")
	}
	v.scheduleFrame(0)
	if v.frameTimer == nil {
		t.Fatal("an active sweep should schedule a frame")
	}
	if err := v.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'm', tcell.ModAlt)); err != nil {
		t.Fatal(err)
	}
	if err := v.Render(); err != nil {
		t.Fatal(err)
	}
	_, _, disabled, _ := pane.GetContent(40, 0)
	if disabled != settled || v.frameTimer != nil || v.elapsed() != time.Duration(0) || len(c.draft) != 0 {
		t.Fatal("Alt+M should settle the sweep, stop frame updates, and leave draft input alone")
	}
}
