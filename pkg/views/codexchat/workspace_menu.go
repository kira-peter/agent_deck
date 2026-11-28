package codexchat

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/gdamore/tcell/v2"
	"github.com/kira-peter/agent_deck/pkg/core"
	visual "github.com/will-to-meaning/terminal_visuals"
)

// SessionPageRequest asks Deck to show the session list for this Codex home.
type SessionPageRequest struct {
	Home string
	Kind core.SourceKind
}

func (SessionPageRequest) Error() string { return "open agent session list" }

type menuKind int

const (
	menuNone menuKind = iota
	menuRoot
	menuTheme
	menuModelWindow
	menuModel
	menuModelText
	menuRestart
	menuCloseWindow
)

const (
	rootTheme = iota
	rootModel
	rootSessions
	rootRestart
	rootClose
	rootCloseAll
)

type workspaceMenu struct {
	kind     menuKind
	list     visual.OptionList
	window   int
	input    []rune
	flashAt  time.Time
	openedAt time.Time
	notice   string
}

type menuRow struct {
	label, detail string
	disabled      bool
}

func (c *chat) busy() bool { return c.working || len(c.prompts) != 0 }

func (w *workspace) restart(index int) bool {
	old := w.tabs[index]
	old.mu.Lock()
	if old.closed || old.busy() || old.connecting {
		old.mu.Unlock()
		return false
	}
	id := old.sessionID
	if old.used && old.threadID != "" {
		id = old.threadID
	}
	c := &chat{
		kind: old.kind, project: old.project, home: old.home, sessionID: id,
		model: old.model, desiredModel: old.desiredModel, models: old.models, modelProvider: old.modelProvider,
		connecting: true, keepItems: true, lastTurnStatus: old.lastTurnStatus,
		showTools: old.showTools, chartCollapsed: old.chartCollapsed, chart: old.chart,
		items: old.items, itemIndex: old.itemIndex, requests: make(map[string]string),
		draft: old.draft, cursor: old.cursor, scroll: old.scroll,
	}
	if c.desiredModel == "" {
		c.desiredModel = c.model
	}
	old.closed = true
	server := old.server
	old.mu.Unlock()
	w.tabs[index] = c
	w.active = index
	w.overview = false
	go func() {
		if server != nil {
			server.close()
		}
		c.connect("", w.wake)
	}()
	return true
}

func (v *workspaceView) openMenu(kind menuKind, selected int) {
	if kind == menuRoot {
		v.fanTrail = nil
		v.fanPane = -1
		if len(v.workspace.tabs) > 0 && !v.workspace.overview {
			v.fanPane = v.workspace.active
		}
	}
	now := time.Now()
	v.fanClosedAt = time.Time{}
	v.menu = workspaceMenu{kind: kind, list: visual.OptionList{Selected: selected}, flashAt: now, openedAt: now}
}

func (v *workspaceView) pushMenu(kind menuKind, selected int) {
	_, rows := v.menuRows()
	v.fanTrail = append(v.fanTrail, fanParent{menu: v.menu, rows: rows})
	v.openMenu(kind, selected)
}

func (v *workspaceView) popMenu() {
	v.restoreTheme()
	if n := len(v.fanTrail); n > 0 {
		v.menu = v.fanTrail[n-1].menu
		v.fanTrail = v.fanTrail[:n-1]
		return
	}
	v.openMenu(menuRoot, 0)
}

func (v *workspaceView) restoreTheme() {
	if v.themeBefore != "" {
		applyTheme(v.themeBefore)
		v.themeBefore = ""
	}
}

func (v *workspaceView) closeMenu() {
	v.restoreTheme()
	if v.menu.kind == menuRoot && !v.workspace.motionOff {
		v.fanClosedAt, v.fanClosedSelection = time.Now(), v.menu.list.Selected
	}
	v.menu = workspaceMenu{}
	v.fanTrail = nil
}

