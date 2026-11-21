package ai

import (
	"fmt"
	"strings"

	"github.com/qqshow/dingtalk-agent/internal/agentdb"
	"github.com/qqshow/dingtalk-agent/internal/analysis"
	"github.com/qqshow/dingtalk-agent/internal/store"
)

const dailySystem = `你是用户的工作助理。用户会给你他某一天的钉钉工作数据（统计、日程、与他相关的聊天记录摘要）以及他现有的未完成事项。
请据此写日报、找出需要他重点关注的事、提取新的待办和待跟踪事项，并判断现有事项的进展。

只输出一个 JSON 对象，不要输出任何其他文字，格式如下：
{
  "summary": "日报全文（Markdown）",
  "focus": ["需要重点关注的事项，每项一句话，按紧急程度排序"],
  "new_items": [
    {"kind": "todo 或 follow_up", "title": "动词开头、可执行的标题", "detail": "背景和原话摘录",
     "owner": "负责人姓名", "due": "YYYY-MM-DD，消息里提到截止时间才写，否则为空字符串",
     "priority": "high、medium 或 low", "conv_title": "来源会话名称，必须与摘要中的会话名完全一致"}
  ],
  "todo_updates": [
    {"id": 现有事项编号, "suggest": "done 或 dropped", "reason": "依据，引用聊天中的关键内容"}
  ]
}

日报（summary）使用中文 Markdown，包含以下小节（没有内容的小节写“无”）：
## 今日概览
消息量、互动人数、参与的群/项目数、会议数和会议时长。直接使用数据中的统计数字，不要自己数。
## 今日会议
逐个列出会议，结合相关聊天说明结论或后续动作；看不出结论的如实写“聊天中未见结论”。
## 项目进展
按项目/群归纳关键进展、决定和风险，每项 1-3 句，注明群名。
## 需要我重点关注
@我且未回复的、等待我回复的单聊、领导/客户的要求、临近截止的、出现问题或风险的事项、未处理的审批、未回复的会议邀请。按紧急程度排序。
## 我今天做了什么
根据我发送的消息和参加的会议归纳。
## 明日安排
列出明日日程；再结合未完成事项的截止日期和优先级，给出明天的时间安排建议（哪个时段做什么、会前要准备什么）。

提取事项（new_items）的规则：
- kind="todo"：需要我亲自完成的事（别人请我做的、我承诺要做的、会议中分给我的）。
- kind="follow_up"：需要我跟踪或催办的事（我安排给别人的、别人承诺会回复我的、等待外部结果的）。
- 不要提取闲聊、通知、机器人消息、已经解决的问题；钉钉审批待办已在钉钉中跟踪，不要重复创建。
- 与现有未完成事项重复的不要再提取。

判断现有事项（todo_updates）的规则：
- 只有聊天记录明确显示已完成（例如对方确认收到、我说已经发了、问题已解决）才建议 done；明确取消或不再需要才建议 dropped。
- 没有明确依据的不要列出。你的建议需要用户确认后才会生效。`

const weeklySystem = `你是用户的工作助理。用户会给你他一周的钉钉工作统计、每天的日报、事项变化和下周日程。请写一份周报。

只输出一个 JSON 对象，不要输出任何其他文字，格式如下：
{
  "summary": "周报全文（Markdown）",
  "focus": ["下周需要重点关注的事项，每项一句话，按重要程度排序"]
}

周报（summary）使用中文 Markdown，包含以下小节（没有内容的小节写“无”）：
## 本周概览
消息量、互动人数、参与的群/项目数、会议数和会议时长。直接使用统计数字，不要自己数。
## 项目进展与风险
按项目归纳本周的进展、关键决定和风险。
## 重要会议与结论
## 本周完成
## 待处理与待跟踪
仍未完成的事项，标出已逾期或临近截止的。
## 下周安排
结合下周日程和未完成事项，给出下周的工作安排建议和需要提前准备的事。

某天没有日报时，只根据统计数字和事项变化概括，不要编造细节。`

