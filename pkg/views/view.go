package views

import (
	"github.com/gdamore/tcell/v2"
)

// View represents a screen view in the application
type View interface {
	// Metadata returns the view's metadata
	Metadata() Metadata

	// Init initializes the view with a screen
	Init(screen tcell.Screen) error

	// Render renders the view to the screen
	Render() error

	// HandleEvent handles keyboard and other events
	HandleEvent(event tcell.Event) error

	// Cleanup cleans up resources
	Cleanup() error
}

// Metadata contains view information
type Metadata struct {
	ID     string // Unique identifier
	NameEn string // English name
	NameZh string // Chinese name
	DescEn string // English description
	DescZh string // Chinese description
	Icon   string // Unicode icon/emoji
}

// ViewFactory is a function that creates a new view instance
type ViewFactory func() View
