package analysis

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// DigestOptions controls the LLM-oriented digest.
type DigestOptions struct {
	Scope       string // "all" (default) or "involved": only single chats, involved groups, project groups
	MaxChars    int    // total budget (default 60000)
	MaxMsgChars int    // per message (default 300)
	SkipBots    bool   // skip conversations with bots / notification accounts (default true)
}

// DigestConv is one conversation in the JSON digest.
type DigestConv struct {
	CID        string       `json:"cid"`
	Title      string       `json:"title"`
	Kind       string       `json:"kind"` // single | group | project
	Messages   int          `json:"message_count"`
	MyMessages int          `json:"my_messages"`
	MentionsMe int          `json:"mentions_me"`
	Priority   int          `json:"priority"`
	Lines      []DigestLine `json:"messages"`
}

type DigestLine struct {
	Time   string `json:"time"`
	Sender string `json:"sender"`
	Text   string `json:"text"`
	Mine   bool   `json:"mine,omitempty"`
	AtMe   bool   `json:"at_me,omitempty"`
	MsgID  int64  `json:"msg_id"`
}

type Digest struct {
	Period    PeriodInfo   `json:"period"`
	Me        Person       `json:"me"`
	Report    *Report      `json:"-"`
	Convs     []DigestConv `json:"conversations"`
	Omitted   []string     `json:"omitted_conversations"`
	Truncated bool         `json:"truncated"`
	Markdown  string       `json:"markdown,omitempty"`
}

func (o *DigestOptions) defaults() {
	if o.MaxChars <= 0 {
		o.MaxChars = 60000
	}
	if o.MaxMsgChars <= 0 {
		o.MaxMsgChars = 300
	}
	if o.Scope == "" {
		o.Scope = "all"
	}
}

// BuildDigest groups the period's messages by conversation, ordered by how
// much they likely matter to me, within a character budget.
func BuildDigest(in *Input, rep *Report, o DigestOptions) *Digest {
	o.defaults()
	s := in.Snap
	d := &Digest{Period: rep.Period, Me: rep.Me, Report: rep, Omitted: []string{}}
	multiDay := in.Period.To.Sub(in.Period.From) > 25*time.Hour

	byCID := map[string]*DigestConv{}
	var order []string
	for _, m := range in.Msgs {
		c := s.ConvByID[m.CID]
		if c == nil || isSelfChat(c, s.MyUID) {
			continue
		}
		text := snippet(m.AgentText(), o.MaxMsgChars)
		if text == "" {
			continue
		}
		dc := byCID[m.CID]
		if dc == nil {
			kind := c.Type
			if c.IsProject {
				kind = "project"
			}
			dc = &DigestConv{CID: c.CID, Title: c.Title, Kind: kind}
			byCID[m.CID] = dc
			order = append(order, m.CID)
		}
		dc.Messages++
		if m.IsMine {
			dc.MyMessages++
		}
		if m.MentionsMe && !m.IsMine {
			dc.MentionsMe++
		}
		tf := "15:04"
		if multiDay {
			tf = "01-02 15:04"
		}
		sender := m.SenderName
		if m.IsMine {
			sender = "我"
		} else if m.IsSystem {
			sender = "[系统]"
		}
		dc.Lines = append(dc.Lines, DigestLine{
			Time: time.UnixMilli(m.CreatedAt).Format(tf), Sender: sender, Text: text,
			Mine: m.IsMine, AtMe: m.MentionsMe && !m.IsMine, MsgID: m.ID,
		})
	}

	var convs []DigestConv
	for _, cid := range order {
		dc := byCID[cid]
		c := s.ConvByID[cid]
		bot := c.IsNotification
		if c.Type == "single" {
			var other int64
			fmt.Sscan(c.OtherUID, &other)
			bot = bot || s.IsBot(other)
		}
		if o.SkipBots && bot {
			d.Omitted = append(d.Omitted, dc.Title+"(机器人/通知)")
			continue
		}
		involved := dc.MyMessages > 0 || dc.MentionsMe > 0
		switch {
		case c.Type == "single":
			dc.Priority = 100
		case dc.MentionsMe > 0:
			dc.Priority = 90
		case involved && c.IsProject:
			dc.Priority = 80
		case involved:
			dc.Priority = 70
		case c.IsProject:
			dc.Priority = 50
		default:
			dc.Priority = 10
		}
		if o.Scope == "involved" && dc.Priority < 50 {
			d.Omitted = append(d.Omitted, dc.Title)
			continue
		}
		convs = append(convs, *dc)
	}
	sort.SliceStable(convs, func(i, j int) bool {
		if convs[i].Priority != convs[j].Priority {
			return convs[i].Priority > convs[j].Priority
		}
		return convs[i].Messages > convs[j].Messages
	})

	// Budget: header + pending sections are always kept.
	head := digestHeader(rep)
	budget := o.MaxChars - runeLen(head)
	var body strings.Builder
	for _, dc := range convs {
		sec := convSection(dc)
		n := runeLen(sec)
		if n > budget {
			// Large low-priority groups: keep the tail (most recent) that fits.
			if budget > 800 && dc.Priority >= 50 {
				dc2 := dc
				for len(dc2.Lines) > 0 && runeLen(convSection(dc2)) > budget {
					dc2.Lines = dc2.Lines[len(dc2.Lines)/4+1:]
				}
				if len(dc2.Lines) > 0 {
					sec = convSection(dc2)
					sec = strings.Replace(sec, "\n", fmt.Sprintf("\n（较早的 %d 条已省略）\n", len(dc.Lines)-len(dc2.Lines)), 1)
					body.WriteString(sec)
					budget -= runeLen(sec)
					d.Convs = append(d.Convs, dc2)
					d.Truncated = true
					continue
				}
			}
			d.Omitted = append(d.Omitted, fmt.Sprintf("%s(%d条)", dc.Title, dc.Messages))
			d.Truncated = true
			continue
		}
		body.WriteString(sec)
		budget -= n
		d.Convs = append(d.Convs, dc)
	}
	if d.Convs == nil {
		d.Convs = []DigestConv{}
	}

	var md strings.Builder
	md.WriteString(head)
	md.WriteString("\n## 会话明细（按重要程度排序）\n")
	md.WriteString("标记说明：「我」= 我发的消息；【@我】= 单独@我的消息\n")
	md.WriteString(body.String())
	if len(d.Omitted) > 0 {
		md.WriteString("\n## 未展开的会话\n")
		md.WriteString(strings.Join(d.Omitted, "、"))
		md.WriteString("\n")
	}
	d.Markdown = md.String()
	return d
}

