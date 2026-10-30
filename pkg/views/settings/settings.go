package settings

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/yourusername/claude-manager/pkg/core"
	"github.com/yourusername/claude-manager/pkg/i18n"
	"github.com/yourusername/claude-manager/pkg/views"
)

// SettingsView displays and allows editing of settings
type SettingsView struct {
	screen tcell.Screen
	config *core.Config
	width  int
	height int
}

func NewSettingsView() views.View {
	return &SettingsView{
		config: core.GetConfig(),
	}
}

func (v *SettingsView) Metadata() views.Metadata {
	return views.Metadata{
		ID:     "settings",
		NameEn: "Settings",
		NameZh: "设置",
		DescEn: "Configure application settings",
		DescZh: "配置应用设置",
		Icon:   "⚙️",
	}
}

func (v *SettingsView) Init(screen tcell.Screen) error {
	v.screen = screen
	v.width, v.height = screen.Size()
	return nil
}

func (v *SettingsView) Render() error {
	v.screen.Clear()

	title := i18n.ViewSettings.Get()
	v.drawCenteredText(1, title, tcell.StyleDefault.
		Foreground(tcell.ColorLightGreen).Bold(true))
	v.drawHorizontalLine(3)

	// Display current settings
	startY := 5
	startX := 10

	settings := []struct {
		label string
		value string
	}{
		{i18n.SettingsClaudeDir.Get(), v.config.GetClaudeDir()},
		{i18n.SettingsLanguage.Get(), v.config.GetLanguage()},
		{i18n.SettingsDateFormat.Get(), v.config.Display.DateFormat},
		{i18n.SettingsSizeUnit.Get(), v.config.Display.SizeUnit},
		{i18n.SettingsItemsPerPage.Get(), fmt.Sprintf("%d", v.config.Display.ItemsPerPage)},
	}

	for i, setting := range settings {
		y := startY + i
		text := fmt.Sprintf("%-25s: %s", setting.label, setting.value)
		v.drawText(startX, y, text, tcell.StyleDefault.Foreground(tcell.ColorWhite))
	}

	// Configuration file location
	configPath := core.GetConfigPath()
	y := startY + len(settings) + 1
	v.drawText(startX, y, fmt.Sprintf("Config file: %s", configPath),
		tcell.StyleDefault.Foreground(tcell.ColorGray))

	// Hints
	v.drawHorizontalLine(v.height - 3)
	v.drawCenteredText(v.height-2, "T: Toggle Language  ESC: Back  Q: Quit",
		tcell.StyleDefault.Foreground(tcell.ColorGray))

	v.screen.Show()
	return nil
}

func (v *SettingsView) HandleEvent(event tcell.Event) error {
	switch ev := event.(type) {
	case *tcell.EventKey:
		switch ev.Key() {
		case tcell.KeyEscape:
			return fmt.Errorf("back")
		case tcell.KeyRune:
			switch ev.Rune() {
			case 'q', 'Q':
				return fmt.Errorf("quit")
			case 't', 'T':
				// Toggle language and save
				mgr := i18n.GetManager()
				newLang := mgr.Toggle()
				langStr := "en"
				if newLang == i18n.LanguageChinese {
					langStr = "zh"
				}
				v.config.SetLanguage(langStr)
			}
		}
	case *tcell.EventResize:
		v.width, v.height = v.screen.Size()
		v.screen.Sync()
	}
	return nil
}

func (v *SettingsView) Cleanup() error {
	return nil
}

func (v *SettingsView) drawText(x, y int, text string, style tcell.Style) {
	for i, r := range text {
		v.screen.SetContent(x+i, y, r, nil, style)
	}
}

func (v *SettingsView) drawCenteredText(y int, text string, style tcell.Style) {
	x := (v.width - len(text)) / 2
	if x < 0 {
		x = 0
	}
	v.drawText(x, y, text, style)
}

func (v *SettingsView) drawHorizontalLine(y int) {
	for x := 2; x < v.width-2; x++ {
		v.screen.SetContent(x, y, '─', nil, tcell.StyleDefault.
			Foreground(tcell.ColorGray))
	}
}