func todoLine(t agentdb.Todo) string {
	kind := "待办"
	if t.Kind == "follow_up" {
		kind = "待跟踪"
	}
	prio := map[string]string{"high": "高", "medium": "中", "low": "低"}[t.Priority]
	var meta []string
	meta = append(meta, kind, prio+"优先级")
	if t.Due != "" {
		meta = append(meta, "截止 "+t.Due)
	}
	if t.Owner != "" {
		meta = append(meta, "负责人 "+t.Owner)
	}
	if t.ConvTitle != "" {
		meta = append(meta, "来源 "+t.ConvTitle)
	}
	if t.SourceDate != "" {
		meta = append(meta, "提出于 "+t.SourceDate)
	}
	line := fmt.Sprintf("- [#%d] %s（%s）", t.ID, t.Title, strings.Join(meta, "，"))
	if d := strings.TrimSpace(t.Detail); d != "" {
		line += "\n  " + snippet(d, 160)
	}
	return line
}

func snippet(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	r := []rune(s)
	if len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}

// dailyUserPrompt is the data part of the daily request.
func dailyUserPrompt(date string, open []agentdb.Todo, digestMD string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "日报日期：%s\n\n", date)
	b.WriteString("## 现有未完成事项（判断进展时用 [#编号] 中的数字作为 id）\n")
	if len(open) == 0 {
		b.WriteString("无\n")
	}
	for _, t := range open {
		b.WriteString(todoLine(t))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(digestMD)
	return b.String()
}

// weeklyInput is everything the weekly report is built from; no raw chat.
type weeklyInput struct {
	From, To  string
	Report    *analysis.Report
	Summaries map[string]*agentdb.Summary // by date
	Days      []string
	Todos     []agentdb.Todo
	NextWeek  []store.Event
	NextLabel string
}

func weeklyUserPrompt(w weeklyInput) string {
	var b strings.Builder
	r := w.Report
	fmt.Fprintf(&b, "周报区间：%s 至 %s\n\n## 本周统计\n", w.From, w.To)
	t := r.Totals
	fmt.Fprintf(&b, "- 消息 %d 条（我发送 %d / 收到 %d），活跃会话 %d 个（单聊 %d、群 %d）\n",
		t.Messages, t.SentByMe, t.Received, t.ActiveConversations, t.ActiveSingleChats, t.ActiveGroups)
	fmt.Fprintf(&b, "- 互动 %d 人，参与群 %d 个，参与项目 %d 个\n", r.People.Count, r.Groups.InvolvedCount, r.Projects.InvolvedCount)
	if r.Calendar.Available {
		fmt.Fprintf(&b, "- 会议 %d 个，共 %d 分钟\n", r.Calendar.MeetingCount, r.Calendar.MeetingMinutes)
	}
	var projects []string
	for _, g := range r.Projects.List {
		if g.Involved {
			projects = append(projects, fmt.Sprintf("%s（%d 条，我 %d 条）", g.Title, g.Messages, g.MyMessages))
		}
	}
	if len(projects) > 0 {
		fmt.Fprintf(&b, "- 参与的项目群：%s\n", strings.Join(projects, "、"))
	}
	if r.Calendar.Available && len(r.Calendar.Meetings) > 0 {
		b.WriteString("\n## 本周会议\n")
		for _, e := range r.Calendar.Meetings {
			fmt.Fprintf(&b, "- %s %s（%d 分钟）\n", e.StartStr, e.Subject, e.Minutes)
		}
	}

	b.WriteString("\n## 每日日报\n")
	for _, d := range w.Days {
		s := w.Summaries[d]
		if s == nil {
			fmt.Fprintf(&b, "\n### %s\n（没有日报）\n", d)
			continue
		}
		fmt.Fprintf(&b, "\n### %s\n%s\n", d, strings.TrimSpace(s.Summary))
	}

	var done, open, dropped []string
	for _, td := range w.Todos {
		switch td.Status {
		case "done":
			done = append(done, todoLine(td))
		case "dropped":
			dropped = append(dropped, todoLine(td))
		default:
			open = append(open, todoLine(td))
		}
	}
	section := func(title string, lines []string) {
		fmt.Fprintf(&b, "\n## %s\n", title)
		if len(lines) == 0 {
			b.WriteString("无\n")
			return
		}
		b.WriteString(strings.Join(lines, "\n"))
		b.WriteString("\n")
	}
	section("本周完成的事项", done)
	section("仍未完成的事项", open)
	section("本周放弃的事项", dropped)

	fmt.Fprintf(&b, "\n## 下周日程（%s）\n", w.NextLabel)
	if len(w.NextWeek) == 0 {
		b.WriteString("无\n")
	}
	for _, e := range w.NextWeek {
		when := e.StartStr
		if e.AllDay {
			when = e.StartStr[:10] + " 全天"
		}
		fmt.Fprintf(&b, "- %s %s\n", when, e.Subject)
	}
	return b.String()
}
