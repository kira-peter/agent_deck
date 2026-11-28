package codexchat

import (
	"path/filepath"
	"strings"
	"unicode"

	"github.com/gdamore/tcell/v2"
	"github.com/kira-peter/agent_deck/pkg/ui/textwidth"
)

func (c *chat) render() {
	width, height := c.screen.Size()
	c.screen.Clear()
	c.screen.HideCursor()
	defer c.screen.Show()
	if width < minPaneWidth || height < minPaneHeight {
		c.draw(0, 0, width, local("Resize terminal · Esc to return", "请放大终端 · Esc 返回"), chartDim)
		return
	}
	status := c.status()
	header := c.agentKind().Label() + "  /  " + filepath.Base(c.project) + "  /  " + status.text
	if c.model != "" {
		header += "  /  " + c.model
	}
	if c.desiredModel != "" && c.desiredModel != c.model {
		header += " → " + c.desiredModel
	}
	panelTitle := baseStyle.Foreground(status.color).Background(inkColor(ink.Surface)).Bold(true)
	if pane, ok := c.screen.(*paneScreen); ok && pane.flashing {
		panelTitle = selectedTab.Foreground(inkColor(ink.Highlight))
	}
	paintRow(c.screen, 0, 1, width, panelTitle)
	c.draw(2, 1, width-4, header, panelTitle)
	c.draw(2, 2, width-4, strings.Repeat("─", width-4), borderStyle)
	if len(c.prompts) > 0 {
		c.renderPrompt(width, height)
		return
	}
	chartRows := min((height-8)/4, 5)
	if height < 18 || width < 52 {
		chartRows = 0
	}
	y := c.drawTokenChart(3, chartRows, width)
	if c.notice != "" {
		c.draw(2, y, width-4, cleanLine(c.notice), baseStyle.Foreground(inkColor(ink.Warning)))
		y++
	}
	c.draw(2, y, width-4, strings.Repeat("─", width-4), borderStyle)
	c.renderTranscript(y+1, height-4, width)
	c.draw(2, height-4, width-4, strings.Repeat("─", width-4), borderStyle)
	c.renderInput(width, height-3)
	tools := local("F2 Show tool details", "F2 展开工具详情")
	if c.showTools {
		tools = local("F2 Hide tool details", "F2 收起工具详情")
	}
	c.draw(2, height-1, width-4, local("Ctrl+C Stop · Enter Send · ", "Ctrl+C 中断 · Enter 发送 · ")+tools+local(" · PgUp/Dn Scroll", " · PgUp/Dn 滚动"), chartDim)
}

func (c *chat) renderTranscript(top, bottom, width int) {
	visible := bottom - top
	if visible <= 0 {
		return
	}
	if len(c.items) == 0 {
		c.draw(2, top, width-4, local("Send a message to start the conversation.", "发送消息开始对话。"), chartDim)
		return
	}
	type row struct {
		text  string
		style tcell.Style
	}
	var rows []row
	for i := len(c.items) - 1; i >= 0 && (c.revealTool || len(rows) < visible+c.scroll); i-- {
		entry := c.items[i]
		label, message, style := c.formatItem(entry, width-4)
		lines := wrap(label+message, width-4)
		for j := len(lines) - 1; j >= 0; j-- {
			rows = append(rows, row{lines[j], style})
		}
		if c.revealTool && entry.Type != "userMessage" && entry.Type != "agentMessage" {
			c.scroll = max(len(rows)-visible, 0)
			break
		}
	}
	c.revealTool = false
	c.scroll = min(c.scroll, max(len(rows)-visible, 0))
	end := min(len(rows), c.scroll+visible)
	for i := end - 1; i >= c.scroll; i-- {
		c.draw(2, top+end-1-i, width-4, rows[i].text, rows[i].style)
	}
}

func (c *chat) formatItem(entry item, width int) (label, message string, style tcell.Style) {
	switch entry.Type {
	case "userMessage":
		var parts []string
		for _, input := range entry.Content {
			if input.Type == "text" {
				parts = append(parts, formatCodexQuestionReply(input.Text))
			} else {
				parts = append(parts, "["+input.Type+"]")
			}
		}
		return local("You  ›  ", "你  ›  "), strings.Join(parts, "\n"), baseStyle.Foreground(inkColor(ink.Secondary))
	case "agentMessage":
		if entry.Delivery == "async" && entry.Text == "" && len(entry.Questions) > 0 {
			questions := make([]string, len(entry.Questions))
			for i, question := range entry.Questions {
				questions[i] = question.Title
			}
			return c.agentKind().Label() + "  ›  ", strings.Join(questions, "\n"), baseStyle
		}
		return c.agentKind().Label() + "  ›  ", entry.Text, baseStyle
	}
	label = local("▸ Tool  ", "▸ 工具  ")
	message = entry.Type
	if entry.Command != "" {
		message = entry.Command
	}
	if entry.Status != "" {
		message += "  [" + entry.Status + "]"
	}
	if c.showTools {
		label = local("▾ Tool  ", "▾ 工具  ")
		if entry.Output != "" {
			message += "\n" + entry.Output
		}
		for _, change := range entry.Changes {
			message += "\n" + change.Path + "\n" + change.Diff
		}
	} else {
		message = textwidth.Truncate(strings.Join(strings.Fields(message), " "), max(width-textwidth.StringWidth(label), 1))
	}
	return label, message, chartDim
}

func (c *chat) renderInput(width, y int) {
	available := width - 7
	start := c.cursor
	cursorX := 0
	for start > 0 {
		cells := textwidth.StringWidth(string(c.draft[start-1]))
		if cursorX+cells >= available {
			break
		}
		cursorX += cells
		start--
	}
	bg := ink.Surface
	if pane, ok := c.screen.(*paneScreen); !ok || pane.focused {
		bg = ink.Raised
	}
	style := baseStyle.Background(inkColor(bg))
	paintRow(c.screen, 1, y, width-2, style)
	c.draw(2, y, 2, "›", style.Foreground(inkColor(ink.Accent)).Bold(true))
	c.draw(5, y, available, string(c.draft[start:]), style)
	if c.ready {
		c.screen.ShowCursor(5+cursorX, y)
	}
}

func cleanLine(text string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, text)
}

func wrap(text string, width int) []string {
	var out []string
	for _, line := range strings.Split(text, "\n") {
		line = cleanLine(strings.TrimSuffix(line, "\r"))
		start, used := 0, 0
		for i, r := range line {
			cells := textwidth.StringWidth(string(r))
			if used+cells > width && i > start {
				out = append(out, line[start:i])
				start, used = i, 0
			}
			used += cells
		}
		out = append(out, line[start:])
	}
	return out
}
