// Package chat is the built-in AI conversation: the user asks questions about
// their DingTalk data, can @ a person or group to bring that conversation's
// recent messages into context, and the model may call the same data tools
// the MCP server offers (read messages, search, report, calendar, to-dos).
package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/qqshow/dingtalk-agent/internal/agentdb"
	"github.com/qqshow/dingtalk-agent/internal/ai"
	"github.com/qqshow/dingtalk-agent/internal/analysis"
	"github.com/qqshow/dingtalk-agent/internal/app"
	"github.com/qqshow/dingtalk-agent/internal/mcp"
	"github.com/qqshow/dingtalk-agent/internal/store"
)

const (
	maxRounds          = 8     // tool-call rounds per answer
	maxToolResultChars = 30000 // per tool result sent to the model
	mentionBudget      = 48000 // chars of @mention context per message
	historyBudget      = 90000 // chars of earlier conversation sent to the model
)

// Tools offered to the model: reading data, plus adding / updating to-dos.
var toolNames = []string{
	"dingtalk_list_conversations", "dingtalk_get_messages", "dingtalk_search", "dingtalk_daily_report",
	"dingtalk_daily_digest", "dingtalk_calendar", "dingtalk_get_summary", "dingtalk_list_todos",
	"dingtalk_add_todos", "dingtalk_update_todo",
}

// Event is streamed to the client while answering.
type Event struct {
	Type      string               `json:"type"` // session | user | tool_start | tool_done | delta | note | done | error
	Session   *agentdb.ChatSession `json:"session,omitempty"`
	Message   *agentdb.ChatMessage `json:"message,omitempty"`
	Text      string               `json:"text,omitempty"`
	Tool      *agentdb.ToolUse     `json:"tool,omitempty"`
	ToolIndex int                  `json:"tool_index,omitempty"`
	Error     string               `json:"error,omitempty"`
}

// Request is one user message.
type Request struct {
	SessionID int64    `json:"session_id"`
	Text      string   `json:"text"`
	Mentions  []string `json:"mentions"` // conversation ids, or "u:<uid>" for a contact
	Days      int      `json:"days"`     // time range for @mention context (default 7)
}

// Service answers chat messages.
type Service struct {
	a     *app.App
	tools map[string]mcp.Tool
	specs []ai.ToolSpec

	mu      sync.Mutex
	noTools map[string]bool // models that rejected tool calls
}

func New(a *app.App, all []mcp.Tool) *Service {
	s := &Service{a: a, tools: map[string]mcp.Tool{}, noTools: map[string]bool{}}
	byName := map[string]mcp.Tool{}
	for _, t := range all {
		byName[t.Name] = t
	}
	for _, n := range toolNames {
		t, ok := byName[n]
		if !ok {
			continue
		}
		s.tools[n] = t
		s.specs = append(s.specs, ai.ToolSpec{Name: t.Name, Description: t.Description, Parameters: t.Schema})
	}
	return s
}

