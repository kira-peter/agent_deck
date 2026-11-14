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
	"github.com/yourusername/claude-manager/pkg/ui/transition"
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

		// Run the selected view with tcell. It sets returning when it played
		// the closing half of the transition, so the menu plays the rest.
		shouldContinue := runView(selectedViewID)

		if !shouldContinue {
			// User chose to quit from view
			return
		}
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
	registry := views.GetRegistry()
	view, err := registry.Get(viewID)
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
		err := view.HandleEvent(ev)

		if err != nil {
			if err.Error() == "back" {
				// Return to selector: close the split over the view first
				closeOver(screen, view, menu)
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
