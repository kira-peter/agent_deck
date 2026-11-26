package selector

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/kira-peter/agentdeck/pkg/i18n"
	"github.com/kira-peter/agentdeck/pkg/ui/animation"
	"github.com/kira-peter/agentdeck/pkg/views"
)

// Selector is the main view selector
type Selector struct {
	screen      tcell.Screen
	viewList    []views.Metadata
	selectedIdx int
	width       int
	height      int
	spinner     *animation.Spinner
	ticker      *time.Ticker
	needsRedraw bool
}

// New creates a new selector
func New(screen tcell.Screen) *Selector {
	registry := views.GetRegistry()
	s := &Selector{
		screen:      screen,
		viewList:    registry.GetAll(),
		selectedIdx: 0,
		spinner:     animation.NewSpinner(),
	}

	// Start spinner animation
	s.spinner.Start()

	// Start refresh ticker for animation
	s.ticker = time.NewTicker(80 * time.Millisecond)

	return s
}

// updateSize updates the terminal dimensions
func (s *Selector) updateSize() {
	s.width, s.height = s.screen.Size()
}

// Render renders the selector interface
func (s *Selector) Render() {
	s.updateSize()
	s.screen.Clear()

	// Title area
	s.renderTitle()

	// View list
	s.renderViewList()

	// Description area
	s.renderDescription()

	// Hints area
	s.renderHints()

	s.screen.Show()
}

// renderTitle renders the title
func (s *Selector) renderTitle() {
	title := i18n.AppTitle.Get()

	// Center the title
	titleY := 2
	s.drawCenteredText(titleY, title, tcell.StyleDefault.
		Foreground(tcell.ColorLightGreen).
		Bold(true))

	// Separator line
	s.drawHorizontalLine(titleY + 2)
}

// renderViewList renders the view list (single column, centered)
func (s *Selector) renderViewList() {
	startY := 6

	if len(s.viewList) == 0 {
		s.drawCenteredText(startY+2, "No views available", tcell.StyleDefault.
			Foreground(tcell.ColorGray))
		return
	}

	mgr := i18n.GetManager()
	isEnglish := mgr.IsEnglish()

	// Calculate max width (always with space for spinner)
	maxWidth := 0
	for idx, view := range s.viewList {
		name := view.NameZh
		if isEnglish {
			name = view.NameEn
		}

		// Always calculate with spinner space to keep consistent width
		text := fmt.Sprintf("⠋ %s (%d)", name, idx+1)
		width := calculateDisplayWidth(text)
		if width > maxWidth {
			maxWidth = width
		}
	}

	// Calculate left-aligned start position (centered as a block)
	startX := (s.width - maxWidth) / 2
	if startX < 2 {
		startX = 2
	}

	// Draw with left alignment
	for idx, view := range s.viewList {
		y := startY + idx*2

		// Determine name based on language
		name := view.NameZh
		if isEnglish {
			name = view.NameEn
		}

		// Format text - ALWAYS include spinner space for alignment
		var text string
		if idx == s.selectedIdx {
			// Show animated spinner for selected item
			frame := s.spinner.GetFrame()
			text = fmt.Sprintf("%c %s (%d)", frame, name, idx+1)
		} else {
			// Empty space (same width as spinner) for unselected items
			text = fmt.Sprintf("  %s (%d)", name, idx+1)
		}

		// Style: selected item uses different color
		style := tcell.StyleDefault.Foreground(tcell.ColorWhite)
		if idx == s.selectedIdx {
			style = tcell.StyleDefault.
				Foreground(tcell.ColorLightGreen).
				Bold(true)
		}

		// Draw text left-aligned at startX
		s.drawTextAt(startX, y, text, style)
	}
}

// renderDescription renders the description of the selected view
func (s *Selector) renderDescription() {
	if s.selectedIdx < 0 || s.selectedIdx >= len(s.viewList) {
		return
	}

	view := s.viewList[s.selectedIdx]
	mgr := i18n.GetManager()

	desc := view.DescZh
	if mgr.IsEnglish() {
		desc = view.DescEn
	}

	descY := s.height - 6
	s.drawHorizontalLine(descY)
	s.drawCenteredText(descY+2, desc, tcell.StyleDefault.
		Foreground(tcell.ColorLightBlue))
}

