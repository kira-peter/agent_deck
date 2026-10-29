package core

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Usage aggregates token consumption, code changes and working time
// recovered from session .jsonl files (the data behind Claude Code's /usage).
type Usage struct {
	InputTokens      int64
	OutputTokens     int64
	CacheReadTokens  int64
	CacheWriteTokens int64
	LinesAdded       int
	LinesRemoved     int
	Duration         time.Duration // sum of per-turn durations
}

// Add accumulates o into u.
func (u *Usage) Add(o Usage) {
	u.InputTokens += o.InputTokens
	u.OutputTokens += o.OutputTokens
	u.CacheReadTokens += o.CacheReadTokens
	u.CacheWriteTokens += o.CacheWriteTokens
	u.LinesAdded += o.LinesAdded
	u.LinesRemoved += o.LinesRemoved
	u.Duration += o.Duration
}

type usageTokens struct {
	Input      int64 `json:"input_tokens"`
	Output     int64 `json:"output_tokens"`
	CacheRead  int64 `json:"cache_read_input_tokens"`
	CacheWrite int64 `json:"cache_creation_input_tokens"`
}

type usageLine struct {
	Type       string `json:"type"`
	Subtype    string `json:"subtype"`
	DurationMs int64  `json:"durationMs"`
	Message    struct {
		ID    string       `json:"id"`
		Usage *usageTokens `json:"usage"`
	} `json:"message"`
	ToolUseResult json.RawMessage `json:"toolUseResult"`
}

type toolPatchResult struct {
	Type            string `json:"type"`
	Content         string `json:"content"`
	StructuredPatch []struct {
		Lines []string `json:"lines"`
	} `json:"structuredPatch"`
}

var (
	markAssistant = []byte(`"type":"assistant"`)
	markTurnEnd   = []byte(`"subtype":"turn_duration"`)
	markPatch     = []byte(`"structuredPatch"`)
)

// ScanSessionUsage computes the usage of a single session .jsonl file.
func ScanSessionUsage(filePath string) (Usage, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return Usage{}, err
	}
	defer file.Close()

	var u Usage
	// One assistant message can be split over several lines that all carry the
	// same usage, so keep only the last usage seen per message id.
	byMessage := make(map[string]usageTokens)

	reader := bufio.NewReaderSize(file, 256*1024)
	for {
		line, readErr := reader.ReadBytes('\n')
		if len(line) > 0 {
			scanUsageLine(line, &u, byMessage)
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return u, readErr
		}
	}

	for _, t := range byMessage {
		u.InputTokens += t.Input
		u.OutputTokens += t.Output
		u.CacheReadTokens += t.CacheRead
		u.CacheWriteTokens += t.CacheWrite
	}
	return u, nil
}

func scanUsageLine(line []byte, u *Usage, byMessage map[string]usageTokens) {
	isAssistant := bytes.Contains(line, markAssistant)
	isTurnEnd := bytes.Contains(line, markTurnEnd)
	isPatch := bytes.Contains(line, markPatch)
	if !isAssistant && !isTurnEnd && !isPatch {
		return
	}

	var l usageLine
	if err := json.Unmarshal(line, &l); err != nil {
		return
	}

	switch {
	case l.Type == "assistant" && l.Message.Usage != nil:
		if l.Message.ID != "" {
			byMessage[l.Message.ID] = *l.Message.Usage
		} else {
			t := l.Message.Usage
			u.InputTokens += t.Input
			u.OutputTokens += t.Output
			u.CacheReadTokens += t.CacheRead
			u.CacheWriteTokens += t.CacheWrite
		}
	case l.Subtype == "turn_duration":
		u.Duration += time.Duration(l.DurationMs) * time.Millisecond
	case isPatch && len(l.ToolUseResult) > 0 && l.ToolUseResult[0] == '{':
		var r toolPatchResult
		if err := json.Unmarshal(l.ToolUseResult, &r); err != nil {
			return
		}
		if len(r.StructuredPatch) == 0 && r.Type == "create" && r.Content != "" {
			u.LinesAdded += strings.Count(r.Content, "\n") + 1
			return
		}
		for _, hunk := range r.StructuredPatch {
			for _, s := range hunk.Lines {
				if strings.HasPrefix(s, "+") {
					u.LinesAdded++
				} else if strings.HasPrefix(s, "-") {
					u.LinesRemoved++
				}
			}
		}
	}
}

// GetProjectUsage sums the usage of every session file of a project.
func (cd *ClaudeDir) GetProjectUsage(encodedName string) Usage {
	projectPath := filepath.Join(cd.Path, "projects", encodedName)
	entries, err := os.ReadDir(projectPath)
	if err != nil {
		return Usage{}
	}

	var total Usage
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}
		if u, err := ScanSessionUsage(filepath.Join(projectPath, entry.Name())); err == nil {
			total.Add(u)
		}
	}
	return total
}

// FillUsage computes Usage for all projects concurrently.
func (cd *ClaudeDir) FillUsage(projects []Project) {
	var wg sync.WaitGroup
	for i := range projects {
		wg.Add(1)
		go func(p *Project) {
			defer wg.Done()
			p.Usage = cd.GetProjectUsage(p.EncodedName)
		}(&projects[i])
	}
	wg.Wait()
}

// FormatTokens renders a token count with a fixed "xxx.xxx" shape and a
// K/M/G unit, e.g. 94 -> "0.094K", 4400000 -> "4.400M".
func FormatTokens(n int64) string {
	f := float64(n)
	switch {
	case n >= 1_000_000_000:
		return fmt.Sprintf("%.3fG", f/1e9)
	case n >= 1_000_000:
		return fmt.Sprintf("%.3fM", f/1e6)
	default:
		return fmt.Sprintf("%.3fK", f/1e3)
	}
}

// FormatHours renders a duration as fractional hours with three decimals,
// e.g. 24m36s -> "0.410h".
func FormatHours(d time.Duration) string {
	return fmt.Sprintf("%.3fh", d.Hours())
}
