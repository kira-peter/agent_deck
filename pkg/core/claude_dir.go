package core

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unicode"
)

// ClaudeDir represents the .claude directory and provides methods to read its contents
type ClaudeDir struct {
	Path string
}

// NewClaudeDir creates a new ClaudeDir instance
func NewClaudeDir(path string) *ClaudeDir {
	return &ClaudeDir{Path: path}
}

// EncodePath converts a file system path to the encoded format used in .claude/projects/
// Examples: D:\jarvis -> D--jarvis, C:\Users\TU -> C--Users-TU
func EncodePath(path string) string {
	normalized := strings.ReplaceAll(path, "\\", "/")
	var segments []string
	for _, seg := range strings.Split(normalized, "/") {
		if seg == "" {
			continue
		}
		segments = append(segments, encodeDirSegment(seg))
	}
	return strings.Join(segments, "-")
}

// encodeDirSegment encodes a single path segment the way Claude Code names
// project directories: every character that is not a letter or digit becomes a
// dash. "config_tool" -> "config-tool", "ls-server" -> "ls-server".
func encodeDirSegment(name string) string {
	var b strings.Builder
	b.Grow(len(name))
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	return b.String()
}

// DecodePath attempts to convert an encoded path back to a file system path
// This is a best-effort approach and may not always be accurate
func DecodePath(encoded string) string {
	// Runs of dashes (e.g. the "D--" drive prefix) collapse to a single
	// separator, so drop the empty segments they produce.
	var parts []string
	for _, p := range strings.Split(encoded, "-") {
		if p != "" {
			parts = append(parts, p)
		}
	}

	// On Windows, try to reconstruct the drive letter
	if runtime.GOOS == "windows" {
		if len(parts) > 0 && len(parts[0]) == 1 {
			// Looks like a drive letter
			return parts[0] + ":\\" + strings.Join(parts[1:], "\\")
		}
	}
	// On Unix, just join the parts with slashes
	return "/" + strings.Join(parts, "/")
}

// DecodePathByExploration explores the file system to find the real path
// corresponding to an encoded project name
func DecodePathByExploration(encoded string) string {
	if runtime.GOOS != "windows" {
		// For Unix systems, use simpler logic for now
		return DecodePath(encoded)
	}

	// Extract drive letter and remaining parts
	// Format: "D--jarvis-claude-manager"
	if len(encoded) < 3 || encoded[1] != '-' || encoded[2] != '-' {
		return DecodePath(encoded) // Fallback
	}

	driveLetter := string(encoded[0])
	remaining := encoded[3:] // Skip "D--" to get "jarvis-claude-manager"

	// Start exploring from the drive root
	result := explorePathRecursive(driveLetter+":\\", remaining)
	if result != "" {
		return result
	}

	// Fallback to simple decode
	return DecodePath(encoded)
}

