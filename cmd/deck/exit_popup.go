package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/gdamore/tcell/v2"
	"github.com/kira-peter/agent_deck/pkg/core"
	"github.com/kira-peter/agent_deck/pkg/i18n"
	"github.com/kira-peter/agent_deck/pkg/ui/textwidth"
	"github.com/kira-peter/agent_deck/pkg/ui/transition"
	"github.com/kira-peter/agent_deck/pkg/views"
	"github.com/kira-peter/agent_deck/pkg/views/codexchat"
	visual "github.com/will-to-meaning/terminal_visuals"
)

type exitAction int

const (
	exitPending exitAction = iota
	exitCancel
	exitWorkspace
	exitConfirm
)

type exitPopup struct {
	direct, animated  bool
	openedAt, focusAt time.Time
}

func newExitPopup() *exitPopup {
	now := time.Now()
	return &exitPopup{animated: codexchat.MotionEnabled(), openedAt: now, focusAt: now.Add(visual.DialogOpenDuration)}
}

func (p exitPopup) refreshInterval(idle time.Duration) time.Duration {
	if p.animated && time.Since(p.focusAt) < visual.ButtonSweepDuration {
		return 33 * time.Millisecond
	}
	return idle
}

func (p *exitPopup) handle(key string, running int) exitAction {
	if running > 0 {
		p.direct = false
	}
	switch key {
	case "esc":
		return exitCancel
	case "left", "right", "up", "down", "tab":
		if running == 0 {
			p.direct = !p.direct
			p.focusAt = time.Now()
			p.openedAt = p.focusAt.Add(-visual.DialogOpenDuration)
		}
	case "enter":
		if p.direct {
			return exitConfirm
		}
		return exitWorkspace
	}
	return exitPending
}

func (p exitPopup) dialog(now time.Time, open, running int) visual.Dialog {
	label := func(en, zh string) string { return (i18n.Text{En: en, Zh: zh}).Get() }
	dialog := visual.Dialog{
		Title:    label("Exit Deck", "退出 Deck"),
		Message:  fmt.Sprintf(label("%d agent window(s) open", "仍有 %d 个 Agent 窗口打开"), open),
		Hint:     label("← → Select · Enter Confirm · Esc Cancel", "← → 选择 · Enter 确认 · Esc 取消"),
		Buttons:  []visual.DialogButton{{Label: label("Go to Workspace", "前往工作区")}},
		OpenAge:  -1,
		FocusAge: -1,
	}
	if p.animated {
		dialog.OpenAge = now.Sub(p.openedAt)
		dialog.FocusAge = now.Sub(p.focusAt)
	}
	if running > 0 {
		dialog.Message = fmt.Sprintf(label("Stop %d running task(s) before exiting.", "请先停止 %d 个正在运行的任务。"), running)
	} else {
		dialog.Buttons = append(dialog.Buttons, visual.DialogButton{Label: label("Quit now", "直接退出"), Danger: true})
		if p.direct {
			dialog.Selected = 1
		}
	}
	return dialog
}

func exitPalette() visual.Palette {
	return visual.PaletteFor(core.GetConfig().GetDisplay().WorkspaceTheme)
}

func dialogTruncate(text string, width int) string {
	if width < 3 {
		return ansi.Truncate(text, width, "")
	}
	return textwidth.Truncate(text, width)
}

func (p exitPopup) overlay(background string, width, height, open, running int) string {
	lines := strings.Split(background, "\n")
	p.dialog(time.Now(), open, running).Draw(width, height, exitPalette(), textwidth.StringWidth, dialogTruncate, func(x, y int, text string, style visual.DialogStyle) {
		if text == "" {
			return
		}
		for y >= len(lines) {
			lines = append(lines, "")
		}
		left := ansi.Cut(lines[y], 0, x)
		right := ansi.Cut(lines[y], x+textwidth.StringWidth(text), width)
		fg, bg := uint32(style.Foreground), uint32(style.Background)
		bold := ""
		if style.Bold {
			bold = "1;"
		}
		color := fmt.Sprintf("\033[%s38;2;%d;%d;%d;48;2;%d;%d;%dm", bold, fg>>16, fg>>8&255, fg&255, bg>>16, bg>>8&255, bg&255)
		lines[y] = left + strings.Repeat(" ", max(x-ansi.StringWidth(left), 0)) + color + text + "\033[0m" + right
	})
	return strings.Join(lines, "\n")
}

func (p exitPopup) draw(screen tcell.Screen, open, running int) {
	width, height := screen.Size()
	p.dialog(time.Now(), open, running).Draw(width, height, exitPalette(), textwidth.StringWidth, dialogTruncate, func(x, y int, text string, style visual.DialogStyle) {
		st := tcell.StyleDefault.Foreground(tcell.NewHexColor(int32(style.Foreground))).
			Background(tcell.NewHexColor(int32(style.Background))).Bold(style.Bold)
		textwidth.Draw(screen, x, y, text, st)
	})
	screen.HideCursor()
	screen.Show()
}

type exitRefresh struct{}

func confirmExitOnScreen(screen *transition.Offscreen, view views.View) exitAction {
	popup := newExitPopup()
	keys := map[tcell.Key]string{tcell.KeyEscape: "esc", tcell.KeyLeft: "left", tcell.KeyRight: "right",
		tcell.KeyUp: "up", tcell.KeyDown: "down", tcell.KeyTab: "tab", tcell.KeyEnter: "enter"}
	for {
		open, running := codexchat.ExitStatus()
		if open == 0 {
			return exitConfirm
		}
		screen.Hold = true
		view.Render()
		screen.Hold = false
		popup.draw(screen, open, running)
		timer := time.AfterFunc(popup.refreshInterval(200*time.Millisecond), func() {
			_ = screen.PostEvent(tcell.NewEventInterrupt(exitRefresh{}))
		})
		event := screen.PollEvent()
		timer.Stop()
		switch event := event.(type) {
		case nil:
			return exitCancel
		case *tcell.EventKey:
			_, running := codexchat.ExitStatus()
			if action := popup.handle(keys[event.Key()], running); action != exitPending {
				return action
			}
		case *tcell.EventInterrupt:
			if _, refresh := event.Data().(exitRefresh); !refresh {
				_ = view.HandleEvent(event)
			}
		case *tcell.EventResize:
			_ = view.HandleEvent(event)
		}
	}
}
