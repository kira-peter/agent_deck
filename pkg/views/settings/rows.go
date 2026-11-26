package settings

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kira-peter/agent_deck/pkg/core"
	"github.com/kira-peter/agent_deck/pkg/i18n"
)

type rowKind int

const (
	kindHeader rowKind = iota // section title, not selectable
	kindCycle                 // cycles through fixed choices (booleans included)
	kindText                  // free text / number input
	kindInfo                  // read-only summary (env, rules)
	kindOpen                  // opens a file or folder
	kindPick                  // opens the model picker
)

// row is one line of the settings page.
type row struct {
	kind  rowKind
	label string
	value string // already formatted for display

	// kindCycle / kindText
	apply   func(input string) error // stores the new value
	clear   func() error             // removes the value; nil = not clearable
	choices []string                 // kindCycle: the values offered on the wheel
	seed    string                   // kindText: initial content of the input box
	isInt   bool                     // kindText: input must be a whole number

	// kindOpen
	path string
}

func (r row) selectable() bool { return r.kind != kindHeader && r.kind != kindInfo }

// secretMarkers flags env names whose values must not be shown in clear.
var secretMarkers = []string{"TOKEN", "KEY", "SECRET", "PASSWORD"}

// maskSecret keeps the first and last 4 characters of a secret.
func maskSecret(name, value string) string {
	upper := strings.ToUpper(name)
	secret := false
	for _, m := range secretMarkers {
		if strings.Contains(upper, m) {
			secret = true
			break
		}
	}
	if !secret {
		return value
	}
	r := []rune(value)
	if len(r) <= 8 {
		return strings.Repeat("*", len(r))
	}
	return string(r[:4]) + strings.Repeat("*", min(len(r)-8, 12)) + string(r[len(r)-4:])
}

// rawString decodes a JSON string value; other types are shown as JSON text.
func rawString(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	return string(raw)
}

// indexOf returns the position of cur in choices, or 0 when it is not there.
func indexOf(choices []string, cur string) int {
	for i, c := range choices {
		if c == cur {
			return i
		}
	}
	return 0
}

// buildRows lays out the whole page from the current config and settings.
func (v *SettingsView) buildRows() []row {
	var rows []row
	cfg := v.config
	disp := cfg.GetDisplay()

	// ── AgentDeck ─────────────────────────────────────────────────
	rows = append(rows, row{kind: kindHeader, label: i18n.SetSecManager.Get()})

	saveDisp := func(fn func(*core.DisplayConfig)) error { return cfg.UpdateDisplay(fn) }
	cycleDisp := func(label string, cur string, choices []string, set func(*core.DisplayConfig, string)) row {
		return row{
			kind: kindCycle, label: label, value: cur,
			choices: choices,
			apply:   func(in string) error { return saveDisp(func(d *core.DisplayConfig) { set(d, in) }) },
		}
	}
	rows = append(rows,
		cycleDisp(i18n.SetDateFormat.Get(), disp.DateFormat, []string{"relative", "absolute"},
			func(d *core.DisplayConfig, s string) { d.DateFormat = s }),
		cycleDisp(i18n.SetSizeUnit.Get(), disp.SizeUnit, []string{"auto", "KB", "MB"},
			func(d *core.DisplayConfig, s string) { d.SizeUnit = s }),
		row{
			kind: kindText, label: i18n.SetItemsPerPage.Get(),
			value: strconv.Itoa(disp.ItemsPerPage), seed: strconv.Itoa(disp.ItemsPerPage), isInt: true,
			apply: func(in string) error {
				n, _ := strconv.Atoi(in)
				return saveDisp(func(d *core.DisplayConfig) { d.ItemsPerPage = max(n, 1) })
			},
		},
	)

	// Codex keeps its settings in config.toml; for now the page only offers
	// to open it.
	if cfg.GetSource() == core.SourceCodex {
		codexDir := cfg.GetCodexDir()
		configPath := filepath.Join(codexDir, "config.toml")
		rows = append(rows,
			row{kind: kindHeader, label: i18n.SetSecCodex.Get()},
			row{kind: kindOpen, label: i18n.SetOpenCodexConfig.Get(), value: configPath, path: configPath},
			row{kind: kindHeader, label: i18n.SetSecFiles.Get()},
			row{kind: kindOpen, label: i18n.SetOpenCodexDir.Get(), value: codexDir, path: codexDir},
			row{kind: kindOpen, label: i18n.SetOpenManager.Get(), value: core.GetConfigPath(), path: core.GetConfigPath()},
		)
		return rows
	}

	// ── Claude Code settings.json ──────────────────────────────────────
	rows = append(rows, row{kind: kindHeader, label: i18n.SetSecClaude.Get()})
	if v.settings == nil {
		rows = append(rows, row{kind: kindInfo, label: "", value: fmt.Sprintf(i18n.SetLoadFailed.Get(), v.loadErr)})
	} else {
		rows = append(rows, v.claudeRows()...)
	}

	// ── env ────────────────────────────────────────────────────────────
	if v.settings != nil {
		rows = append(rows, row{kind: kindHeader, label: i18n.SetSecEnv.Get()})
		envs := v.settings.Entries("env")
		if len(envs) == 0 {
			rows = append(rows, row{kind: kindInfo, value: i18n.SetNoEnv.Get()})
		}
		for _, e := range envs {
			key := e.Key
			kind := kindText
			if key == "ANTHROPIC_MODEL" {
				kind = kindPick
			}
			rows = append(rows, row{
				kind: kind, label: key,
				value: maskSecret(key, rawString(e.Value)),
				seed:  rawString(e.Value),
				apply: func(in string) error { return v.saveSetting("env."+key, in) },
				clear: func() error { return v.clearSetting("env." + key) },
			})
		}
	}

	// ── Files ──────────────────────────────────────────────────────────
	claudeDir := cfg.GetClaudeDir()
	rows = append(rows,
		row{kind: kindHeader, label: i18n.SetSecFiles.Get()},
		row{kind: kindOpen, label: i18n.SetOpenClaudeDir.Get(), value: claudeDir, path: claudeDir},
		row{kind: kindOpen, label: i18n.SetOpenSettings.Get(), value: filepath.Join(claudeDir, "settings.json"),
			path: filepath.Join(claudeDir, "settings.json")},
		row{kind: kindOpen, label: i18n.SetOpenManager.Get(), value: core.GetConfigPath(), path: core.GetConfigPath()},
	)
	return rows
}