// explorePathRecursive recursively explores directories to find matching path
func explorePathRecursive(currentPath string, remaining string) string {
	// Base case: no more parts to match
	if remaining == "" {
		return strings.TrimSuffix(currentPath, "\\")
	}

	// List directories in current path
	entries, err := os.ReadDir(currentPath)
	if err != nil {
		return "" // Can't read directory
	}

	// Collect all matching directories with their encoded names
	type candidate struct {
		dirName      string
		encodedName  string
		newRemaining string
	}
	var candidates []candidate

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dirName := entry.Name()

		// The encoded form replaces every non-alphanumeric character with a
		// dash, so "config_tool" appears in the encoded path as "config-tool".
		encodedName := encodeDirSegment(dirName)

		// Check if remaining starts with this encoded name
		if strings.HasPrefix(remaining, encodedName) {
			// Check if it's followed by a dash or is the end
			afterMatch := remaining[len(encodedName):]
			if afterMatch == "" || strings.HasPrefix(afterMatch, "-") {
				candidates = append(candidates, candidate{
					dirName:      dirName,
					encodedName:  encodedName,
					newRemaining: strings.TrimPrefix(afterMatch, "-"),
				})
			}
		}
	}

	// Sort candidates by encoded name length (longest first) for greedy matching
	// This ensures we match "ls-server" before "ls" if both exist
	for i := 0; i < len(candidates); i++ {
		for j := i + 1; j < len(candidates); j++ {
			if len(candidates[j].encodedName) > len(candidates[i].encodedName) {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}

	// Try each candidate (longest first)
	for _, cand := range candidates {
		newPath := filepath.Join(currentPath, cand.dirName)
		result := explorePathRecursive(newPath, cand.newRemaining)
		if result != "" {
			return result
		}
	}

	return "" // No match found
}

// encodeFileName encodes a single file/directory name (not a full path)
// Note: This preserves existing dashes in directory names
func encodeFileName(name string) string {
	// Only replace path separators
	encoded := strings.ReplaceAll(name, "\\", "-")
	encoded = strings.ReplaceAll(encoded, "/", "-")
	return encoded
}

// GetProjects returns a list of all projects in the .claude directory
func (cd *ClaudeDir) GetProjects() ([]Project, error) {
	projectsDir := filepath.Join(cd.Path, "projects")

	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Project{}, nil
		}
		return nil, err
	}

	var projects []Project
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		encodedName := entry.Name()
		projectPath := filepath.Join(projectsDir, encodedName)

		project, err := cd.LoadProject(encodedName, projectPath)
		if err != nil {
			// Skip projects that can't be loaded
			continue
		}

		projects = append(projects, project)
	}

	return projects, nil
}

// LoadProject loads a single project's metadata
func (cd *ClaudeDir) LoadProject(encodedName, projectPath string) (Project, error) {
	project := Project{
		EncodedName: encodedName,
		Path:        "", // Will be filled from sessions-index.json
	}

	// Try to load sessions-index.json for metadata and real project path
	indexPath := filepath.Join(projectPath, "sessions-index.json")
	if data, err := os.ReadFile(indexPath); err == nil {
		var sessions []SessionIndex
		if err := json.Unmarshal(data, &sessions); err == nil {
			project.SessionCount = len(sessions)

			// Get real project path from the first session
			if len(sessions) > 0 && sessions[0].ProjectPath != "" {
				project.Path = sessions[0].ProjectPath
			}

			// Find last activity time
			for _, s := range sessions {
				if s.Modified != "" {
					if t, err := time.Parse(time.RFC3339, s.Modified); err == nil {
						if t.After(project.LastActivity) {
							project.LastActivity = t
						}
					}
				}
			}
		}
	}

	// Fallback to exploration-based decode if we couldn't get the real path from sessions-index
	if project.Path == "" {
		project.Path = DecodePathByExploration(encodedName)
	}

	// Calculate total size by scanning .jsonl files
	entries, err := os.ReadDir(projectPath)
	if err != nil {
		return project, err
	}

	var (
		jsonlCount int
		latestMod  time.Time
	)

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".jsonl") {
			info, err := entry.Info()
			if err == nil {
				project.TotalSize += info.Size()
				jsonlCount++
				if info.ModTime().After(latestMod) {
					latestMod = info.ModTime()
				}
			}
		}
	}

	// Newer Claude Code versions no longer write sessions-index.json, so fall
	// back to what the .jsonl files themselves tell us.
	if project.SessionCount == 0 {
		project.SessionCount = jsonlCount
	}
	if project.LastActivity.IsZero() {
		project.LastActivity = latestMod
	}

	return project, nil
}

// GetSessions returns all sessions across all projects
func (cd *ClaudeDir) GetSessions() ([]Session, error) {
	projects, err := cd.GetProjects()
	if err != nil {
		return nil, err
	}

	var allSessions []Session
	for _, project := range projects {
		sessions, err := cd.GetEncodedProjectSessions(project.EncodedName)
		if err != nil {
			continue
		}
		allSessions = append(allSessions, sessions...)
	}

	return allSessions, nil
}

