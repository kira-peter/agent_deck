package codexchat

import (
	"errors"
	"slices"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/kira-peter/agent_deck/pkg/core"
	"github.com/kira-peter/agent_deck/pkg/views"
)

var liveWorkspace = workspace{split: true}

var errBack = errors.New("back")
var errQuit = errors.New("quit")

const ViewID = "workspace"

type workspace struct {
	// The terminal event loop owns tabs, focus, and split mode. Each session
	// owns its protocol state and keeps receiving events between views.
	tabs      []*chat
	active    int
	split     bool
	overview  bool
	motionOff bool
	before    *paneLayout
	retiring  sync.WaitGroup

	wakeMu sync.Mutex
	screen tcell.Screen
	queued bool
}

func init() {
	views.Register(func() views.View { return &workspaceView{workspace: &liveWorkspace} })
}

// Open opens or focuses an agent session, then shows the workspace.
func Open(screen tcell.Screen, project string, source core.Source, session *core.Session) error {
	liveWorkspace.openFor(source.Kind(), project, source.Dir(), session)
	return Show(screen)
}

// OpenSessions adds a batch of existing sessions and presents them together.
func OpenSessions(screen tcell.Screen, source core.Source, sessions []core.Session) error {
	liveWorkspace.openSessions(source.Kind(), source.Dir(), sessions)
	return Show(screen)
}

func (w *workspace) openSessions(kind core.SourceKind, home string, sessions []core.Session) {
	first := -1
	for i := range sessions {
		session := &sessions[i]
		if session.ID == "" {
			continue
		}
		w.openFor(kind, session.ProjectPath, home, session)
		if first < 0 {
			first = w.active
		}
	}
	if first >= 0 {
		w.active = first
	}
	if len(sessions) > 1 && len(w.tabs) > 1 {
		w.split = true
	}
}

// Show temporarily presents the workspace on a Projects view's screen.
func Show(screen tcell.Screen) error {
	v := &workspaceView{workspace: &liveWorkspace}
	_ = v.Init(screen)
	defer v.Cleanup()
	for {
		_ = v.Render()
		if err := v.HandleEvent(screen.PollEvent()); err != nil {
			if err == errBack {
				return nil
			}
			return err
		}
	}
}

// HasSessions reports whether Deck is hosting any sessions.
func HasSessions() bool { return len(liveWorkspace.tabs) > 0 }

// ExitStatus reports the open windows and how many have an active turn.
func ExitStatus() (open, running int) { return liveWorkspace.exitStatus() }

func (w *workspace) exitStatus() (open, running int) {
	for _, c := range w.tabs {
		c.mu.Lock()
		if c.working {
			running++
		}
		c.mu.Unlock()
	}
	return len(w.tabs), running
}

// FocusForExit shows the active work first when the user jumps to Workspace.
func FocusForExit() { liveWorkspace.focusForExit() }

// MotionEnabled reports the Workspace animation preference.
func MotionEnabled() bool { return !liveWorkspace.motionOff }

func (w *workspace) focusForExit() {
	w.overview = false
	for offset := range w.tabs {
		index := (w.active + offset) % len(w.tabs)
		c := w.tabs[index]
		c.mu.Lock()
		working := c.working
		c.mu.Unlock()
		if working {
			w.active = index
			return
		}
	}
}

// Shutdown stops app-server processes on Deck exit.
func Shutdown() {
	var closing sync.WaitGroup
	for _, c := range liveWorkspace.tabs {
		closing.Go(c.close)
	}
	closing.Wait()
	liveWorkspace.retiring.Wait()
}

func (w *workspace) openFor(kind core.SourceKind, project, home string, session *core.Session) {
	w.overview = false
	if session != nil && session.ID != "" {
		for i, c := range w.tabs {
			c.mu.Lock()
			matches := c.agentKind() == kind && c.home == home && (c.threadID == session.ID || c.sessionID == session.ID)
			c.mu.Unlock()
			if matches {
				w.active = i
				return
			}
		}
	}
	c := &chat{
		kind: kind, project: project, home: home, connecting: true,
		itemIndex: make(map[string]int), requests: make(map[string]string),
	}
	var historyPath string
	if session != nil {
		c.sessionID, historyPath = session.ID, session.FilePath
	}
	w.rememberLayout()
	w.tabs = append(w.tabs, c)
	w.active = len(w.tabs) - 1
	go c.connect(historyPath, w.wake)
}