// claudeRows are the well-known keys of settings.json. Keys that are not set
// show "(not set)" and are created on first edit.
func (v *SettingsView) claudeRows() []row {
	s := v.settings
	notSet := i18n.SetNotSet.Get()

	show := func(path string) (string, bool) {
		raw, ok := s.Lookup(path)
		if !ok {
			return notSet, false
		}
		return rawString(raw), true
	}

	text := func(label, path string) row {
		val, set := show(path)
		seed := ""
		if set {
			seed = val
		}
		return row{
			kind: kindText, label: label, value: val, seed: seed,
			apply: func(in string) error { return v.saveSetting(path, in) },
			clear: func() error { return v.clearSetting(path) },
		}
	}
	cycle := func(label, path string, choices []string) row {
		val, _ := show(path)
		return row{
			kind: kindCycle, label: label, value: val,
			choices: choices,
			apply:   func(in string) error { return v.saveSetting(path, in) },
			clear:   func() error { return v.clearSetting(path) },
		}
	}
	boolean := func(label, path string) row {
		val, _ := show(path)
		return row{
			kind: kindCycle, label: label, value: val,
			choices: []string{"true", "false"},
			apply:   func(in string) error { return v.saveSetting(path, in == "true") },
			clear:   func() error { return v.clearSetting(path) },
		}
	}

	bashLimit := text(i18n.SetBashOutput.Get(), "bashOutputMaxChars")
	bashLimit.isInt = true
	bashLimit.apply = func(in string) error {
		n, _ := strconv.Atoi(in)
		return v.saveSetting("bashOutputMaxChars", n)
	}

	rules := s.StringList("permissions.allow")
	allow := row{
		kind: kindOpen, label: i18n.SetAllowRules.Get(),
		value: fmt.Sprintf(i18n.SetEntries.Get(), len(rules)) + "  (" + i18n.SetExternal.Get() + ")",
		path:  s.Path,
	}

	return []row{
		text(i18n.SetModel.Get(), "model"),
		text(i18n.SetClaudeLang.Get(), "language"),
		cycle(i18n.SetTheme.Get(), "theme", []string{"dark", "light", "dark-daltonized", "light-daltonized"}),
		boolean(i18n.SetAutoCompact.Get(), "autoCompact"),
		boolean(i18n.SetAutoMemory.Get(), "autoMemoryEnabled"),
		boolean(i18n.SetAutoContinue.Get(), "autoContinueAtUsageLimit"),
		cycle(i18n.SetDefaultMode.Get(), "permissions.defaultMode",
			[]string{"default", "acceptEdits", "plan", "bypassPermissions"}),
		bashLimit,
		allow,
	}
}

// saveSetting writes one value into settings.json and reloads the page data.
func (v *SettingsView) saveSetting(path string, value any) error {
	if err := v.settings.Set(path, value); err != nil {
		return err
	}
	return v.settings.Save()
}

func (v *SettingsView) clearSetting(path string) error {
	if err := v.settings.Unset(path); err != nil {
		return err
	}
	return v.settings.Save()
}