func kindLabel(k string) string {
	switch k {
	case "single":
		return "单聊"
	case "project":
		return "项目群"
	}
	return "群"
}

func convSection(dc DigestConv) string {
	var b strings.Builder
	fmt.Fprintf(&b, "\n### [%s] %s（%d 条", kindLabel(dc.Kind), dc.Title, dc.Messages)
	if dc.Kind != "single" {
		fmt.Fprintf(&b, "，我发言 %d", dc.MyMessages)
		if dc.MentionsMe > 0 {
			fmt.Fprintf(&b, "，@我 %d", dc.MentionsMe)
		}
	}
	b.WriteString("）\n")
	for _, l := range dc.Lines {
		mark := ""
		if l.AtMe {
			mark = "【@我】"
		}
		fmt.Fprintf(&b, "%s %s: %s%s\n", l.Time, l.Sender, mark, l.Text)
	}
	return b.String()
}

func digestHeader(r *Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# 钉钉消息摘要 %s\n", r.Period.Label)
	fmt.Fprintf(&b, "我：%s（uid %d）｜数据截至 %s\n\n", r.Me.Name, r.Me.UID, r.DataAsOf)
	t := r.Totals
	fmt.Fprintf(&b, "**统计**：消息 %d 条（我发送 %d / 收到 %d）；活跃会话 %d 个（单聊 %d、群 %d）；"+
		"互动 %d 人；参与群 %d 个（活跃 %d）；参与项目群 %d 个（活跃 %d）\n",
		t.Messages, t.SentByMe, t.Received, t.ActiveConversations, t.ActiveSingleChats, t.ActiveGroups,
		r.People.Count, r.Groups.InvolvedCount, r.Groups.ActiveCount, r.Projects.InvolvedCount, r.Projects.ActiveCount)
	if r.Calendar.Available {
		fmt.Fprintf(&b, "**会议**：%d 个，共 %d 分钟\n", r.Calendar.MeetingCount, r.Calendar.MeetingMinutes)
	}

	if len(r.People.List) > 0 {
		var names []string
		for i, p := range r.People.List {
			if i >= 20 {
				names = append(names, fmt.Sprintf("等 %d 人", len(r.People.List)))
				break
			}
			names = append(names, p.Name)
		}
		fmt.Fprintf(&b, "**互动的人**：%s\n", strings.Join(names, "、"))
	}
	var inv []string
	for _, g := range r.Groups.List {
		if g.Involved {
			tag := ""
			if g.IsProject {
				tag = "[项目]"
			}
			inv = append(inv, tag+g.Title)
		}
	}
	if len(inv) > 0 {
		fmt.Fprintf(&b, "**参与的群**：%s\n", strings.Join(inv, "、"))
	}

	b.WriteString(calendarMarkdown(r.Calendar, strings.Contains(r.Period.DateKey, "~")))

	if len(r.Awaiting) > 0 {
		b.WriteString("\n## 等待我回复\n")
		for _, p := range r.Awaiting {
			fmt.Fprintf(&b, "- [%s] %s — %s %s：%s（未回复 %d 条）\n",
				p.Reason, p.ConvTitle, cut(p.LastAt, 11, 16), p.LastSender, p.LastText, p.Unanswered)
		}
	}
	if len(r.MentionsMe) > 0 {
		b.WriteString("\n## @我的消息\n")
		for _, m := range r.MentionsMe {
			st := "未回复"
			if m.Replied {
				st = "已在群内回复"
			}
			fmt.Fprintf(&b, "- %s %s / %s：%s（%s）\n", cut(m.Time, 5, 16), m.ConvTitle, m.Sender, m.Text, st)
		}
	}
	return b.String()
}

func runeLen(s string) int { return len([]rune(s)) }

func cut(s string, a, b int) string {
	if len(s) < b {
		return s
	}
	return s[a:b]
}
