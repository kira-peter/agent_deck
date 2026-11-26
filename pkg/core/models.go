package core

import "time"

// Project represents a Claude project directory
type Project struct {
	Path         string    // Original path (e.g., D:\jarvis)
	EncodedName  string    // Encoded name (e.g., D--jarvis)
	SessionCount int       // Number of sessions in this project
	TotalSize    int64     // Total size in bytes
	LastActivity time.Time // Last modification time
	Sessions     []Session // List of sessions (loaded on demand)
	Usage        Usage     // Aggregated usage (filled by FillUsage)
}

// Session represents a conversation session
type Session struct {
	ID           string    // Session UUID
	ProjectPath  string    // Path to the project
	FirstPrompt  string    // First user message
	MessageCount int       // Number of messages in the session
	Created      time.Time // Creation timestamp
	Modified     time.Time // Last modification timestamp
	Size         int64     // Session file size in bytes
	FilePath     string    // Path to .jsonl file
	HasSubagents bool      // Whether this session has subagent directories
	GitBranch    string    // Git branch if available
}

// MemoryKind says which kind of memory a Memory is.
type MemoryKind string

const (
	MemoryUser    MemoryKind = "user"    // ~/.claude/CLAUDE.md, ~/.codex/AGENTS.md
	MemoryProject MemoryKind = "project" // <project>/CLAUDE.md, <project>/AGENTS.md
	MemoryAuto    MemoryKind = "auto"    // projects/<p>/memory/MEMORY.md (index)
	MemoryTopic   MemoryKind = "topic"   // projects/<p>/memory/<topic>.md
)

// Memory represents a project's memory file
type Memory struct {
	Kind         MemoryKind // Where the file comes from
	Name         string     // File name, e.g. MEMORY.md or CLAUDE.md
	ProjectPath  string     // Path to the project ("" for the user-level file)
	FilePath     string     // Path to the file
	Size         int64      // File size in bytes
	EntryCount   int        // Number of memory entries (parsed from markdown)
	LastModified time.Time  // Last modification time
	Preview      string     // First few lines of content
}

// Overview contains summary statistics
type Overview struct {
	ProjectCount   int             // Total number of projects
	SessionCount   int             // Total number of sessions
	MemoryCount    int             // Total number of memory files
	TotalSize      int64           // Total size of all data
	RecentSessions []Session       // Recently modified sessions
	ActiveSessions []ActiveSession // Currently running sessions
}

// ActiveSession represents a currently running Claude Code session
type ActiveSession struct {
	PID       int       // Process ID
	SessionID string    // Session UUID
	CWD       string    // Current working directory
	StartedAt time.Time // Start timestamp
	Version   string    // Claude Code version
	Status    string    // Session status (idle, working, etc.)
	Kind      string    // Session kind (interactive, etc.)
}

// SessionMessage represents a single message in a session conversation
type SessionMessage struct {
	Role    string    // "user" or "assistant"
	Content string    // extracted text content
	Time    time.Time // message timestamp
}

// SessionIndex represents the sessions-index.json structure
type SessionIndex struct {
	SessionID    string `json:"sessionId"`
	FullPath     string `json:"fullPath"`
	FileMtime    int64  `json:"fileMtime"`
	FirstPrompt  string `json:"firstPrompt"`
	MessageCount int    `json:"messageCount"`
	Created      string `json:"created"`
	Modified     string `json:"modified"`
	GitBranch    string `json:"gitBranch,omitempty"`
	ProjectPath  string `json:"projectPath"`
}

// ActiveSessionFile represents the structure of session JSON files in .claude/sessions/
type ActiveSessionFile struct {
	PID       int    `json:"pid"`
	SessionID string `json:"sessionId"`
	CWD       string `json:"cwd"`
	StartedAt int64  `json:"startedAt"`
	Version   string `json:"version"`
	Kind      string `json:"kind"`
	Status    string `json:"status"`
	Name      string `json:"name,omitempty"`
}
