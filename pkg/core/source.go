package core

import (
	"fmt"
	"os"
	"path/filepath"
)

// SourceKind names a coding agent whose local data the manager reads.
type SourceKind string

const (
	SourceClaude SourceKind = "claude"
	SourceCodex  SourceKind = "codex"
)

// Label is the product name shown in the UI.
func (k SourceKind) Label() string {
	if k == SourceCodex {
		return "Codex"
	}
	return "Claude Code"
}

// Source is the local data of one coding agent. Views talk to it instead of a
// concrete directory layout, so the same screens work for Claude Code and
// Codex.
type Source interface {
	Kind() SourceKind
	// Dir is the agent's data directory (~/.claude, ~/.codex).
	Dir() string
	// ConfigPath is the agent's main config file.
	ConfigPath() string

	GetProjects() ([]Project, error)
	// GetProjectSessions lists the sessions of p.
	GetProjectSessions(p Project) ([]Session, error)
	GetSessions() ([]Session, error)
	GetSessionMessages(s Session, maxMessages int) ([]SessionMessage, error)
	GetOverview() (Overview, error)
	// FillUsage sets Usage on every project.
	FillUsage(projects []Project)

	// CanDelete reports whether sessions and projects can be deleted.
	CanDelete() bool
	DeleteSession(s Session) error
	DeleteProject(p Project) error

	// LaunchArgs is the command line that opens the agent in projectPath,
	// resuming sessionID when it is not empty. title, when supported, names
	// the terminal tab.
	LaunchArgs(projectPath, sessionID, title string) (exe string, args []string)
	// HasLineStats reports whether Usage.LinesAdded/LinesRemoved are known.
	HasLineStats() bool
}

// CurrentSource returns the source selected in the config.
func CurrentSource() Source {
	cfg := GetConfig()
	if cfg.GetSource() == SourceCodex {
		return NewCodexDir(cfg.GetCodexDir())
	}
	return NewClaudeDir(cfg.GetClaudeDir())
}

// SourceAvailable reports whether the data directory of kind exists.
func SourceAvailable(kind SourceKind) bool {
	cfg := GetConfig()
	if kind == SourceCodex {
		return CodexDirValid(cfg.GetCodexDir())
	}
	return ExistsAndValid(cfg.GetClaudeDir())
}

// ── Claude Code ────────────────────────────────────────────────────────────

func (cd *ClaudeDir) Kind() SourceKind   { return SourceClaude }
func (cd *ClaudeDir) Dir() string        { return cd.Path }
func (cd *ClaudeDir) ConfigPath() string { return filepath.Join(cd.Path, "settings.json") }
func (cd *ClaudeDir) CanDelete() bool    { return true }
func (cd *ClaudeDir) HasLineStats() bool { return true }

// GetProjectSessions lists the sessions of p.
func (cd *ClaudeDir) GetProjectSessions(p Project) ([]Session, error) {
	return cd.GetEncodedProjectSessions(p.EncodedName)
}

func (cd *ClaudeDir) GetSessionMessages(s Session, maxMessages int) ([]SessionMessage, error) {
	return cd.GetSessionMessagesFromFile(s.FilePath, maxMessages)
}

func (cd *ClaudeDir) DeleteSession(s Session) error { return DeleteSessionFiles(s) }

// DeleteProject removes the project's directory from projects/.
func (cd *ClaudeDir) DeleteProject(p Project) error {
	if p.EncodedName == "" {
		return fmt.Errorf("project has no directory")
	}
	return os.RemoveAll(filepath.Join(cd.Path, "projects", p.EncodedName))
}

func (cd *ClaudeDir) LaunchArgs(projectPath, sessionID, title string) (string, []string) {
	var args []string
	if title != "" {
		// --name sets the terminal tab title instead of the first message
		args = append(args, "--name", title)
	}
	if sessionID != "" {
		args = append(args, "--resume", sessionID)
	}
	return "claude", args
}
