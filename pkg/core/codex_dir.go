package core

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// CodexDir reads the local data of the Codex CLI (~/.codex). Codex stores
// every session as sessions/YYYY/MM/DD/rollout-<time>-<id>.jsonl, so unlike
// Claude Code there are no project folders: projects are the distinct working
// directories (session_meta.cwd) of the sessions. The files are only read;
// deleting and archiving are left to the codex CLI.
type CodexDir struct {
	Path string

	once     sync.Once
	sessions []Session
	usage    map[string]Usage // by session id
	err      error
}

func NewCodexDir(path string) *CodexDir { return &CodexDir{Path: path} }

// CodexDirValid reports whether dir looks like a Codex data directory.
func CodexDirValid(dir string) bool {
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return false
	}
	for _, name := range []string{"sessions", "config.toml", "session_index.jsonl"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}
	return false
}

func (cx *CodexDir) Kind() SourceKind   { return SourceCodex }
func (cx *CodexDir) Dir() string        { return cx.Path }
func (cx *CodexDir) ConfigPath() string { return filepath.Join(cx.Path, "config.toml") }
func (cx *CodexDir) HasLineStats() bool { return false }

// Deleting and archiving go through the codex CLI (`codex delete`, `codex
// archive`), so Codex's own sqlite indexes stay consistent with the files.
func (cx *CodexDir) CanDelete() bool { return true }

// DeleteSession permanently deletes s with `codex delete`.
func (cx *CodexDir) DeleteSession(s Session) error {
	// Without a terminal codex cannot ask for confirmation and refuses;
	// --force skips the prompt (the user already confirmed in our popup, or
	// chose D to delete without one). It requires a UUID, which s.ID is.
	if err := runCodex(cx.Path, "delete", "--force", s.ID); err != nil {
		return err
	}
	// trust the files, not just the exit code
	if _, err := os.Stat(s.FilePath); err == nil {
		return fmt.Errorf("codex delete reported success but %s is still there", filepath.Base(s.FilePath))
	}
	cx.forget(s.ID)
	return nil
}

// DeleteProject deletes every session whose working directory is p. It
// stops at the first failure; sessions deleted before that stay deleted.
func (cx *CodexDir) DeleteProject(p Project) error {
	sessions, err := cx.GetProjectSessions(p)
	if err != nil {
		return err
	}
	for i, s := range sessions {
		if err := cx.DeleteSession(s); err != nil {
			return fmt.Errorf("%d/%d: %w", i+1, len(sessions), err)
		}
	}
	return nil
}

// ArchiveDir is where `codex archive` moves rollouts.
func (cx *CodexDir) ArchiveDir(Session) string {
	return filepath.Join(cx.Path, "archived_sessions")
}

// ArchiveSession archives s with `codex archive`; the rollout moves to
// archived_sessions/ and drops out of the session list.
func (cx *CodexDir) ArchiveSession(s Session) error {
	if err := runCodex(cx.Path, "archive", s.ID); err != nil {
		return err
	}
	cx.forget(s.ID)
	return nil
}

// forget drops a session from the cached scan so lists reloaded from this
// CodexDir no longer show it.
func (cx *CodexDir) forget(id string) {
	cx.load() // make sure the cache is filled before editing it
	for i, s := range cx.sessions {
		if s.ID == id {
			cx.sessions = append(cx.sessions[:i], cx.sessions[i+1:]...)
			break
		}
	}
	delete(cx.usage, id)
}

// codexTimeout bounds one codex CLI call; delete/archive normally take well
// under a second.
const codexTimeout = 30 * time.Second

// runCodex runs `codex <args>` against the Codex home dir and turns a
// failure into an error carrying codex's own message.
func runCodex(home string, args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), codexTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "codex", args...)
	cmd.Env = append(os.Environ(), "CODEX_HOME="+home)
	cmd.Stdin = nil
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = io.Discard
	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("codex %s timed out", args[0])
	}
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		msg = strings.TrimPrefix(msg, "Error: ")
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("codex %s: %s", args[0], msg)
	}
	return nil
}

