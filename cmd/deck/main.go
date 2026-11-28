package main

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gdamore/tcell/v2"
	"github.com/kira-peter/agent_deck/pkg/core"
	"github.com/kira-peter/agent_deck/pkg/i18n"
	"github.com/kira-peter/agent_deck/pkg/ui/selector"
	"github.com/kira-peter/agent_deck/pkg/ui/transition"
	"github.com/kira-peter/agent_deck/pkg/views"
	"github.com/kira-peter/agent_deck/pkg/views/codexchat"
	"github.com/kira-peter/agent_deck/pkg/views/sessions"

	// Import views to trigger their init() registration
	_ "github.com/kira-peter/agent_deck/pkg/views/guide"
	_ "github.com/kira-peter/agent_deck/pkg/views/memory"
	_ "github.com/kira-peter/agent_deck/pkg/views/overview"
	_ "github.com/kira-peter/agent_deck/pkg/views/projects"
	_ "github.com/kira-peter/agent_deck/pkg/views/settings"
)

// tickMsg is sent on every tick to update animations
type tickMsg time.Time

// model represents the application state
type model struct {
	selector       *selector.BubbleTeaSelector
	selectedViewID string // 保存用户选择的视图 ID
	ready          bool
	width          int
	height         int
	exitPopup      *exitPopup
}

// lastViewID remembers the view opened last, so the selector highlights it
// again when the user comes back from that view.
var lastViewID string

// returning is set when the selector is shown again after a view, so it opens
// with the page transition played backwards.
var returning bool

func initialModel() model {
	sel := selector.NewBubbleTeaSelector()
	sel.SelectByID(lastViewID)
	return model{
		selector: sel,
		ready:    false,
	}
}

func (m model) Init() tea.Cmd {
	return tickCmd(80 * time.Millisecond)
}