// GetEncodedProjectSessions returns all sessions for the project stored in
// projects/<encodedName>.
// It first tries sessions-index.json; if absent, falls back to scanning .jsonl files directly.
func (cd *ClaudeDir) GetEncodedProjectSessions(encodedName string) ([]Session, error) {
	projectPath := filepath.Join(cd.Path, "projects", encodedName)
	indexPath := filepath.Join(projectPath, "sessions-index.json")

	if data, err := os.ReadFile(indexPath); err == nil {
		var sessionsIndex []SessionIndex
		if err := json.Unmarshal(data, &sessionsIndex); err != nil {
			return nil, err
		}

		var sessions []Session
		for _, si := range sessionsIndex {
			session := Session{
				ID:           si.SessionID,
				ProjectPath:  si.ProjectPath,
				FirstPrompt:  si.FirstPrompt,
				MessageCount: si.MessageCount,
				GitBranch:    si.GitBranch,
				FilePath:     si.FullPath,
			}
			if si.Created != "" {
				if t, err := time.Parse(time.RFC3339, si.Created); err == nil {
					session.Created = t
				}
			}
			if si.Modified != "" {
				if t, err := time.Parse(time.RFC3339, si.Modified); err == nil {
					session.Modified = t
				}
			}
			if info, err := os.Stat(si.FullPath); err == nil {
				session.Size = info.Size()
			}
			sessionDir := filepath.Join(projectPath, si.SessionID)
			if _, err := os.Stat(filepath.Join(sessionDir, "subagents")); err == nil {
				session.HasSubagents = true
			}
			sessions = append(sessions, session)
		}
		return sessions, nil
	}

	// Fallback: scan .jsonl files directly
	return cd.loadSessionsFromJsonl(projectPath, encodedName)
}

// loadSessionsFromJsonl scans .jsonl files in a project directory to build session list.
func (cd *ClaudeDir) loadSessionsFromJsonl(projectPath, encodedName string) ([]Session, error) {
	entries, err := os.ReadDir(projectPath)
	if err != nil {
		return nil, err
	}

	var sessions []Session
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}

		filePath := filepath.Join(projectPath, entry.Name())
		session, err := cd.parseJsonlSession(filePath, projectPath, encodedName)
		if err != nil {
			continue
		}
		sessions = append(sessions, session)
	}
	return sessions, nil
}

// parseJsonlSession reads a single .jsonl file and extracts session metadata.
func (cd *ClaudeDir) parseJsonlSession(filePath, projectPath, encodedName string) (Session, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return Session{}, err
	}

	session := Session{
		FilePath: filePath,
		Size:     info.Size(),
		Modified: info.ModTime(),
	}

	file, err := os.Open(filePath)
	if err != nil {
		return Session{}, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 2*1024*1024), 2*1024*1024)

	type rawLine struct {
		Type      string `json:"type"`
		SessionID string `json:"sessionId"`
		Timestamp string `json:"timestamp"`
		GitBranch string `json:"gitBranch"`
		CWD       string `json:"cwd"`
		Message   struct {
			Role    string          `json:"role"`
			Content json.RawMessage `json:"content"`
		} `json:"message"`
	}

	firstUserSeen := false
	for scanner.Scan() {
		var raw rawLine
		if err := json.Unmarshal(scanner.Bytes(), &raw); err != nil {
			continue
		}

		// Pick up session ID from any line that carries it
		if session.ID == "" && raw.SessionID != "" {
			session.ID = raw.SessionID
		}

		if raw.Type == "user" || raw.Type == "assistant" {
			session.MessageCount++

			if raw.Type == "user" && !firstUserSeen {
				firstUserSeen = true
				if raw.CWD != "" {
					session.ProjectPath = raw.CWD
				}
				if raw.GitBranch != "" {
					session.GitBranch = raw.GitBranch
				}
				if raw.Timestamp != "" {
					if t, err := time.Parse(time.RFC3339, raw.Timestamp); err == nil {
						session.Created = t
					}
				}
				session.FirstPrompt = extractTextContent(raw.Message.Content)
			}
		}
	}

	// Derive session ID from filename if not found in content
	if session.ID == "" {
		base := strings.TrimSuffix(filepath.Base(filePath), ".jsonl")
		session.ID = base
	}

	// Fallback project path
	if session.ProjectPath == "" {
		session.ProjectPath = DecodePathByExploration(encodedName)
	}

	// Check for subagents directory
	if session.ID != "" {
		subagentsDir := filepath.Join(projectPath, session.ID, "subagents")
		if _, err := os.Stat(subagentsDir); err == nil {
			session.HasSubagents = true
		}
	}

	return session, nil
}