// LaunchArgs opens codex in projectPath. Codex has no option to name the
// terminal tab, so title is ignored.
func (cx *CodexDir) LaunchArgs(projectPath, sessionID, title string) (string, []string) {
	if sessionID != "" {
		return "codex", []string{"resume", sessionID}
	}
	return "codex", []string{"-C", projectPath}
}

// load scans every rollout once per CodexDir.
func (cx *CodexDir) load() ([]Session, error) {
	cx.once.Do(func() {
		titles := cx.readTitles()
		root := filepath.Join(cx.Path, "sessions")
		var files []string
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil // skip unreadable parts
			}
			if !d.IsDir() && strings.HasPrefix(d.Name(), "rollout-") && strings.HasSuffix(d.Name(), ".jsonl") {
				files = append(files, path)
			}
			return nil
		})
		if err != nil && !os.IsNotExist(err) {
			cx.err = err
			return
		}

		results := make([]codexRollout, len(files))
		var wg sync.WaitGroup
		sem := make(chan struct{}, 8)
		for i, f := range files {
			wg.Add(1)
			go func(i int, f string) {
				defer wg.Done()
				sem <- struct{}{}
				results[i] = parseCodexRollout(f)
				<-sem
			}(i, f)
		}
		wg.Wait()

		cx.usage = make(map[string]Usage)
		for _, r := range results {
			if !r.ok {
				continue
			}
			if t := titles[r.session.ID]; t != "" {
				r.session.FirstPrompt = t
			}
			cx.sessions = append(cx.sessions, r.session)
			cx.usage[r.session.ID] = r.usage
		}
		sort.Slice(cx.sessions, func(i, j int) bool {
			return cx.sessions[i].Modified.After(cx.sessions[j].Modified)
		})
	})
	return cx.sessions, cx.err
}

// readTitles reads session_index.jsonl: one line per rename, the last one for
// an id wins.
func (cx *CodexDir) readTitles() map[string]string {
	titles := make(map[string]string)
	f, err := os.Open(filepath.Join(cx.Path, "session_index.jsonl"))
	if err != nil {
		return titles
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		var e struct {
			ID   string `json:"id"`
			Name string `json:"thread_name"`
		}
		if json.Unmarshal(sc.Bytes(), &e) == nil && e.ID != "" && strings.TrimSpace(e.Name) != "" {
			titles[e.ID] = strings.TrimSpace(e.Name)
		}
	}
	return titles
}

type codexRollout struct {
	session Session
	usage   Usage
	ok      bool
}