func (v *workspaceView) previewWindow() int {
	var index int
	switch v.menu.kind {
	case menuModelWindow, menuRestart, menuCloseWindow:
		index = v.menu.list.Selected
	case menuModel, menuModelText:
		index = v.menu.window
	default:
		return -1
	}
	if index < 0 || index >= len(v.workspace.tabs) {
		return -1
	}
	return index
}

func (v *workspaceView) menuRows() (string, []menuRow) {
	w := v.workspace
	switch v.menu.kind {
	case menuRoot:
		return local("Workspace options", "工作区选项"), v.rootMenuRows()
	case menuTheme:
		var rows []menuRow
		for _, theme := range visual.Themes() {
			row := menuRow{label: themeName(theme.ID)}
			if v.themeBefore == theme.ID {
				row.detail = "✓"
			}
			rows = append(rows, row)
		}
		return local("Theme", "主题"), rows
	case menuModelWindow, menuRestart, menuCloseWindow:
		rows := make([]menuRow, 0, len(w.tabs))
		for i, c := range w.tabs {
			c.mu.Lock()
			status := c.status().text
			blocked := v.menu.kind == menuRestart && (c.busy() || c.connecting) || v.menu.kind == menuCloseWindow && c.busy()
			c.mu.Unlock()
			rows = append(rows, menuRow{fmt.Sprintf("%d  %s", i+1, filepath.Base(c.project)), status, blocked})
		}
		if v.menu.kind == menuRestart {
			return local("Restart a window", "选择要重启的窗口"), rows
		}
		if v.menu.kind == menuCloseWindow {
			return local("Close window", "关闭窗口"), rows
		}
		return local("Select a window", "选择要修改模型的窗口"), rows
	case menuModel, menuModelText:
		c := w.tabs[v.menu.window]
		c.mu.Lock()
		defer c.mu.Unlock()
		title := local("Model · ", "模型 · ") + filepath.Base(c.project)
		if v.menu.kind == menuModelText {
			return title, []menuRow{{label: "› " + string(v.menu.input)}}
		}
		rows := []menuRow{{label: local("Enter a model ID…", "手动输入模型 ID…")}}
		models := slices.Clone(c.models)
		for _, id := range []string{c.model, c.desiredModel} {
			if id != "" && !slices.Contains(models, id) {
				models = append(models, id)
			}
		}
		for _, id := range models {
			detail := ""
			switch {
			case id == c.model:
				detail = local("Current", "当前")
			case id == c.desiredModel:
				detail = local("Pending restart", "待重启")
			}
			rows = append(rows, menuRow{label: id, detail: detail})
		}
		return title, rows
	}
	return "", nil
}