// GetActiveSessions returns currently running Claude Code sessions
func (cd *ClaudeDir) GetActiveSessions() ([]ActiveSession, error) {
	sessionsDir := filepath.Join(cd.Path, "sessions")

	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []ActiveSession{}, nil
		}
		return nil, err
	}

	var activeSessions []ActiveSession
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		sessionPath := filepath.Join(sessionsDir, entry.Name())
		data, err := os.ReadFile(sessionPath)
		if err != nil {
			continue
		}

		var asf ActiveSessionFile
		if err := json.Unmarshal(data, &asf); err != nil {
			continue
		}

		activeSessions = append(activeSessions, ActiveSession{
			PID:       asf.PID,
			SessionID: asf.SessionID,
			CWD:       asf.CWD,
			StartedAt: time.Unix(0, asf.StartedAt*int64(time.Millisecond)),
			Version:   asf.Version,
			Status:    asf.Status,
			Kind:      asf.Kind,
		})
	}

	return activeSessions, nil
}

// GetMemories returns every memory file Claude Code reads: the user-level
// CLAUDE.md, each project's CLAUDE.md, and the auto-memory folder
// (MEMORY.md index plus topic files) under projects/<project>/memory.
func (cd *ClaudeDir) GetMemories() ([]Memory, error) {
	var memories []Memory

	if m, ok := newMemory(MemoryUser, "", filepath.Join(cd.Path, "CLAUDE.md")); ok {
		memories = append(memories, m)
	}

	projectsDir := filepath.Join(cd.Path, "projects")
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return memories, nil
		}
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		projectPath := DecodePathByExploration(entry.Name())

		if m, ok := newMemory(MemoryProject, projectPath, filepath.Join(projectPath, "CLAUDE.md")); ok {
			memories = append(memories, m)
		}

		memDir := filepath.Join(projectsDir, entry.Name(), "memory")
		files, err := os.ReadDir(memDir)
		if err != nil {
			continue
		}
		for _, f := range files {
			if f.IsDir() || !strings.HasSuffix(strings.ToLower(f.Name()), ".md") {
				continue
			}
			kind := MemoryTopic
			if f.Name() == "MEMORY.md" {
				kind = MemoryAuto
			}
			if m, ok := newMemory(kind, projectPath, filepath.Join(memDir, f.Name())); ok {
				memories = append(memories, m)
			}
		}
	}

	return memories, nil
}

// newMemory stats path and fills a Memory; ok is false if it is not a file.
func newMemory(kind MemoryKind, projectPath, path string) (Memory, bool) {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return Memory{}, false
	}
	m := Memory{
		Kind:         kind,
		Name:         filepath.Base(path),
		ProjectPath:  projectPath,
		FilePath:     path,
		Size:         info.Size(),
		LastModified: info.ModTime(),
	}
	if preview, err := readFirstLines(path, 5); err == nil {
		m.Preview = preview
	}
	if count, err := countMemoryEntries(path); err == nil {
		m.EntryCount = count
	}
	return m, true
}

// GetOverview returns summary statistics
func (cd *ClaudeDir) GetOverview() (Overview, error) {
	overview := Overview{}

	// Get projects
	projects, err := cd.GetProjects()
	if err == nil {
		overview.ProjectCount = len(projects)
		for _, p := range projects {
			overview.TotalSize += p.TotalSize
		}
	}

	// Get sessions
	sessions, err := cd.GetSessions()
	if err == nil {
		overview.SessionCount = len(sessions)

		// Sort by modified time and take the 5 most recent
		// For now, just take the first 5
		if len(sessions) > 5 {
			overview.RecentSessions = sessions[:5]
		} else {
			overview.RecentSessions = sessions
		}
	}

	// Get active sessions
	activeSessions, err := cd.GetActiveSessions()
	if err == nil {
		overview.ActiveSessions = activeSessions
	}

	// Get memories
	memories, err := cd.GetMemories()
	if err == nil {
		overview.MemoryCount = len(memories)
	}

	return overview, nil
}