// Send answers one message, streaming events to emit.
func (s *Service) Send(ctx context.Context, req Request, emit func(Event)) error {
	cfg := s.a.Cfg.AI()
	if !cfg.Ready() {
		return ai.ErrNotReady
	}
	req.Text = strings.TrimSpace(req.Text)
	if req.Text == "" {
		return errors.New("请输入问题")
	}
	if req.Days <= 0 || req.Days > 93 {
		req.Days = 7
	}
	snap, err := s.a.Snapshot(true)
	if err != nil {
		return err
	}

	var sess *agentdb.ChatSession
	if req.SessionID > 0 {
		if sess, err = s.a.Agent.GetChatSession(req.SessionID); err != nil {
			return err
		}
	} else if sess, err = s.a.Agent.CreateChatSession(titleOf(req.Text)); err != nil {
		return err
	}
	emit(Event{Type: "session", Session: sess})

	history, err := s.a.Agent.ChatMessages(sess.ID)
	if err != nil {
		return err
	}
	mentions, mctx := mentionContext(snap, req.Mentions, req.Days, time.Now())
	userMsg, err := s.a.Agent.AddChatMessage(agentdb.ChatMessage{SessionID: sess.ID, Role: "user", Content: req.Text,
		Context: mctx, Meta: agentdb.ChatMeta{Mentions: mentions, Days: req.Days}})
	if err != nil {
		return err
	}
	emit(Event{Type: "user", Message: userMsg})

	client := &ai.Client{BaseURL: cfg.BaseURL, APIKey: cfg.APIKey, Model: cfg.Model}
	s.mu.Lock()
	useTools := !s.noTools[cfg.BaseURL+"|"+cfg.Model]
	s.mu.Unlock()

	meta := agentdb.ChatMeta{Model: cfg.Model}
	var answer strings.Builder
	finish := func(errMsg string) error {
		meta.Error = errMsg
		content := answer.String()
		if content == "" && errMsg == "" {
			content = "（没有生成回答）"
		}
		m, err := s.a.Agent.AddChatMessage(agentdb.ChatMessage{SessionID: sess.ID, Role: "assistant", Content: content, Meta: meta})
		if err != nil {
			return err
		}
		emit(Event{Type: "done", Message: m})
		return nil
	}

	msgs := s.buildMessages(snap, history, *userMsg, useTools)
	for round := 0; ; round++ {
		var tools []ai.ToolSpec
		if useTools && round < maxRounds {
			tools = s.specs
		}
		answer.Reset()
		reply, _, err := client.Converse(ctx, msgs, tools, func(t string) {
			answer.WriteString(t)
			emit(Event{Type: "delta", Text: t})
		})
		if errors.Is(err, ai.ErrToolsUnsupported) && len(tools) > 0 {
			slog.Warn("model rejected tools, retrying without", "model", cfg.Model, "err", err)
			s.mu.Lock()
			s.noTools[cfg.BaseURL+"|"+cfg.Model] = true
			s.mu.Unlock()
			useTools = false
			meta.Note = "当前模型不支持调用工具，只根据 @ 的人或群的聊天记录回答"
			emit(Event{Type: "note", Text: meta.Note})
			msgs = s.buildMessages(snap, history, *userMsg, false)
			continue
		}
		if err != nil {
			if ctx.Err() != nil {
				meta.Note = "已停止生成"
				return finish("")
			}
			return finish(err.Error())
		}
		if len(reply.ToolCalls) == 0 {
			return finish("")
		}
		msgs = append(msgs, *reply)
		for _, call := range reply.ToolCalls {
			use := agentdb.ToolUse{Name: call.Name, Args: call.Arguments}
			use.Title = describeCall(snap, call.Name, call.Arguments)
			idx := len(meta.Tools)
			meta.Tools = append(meta.Tools, use)
			emit(Event{Type: "tool_start", Tool: &use, ToolIndex: idx})
			result := s.runTool(ctx, call)
			use.Summary = summarize(result)
			if strings.HasPrefix(result, "错误：") {
				use.Error = strings.TrimPrefix(result, "错误：")
			}
			meta.Tools[idx] = use
			emit(Event{Type: "tool_done", Tool: &use, ToolIndex: idx})
			msgs = append(msgs, ai.ChatMessage{Role: "tool", ToolCallID: call.ID, Content: result})
		}
		if ctx.Err() != nil {
			meta.Note = "已停止生成"
			return finish("")
		}
	}
}

func (s *Service) runTool(ctx context.Context, call ai.ToolCall) string {
	t, ok := s.tools[call.Name]
	if !ok {
		return "错误：没有这个工具 " + call.Name
	}
	args := map[string]any{}
	if strings.TrimSpace(call.Arguments) != "" {
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			return "错误：参数不是有效的 JSON: " + err.Error()
		}
	}
	if _, ok := args["refresh"]; !ok {
		args["refresh"] = false // the snapshot was refreshed when the message arrived
	}
	tctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	out, err := t.Run(tctx, args)
	if err != nil {
		return "错误：" + err.Error()
	}
	if r := []rune(out); len(r) > maxToolResultChars {
		out = string(r[:maxToolResultChars]) + "\n…（内容过长已截断，可缩小日期范围或指定会话后重试）"
	}
	return out
}