func (v *workspaceView) handleMenu(event *tcell.EventKey) error {
	w := v.workspace
	m := &v.menu
	key := event.Key()
	if m.kind != menuModelText {
		if key == tcell.KeyRight && m.kind != menuTheme {
			key = tcell.KeyEnter
		}
		if key == tcell.KeyLeft {
			key = tcell.KeyEscape
		}
	}
	if event.Key() == tcell.KeyF1 {
		v.closeMenu()
		return nil
	}
	if key == tcell.KeyEscape {
		switch m.kind {
		case menuRoot:
			v.closeMenu()
		default:
			v.popMenu()
		}
		return nil
	}
	if m.kind == menuModelText {
		switch event.Key() {
		case tcell.KeyRune:
			if len(m.input) < 160 && !unicode.IsControl(event.Rune()) {
				m.input = append(m.input, event.Rune())
			}
		case tcell.KeyBackspace, tcell.KeyBackspace2:
			m.input = m.input[:max(len(m.input)-1, 0)]
		case tcell.KeyEnter:
			id := strings.TrimSpace(string(m.input))
			if id == "" || strings.ContainsFunc(id, unicode.IsSpace) {
				m.notice = local("Enter a model ID without spaces.", "请输入不带空格的模型 ID。")
			} else {
				v.stageModel(m.window, id)
			}
		}
		return nil
	}
	_, rows := v.menuRows()
	switch key {
	case tcell.KeyUp, tcell.KeyDown, tcell.KeyPgUp, tcell.KeyPgDn:
		delta := map[tcell.Key]int{tcell.KeyUp: -1, tcell.KeyDown: 1, tcell.KeyPgUp: -5, tcell.KeyPgDn: 5}[event.Key()]
		m.list.Move(len(rows), delta)
		m.flashAt, m.notice = time.Now(), ""
		if m.kind == menuTheme {
			applyTheme(visual.Themes()[m.list.Selected].ID)
		}
	case tcell.KeyEnter:
		i := m.list.Selected
		if i < 0 || i >= len(rows) {
			return nil
		}
		if rows[i].disabled {
			m.notice = local("Unavailable while a window is working or connecting.", "窗口正在工作或连接，暂时无法执行。")
			if m.kind == menuRoot {
				m.notice = local("Open a session first.", "请先打开一个会话。")
				if i == rootCloseAll {
					m.notice = closeAllBlocked()
				}
			}
			return nil
		}
		switch m.kind {
		case menuRoot:
			switch i {
			case rootTheme:
				v.themeBefore = currentTheme
				v.pushMenu(menuTheme, slices.IndexFunc(visual.Themes(), func(t visual.Theme) bool { return t.ID == currentTheme }))
			case rootModel:
				v.pushMenu(menuModelWindow, w.active)
			case rootSessions:
				source := core.CurrentSource()
				home, kind := source.Dir(), source.Kind()
				if len(w.tabs) > 0 && w.tabs[w.active].home != "" {
					active := w.tabs[w.active]
					home, kind = active.home, active.agentKind()
				}
				v.closeMenu()
				return SessionPageRequest{Home: home, Kind: kind}
			case rootRestart:
				v.pushMenu(menuRestart, w.active)
			case rootClose:
				if v.fanPane < 0 {
					v.pushMenu(menuCloseWindow, w.active)
				} else {
					v.closeSelectedWindow(v.fanPane)
				}
			case rootCloseAll:
				v.closeMenu()
				v.fanClosedAt = time.Time{}
				v.closingAll = true
				w.overview = false
				return v.advanceCloseAll()
			}
		case menuTheme:
			id := visual.Themes()[i].ID
			applyTheme(id)
			v.themeBefore = ""
			if err := core.GetConfig().UpdateDisplay(func(d *core.DisplayConfig) { d.WorkspaceTheme = id }); err != nil {
				v.message = err.Error()
			}
			v.popMenu()
		case menuModelWindow:
			c := w.tabs[i]
			c.mu.Lock()
			if c.ready && c.server != nil && len(c.models) == 0 {
				params := map[string]any{"limit": 100}
				if c.modelProvider != "" {
					params["modelProvider"] = c.modelProvider
				}
				c.request("model/list", params)
			}
			c.mu.Unlock()
			v.pushMenu(menuModel, 0)
			v.menu.window = i
		case menuModel:
			if i == 0 {
				window := m.window
				v.pushMenu(menuModelText, 0)
				v.menu.window = window
			} else {
				v.stageModel(m.window, rows[i].label)
			}
		case menuRestart:
			if w.restart(i) {
				v.closeMenu()
				v.focusSince = time.Now()
			} else {
				m.notice = local("This window cannot be restarted right now.", "该窗口暂时无法重启。")
			}
		case menuCloseWindow:
			v.closeSelectedWindow(i)
		}
	}
	return nil
}

func (v *workspaceView) stageModel(index int, id string) {
	c := v.workspace.tabs[index]
	c.mu.Lock()
	c.desiredModel = id
	c.mu.Unlock()
	if len(v.fanTrail) > 0 {
		v.fanTrail = v.fanTrail[:1]
		v.fanTrail[0].menu.list.Selected = rootRestart
	}
	v.openMenu(menuRestart, index)
	v.menu.notice = local("Model saved. Restart this window to apply it.", "模型已设置。重启此窗口后生效。")
}
