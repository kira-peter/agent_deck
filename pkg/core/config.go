package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// Config represents the application configuration
type Config struct {
	Version   string        `json:"version"`
	Language  string        `json:"language"`  // "en" or "zh"
	ClaudeDir string        `json:"claudeDir"` // Path to .claude directory
	CodexDir  string        `json:"codexDir"`  // Path to .codex directory
	Source    SourceKind    `json:"source"`    // agent whose data is shown
	Display   DisplayConfig `json:"display"`
	mu        sync.RWMutex
}

// DisplayConfig contains display-related settings
type DisplayConfig struct {
	DateFormat   string `json:"dateFormat"`   // "relative" or "absolute"
	SizeUnit     string `json:"sizeUnit"`     // "auto", "KB", "MB"
	ItemsPerPage int    `json:"itemsPerPage"` // Number of items per page in lists
}

var (
	globalConfig *Config
	configMutex  sync.RWMutex
)

// GetConfig returns the global configuration instance
func GetConfig() *Config {
	configMutex.RLock()
	defer configMutex.RUnlock()

	if globalConfig == nil {
		// Load or create default config
		cfg, err := LoadConfig()
		if err != nil {
			cfg = DefaultConfig()
		}
		globalConfig = cfg
	}

	return globalConfig
}

// DefaultConfig returns a configuration with default values
func DefaultConfig() *Config {
	claudeDir := GetDefaultClaudeDir()

	return &Config{
		Version:   "1.0",
		Language:  "en", // Default to English
		ClaudeDir: claudeDir,
		CodexDir:  GetDefaultCodexDir(),
		Source:    SourceClaude,
		Display: DisplayConfig{
			DateFormat:   "relative",
			SizeUnit:     "auto",
			ItemsPerPage: 20,
		},
	}
}

// GetDefaultClaudeDir returns the default .claude directory path
func GetDefaultClaudeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude")
}

// GetDefaultCodexDir returns the default .codex directory path, honouring
// CODEX_HOME like Codex itself does.
func GetDefaultCodexDir() string {
	if env := os.Getenv("CODEX_HOME"); env != "" {
		return env
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".codex")
}

// GetConfigDir returns the configuration directory path (~/.claude-manager)
func GetConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude-manager")
}

// GetConfigPath returns the full path to the config file
func GetConfigPath() string {
	return filepath.Join(GetConfigDir(), "config.json")
}

// LoadConfig loads configuration from file
func LoadConfig() (*Config, error) {
	configPath := GetConfigPath()

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Config doesn't exist, return default
			return DefaultConfig(), nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Validate and set defaults for missing fields
	if cfg.Language == "" {
		cfg.Language = "en"
	}
	if cfg.ClaudeDir == "" {
		cfg.ClaudeDir = GetDefaultClaudeDir()
	}
	if cfg.CodexDir == "" {
		cfg.CodexDir = GetDefaultCodexDir()
	}
	if cfg.Source != SourceCodex {
		cfg.Source = SourceClaude
	}
	if cfg.Display.DateFormat == "" {
		cfg.Display.DateFormat = "relative"
	}
	if cfg.Display.SizeUnit == "" {
		cfg.Display.SizeUnit = "auto"
	}
	if cfg.Display.ItemsPerPage == 0 {
		cfg.Display.ItemsPerPage = 20
	}

	return &cfg, nil
}

// Save writes the configuration to file
func (c *Config) Save() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	configDir := GetConfigDir()

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	// Marshal config to JSON with indentation
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	// Write to file
	configPath := GetConfigPath()
	return os.WriteFile(configPath, data, 0644)
}

// SetLanguage sets the language and saves the configuration
func (c *Config) SetLanguage(lang string) error {
	c.mu.Lock()
	c.Language = lang
	c.mu.Unlock()

	return c.Save()
}

// SetClaudeDir sets the Claude directory path and saves the configuration
func (c *Config) SetClaudeDir(dir string) error {
	c.mu.Lock()
	c.ClaudeDir = dir
	c.mu.Unlock()

	return c.Save()
}

// GetDisplay returns a copy of the display options (thread-safe)
func (c *Config) GetDisplay() DisplayConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Display
}

// UpdateDisplay applies fn to the display options and saves the configuration
func (c *Config) UpdateDisplay(fn func(*DisplayConfig)) error {
	c.mu.Lock()
	fn(&c.Display)
	c.mu.Unlock()

	return c.Save()
}

// VisibleRows caps the number of list rows that fit on screen (avail) by the
// "items per page" display option.
func VisibleRows(avail int) int {
	if n := GetConfig().GetDisplay().ItemsPerPage; n > 0 && n < avail {
		avail = n
	}
	return max(avail, 1)
}

// GetLanguage returns the current language setting (thread-safe)
func (c *Config) GetLanguage() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Language
}

// GetClaudeDir returns the Claude directory path (thread-safe)
func (c *Config) GetClaudeDir() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ClaudeDir
}

// GetCodexDir returns the Codex directory path (thread-safe)
func (c *Config) GetCodexDir() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.CodexDir
}

// GetSource returns the agent whose data is shown (thread-safe)
func (c *Config) GetSource() SourceKind {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Source
}

// SetSource switches the agent whose data is shown and saves the config.
func (c *Config) SetSource(kind SourceKind) error {
	c.mu.Lock()
	c.Source = kind
	c.mu.Unlock()
	return c.Save()
}

// DetectClaudeDirs returns a list of possible .claude directory locations
func DetectClaudeDirs() []string {
	var locations []string

	// Default location
	defaultDir := GetDefaultClaudeDir()
	if ExistsAndValid(defaultDir) {
		locations = append(locations, defaultDir)
	}

	// From environment variable
	if envDir := os.Getenv("CLAUDE_DIR"); envDir != "" {
		if ExistsAndValid(envDir) {
			locations = append(locations, envDir)
		}
	}

	return locations
}

// ExistsAndValid checks if a directory exists and looks like a valid .claude directory
func ExistsAndValid(dir string) bool {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return false
	}

	// Check for characteristic files/directories
	settingsPath := filepath.Join(dir, "settings.json")
	projectsPath := filepath.Join(dir, "projects")

	_, err1 := os.Stat(settingsPath)
	_, err2 := os.Stat(projectsPath)

	return err1 == nil || err2 == nil
}