// ------------------------------------------------------------- messages

func (s *Service) systemPrompt(snap *store.Snapshot, tools bool) string {
	now := time.Now()
	week := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}[now.Weekday()]
	var b strings.Builder
	fmt.Fprintf(&b, "你是%s的工作助理，可以查阅他电脑上的钉钉聊天记录、日程和待办（数据只读，你不能替他在钉钉里发送消息）。\n", snap.MyName())
	fmt.Fprintf(&b, "现在是 %s %s %s。\n\n", now.Format("2006-01-02"), week, now.Format("15:04"))
	b.WriteString("规则：\n")
	b.WriteString("- 用户 @ 的人或群，其近期聊天记录附在消息开头的「引用资料」中，优先使用。\n")
	if tools {
		b.WriteString("- 资料不足时调用工具查询：按名称查会话用 dingtalk_list_conversations，读某个会话的消息用 dingtalk_get_messages，" +
			"关键字查找用 dingtalk_search，某天或某段时间的整体情况用 dingtalk_daily_report / dingtalk_daily_digest，日程用 dingtalk_calendar，待办用 dingtalk_list_todos。\n")
		b.WriteString("- 用户要求记下待办或需要跟踪的事时，调用 dingtalk_add_todos 写入，并说明写入了哪些事项；不要未经要求就写入。\n")
		b.WriteString("- 用户问某个问题以前是怎么解决的、某件事的来龙去脉、谁熟悉某方面、某个文件或链接在哪时，把聊天记录当作知识库来查：" +
			"用 dingtalk_search 在全部历史中搜索（不限日期），换几个关键词多搜几次（同义词、报错信息里的关键片段、产品或模块名、较短的词）；" +
			"找到相关消息后，用 dingtalk_get_messages 读取那个会话在当天（必要时前后几天）的消息，弄清前因后果。" +
			"回答时按「问题现象 → 原因 → 解决办法 → 谁处理的 → 什么时候」整理，注明出处（会话名和时间）；" +
			"有多次类似问题时分别列出；搜不到就如实说明，并建议用户换个说法或 @ 可能相关的群。\n")
	} else {
		b.WriteString("- 你无法调用工具，只能根据引用资料回答；资料不足时请提示用户 @ 相关的人或群，或调整时间范围。\n")
	}
	b.WriteString("- 引用具体内容时注明会话名和时间。不要编造数据中没有出现的事实，不确定时如实说明。\n")
	b.WriteString("- 用户要你起草回复或消息时，直接给出可以复制发送的文本。\n")
	b.WriteString("- 用中文回答，使用 Markdown，简洁、直接。")
	return b.String()
}

func userContent(m agentdb.ChatMessage, withContext bool) string {
	if !withContext || strings.TrimSpace(m.Context) == "" {
		return m.Content
	}
	return "【引用资料】\n" + m.Context + "\n\n【问题】\n" + m.Content
}

// buildMessages assembles system prompt, earlier turns and the new message.
// @mention context is kept for the newest messages and dropped from older
// ones when the history grows too long.
func (s *Service) buildMessages(snap *store.Snapshot, history []agentdb.ChatMessage, cur agentdb.ChatMessage, tools bool) []ai.ChatMessage {
	sys := s.systemPrompt(snap, tools)
	curContent := userContent(cur, true)
	if !tools {
		if hits := keywordContext(snap, cur.Content); hits != "" {
			curContent = "【引用资料：按问题中的关键词搜索到的聊天记录】\n" + hits + "\n\n" + curContent
		}
	}
	if !tools && strings.TrimSpace(cur.Context) == "" && len(history) == 0 && !strings.Contains(curContent, "【引用资料") {
		// Without tools the model would know nothing: give it today's overview.
		if d, err := s.a.Digest("today", "", "", analysis.DigestOptions{Scope: "involved", SkipBots: true, MaxChars: 20000}, false); err == nil {
			curContent = "【引用资料：今天与我相关的消息摘要】\n" + d.Markdown + "\n\n【问题】\n" + cur.Content
		}
	}

	var turns []ai.ChatMessage
	used := 0
	contexts := 0
	for i := len(history) - 1; i >= 0; i-- {
		m := history[i]
		if m.Role != "user" && m.Role != "assistant" {
			continue
		}
		content := m.Content
		if m.Role == "user" && m.Context != "" && contexts < 2 && used+len([]rune(m.Context)) < historyBudget/2 {
			content = userContent(m, true)
			contexts++
		}
		if m.Role == "assistant" && m.Meta.Error != "" && strings.TrimSpace(content) == "" {
			continue
		}
		n := len([]rune(content))
		if used+n > historyBudget {
			break
		}
		used += n
		turns = append(turns, ai.ChatMessage{Role: m.Role, Content: content})
	}
	out := []ai.ChatMessage{{Role: "system", Content: sys}}
	for i := len(turns) - 1; i >= 0; i-- {
		out = append(out, turns[i])
	}
	return append(out, ai.ChatMessage{Role: "user", Content: curContent})
}