func tickCmd(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// frameMsg drives the page transition at a smoother rate than the spinner.
type frameMsg time.Time

func frameCmd() tea.Cmd {
	return tea.Tick(16*time.Millisecond, func(t time.Time) tea.Msg {
		return frameMsg(t)
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.exitPopup != nil {
			_, running := codexchat.ExitStatus()
			switch m.exitPopup.handle(msg.String(), running) {
			case exitCancel:
				m.exitPopup = nil
			case exitWorkspace:
				codexchat.FocusForExit()
				m.selectedViewID = codexchat.ViewID
				return m, tea.Quit
			case exitConfirm:
				return m, tea.Quit
			}
			return m, nil
		}
		if m.selector.Animating() {
			if msg.String() == "ctrl+c" {
				m.selector.StopTransition()
				return m.requestExit()
			}
			return m, nil // ignore keys while the transition plays
		}
		result := m.selector.HandleKey(msg)

		if result == -2 {
			return m.requestExit()
		} else if result >= 0 {
			// 用户按 Enter 选中视图：先播放转场动画，结束后再退出
			m.selectedViewID = m.selector.GetSelectedViewID()
			if m.selector.StartTransition(false) {
				return m, frameCmd()
			}
			return m, tea.Quit // 退出 Bubble Tea，返回 main 继续处理
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.selector.SetSize(msg.Width, msg.Height)
		if !m.ready {
			m.ready = true
			// the view already closed the split onto the line; shrink it back
			if returning && m.selector.StartTransition(true) {
				returning = false
				return m, frameCmd()
			}
			returning = false
		}
		return m, nil

	case frameMsg:
		if !m.selector.TransitionDone() {
			return m, frameCmd()
		}
		if m.selectedViewID != "" {
			// stretch finished: hand the line frame to the view, which plays
			// the split while uncovering itself
			if lines, row, ok := m.selector.LineFrame(); ok {
				transition.Hand(&transition.Frame{Rows: transition.FromANSI(lines, m.width), Line: row})
			}
			return m, tea.Quit
		}
		return m, nil

	case tickMsg:
		interval := 80 * time.Millisecond
		if m.exitPopup != nil {
			interval = m.exitPopup.refreshInterval(interval)
		}
		return m, tickCmd(interval)
	}

	return m, nil
}

func (m model) View() string {
	background := m.selector.View()
	if m.exitPopup == nil {
		return background
	}
	open, running := codexchat.ExitStatus()
	return m.exitPopup.overlay(background, m.width, m.height, open, running)
}

func (m model) requestExit() (tea.Model, tea.Cmd) {
	m.selectedViewID = ""
	if open, _ := codexchat.ExitStatus(); open == 0 {
		return m, tea.Quit
	}
	m.exitPopup = newExitPopup()
	return m, nil
}

func main() {
	defer codexchat.Shutdown()
	// Initialize random seed
	rand.Seed(time.Now().UnixNano())

	// Initialize configuration
	config := core.GetConfig()

	// Start when either agent's data folder exists; if the saved choice is
	// missing, fall back to the other one.
	claudeOK := core.SourceAvailable(core.SourceClaude)
	codexOK := core.SourceAvailable(core.SourceCodex)
	if !claudeOK && !codexOK {
		fmt.Printf("Error: neither Claude Code (%s) nor Codex (%s) data was found.\n",
			config.GetClaudeDir(), config.GetCodexDir())
		fmt.Println("Please run Claude Code or Codex at least once.")
		os.Exit(1)
	}
	if config.GetSource() == core.SourceCodex && !codexOK {
		_ = config.SetSource(core.SourceClaude)
	} else if config.GetSource() == core.SourceClaude && !claudeOK {
		_ = config.SetSource(core.SourceCodex)
	}

	// Initialize i18n with config language
	mgr := i18n.GetManager()
	if config.GetLanguage() == "zh" {
		mgr.SetLanguage(i18n.LanguageChinese)
	} else {
		mgr.SetLanguage(i18n.LanguageEnglish)
	}

	// Main loop: selector -> view -> back to selector
	for {
		// Run Bubble Tea selector
		selectedViewID := runSelector()

		if selectedViewID != "" {
			lastViewID = selectedViewID
			if runView(selectedViewID) {
				continue
			}
		}
		return
	}
}

// runSelector runs the Bubble Tea selector and returns the selected view ID
func runSelector() string {
	p := tea.NewProgram(
		initialModel(),
		tea.WithAltScreen(),
	)

	finalModel, err := p.Run()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	m := finalModel.(model)
	return m.selectedViewID
}

// runView runs a view with tcell and returns whether to continue (true = back to selector, false = quit)
func runView(viewID string) bool {
	// Initialize tcell screen
	raw, err := tcell.NewScreen()
	if err != nil {
		log.Fatalf("Failed to create screen: %v", err)
	}
	if err := raw.Init(); err != nil {
		log.Fatalf("Failed to initialize screen: %v", err)
	}
	defer raw.Fini()

	// The menu hands over the frame its transition ended on. While it is
	// set, the view draws into the buffer without showing, then the split
	// uncovers it.
	menu := transition.Take()
	if menu != nil {
		if w, h := raw.Size(); h != len(menu.Rows) || (h > 0 && w != len(menu.Rows[0])) {
			menu = nil // window changed size in between: skip the animation
		}
	}
	screen := &transition.Offscreen{Screen: raw, Hold: menu != nil}

	screen.SetStyle(tcell.StyleDefault.
		Background(tcell.ColorBlack).
		Foreground(tcell.ColorWhite))
	if menu != nil {
		// keep the line on screen while the view loads
		transition.ShowLine(screen.Screen, menu)
	} else {
		screen.Clear()
	}

	// Load the view
	view, err := views.GetRegistry().Get(viewID)
	if err != nil {
		raw.Fini()
		fmt.Printf("Error loading view: %v\n", err)
		return false
	}

	if err := view.Init(screen); err != nil {
		raw.Fini()
		fmt.Printf("Error initializing view: %v\n", err)
		return false
	}

	// View event loop
	view.Render()
	if menu != nil {
		frame := transition.Capture(raw)
		screen.Hold = false
		transition.Open(raw, menu, frame)
	}
	for {
		ev := screen.PollEvent()
		if ev == nil {
			view.Cleanup()
			return false
		}
		var err error
		if key, ok := ev.(*tcell.EventKey); ok && key.Key() == tcell.KeyCtrlC && viewID != codexchat.ViewID {
			err = errors.New("quit")
		} else {
			err = view.HandleEvent(ev)
		}
		var request codexchat.SessionPageRequest
		if errors.As(err, &request) {
			next := sessions.NewAgentSessionsView(request.Kind, request.Home)
			view.Cleanup()
			if err := next.Init(screen); err != nil {
				raw.Fini()
				fmt.Printf("Error opening sessions: %v\n", err)
				return true
			}
			view = next
			viewID = next.Metadata().ID
			lastViewID = viewID
			menu = nil
			view.Render()
			continue
		}

		if err != nil {
			if err.Error() == "back" {
				// Return to selector: close the split over the view first
				closeOver(screen, view, menu)
				view.Cleanup()
				return true
			} else if err.Error() == "quit" {
				switch confirmExitOnScreen(screen, view) {
				case exitConfirm:
					view.Cleanup()
					return false
				case exitWorkspace:
					codexchat.FocusForExit()
					if viewID != codexchat.ViewID {
						view.Cleanup()
						view, err = views.GetRegistry().Get(codexchat.ViewID)
						if err != nil {
							fmt.Printf("Error opening workspace: %v\n", err)
							return true
						}
						if err := view.Init(screen); err != nil {
							fmt.Printf("Error opening workspace: %v\n", err)
							return true
						}
						viewID, lastViewID, menu = codexchat.ViewID, codexchat.ViewID, nil
					}
				}
			}
			// Other errors: continue
		}

		view.Render()
	}
}

// closeOver plays the split backwards over the view, ending on the line the
// menu then shrinks into its entry. The menu backdrop outside the lines is the
// frame the menu handed over; without one (or after a resize) there is no
// animation and the menu plays nothing either.
func closeOver(screen *transition.Offscreen, view views.View, menu *transition.Frame) {
	w, h := screen.Size()
	if menu == nil || h != len(menu.Rows) || (h > 0 && w != len(menu.Rows[0])) {
		returning = false
		return
	}
	screen.Hold = true
	view.Render()
	frame := transition.Capture(screen.Screen)
	screen.Hold = false
	transition.Close(screen.Screen, menu, frame)
	returning = true
}
