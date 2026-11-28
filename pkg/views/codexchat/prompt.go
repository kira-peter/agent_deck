package codexchat

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/gdamore/tcell/v2"
)

type promptOption struct {
	Label, Description string
}

type promptQuestion struct {
	ID, Header, Question string
	Options              []promptOption
	MultiSelect          bool
	IsOther, IsSecret    bool
}

type sessionPrompt struct {
	id, title, detail string
	questions         []promptQuestion
	reply             func(map[string][]string)
	answers           map[string][]string
	page, choice      int
	line              int
	editing           bool
	async             bool
	input             []rune
}

func (c *chat) removePrompt(id string) {
	c.prompts = slices.DeleteFunc(c.prompts, func(p sessionPrompt) bool { return p.id == id })
}

func (c *chat) queuePermission(id, detail string, allow bool, reply func(bool)) {
	question := promptQuestion{ID: "decision", Options: []promptOption{{Label: local("Decline", "拒绝")}}}
	if allow {
		question.Options = append([]promptOption{{Label: local("Allow once", "允许本次")}}, question.Options...)
	}
	c.prompts = append(c.prompts, sessionPrompt{
		id: id, title: local("Approval required", "请求审批"), detail: detail,
		questions: []promptQuestion{question}, choice: len(question.Options) - 1,
		reply: func(answers map[string][]string) {
			reply(allow && slices.Equal(answers[question.ID], []string{question.Options[0].Label}))
		},
	})
}

func (c *chat) queueCodexQuestions(msg rpcMessage) {
	var params struct {
		ThreadID  string
		Questions []promptQuestion
	}
	reply := func(answers map[string][]string) {
		type answer struct {
			Answers []string `json:"answers"`
		}
		result := make(map[string]answer, len(answers))
		for id, values := range answers {
			result[id] = answer{values}
		}
		c.send(map[string]any{"id": msg.ID, "result": map[string]any{"answers": result}})
	}
	if json.Unmarshal(msg.Params, &params) != nil || params.ThreadID != c.threadID || len(params.Questions) == 0 {
		reply(nil)
		return
	}
	c.queueQuestions(string(msg.ID), params.Questions, reply)
}

func (c *chat) queueQuestions(id string, questions []promptQuestion, reply func(map[string][]string)) {
	c.prompts = append(c.prompts, sessionPrompt{
		id: id, title: local("Your input is needed", "等待你的选择"),
		questions: questions, answers: make(map[string][]string), reply: reply,
		editing: len(questions[0].Options) == 0,
	})
}

func (c *chat) finishPrompt(answers map[string][]string) {
	reply := c.prompts[0].reply
	c.prompts = c.prompts[1:]
	reply(answers)
}

func (p *sessionPrompt) question() promptQuestion { return p.questions[p.page] }

func (q promptQuestion) choices() []promptOption {
	if q.IsOther && len(q.Options) > 0 {
		return append(slices.Clone(q.Options), promptOption{Label: local("Other…", "其他…")})
	}
	return q.Options
}

