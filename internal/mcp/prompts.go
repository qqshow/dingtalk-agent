package mcp

import (
	"strings"

	"github.com/qqshow/dingtalk-agent/internal/analysis"
)

type PromptArg struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

type Prompt struct {
	Name        string
	Title       string
	Description string
	Arguments   []PromptArg
	Template    string
}

func (p Prompt) Render(args map[string]string) string {
	out := p.Template
	for _, a := range p.Arguments {
		v := strings.TrimSpace(args[a.Name])
		// Resolve relative dates so saved summaries/todos get a concrete date.
		if d, err := analysis.ParseDay(v); err == nil {
			v = d.Format("2006-01-02")
		}
		out = strings.ReplaceAll(out, "{{"+a.Name+"}}", v)
	}
	return out
}

// DailySummaryInstructions is the recommended workflow for a daily run; it is
// also printed by `dingtalk-agent prompt`.
const DailySummaryInstructions = `请为我生成 {{date}} 的钉钉工作日报，并整理待办。按以下步骤使用 dingtalk 工具：

1. 调用 dingtalk_daily_report(date="{{date}}") 获取统计数据（其中 calendar 字段含当天会议、明日日程、未处理的审批待办）。
2. 调用 dingtalk_daily_digest(date="{{date}}") 读取按会话分组的消息全文（开头含今日日程/明日日程）；如果有被省略但看起来重要的会话，用 dingtalk_get_messages 单独读取。
3. 调用 dingtalk_list_todos() 查看现有未完成事项；如果聊天记录显示某事项已完成，用 dingtalk_update_todo 把它设为 done。
4. 写一份中文日报（Markdown），包含：
   - **今日概览**：消息量、互动人数、参与的群/项目数、会议数和会议时长（直接用统计数据，不要自己数）。
   - **今日会议**：逐个列出会议，结合会议前后相关群聊/单聊里的讨论（以及会议备注里的 AI 概要，如有）说明会议的结论或后续动作；看不出结论的如实写“聊天中未见结论”。
   - **项目进展**：按项目/群归纳关键进展、决定和风险，每项 1-3 句，注明群名。
   - **需要我重点关注**：@我且未回复的、等待我回复的单聊、领导/客户的要求、截止日期临近、出现问题或风险的事项、未处理的审批待办、我还没回复的会议邀请。按紧急程度排序。
   - **我今天做了什么**：根据我发送的消息和参加的会议归纳。
   - **明日安排**：列出明日日程，并指出需要提前准备的内容（结合相关聊天）。
5. 提取事项并调用 dingtalk_add_todos 写入：
   - kind="todo"：需要我亲自完成的事（别人请我做的、我承诺要做的）。
   - kind="follow_up"：需要我跟踪/催办的事（我安排给别人的、别人承诺会给我回复的、等待外部结果的）。
   - 每项写清楚 title（动词开头，可执行）、detail（背景+原话摘录）、owner、due（消息里有提到才写）、priority、cid、conv_title、msg_id、source_date="{{date}}"。
   - 会议中明确的行动项也要提取；需要为明日会议准备材料的，作为 todo，due 写会议日期。
   - 不要把闲聊、通知、机器人消息、已解决的问题列为事项；审批流程待办已在钉钉里跟踪，不必重复创建。
6. 调用 dingtalk_save_summary(date="{{date}}", summary=日报全文, focus=[重点关注事项，每项一句话])。
7. 最后在回复中给出日报全文，并列出新增的待办和待跟踪事项。`

const WeeklyInstructions = `请生成 {{from}} 至 {{to}} 的钉钉工作周报：

1. 调用 dingtalk_daily_report(from="{{from}}", to="{{to}}") 获取本周统计。
2. 对区间内每一天调用 dingtalk_get_summary 读取已保存的日报；缺失的日期用 dingtalk_daily_digest(date=...) 补读。
3. 调用 dingtalk_list_todos(status="all") 查看本周新增/完成/仍未完成的事项。
4. 调用 dingtalk_calendar(from="{{from}}", to="{{to}}") 查看本周会议；再查下周日程作为“下周安排”。
5. 输出周报：本周概览（数据、会议数与时长）、各项目进展与风险、重要会议及结论、完成的事项、仍待处理/待跟踪的事项、下周安排与需要关注的重点。`

func buildPrompts() []Prompt {
	return []Prompt{
		{
			Name: "daily_summary", Title: "钉钉每日总结",
			Description: "生成某天的工作日报：统计、项目进展、需重点关注事项，并自动写入待办和待跟踪事项",
			Arguments:   []PromptArg{{Name: "date", Description: "YYYY-MM-DD / today / yesterday，默认 today"}},
			Template:    DailySummaryInstructions,
		},
		{
			Name: "weekly_summary", Title: "钉钉周报",
			Description: "汇总一段时间的日报与待办生成周报",
			Arguments: []PromptArg{
				{Name: "from", Description: "开始日期 YYYY-MM-DD", Required: true},
				{Name: "to", Description: "结束日期 YYYY-MM-DD，默认 today"},
			},
			Template: WeeklyInstructions,
		},
	}
}