// keywordContext searches the whole history for terms the user put in
// 「」 or quotes; used when the model cannot call the search tool itself.
func keywordContext(snap *store.Snapshot, text string) string {
	terms := quotedTerms(text)
	var b strings.Builder
	for _, t := range terms {
		msgs, _ := snap.Range(store.RangeQuery{Keyword: t, Desc: true, Limit: 80})
		found, n := newestWithin(snap, msgs, 12000, true)
		if n == 0 {
			fmt.Fprintf(&b, "### 搜索「%s」\n（没有找到）\n\n", t)
			continue
		}
		fmt.Fprintf(&b, "### 搜索「%s」（%d 条，按时间排列）\n%s\n\n", t, n, found)
	}
	return strings.TrimSpace(b.String())
}

// quotedTerms returns up to 3 terms quoted with 「」, “” or "".
func quotedTerms(text string) []string {
	var terms []string
	for _, pair := range [][2]string{{"「", "」"}, {"\u201c", "\u201d"}, {`"`, `"`}} {
		rest := text
		for {
			i := strings.Index(rest, pair[0])
			if i < 0 {
				break
			}
			rest = rest[i+len(pair[0]):]
			j := strings.Index(rest, pair[1])
			if j < 0 {
				break
			}
			t := strings.TrimSpace(rest[:j])
			if t != "" && len([]rune(t)) <= 40 && strings.Trim(t, "…. ") != "" {
				terms = append(terms, t)
			}
			rest = rest[j+len(pair[1]):]
		}
	}
	if len(terms) > 3 {
		terms = terms[:3]
	}
	return terms
}

// --------------------------------------------------------------- mentions

func formatLine(snap *store.Snapshot, m *store.Message, withConv bool) string {
	sender := m.SenderName
	if m.IsMine {
		sender = "我"
	}
	mark := ""
	if m.MentionsMe && !m.IsMine {
		mark = "【@我】"
	}
	t := m.CreatedAtStr
	if len(t) >= 16 {
		t = t[5:16]
	}
	text := strings.Join(strings.Fields(m.AgentText()), " ")
	if r := []rune(text); len(r) > 500 {
		text = string(r[:500]) + "…"
	}
	if withConv {
		return fmt.Sprintf("%s [%s] %s: %s%s", t, snap.ConvTitle(m.CID), sender, mark, text)
	}
	return fmt.Sprintf("%s %s: %s%s", t, sender, mark, text)
}

// newestWithin formats messages (newest first) into chronological lines,
// keeping the most recent ones that fit in budget chars.
func newestWithin(snap *store.Snapshot, msgs []*store.Message, budget int, withConv bool) (string, int) {
	var lines []string
	used := 0
	for _, m := range msgs {
		if strings.TrimSpace(m.AgentText()) == "" {
			continue
		}
		l := formatLine(snap, m, withConv)
		n := len([]rune(l)) + 1
		if used+n > budget {
			break
		}
		used += n
		lines = append(lines, l)
	}
	for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
		lines[i], lines[j] = lines[j], lines[i]
	}
	return strings.Join(lines, "\n"), len(lines)
}