func (c *chat) connect(historyPath string, wake func()) {
	if c.agentKind() == core.SourceClaude {
		c.connectClaude(historyPath, wake)
		return
	}
	var samples []core.CodexTokenSample
	var historyErr error
	if historyPath != "" {
		samples, historyErr = core.ReadCodexTokenSamples(historyPath)
	}
	c.mu.Lock()
	defer wake()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	for _, sample := range samples {
		c.chart.record(sample)
	}
	if historyErr != nil {
		c.notice = local("Token history: ", "Token 历史：") + historyErr.Error()
	}
	server, err := startServer(c.project, c.home, func(event processEvent) {
		c.applyProcessEvent(event)
		wake()
	})
	if err != nil {
		c.notice = err.Error()
		c.connecting = false
		return
	}
	c.server = server
	c.request("initialize", map[string]any{
		"clientInfo":   map[string]string{"name": "agentdeck", "title": "AgentDeck", "version": "0.1"},
		"capabilities": map[string]bool{"experimentalApi": true},
	})
}

func (c *chat) applyProcessEvent(event processEvent) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	if event.err != nil {
		c.notice, c.ready, c.working = event.err.Error(), false, false
		c.connecting = false
		c.interruptPending = false
		c.prompts = nil
		return
	}
	c.receive(event.message)
}

func (c *chat) close() {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	server := c.server
	stopClaude := c.stopClaude
	c.mu.Unlock()
	if stopClaude != nil {
		stopClaude()
	}
	if server != nil {
		server.close()
	}
}

type workspaceUpdate struct{ owner *workspace }

func (w *workspace) attach(screen tcell.Screen) {
	w.wakeMu.Lock()
	defer w.wakeMu.Unlock()
	w.screen, w.queued = screen, false
}

func (w *workspace) wake() {
	w.wakeMu.Lock()
	defer w.wakeMu.Unlock()
	if w.screen != nil && !w.queued {
		w.queued = w.screen.PostEvent(tcell.NewEventInterrupt(workspaceUpdate{owner: w})) == nil
	}
}

type workspaceView struct {
	workspace          *workspace
	screen             tcell.Screen
	message            string
	menu               workspaceMenu
	focusSince         time.Time
	frameTimer         *time.Timer
	fanClosedAt        time.Time
	fanClosedSelection int
	fanPane            int
	fanTrail           []fanParent
	themeBefore        string
	motion             *paneMotion
	closingAll         bool
	closingWindow      <-chan struct{}
}

func (v *workspaceView) Metadata() views.Metadata {
	return views.Metadata{
		ID: ViewID, NameEn: "Workspace", NameZh: "工作区",
		DescEn: "Switch between Codex and Claude Code sessions or view them side by side",
		DescZh: "切换 Codex 和 Claude Code 会话或分屏查看",
	}
}

func (v *workspaceView) Init(screen tcell.Screen) error {
	v.screen = screen
	v.focusSince = time.Now()
	applyTheme(core.GetConfig().GetDisplay().WorkspaceTheme)
	v.workspace.attach(screen)
	return nil
}

func (v *workspaceView) Cleanup() error {
	v.restoreTheme()
	v.stopFrame()
	v.workspace.attach(nil)
	v.screen.HideCursor()
	return nil
}