// codexLine is the part of a rollout line the manager needs.
type codexLine struct {
	Timestamp string          `json:"timestamp"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}

type codexPayload struct {
	Type string `json:"type"`
	// session_meta
	ID        string `json:"id"`
	CWD       string `json:"cwd"`
	Timestamp string `json:"timestamp"`
	Git       *struct {
		Branch string `json:"branch"`
	} `json:"git"`
	// response_item message
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
	// event_msg token_count
	Info *struct {
		Total codexTokenTotals `json:"total_token_usage"`
	} `json:"info"`
}

type codexTokenTotals struct {
	Input  int64 `json:"input_tokens"`
	Cached int64 `json:"cached_input_tokens"`
	Write  int64 `json:"cache_write_input_tokens"`
	Output int64 `json:"output_tokens"`
}

// CodexTokenSample is a timestamped snapshot of cumulative model usage.
type CodexTokenSample struct {
	At            time.Time
	Input, Output int64
}

// ReadCodexTokenSamples reads the historical counters from a Codex rollout.
// Input includes cached input, consistent with the app-server counters.
func ReadCodexTokenSamples(path string) ([]CodexTokenSample, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var samples []CodexTokenSample
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 16*1024*1024)
	for sc.Scan() {
		if !bytes.Contains(sc.Bytes(), []byte(`"token_count"`)) {
			continue
		}
		var line codexLine
		if json.Unmarshal(sc.Bytes(), &line) != nil || line.Type != "event_msg" {
			continue
		}
		var p codexPayload
		if json.Unmarshal(line.Payload, &p) != nil || p.Type != "token_count" || p.Info == nil {
			continue
		}
		at, err := time.Parse(time.RFC3339Nano, line.Timestamp)
		if err == nil {
			samples = append(samples, CodexTokenSample{At: at, Input: p.Info.Total.Input, Output: p.Info.Total.Output})
		}
	}
	return samples, sc.Err()
}

func parseCodexRollout(path string) codexRollout {
	var r codexRollout
	info, err := os.Stat(path)
	if err != nil {
		return r
	}
	f, err := os.Open(path)
	if err != nil {
		return r
	}
	defer f.Close()

	s := &r.session
	s.FilePath = path
	s.Size = info.Size()
	s.Modified = info.ModTime()

	var first, last time.Time
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 16*1024*1024)
	for sc.Scan() {
		var line codexLine
		if json.Unmarshal(sc.Bytes(), &line) != nil {
			continue
		}
		if t, err := time.Parse(time.RFC3339Nano, line.Timestamp); err == nil {
			if first.IsZero() {
				first = t
			}
			last = t
		}
		switch line.Type {
		case "session_meta":
			var p codexPayload
			if json.Unmarshal(line.Payload, &p) != nil {
				continue
			}
			s.ID, s.ProjectPath = p.ID, p.CWD
			if t, err := time.Parse(time.RFC3339Nano, p.Timestamp); err == nil {
				s.Created = t
			}
			if p.Git != nil {
				s.GitBranch = p.Git.Branch
			}
			r.ok = s.ID != ""
		case "response_item":
			var p codexPayload
			if json.Unmarshal(line.Payload, &p) != nil || p.Type != "message" {
				continue
			}
			if p.Role != "user" && p.Role != "assistant" {
				continue
			}
			text := codexText(p.Content)
			if p.Role == "user" && isCodexInjected(text) {
				continue
			}
			s.MessageCount++
			if p.Role == "user" && s.FirstPrompt == "" {
				s.FirstPrompt = strings.Join(strings.Fields(text), " ")
			}
		case "event_msg":
			var p codexPayload
			if json.Unmarshal(line.Payload, &p) != nil || p.Type != "token_count" || p.Info == nil {
				continue
			}
			// running totals: the last one is the session total
			t := p.Info.Total
			r.usage = Usage{
				InputTokens:      t.Input - t.Cached,
				CacheReadTokens:  t.Cached,
				CacheWriteTokens: t.Write,
				OutputTokens:     t.Output,
			}
		}
	}
	if s.Created.IsZero() {
		s.Created = first
	}
	if !first.IsZero() && last.After(first) {
		r.usage.Duration = last.Sub(first)
	}
	return r
}

// codexText joins the text parts of a message's content array.
func codexText(raw json.RawMessage) string {
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &parts) != nil {
		return ""
	}
	var out []string
	for _, p := range parts {
		if t := strings.TrimSpace(p.Text); t != "" {
			out = append(out, t)
		}
	}
	return strings.Join(out, "\n")
}

// isCodexInjected reports whether a "user" message was written by Codex
// itself (environment context, AGENTS.md instructions, ...) rather than typed
// by the user.
func isCodexInjected(text string) bool {
	t := strings.TrimSpace(text)
	if t == "" {
		return true
	}
	for _, prefix := range []string{"<environment_context>", "<user_instructions>", "<turn_aborted>", "# AGENTS.md instructions"} {
		if strings.HasPrefix(t, prefix) {
			return true
		}
	}
	return false
}

// codexInstructionFiles are the instruction files Codex looks for, in a
// folder. Only the ones that exist are listed; the manager does not try to
// model which one wins.
var codexInstructionFiles = []string{"AGENTS.override.md", "AGENTS.md"}

// GetMemories lists the AGENTS.md files Codex reads: the global one in the
// Codex home and one per project root. Codex's own memories_1.sqlite is an
// internal database and is not read.
func (cx *CodexDir) GetMemories() ([]Memory, error) {
	var out []Memory
	for _, name := range codexInstructionFiles {
		if m, ok := newMemory(MemoryUser, "", filepath.Join(cx.Path, name)); ok {
			out = append(out, m)
		}
	}
	projects, err := cx.GetProjects()
	if err != nil {
		return out, err
	}
	seen := map[string]bool{}
	for _, p := range projects {
		dir := filepath.Clean(p.Path)
		// a session started in the Codex home is not a project
		if dir == "" || seen[projectKey(dir)] || projectKey(dir) == projectKey(cx.Path) {
			continue
		}
		seen[projectKey(dir)] = true
		for _, name := range codexInstructionFiles {
			if m, ok := newMemory(MemoryProject, p.Path, filepath.Join(dir, name)); ok {
				out = append(out, m)
			}
		}
	}
	return out, nil
}

// GetSessions returns all sessions, most recent first.
func (cx *CodexDir) GetSessions() ([]Session, error) {
	sessions, err := cx.load()
	return append([]Session(nil), sessions...), err
}

// GetProjects groups the sessions by working directory.
func (cx *CodexDir) GetProjects() ([]Project, error) {
	sessions, err := cx.load()
	if err != nil {
		return nil, err
	}
	byKey := make(map[string]*Project)
	var order []string
	for _, s := range sessions {
		key := projectKey(s.ProjectPath)
		p, ok := byKey[key]
		if !ok {
			p = &Project{Path: s.ProjectPath, EncodedName: key}
			byKey[key] = p
			order = append(order, key)
		}
		p.SessionCount++
		p.TotalSize += s.Size
		if s.Modified.After(p.LastActivity) {
			p.LastActivity = s.Modified
		}
	}
	projects := make([]Project, 0, len(order))
	for _, k := range order {
		projects = append(projects, *byKey[k])
	}
	return projects, nil
}

// projectKey makes paths that differ only in case or a trailing separator
// count as one project (Windows paths are case-insensitive).
func projectKey(path string) string {
	p := filepath.Clean(path)
	if os.PathSeparator == '\\' {
		p = strings.ToLower(p)
	}
	return p
}

func (cx *CodexDir) GetProjectSessions(p Project) ([]Session, error) {
	sessions, err := cx.load()
	if err != nil {
		return nil, err
	}
	key := projectKey(p.Path)
	var out []Session
	for _, s := range sessions {
		if projectKey(s.ProjectPath) == key {
			out = append(out, s)
		}
	}
	return out, nil
}

func (cx *CodexDir) FillUsage(projects []Project) {
	sessions, _ := cx.load()
	for i := range projects {
		key := projectKey(projects[i].Path)
		var u Usage
		for _, s := range sessions {
			if projectKey(s.ProjectPath) == key {
				u.Add(cx.usage[s.ID])
			}
		}
		projects[i].Usage = u
	}
}

func (cx *CodexDir) GetOverview() (Overview, error) {
	var o Overview
	projects, err := cx.GetProjects()
	if err != nil {
		return o, err
	}
	o.ProjectCount = len(projects)
	for _, p := range projects {
		o.TotalSize += p.TotalSize
	}
	sessions, _ := cx.load()
	o.SessionCount = len(sessions)
	o.RecentSessions = sessions[:min(5, len(sessions))]
	if memories, err := cx.GetMemories(); err == nil {
		o.MemoryCount = len(memories)
	}
	return o, nil
}

// GetSessionMessages returns the user and assistant messages of s, skipping
// what Codex injects on its own.
func (cx *CodexDir) GetSessionMessages(s Session, maxMessages int) ([]SessionMessage, error) {
	f, err := os.Open(s.FilePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var msgs []SessionMessage
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 16*1024*1024)
	for sc.Scan() {
		if maxMessages > 0 && len(msgs) >= maxMessages {
			break
		}
		var line codexLine
		if json.Unmarshal(sc.Bytes(), &line) != nil || line.Type != "response_item" {
			continue
		}
		var p codexPayload
		if json.Unmarshal(line.Payload, &p) != nil || p.Type != "message" {
			continue
		}
		if p.Role != "user" && p.Role != "assistant" {
			continue
		}
		text := codexText(p.Content)
		if text == "" || (p.Role == "user" && isCodexInjected(text)) {
			continue
		}
		m := SessionMessage{Role: p.Role, Content: text}
		m.Time, _ = time.Parse(time.RFC3339Nano, line.Timestamp)
		msgs = append(msgs, m)
	}
	return msgs, sc.Err()
}