// target is a resolved @mention.
type target struct {
	key  string // what the page sent: a cid or "u:<uid>"
	conv *store.Conversation
	uid  int64  // person
	name string //
}

// resolveMentions turns mention keys into targets. A key is a conversation
// id, or "u:<uid>" for a contact (who may have no single chat with us).
func resolveMentions(snap *store.Snapshot, keys []string) []target {
	var out []target
	seen := map[string]bool{}
	for _, key := range keys {
		var t target
		if strings.HasPrefix(key, "u:") {
			uid, err := strconv.ParseInt(strings.TrimPrefix(key, "u:"), 10, 64)
			if err != nil || uid <= 0 || snap.IsBot(uid) {
				continue
			}
			t = target{key: key, uid: uid, name: snap.UserName("", uid)}
			if cid := snap.SingleCIDFor(uid); cid != "" {
				t.conv = snap.ConvByID[cid]
			}
		} else {
			c := snap.ConvByID[key]
			if c == nil {
				continue
			}
			t = target{key: key, conv: c, name: c.Title}
			if c.Type == "single" {
				t.uid, _ = strconv.ParseInt(c.OtherUID, 10, 64)
			}
		}
		id := t.key
		if t.uid > 0 {
			id = "u:" + strconv.FormatInt(t.uid, 10)
		} else if t.conv != nil {
			id = t.conv.CID
		}
		if !seen[id] {
			seen[id] = true
			out = append(out, t)
		}
	}
	return out
}

// mentionContext loads recent messages for each @mention. For a person: our
// single chat (if any), what they said in groups, and — when there is no
// single chat — group messages that @ them. For a group: its messages.
func mentionContext(snap *store.Snapshot, keys []string, days int, now time.Time) ([]agentdb.Mention, string) {
	targets := resolveMentions(snap, keys)
	if len(targets) == 0 {
		return nil, ""
	}
	y, mo, d := now.Date()
	from := time.Date(y, mo, d, 0, 0, 0, 0, time.Local).AddDate(0, 0, -(days - 1))
	q := store.RangeQuery{From: from.UnixMilli(), To: now.Add(time.Minute).UnixMilli(), Desc: true}
	label := fmt.Sprintf("近 %d 天（%s 起）", days, from.Format("01-02"))
	if days == 1 {
		label = "今天"
	}
	per := mentionBudget / len(targets)

	var mentions []agentdb.Mention
	var b strings.Builder
	for _, t := range targets {
		if t.uid == 0 { // group
			c := t.conv
			mentions = append(mentions, agentdb.Mention{CID: c.CID, Title: c.Title, Kind: "group"})
			kind := "群聊"
			if c.IsProject {
				kind = "项目群"
			}
			fmt.Fprintf(&b, "### @%s（%s，%d 人，%s）\n", c.Title, kind, c.MemberCount, label)
			qq := q
			qq.CIDs, qq.Limit = []string{c.CID}, 1500
			msgs, _ := snap.Range(qq)
			text, n := newestWithin(snap, msgs, per, false)
			if n == 0 {
				text = "（这段时间没有消息）"
			}
			b.WriteString(text + "\n\n")
			continue
		}

		m := agentdb.Mention{CID: t.key, Title: t.name, Kind: "person"}
		if t.conv != nil {
			m.CID = t.conv.CID
		}
		mentions = append(mentions, m)
		groupBudget := per / 3
		if t.conv != nil {
			fmt.Fprintf(&b, "### @%s（单聊，%s）\n", t.name, label)
			qq := q
			qq.CIDs, qq.Limit = []string{t.conv.CID}, 800
			msgs, _ := snap.Range(qq)
			text, n := newestWithin(snap, msgs, per*2/3, false)
			if n == 0 {
				text = "（这段时间没有单聊消息）"
			}
			b.WriteString(text + "\n")
		} else {
			fmt.Fprintf(&b, "### @%s（我和他没有单聊，以下是他在群里的发言和群里 @ 他的消息，%s）\n", t.name, label)
			groupBudget = per * 2 / 3
		}

		qq := q
		qq.SenderID, qq.Limit = t.uid, 600
		all, _ := snap.Range(qq)
		var inGroups []*store.Message
		for _, msg := range all {
			if t.conv == nil || msg.CID != t.conv.CID {
				inGroups = append(inGroups, msg)
			}
		}
		text, n := newestWithin(snap, inGroups, groupBudget, true)
		if n > 0 {
			fmt.Fprintf(&b, "\n%s 在群里的发言：\n%s\n", t.name, text)
		} else if t.conv == nil {
			b.WriteString("（这段时间他没有在群里发言）\n")
		}

		if t.conv == nil && t.name != "" {
			qq := q
			qq.Keyword, qq.Limit = "@"+t.name, 300
			hits, _ := snap.Range(qq)
			var atHim []*store.Message
			for _, msg := range hits {
				if msg.SenderID != t.uid {
					atHim = append(atHim, msg)
				}
			}
			if text, n := newestWithin(snap, atHim, per/3, true); n > 0 {
				fmt.Fprintf(&b, "\n群里 @%s 的消息：\n%s\n", t.name, text)
			}
		}
		b.WriteString("\n")
	}
	return mentions, strings.TrimSpace(b.String())
}