func (v *workspaceView) HandleEvent(event tcell.Event) error {
	w := v.workspace
	switch event := event.(type) {
	case nil:
		return errBack
	case *tcell.EventResize:
		v.screen.Sync()
	case *tcell.EventInterrupt:
		if update, ok := event.Data().(workspaceUpdate); ok && update.owner == w {
			w.wakeMu.Lock()
			w.queued = false
			w.wakeMu.Unlock()
			if v.closingAll {
				return v.advanceCloseAll()
			}
		}
	case *tcell.EventKey:
		if v.closingAll {
			return nil
		}
		if event.Key() == tcell.KeyCtrlC {
			if v.menu.kind != menuNone {
				v.closeMenu()
				return nil
			}
			if len(w.tabs) > 0 && !w.overview {
				c := w.tabs[w.active]
				c.mu.Lock()
				if !c.interrupt() && len(c.draft) > 0 {
					c.draft, c.cursor = nil, 0
				}
				c.mu.Unlock()
				return nil
			}
			return errQuit
		}
		if event.Key() == tcell.KeyRune && (event.Rune() == 'm' || event.Rune() == 'M') && event.Modifiers()&tcell.ModAlt != 0 {
			w.motionOff = !w.motionOff
			v.focusSince = time.Now()
			return nil
		}
		if v.menu.kind != menuNone {
			return v.handleMenu(event)
		}
		if event.Key() == tcell.KeyEscape {
			if len(w.tabs) > 0 && !w.overview {
				c := w.tabs[w.active]
				c.mu.Lock()
				if len(c.prompts) > 0 {
					c.handlePrompt(event)
					c.mu.Unlock()
					return nil
				}
				c.mu.Unlock()
			}
			return errBack
		}
		v.message = ""
		if event.Key() == tcell.KeyF1 {
			v.openMenu(menuRoot, 0)
			return nil
		}
		if len(w.tabs) == 0 {
			return nil
		}
		if event.Key() == tcell.KeyTab || (w.overview && event.Key() == tcell.KeyEnter) {
			for offset := 1; offset <= len(w.tabs); offset++ {
				index := (w.active + offset) % len(w.tabs)
				c := w.tabs[index]
				c.mu.Lock()
				waiting := len(c.prompts) != 0
				c.mu.Unlock()
				if waiting {
					w.active, w.overview = index, false
					v.focusSince = time.Now()
					return nil
				}
			}
		}
		c := w.tabs[w.active]
		switch event.Key() {
		case tcell.KeyF2:
			if w.overview {
				v.message = local("F4/F5 focuses a window before viewing its tools.", "按 F4/F5 选中一个窗口，再查看它的工具详情。")
				return nil
			}
			c.mu.Lock()
			v.message = c.toggleToolDetails()
			c.mu.Unlock()
		case tcell.KeyF4:
			w.active = (w.active + len(w.tabs) - 1) % len(w.tabs)
			w.overview = false
			v.focusSince = time.Now()
		case tcell.KeyF5:
			w.active = (w.active + 1) % len(w.tabs)
			w.overview = false
			v.focusSince = time.Now()
		case tcell.KeyF6:
			w.split = !w.split
			w.overview = false
			v.focusSince = time.Now()
		case tcell.KeyF7:
			if w.overview {
				return nil
			}
			v.closeSelectedWindow(w.active)
		case tcell.KeyF8:
			w.split, w.overview = true, true
		default:
			if w.overview {
				return nil
			}
			c.mu.Lock()
			c.handleKey(event)
			c.mu.Unlock()
		}
	}
	return nil
}

func closeAllBlocked() string {
	return local("Stop all running tasks before closing the windows.", "请先停止所有正在运行的任务。")
}

func (v *workspaceView) advanceCloseAll() error {
	w := v.workspace
	v.startPaneMotion()
	if v.motion != nil {
		return nil
	}
	if v.closingWindow != nil {
		select {
		case <-v.closingWindow:
			v.closingWindow = nil
		default:
			return nil
		}
	}
	open, running := w.exitStatus()
	if open == 0 {
		return errQuit
	}
	if running > 0 || !v.closeSelectedWindow(w.active) {
		v.closingAll = false
		v.message = closeAllBlocked()
		return nil
	}
	w.wake()
	return nil
}

func (v *workspaceView) closeSelectedWindow(index int) bool {
	w := v.workspace
	c := w.tabs[index]
	c.mu.Lock()
	if c.working || !v.closingAll && len(c.prompts) > 0 {
		c.mu.Unlock()
		message := local("This window is working and cannot be closed.", "该窗口正在工作，无法关闭。")
		if v.menu.kind != menuNone {
			v.menu.notice = message
		} else {
			v.message = message
		}
		return false
	}
	c.closed = true
	server := c.server
	c.mu.Unlock()
	w.active = index
	w.rememberLayout()
	w.tabs = slices.Delete(w.tabs, index, index+1)
	w.active = max(0, min(index, len(w.tabs)-1))
	v.closeMenu()
	v.fanClosedAt = time.Time{}
	v.focusSince = time.Now()
	if server != nil {
		if v.closingAll {
			done := make(chan struct{})
			v.closingWindow = done
			w.retiring.Go(func() {
				server.close()
				close(done)
				w.wake()
			})
		} else {
			w.retiring.Go(server.close)
		}
	}
	return true
}
