package i18n

import (
	"sync"
)

// Language represents a language type
type Language string

const (
	// LanguageChinese represents Chinese
	LanguageChinese Language = "zh"
	// LanguageEnglish represents English
	LanguageEnglish Language = "en"
)

// Manager manages the current language setting
type Manager struct {
	mu      sync.RWMutex
	current Language
}

var (
	globalManager *Manager
	once          sync.Once
)

// GetManager returns the global language manager instance
func GetManager() *Manager {
	once.Do(func() {
		globalManager = &Manager{
			current: LanguageEnglish, // Default to English
		}
	})
	return globalManager
}

// GetCurrent returns the current language
func (m *Manager) GetCurrent() Language {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current
}

// SetLanguage sets the language
func (m *Manager) SetLanguage(lang Language) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.current = lang
}

// Toggle switches between Chinese and English
func (m *Manager) Toggle() Language {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.current == LanguageChinese {
		m.current = LanguageEnglish
	} else {
		m.current = LanguageChinese
	}

	return m.current
}

// IsEnglish returns true if current language is English
func (m *Manager) IsEnglish() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current == LanguageEnglish
}

// IsChinese returns true if current language is Chinese
func (m *Manager) IsChinese() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current == LanguageChinese
}