// Helper functions

func readFirstLines(filePath string, n int) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for i := 0; i < n && scanner.Scan(); i++ {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return strings.Join(lines, "\n"), nil
}

func countMemoryEntries(filePath string) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	count := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// Count lines starting with "- [" (markdown list items with links)
		if strings.HasPrefix(strings.TrimSpace(line), "- [") {
			count++
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, err
	}

	return count, nil
}

// SearchSessions searches for sessions containing the given keyword
func (cd *ClaudeDir) SearchSessions(keyword string) ([]Session, error) {
	sessions, err := cd.GetSessions()
	if err != nil {
		return nil, err
	}

	var results []Session
	keyword = strings.ToLower(keyword)

	for _, session := range sessions {
		// Search in first prompt
		if strings.Contains(strings.ToLower(session.FirstPrompt), keyword) {
			results = append(results, session)
			continue
		}

		// Search in project path
		if strings.Contains(strings.ToLower(session.ProjectPath), keyword) {
			results = append(results, session)
			continue
		}

		// Search in session ID
		if strings.Contains(strings.ToLower(session.ID), keyword) {
			results = append(results, session)
		}
	}

	return results, nil
}

// GetSessionContent reads the first N lines of a session's .jsonl file
func (cd *ClaudeDir) GetSessionContent(filePath string, maxLines int) (string, error) {
	return readFirstLines(filePath, maxLines)
}

// GetSessionMessagesFromFile parses a session .jsonl file and returns user/assistant messages.
// It handles both plain-string content (user) and array content blocks (assistant).
func (cd *ClaudeDir) GetSessionMessagesFromFile(filePath string, maxMessages int) ([]SessionMessage, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 2*1024*1024), 2*1024*1024) // 2MB per line

	var messages []SessionMessage

	for scanner.Scan() {
		if maxMessages > 0 && len(messages) >= maxMessages {
			break
		}
		line := scanner.Bytes()

		var raw struct {
			Type    string `json:"type"`
			Message struct {
				Role    string          `json:"role"`
				Content json.RawMessage `json:"content"`
			} `json:"message"`
			Timestamp string `json:"timestamp"`
		}

		if err := json.Unmarshal(line, &raw); err != nil {
			continue
		}

		if raw.Type != "user" && raw.Type != "assistant" {
			continue
		}

		content := extractTextContent(raw.Message.Content)
		if content == "" {
			continue
		}

		msg := SessionMessage{
			Role:    raw.Message.Role,
			Content: content,
		}
		if raw.Timestamp != "" {
			msg.Time, _ = time.Parse(time.RFC3339, raw.Timestamp)
		}

		messages = append(messages, msg)
	}

	return messages, scanner.Err()
}

// extractTextContent extracts displayable text from a JSON content field.
// Content can be a plain string or an array of typed content blocks.
func extractTextContent(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	// Plain string (common for user messages)
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return strings.TrimSpace(s)
	}
	// Array of content blocks (assistant messages, or user tool_result arrays)
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return ""
	}
	var parts []string
	for _, b := range blocks {
		if b.Type == "text" && strings.TrimSpace(b.Text) != "" {
			parts = append(parts, strings.TrimSpace(b.Text))
		}
	}
	return strings.Join(parts, "\n")
}

// Utility function for formatting sizes
func FormatSize(bytes int64) string {
	switch GetConfig().GetDisplay().SizeUnit {
	case "KB":
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	case "MB":
		return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
	}

	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// Utility function for formatting time relative to now
func FormatRelativeTime(t time.Time) string {
	// Check for zero/uninitialized time
	if t.IsZero() || t.Year() == 1 {
		return "N/A"
	}
	if GetConfig().GetDisplay().DateFormat == "absolute" {
		return t.Local().Format("2006-01-02 15:04")
	}

	duration := time.Since(t)

	if duration < time.Minute {
		return "just now"
	} else if duration < time.Hour {
		minutes := int(duration.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else if duration < 7*24*time.Hour {
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	} else {
		return t.Format("2006-01-02")
	}
}
