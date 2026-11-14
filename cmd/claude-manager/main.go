package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gdamore/tcell/v2"
	"github.com/yourusername/claude-manager/pkg/core"
	"github.com/yourusername/claude-manager/pkg/i18n"
	"github.com/yourusername/claude-manager/pkg/ui/selector"
	"github.com/yourusername/claude-manager/pkg/views"

	// Import views to trigger their init() registration
	_ "github.com/yourusername/claude-manager/pkg/views/guide"
	_ "github.com/yourusername/claude-manager/pkg/views/memory"
	_ "github.com/yourusername/claude-manager/pkg/views/overview"
	_ "github.com/yourusername/claude-manager/pkg/views/projects"
	_ "github.com/yourusername/claude-manager/pkg/views/sessions"
	_ "github.com/yourusername/claude-manager/pkg/views/settings"
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
	return tickCmd()
}

func tickCmd() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(t time.Time) tea.Msg {
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
		if m.selector.Animating() {
			if msg.String() == "ctrl+c" {
				return m, tea.Quit
			}
			return m, nil // ignore keys while the transition plays
		}
		result := m.selector.HandleKey(msg)

		if result == -2 {
			// 用户按 q 或 ESC，不选择任何视图，直接退出
			return m, tea.Quit
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
			if returning && m.selector.StartTransition(true) {
				returning = false
				return m, frameCmd()
			}
		}
		return m, nil

	case frameMsg:
		if !m.selector.TransitionDone() {
			return m, frameCmd()
		}
		if m.selectedViewID != "" {
			return m, tea.Quit // forward transition finished: open the view
		}
		return m, nil

	case tickMsg:
		// Animation tick - selector will update internally
		return m, tickCmd()
	}

	return m, nil
}

func (m model) View() string {
	return m.selector.View()
}

func main() {
	// Initialize random seed
	rand.Seed(time.Now().UnixNano())

	// Initialize configuration
	config := core.GetConfig()

	// Check if Claude directory exists
	if !core.ExistsAndValid(config.GetClaudeDir()) {
		fmt.Printf("Error: Claude directory not found at %s\n", config.GetClaudeDir())
		fmt.Println("Please ensure Claude Code is installed and has been run at least once.")
		os.Exit(1)
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

		if selectedViewID == "" {
			// User quit without selecting a view
			return
		}

		lastViewID = selectedViewID

		// Run the selected view with tcell
		shouldContinue := runView(selectedViewID)

		if !shouldContinue {
			// User chose to quit from view
			return
		}
		// Loop continues - return to selector, playing the transition back
		returning = true
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
	screen, err := tcell.NewScreen()
	if err != nil {
		log.Fatalf("Failed to create screen: %v", err)
	}
	if err := screen.Init(); err != nil {
		log.Fatalf("Failed to initialize screen: %v", err)
	}
	defer screen.Fini()

	screen.SetStyle(tcell.StyleDefault.
		Background(tcell.ColorBlack).
		Foreground(tcell.ColorWhite))
	screen.Clear()

	// Load the view
	registry := views.GetRegistry()
	view, err := registry.Get(viewID)
	if err != nil {
		screen.Fini()
		fmt.Printf("Error loading view: %v\n", err)
		return false
	}

	if err := view.Init(screen); err != nil {
		screen.Fini()
		fmt.Printf("Error initializing view: %v\n", err)
		return false
	}

	// View event loop
	view.Render()
	for {
		ev := screen.PollEvent()
		err := view.HandleEvent(ev)

		if err != nil {
			if err.Error() == "back" {
				// Return to selector
				view.Cleanup()
				return true
			} else if err.Error() == "quit" {
				// Quit application
				view.Cleanup()
				return false
			}
			// Other errors: continue
		}

		view.Render()
	}
}