func (c *chat) handlePrompt(event *tcell.EventKey) {
	p := &c.prompts[0]
	q := p.question()
	options := q.choices()
	key := event.Key()
	if key == tcell.KeyEscape {
		if p.editing && len(options) > 0 {
			p.editing, p.input = false, nil
		} else {
			c.finishPrompt(nil)
		}
		return
	}
	if key == tcell.KeyPgUp || key == tcell.KeyPgDn {
		if key == tcell.KeyPgUp {
			p.line = max(p.line-4, 0)
		} else {
			p.line += 4
		}
		return
	}
	if !p.editing {
		switch key {
		case tcell.KeyUp:
			p.choice = (p.choice + len(options) - 1) % len(options)
			return
		case tcell.KeyDown:
			p.choice = (p.choice + 1) % len(options)
			return
		case tcell.KeyLeft:
			if p.page > 0 {
				p.goTo(p.page - 1)
			}
			return
		case tcell.KeyCtrlY, tcell.KeyCtrlN:
			if p.answers == nil {
				if key == tcell.KeyCtrlY && len(options) == 2 {
					c.finishPrompt(map[string][]string{q.ID: {options[0].Label}})
				} else if key == tcell.KeyCtrlN {
					c.finishPrompt(nil)
				}
			}
			return
		}
		if event.Rune() >= '1' && event.Rune() <= '9' && int(event.Rune()-'1') < len(options) {
			p.choice = int(event.Rune() - '1')
			if q.MultiSelect {
				key = tcell.KeyRune
				event = tcell.NewEventKey(key, ' ', 0)
			} else {
				key = tcell.KeyEnter
			}
		}
		if q.MultiSelect && event.Rune() == ' ' && p.choice < len(q.Options) {
			value := options[p.choice].Label
			values := p.answers[q.ID]
			if i := slices.Index(values, value); i >= 0 {
				p.answers[q.ID] = slices.Delete(values, i, i+1)
			} else {
				p.answers[q.ID] = append(values, value)
			}
			return
		}
	}
	if p.editing {
		switch key {
		case tcell.KeyRune:
			p.input = append(p.input, event.Rune())
		case tcell.KeyBackspace, tcell.KeyBackspace2:
			p.input = p.input[:max(0, len(p.input)-1)]
		}
	}
	if key != tcell.KeyEnter {
		return
	}
	if p.answers == nil {
		c.finishPrompt(map[string][]string{q.ID: {options[p.choice].Label}})
		return
	}
	switch {
	case p.editing:
		value := strings.TrimSpace(string(p.input))
		if value == "" {
			return
		}
		p.answers[q.ID] = []string{value}
	case p.choice >= len(q.Options):
		p.editing = true
		return
	case !q.MultiSelect:
		p.answers[q.ID] = []string{options[p.choice].Label}
	case len(p.answers[q.ID]) == 0:
		return
	}
	if p.page+1 == len(p.questions) {
		c.finishPrompt(p.answers)
	} else {
		p.goTo(p.page + 1)
	}
}

func (p *sessionPrompt) goTo(page int) {
	p.page, p.choice, p.line, p.input = page, 0, 0, nil
	q := p.question()
	p.editing = len(q.Options) == 0
	if answers := p.answers[q.ID]; len(answers) > 0 {
		if i := slices.IndexFunc(q.Options, func(o promptOption) bool { return o.Label == answers[0] }); i >= 0 {
			p.choice = i
		} else {
			p.choice, p.editing, p.input = len(q.Options), true, []rune(answers[0])
		}
	}
}

func (c *chat) renderPrompt(width, height int) {
	p := &c.prompts[0]
	q := p.question()
	options := q.choices()
	style := baseStyle.Foreground(inkColor(ink.Warning))
	title := c.agentKind().Label() + " · " + p.title
	if len(p.questions) > 1 {
		title += fmt.Sprintf(" (%d/%d)", p.page+1, len(p.questions))
	}
	c.draw(2, 3, width-4, title, style.Bold(true))
	detail := strings.TrimSpace(strings.Join([]string{q.Question, p.detail}, "\n"))
	lines := wrap(detail, width-4)
	rows := min(max(height-10, 2), len(lines))
	p.line = min(p.line, max(len(lines)-rows, 0))
	for i, line := range lines[p.line : p.line+rows] {
		c.draw(2, 4+i, width-4, line, baseStyle)
	}
	y := 4 + rows
	if p.editing {
		input := string(p.input)
		if q.IsSecret {
			input = strings.Repeat("•", len(p.input))
		}
		c.draw(2, y, width-4, "› "+input+"▏", selectedTab)
		c.draw(2, height-1, width-4, local("Type · Enter · Esc", "输入 · Enter 确认 · Esc"), style)
		return
	}
	count := min(height-y-2, len(options))
	start := min(max(p.choice-count+1, 0), len(options)-count)
	for i := start; i < start+count; i++ {
		optionStyle, marker := baseStyle, "  "
		if i == p.choice {
			optionStyle, marker = selectedTab.Bold(true), "› "
		}
		label := options[i].Label
		if q.MultiSelect && i < len(q.Options) {
			check := "[ ] "
			if slices.Contains(p.answers[q.ID], label) {
				check = "[✓] "
			}
			label = check + label
		}
		c.draw(2, y+i-start, width-4, fmt.Sprintf("%s%d. %s", marker, i+1, label), optionStyle)
	}
	if info := options[p.choice].Description; info != "" {
		c.draw(2, height-2, width-4, cleanLine(info), chartDim)
	}
	hint := local("↑↓ Select · Enter Confirm · Esc", "↑↓ 选择 · Enter 确认 · Esc")
	if q.MultiSelect {
		hint = local("↑↓ Move · Space Toggle · Enter · Esc", "↑↓ 移动 · 空格勾选 · Enter · Esc")
	} else if len(lines) > rows {
		hint += " · PgUp/Dn"
	}
	c.draw(2, height-1, width-4, hint, style)
}