// ------------------------------------------------------------- display

func titleOf(text string) string {
	t := strings.Join(strings.Fields(text), " ")
	if r := []rune(t); len(r) > 30 {
		t = string(r[:30]) + "…"
	}
	return t
}

// describeCall turns a tool call into a short human description.
func describeCall(snap *store.Snapshot, name, rawArgs string) string {
	args := map[string]any{}
	json.Unmarshal([]byte(rawArgs), &args)
	get := func(k string) string {
		if v, ok := args[k]; ok && v != nil {
			return strings.TrimSpace(fmt.Sprint(v))
		}
		return ""
	}
	period := get("date")
	if f, t := get("from"), get("to"); f != "" || t != "" {
		period = strings.Trim(f+" ~ "+t, " ~")
	}
	conv := ""
	if cid := get("cid"); cid != "" {
		conv = "「" + snap.ConvTitle(cid) + "」"
	}
	with := func(s string) string {
		if period != "" {
			return s + "（" + period + "）"
		}
		return s
	}
	switch name {
	case "dingtalk_list_conversations":
		if k := get("keyword"); k != "" {
			return "查找会话：" + k
		}
		return "列出会话"
	case "dingtalk_get_messages":
		return with("读取" + conv + "的消息")
	case "dingtalk_search":
		s := "搜索「" + get("query") + "」"
		if conv != "" {
			s += "，限定" + conv
		}
		return with(s)
	case "dingtalk_daily_report":
		return with("查看统计")
	case "dingtalk_daily_digest":
		return with("阅读消息摘要")
	case "dingtalk_calendar":
		return with("查看日程")
	case "dingtalk_get_summary":
		return with("读取日报")
	case "dingtalk_list_todos":
		return "查看待办"
	case "dingtalk_add_todos":
		if items, ok := args["items"].([]any); ok {
			return fmt.Sprintf("添加 %d 个待办", len(items))
		}
		return "添加待办"
	case "dingtalk_update_todo":
		return "更新待办 #" + get("id")
	}
	return name
}

func summarize(result string) string {
	if strings.HasPrefix(result, "错误：") {
		return result
	}
	trimmed := strings.TrimSpace(result)
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		var v struct {
			Total int `json:"total"`
		}
		if json.Unmarshal([]byte(trimmed), &v) == nil && v.Total > 0 {
			return fmt.Sprintf("找到 %d 个", v.Total)
		}
		return fmt.Sprintf("已获取数据（%d 字）", len([]rune(result)))
	}
	first := strings.TrimSpace(strings.SplitN(trimmed, "\n", 2)[0])
	first = strings.TrimLeft(first, "# ")
	if r := []rune(first); len(r) > 60 {
		first = string(r[:60]) + "…"
	}
	return fmt.Sprintf("%s（%d 字）", first, len([]rune(result)))
}
