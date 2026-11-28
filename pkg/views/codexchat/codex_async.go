package codexchat

import (
	"encoding/json"
	"strconv"
	"strings"
)

const (
	codexQuestionReplyStart = "<send_user_message_question_reply>\n"
	codexQuestionReplyEnd   = "\n</send_user_message_question_reply>"
)

type codexAsyncQuestion struct {
	Title   string   `json:"title"`
	Options []string `json:"options"`
}

type codexAsyncReply struct {
	item item
	text string
}

type codexQuestionAnswer struct {
	Answer         string `json:"answer"`
	Question       string `json:"question"`
	QuestionItemID string `json:"questionItemId"`
}

func (c *chat) queueCodexAsyncQuestions(entry item) {
	if entry.ID == "" || entry.Type != "agentMessage" || len(entry.Questions) == 0 || c.asyncSeen[entry.ID] {
		return
	}
	if c.asyncSeen == nil {
		c.asyncSeen = make(map[string]bool)
	}
	c.asyncSeen[entry.ID] = true
	questions := make([]promptQuestion, len(entry.Questions))
	for i, question := range entry.Questions {
		id, _ := json.Marshal([]any{"request_user_input_async", entry.ID, i})
		questions[i] = promptQuestion{ID: string(id), Question: question.Title, IsOther: true}
		for _, label := range question.Options {
			questions[i].Options = append(questions[i].Options, promptOption{Label: label})
		}
	}
	c.queueQuestions(entry.ID, questions, func(answers map[string][]string) {
		if len(answers) == 0 {
			return
		}
		var reply []codexQuestionAnswer
		for _, question := range questions {
			if values := answers[question.ID]; len(values) > 0 {
				reply = append(reply, codexQuestionAnswer{values[0], question.Question, question.ID})
			}
		}
		body, _ := json.Marshal(reply)
		c.asyncReplies = append(c.asyncReplies, codexAsyncReply{entry, codexQuestionReplyStart + string(body) + codexQuestionReplyEnd})
		c.sendCodexAsyncReply()
	})
	c.prompts[len(c.prompts)-1].async = true
}

func (c *chat) codexInput(text string) map[string]any {
	return map[string]any{"threadId": c.threadID, "input": []map[string]string{{"type": "text", "text": text}}}
}

func (c *chat) sendCodexAsyncReply() {
	if len(c.asyncReplies) == 0 || c.asyncReplyRequest != "" || !c.ready || (c.working && (c.turnID == "" || c.turnID == c.asyncRetryAfter)) {
		return
	}
	method := "turn/start"
	params := c.codexInput(c.asyncReplies[0].text)
	c.asyncRetryAfter = ""
	if c.working {
		method = "turn/steer"
		params["expectedTurnId"] = c.turnID
		c.asyncRetryAfter = c.turnID
	}
	if c.request(method, params) {
		c.asyncReplyRequest = strconv.Itoa(c.nextID)
		c.notice = ""
		c.working = true
	}
}

func (c *chat) finishCodexAsyncReply(method string, msg rpcMessage) {
	c.asyncReplyRequest = ""
	if msg.Error != nil && method == "turn/steer" {
		// The turn may have ended while the reply was being sent. Once
		// its completion arrives, submit the same answer as a new turn.
		c.notice = local("Answer queued for the next turn.", "答案将在下一轮发送。")
		c.sendCodexAsyncReply()
		return
	}
	reply := c.asyncReplies[0]
	c.asyncReplies = c.asyncReplies[1:]
	c.asyncRetryAfter = ""
	if msg.Error != nil {
		c.working, c.interruptPending = false, false
		delete(c.asyncSeen, reply.item.ID)
		c.queueCodexAsyncQuestions(reply.item)
		c.prompts[len(c.prompts)-1].detail = msg.Error.Message
		return
	}
	var result struct{ Turn turn }
	if method == "turn/start" && c.working && json.Unmarshal(msg.Result, &result) == nil && result.Turn.ID != "" {
		c.turnID = result.Turn.ID
		c.flushInterrupt()
	}
	c.sendCodexAsyncReply()
}

func formatCodexQuestionReply(text string) string {
	if !strings.HasPrefix(text, codexQuestionReplyStart) || !strings.HasSuffix(text, codexQuestionReplyEnd) {
		return text
	}
	body := strings.TrimSuffix(strings.TrimPrefix(text, codexQuestionReplyStart), codexQuestionReplyEnd)
	var answers []codexQuestionAnswer
	if json.Unmarshal([]byte(body), &answers) != nil || len(answers) == 0 {
		return text
	}
	lines := make([]string, len(answers))
	for i, answer := range answers {
		lines[i] = answer.Question + " → " + answer.Answer
	}
	return strings.Join(lines, "\n")
}