// renderHints renders keyboard hints
func (s *Selector) renderHints() {
	hints := i18n.HintNavigation.Get()

	hintsY := s.height - 3
	s.drawHorizontalLine(hintsY)
	s.drawCenteredText(hintsY+1, hints, tcell.StyleDefault.
		Foreground(tcell.ColorGray))
}

// HandleEvent handles keyboard events
// Returns: selected view ID, or special values:
//
//	-1: no selection (continue)
//	-2: quit
//	-3: refresh (language changed)
func (s *Selector) HandleEvent(ev tcell.Event) int {
	switch ev := ev.(type) {
	case *tcell.EventKey:
		switch ev.Key() {
		case tcell.KeyEscape, tcell.KeyCtrlC:
			return -2 // Quit

		case tcell.KeyUp:
			s.movePrev()
			return -1

		case tcell.KeyDown:
			s.moveNext()
			return -1

		case tcell.KeyLeft:
			s.moveLeft()
			return -1

		case tcell.KeyRight:
			s.moveRight()
			return -1

		case tcell.KeyEnter:
			return s.selectedIdx // Select

		case tcell.KeyRune:
			switch ev.Rune() {
			case 'q', 'Q':
				return -2 // Quit

			case 'k': // Vim-style up
				s.movePrev()
				return -1

			case 'j': // Vim-style down
				s.moveNext()
				return -1

			case 'h': // Vim-style left
				s.moveLeft()
				return -1

			case 'l': // Vim-style right
				s.moveRight()
				return -1

			case 't', 'T': // Toggle language
				mgr := i18n.GetManager()
				mgr.Toggle()
				return -3 // Refresh

			case '1', '2', '3', '4', '5', '6', '7', '8', '9':
				idx := int(ev.Rune() - '1')
				if idx < len(s.viewList) {
					return idx
				}

			case '0':
				if len(s.viewList) >= 10 {
					return 9
				}
			}
		}

	case *tcell.EventResize:
		s.screen.Sync()
		return -1
	}

	return -1
}

// Movement functions
func (s *Selector) movePrev() {
	if s.selectedIdx > 0 {
		s.selectedIdx--
	}
}

func (s *Selector) moveNext() {
	if s.selectedIdx < len(s.viewList)-1 {
		s.selectedIdx++
	}
}

func (s *Selector) moveLeft() {
	// Single column layout - left key moves up
	s.movePrev()
}

func (s *Selector) moveRight() {
	// Single column layout - right key moves down
	s.moveNext()
}

// Helper drawing functions

func (s *Selector) drawText(x, y int, text string, style tcell.Style) {
	for i, r := range text {
		s.screen.SetContent(x+i, y, r, nil, style)
	}
}

func (s *Selector) drawTextAt(x, y int, text string, style tcell.Style) {
	// Draw text character by character at specified position
	xOffset := 0
	for _, r := range text {
		s.screen.SetContent(x+xOffset, y, r, nil, style)

		// Determine character width
		var charWidth int
		if r >= 0x4E00 && r <= 0x9FFF { // CJK
			charWidth = 2
		} else if r >= 0x3400 && r <= 0x4DBF {
			charWidth = 2
		} else if r >= 0x20000 && r <= 0x2A6DF {
			charWidth = 2
		} else if r >= 0x2A700 && r <= 0x2B73F {
			charWidth = 2
		} else if r >= 0x2B740 && r <= 0x2B81F {
			charWidth = 2
		} else if r >= 0x2B820 && r <= 0x2CEAF {
			charWidth = 2
		} else if r >= 0x3000 && r <= 0x303F {
			charWidth = 2
		} else if r >= 0xFF00 && r <= 0xFFEF {
			charWidth = 2
		} else if r >= 0x1F300 && r <= 0x1F9FF { // Emoji
			charWidth = 2
		} else if r >= 0x2600 && r <= 0x26FF {
			charWidth = 2
		} else if r >= 0x2700 && r <= 0x27BF {
			charWidth = 2
		} else if r >= 0xFE00 && r <= 0xFE0F { // Variation Selectors
			charWidth = 0
		} else if r >= 0x2800 && r <= 0x28FF { // Braille (spinner)
			charWidth = 1
		} else {
			charWidth = 1
		}

		// Fill extra space for wide characters with background
		if charWidth == 2 {
			s.screen.SetContent(x+xOffset+1, y, ' ', nil, style)
		}

		xOffset += charWidth
	}
}

func (s *Selector) drawCenteredText(y int, text string, style tcell.Style) {
	// Calculate display width (handle Chinese characters)
	displayWidth := calculateDisplayWidth(text)
	x := (s.width - displayWidth) / 2
	if x < 0 {
		x = 0
	}

	// Draw text character by character
	xOffset := 0
	for _, r := range text {
		s.screen.SetContent(x+xOffset, y, r, nil, style)

		// Determine character width
		var charWidth int
		if r >= 0x4E00 && r <= 0x9FFF { // CJK
			charWidth = 2
		} else if r >= 0x3400 && r <= 0x4DBF {
			charWidth = 2
		} else if r >= 0x20000 && r <= 0x2A6DF {
			charWidth = 2
		} else if r >= 0x2A700 && r <= 0x2B73F {
			charWidth = 2
		} else if r >= 0x2B740 && r <= 0x2B81F {
			charWidth = 2
		} else if r >= 0x2B820 && r <= 0x2CEAF {
			charWidth = 2
		} else if r >= 0x3000 && r <= 0x303F {
			charWidth = 2
		} else if r >= 0xFF00 && r <= 0xFFEF {
			charWidth = 2
		} else if r >= 0x1F300 && r <= 0x1F9FF { // Emoji
			charWidth = 2
		} else if r >= 0x2600 && r <= 0x26FF {
			charWidth = 2
		} else if r >= 0x2700 && r <= 0x27BF {
			charWidth = 2
		} else if r >= 0xFE00 && r <= 0xFE0F { // Variation Selectors
			charWidth = 0
		} else if r >= 0x2800 && r <= 0x28FF { // Braille (spinner)
			charWidth = 1
		} else {
			charWidth = 1
		}

		// Fill extra space for wide characters with background
		if charWidth == 2 {
			s.screen.SetContent(x+xOffset+1, y, ' ', nil, style)
		}

		xOffset += charWidth
	}
}

func (s *Selector) drawHorizontalLine(y int) {
	for x := 2; x < s.width-2; x++ {
		s.screen.SetContent(x, y, '─', nil, tcell.StyleDefault.
			Foreground(tcell.ColorGray))
	}
}

// calculateDisplayWidth calculates the display width of a string
// Chinese characters and emoji count as 2, others as 1
func calculateDisplayWidth(text string) int {
	width := 0
	for _, r := range text {
		if r >= 0x4E00 && r <= 0x9FFF { // CJK Unified Ideographs
			width += 2
		} else if r >= 0x3400 && r <= 0x4DBF { // CJK Extension A
			width += 2
		} else if r >= 0x20000 && r <= 0x2A6DF { // CJK Extension B
			width += 2
		} else if r >= 0x2A700 && r <= 0x2B73F { // CJK Extension C
			width += 2
		} else if r >= 0x2B740 && r <= 0x2B81F { // CJK Extension D
			width += 2
		} else if r >= 0x2B820 && r <= 0x2CEAF { // CJK Extension E
			width += 2
		} else if r >= 0x3000 && r <= 0x303F { // CJK Symbols and Punctuation
			width += 2
		} else if r >= 0xFF00 && r <= 0xFFEF { // Halfwidth and Fullwidth Forms
			width += 2
		} else if r >= 0x1F300 && r <= 0x1F9FF { // Emoji
			width += 2
		} else if r >= 0x2600 && r <= 0x26FF { // Misc Symbols (emoji)
			width += 2
		} else if r >= 0x2700 && r <= 0x27BF { // Dingbats (emoji)
			width += 2
		} else if r >= 0xFE00 && r <= 0xFE0F { // Variation Selectors (emoji modifiers)
			width += 0 // Don't count modifiers
		} else if r >= 0x2800 && r <= 0x28FF { // Braille Patterns (spinner animation)
			width += 1
		} else {
			width += 1
		}
	}
	return width
}

// GetSelectedViewID returns the ID of the currently selected view
func (s *Selector) GetSelectedViewID() string {
	if s.selectedIdx >= 0 && s.selectedIdx < len(s.viewList) {
		return s.viewList[s.selectedIdx].ID
	}
	return ""
}

// Cleanup stops the spinner and ticker
func (s *Selector) Cleanup() {
	if s.spinner != nil {
		s.spinner.Stop()
	}
	if s.ticker != nil {
		s.ticker.Stop()
	}
}

// GetTicker returns the refresh ticker for animation
func (s *Selector) GetTicker() *time.Ticker {
	return s.ticker
}
