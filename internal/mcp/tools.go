package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/qqshow/dingtalk-agent/internal/agentdb"
	"github.com/qqshow/dingtalk-agent/internal/analysis"
	"github.com/qqshow/dingtalk-agent/internal/store"
	"github.com/qqshow/dingtalk-agent/internal/textutil"
)

type Tool struct {
	Name        string
	Title       string
	Description string
	Schema      map[string]any
	ReadOnly    bool
	Run         func(ctx context.Context, args map[string]any) (string, error)
}

// ------------------------------------------------------------ arg helpers

func str(a map[string]any, k string) string {
	if v, ok := a[k]; ok && v != nil {
		return strings.TrimSpace(fmt.Sprint(v))
	}
	return ""
}

func num(a map[string]any, k string, def int) int {
	switch v := a[k].(type) {
	case float64:
		return int(v)
	case string:
		var n int
		if _, err := fmt.Sscan(v, &n); err == nil {
			return n
		}
	}
	return def
}

func boolean(a map[string]any, k string, def bool) bool {
	switch v := a[k].(type) {
	case bool:
		return v
	case string:
		return v == "true" || v == "1"
	}
	return def
}

func toJSON(v any) string {
	var b strings.Builder
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", " ")
	enc.Encode(v)
	return b.String()
}

func obj(props map[string]any, required ...string) map[string]any {
	s := map[string]any{"type": "object", "properties": props}
	if len(required) > 0 {
		s["required"] = required
	}
	return s
}

func prop(typ, desc string) map[string]any { return map[string]any{"type": typ, "description": desc} }

var (
	pDate    = prop("string", "日期：YYYY-MM-DD、today、yesterday 或 -N（N 天前）。默认 today")
	pFrom    = prop("string", "区间开始日期（含），与 to 搭配用于周报等多日范围")
	pTo      = prop("string", "区间结束日期（含）")
	pRefresh = prop("boolean", "数据超过5分钟时先同步最新钉钉数据，默认 true")
)

func periodProps(extra map[string]any) map[string]any {
	p := map[string]any{"date": pDate, "from": pFrom, "to": pTo, "refresh": pRefresh}
	for k, v := range extra {
		p[k] = v
	}
	return p
}

// ------------------------------------------------------------------ tools

func (s *Server) buildTools() []Tool {
	a := s.a
	todoItem := obj(map[string]any{
		"kind":        map[string]any{"type": "string", "enum": []string{"todo", "follow_up"}, "description": "todo=我需要做的事；follow_up=需要我跟踪/催办他人的事"},
		"title":       prop("string", "简短、可执行的标题"),
		"detail":      prop("string", "背景与细节（可引用原话）"),
		"priority":    map[string]any{"type": "string", "enum": []string{"high", "medium", "low"}},
		"due":         prop("string", "截止日期 YYYY-MM-DD（如消息中有提及）"),
		"owner":       prop("string", "负责人或需要跟进的人"),
		"cid":         prop("string", "来源会话 cid"),
		"conv_title":  prop("string", "来源会话名称"),
		"msg_id":      prop("integer", "来源消息 id"),
		"source_date": prop("string", "来源日期 YYYY-MM-DD"),
		"dedupe_key":  prop("string", "可选去重键；相同键的事项只会更新不会重复创建"),
	}, "title")

	return []Tool{
		{
			Name: "dingtalk_daily_report", Title: "每日统计", ReadOnly: true,
			Description: "获取某天（或日期区间）的结构化统计：消息量、我发送/收到、互动的人（及次数）、活跃/参与的群、" +
				"参与的项目群、@我的消息（是否已回复）、等待我回复的会话，以及已保存的总结和未完成待办。",
			Schema: obj(periodProps(nil)),
			Run: func(ctx context.Context, x map[string]any) (string, error) {
				r, err := a.Report(str(x, "date"), str(x, "from"), str(x, "to"), boolean(x, "refresh", true))
				if err != nil {
					return "", err
				}
				return toJSON(r), nil
			},
		},
		{
			Name: "dingtalk_daily_digest", Title: "每日消息摘要", ReadOnly: true,
			Description: "获取某天（或区间）按会话分组、按重要程度排序的消息全文（Markdown），开头含统计、等待我回复和@我的消息。" +
				"用于生成每日总结、识别需要关注的内容、提取待办和待跟踪事项。内容过长时低优先级会话会被省略并列出名称，" +
				"可用 dingtalk_get_messages 单独读取。",
			Schema: obj(periodProps(map[string]any{
				"scope":     map[string]any{"type": "string", "enum": []string{"all", "involved"}, "description": "all=全部会话；involved=只看单聊、我参与的群和项目群。默认 all"},
				"max_chars": prop("integer", "输出字符上限，默认 60000"),
			})),
			Run: func(ctx context.Context, x map[string]any) (string, error) {
				d, err := a.Digest(str(x, "date"), str(x, "from"), str(x, "to"), analysis.DigestOptions{
					Scope: str(x, "scope"), MaxChars: num(x, "max_chars", 60000), SkipBots: true,
				}, boolean(x, "refresh", true))
				if err != nil {
					return "", err
				}
				return d.Markdown, nil
			},
		},
		{
			Name: "dingtalk_list_conversations", Title: "会话列表", ReadOnly: true,
			Description: "按名称关键字列出会话（单聊/群），返回 cid、名称、类型、是否项目群、最后活跃时间。",
			Schema: obj(map[string]any{
				"keyword": prop("string", "名称关键字（可选）"),
				"type":    map[string]any{"type": "string", "enum": []string{"single", "group"}},
				"limit":   prop("integer", "默认 30"),
			}),
			Run: func(ctx context.Context, x map[string]any) (string, error) {
				snap, err := a.Snapshot(false)
				if err != nil {
					return "", err
				}
				total, list := snap.ConversationList(str(x, "keyword"), str(x, "type"), num(x, "limit", 30), 0)
				type row struct {
					CID, Title, Type string
					IsProject        bool   `json:"is_project"`
					LastActive       string `json:"last_active"`
					Members          int64  `json:"members"`
				}
				rows := []row{}
				for _, c := range list {
					rows = append(rows, row{c.CID, c.Title, c.Type, c.IsProject, store.FormatTS(c.LastModify), c.MemberCount})
				}
				return toJSON(map[string]any{"total": total, "conversations": rows}), nil
			},
		},
		{
			Name: "dingtalk_get_messages", Title: "读取会话消息", ReadOnly: true,
			Description: "读取单个会话在某天/区间内的全部消息（纯文本，每行：时间 发送人: 内容）。cid 可从日报、摘要或会话列表获得。",
			Schema: obj(periodProps(map[string]any{
				"cid":   prop("string", "会话 cid"),
				"limit": prop("integer", "最多返回条数，默认 500"),
			}), "cid"),
			Run: func(ctx context.Context, x map[string]any) (string, error) {
				p, err := analysis.ParsePeriod(str(x, "date"), str(x, "from"), str(x, "to"))
				if err != nil {
					return "", err
				}
				snap, err := a.Snapshot(boolean(x, "refresh", true))
				if err != nil {
					return "", err
				}
				cid := str(x, "cid")
				if snap.ConvByID[cid] == nil {
					return "", fmt.Errorf("unknown cid %q; use dingtalk_list_conversations", cid)
				}
				msgs, err := snap.Range(store.RangeQuery{From: p.FromMS(), To: p.ToMS(), CIDs: []string{cid},
					Limit: num(x, "limit", 500), Desc: true})
				if err != nil {
					return "", err
				}
				var b strings.Builder
				fmt.Fprintf(&b, "# %s（%s，共 %d 条）\n", snap.ConvTitle(cid), p.Label, len(msgs))
				for i := len(msgs) - 1; i >= 0; i-- {
					m := msgs[i]
					sender := m.SenderName
					if m.IsMine {
						sender = "我"
					}
					mark := ""
					if m.MentionsMe && !m.IsMine {
						mark = "【@我】"
					}
					fmt.Fprintf(&b, "%s %s: %s%s\n", cut(m.CreatedAtStr, 5, 16), sender, mark, m.AgentText())
				}
				return b.String(), nil
			},
		},
		{
			Name: "dingtalk_calendar", Title: "日程", ReadOnly: true,
			Description: "读取钉钉日历：某天/区间内的会议（时间、会议室/地点、组织者、是否视频会议、我是否回复邀请、备注——" +
				"有时含 AI 会议概要）、审批/流程待办、AI听记、Teambition任务。查明天或下周的安排时传对应日期。" +
				"本地数据不含完整参会人名单。",
			Schema: obj(periodProps(map[string]any{
				"include_closed": prop("boolean", "包含已取消的会议和已完成的待办，默认 false"),
			})),
			Run: func(ctx context.Context, x map[string]any) (string, error) {
				res, err := a.Calendar(str(x, "date"), str(x, "from"), str(x, "to"),
					boolean(x, "include_closed", false), boolean(x, "refresh", true))
				if err != nil {
					return "", err
				}
				if !res.Available {
					return "本机没有可用的钉钉日历数据", nil
				}
				multi := strings.Contains(res.Period, "~")
				var b strings.Builder
				fmt.Fprintf(&b, "# 日程 %s（%d 项）\n", res.Period, len(res.Events))
				kindName := map[string]string{"meeting": "会议", "approval_todo": "审批待办", "ai_minutes": "AI听记", "task": "任务", "leave": "请假"}
				for _, e := range res.Events {
					fmt.Fprintf(&b, "- [%s] %s %s", kindName[e.Kind], e.TimeRange(multi), e.Subject)
					meta := analysis.EventMetadata(e)
					if e.Status != "normal" && e.Status != "" {
						meta = append(meta, map[string]string{"cancelled": "已取消", "done": "已完成",
							"in_progress": "处理中", "open": "未处理"}[e.Status])
					}
					if len(meta) > 0 {
						b.WriteString("（" + strings.Join(meta, "，") + "）")
					}
					if e.Comment != "" {
						b.WriteString("\n  备注：" + textutil.Snippet(e.Comment, 500))
					}
					b.WriteString("\n")
				}
				return b.String(), nil
			},
		},
		{
			Name: "dingtalk_search", Title: "搜索消息", ReadOnly: true,
			Description: "在全部聊天记录中按关键字搜索（可限定日期区间或会话），按时间倒序返回。",
			Schema: obj(map[string]any{
				"query": prop("string", "关键字"),
				"from":  pFrom, "to": pTo,
				"cid":   prop("string", "限定会话（可选）"),
				"limit": prop("integer", "默认 50"),
			}, "query"),
			Run: func(ctx context.Context, x map[string]any) (string, error) {
				snap, err := a.Snapshot(false)
				if err != nil {
					return "", err
				}
				q := store.RangeQuery{Keyword: str(x, "query"), Desc: true, Limit: num(x, "limit", 50)}
				if q.Keyword == "" {
					return "", errors.New("query is required")
				}
				if c := str(x, "cid"); c != "" {
					q.CIDs = []string{c}
				}
				if str(x, "from") != "" || str(x, "to") != "" {
					p, err := analysis.ParsePeriod("", str(x, "from"), str(x, "to"))
					if err != nil {
						return "", err
					}
					q.From, q.To = p.FromMS(), p.ToMS()
				}
				msgs, err := snap.Range(q)
				if err != nil {
					return "", err
				}
				var b strings.Builder
				fmt.Fprintf(&b, "找到 %d 条（最多显示 %d）\n", len(msgs), q.Limit)
				for _, m := range msgs {
					fmt.Fprintf(&b, "- %s [%s] %s: %s  (cid=%s)\n", cut(m.CreatedAtStr, 0, 16), snap.ConvTitle(m.CID),
						m.SenderName, textutil.Snippet(m.AgentText(), 300), m.CID)
				}
				return b.String(), nil
			},
		},
		{
			Name: "dingtalk_sync", Title: "同步数据",
			Description: "立即从钉钉客户端解密最新聊天数据（约 2 秒）。其他工具在 refresh=true 时会自动按需同步。",
			Schema:      obj(map[string]any{}),
			Run: func(ctx context.Context, x map[string]any) (string, error) {
				st, err := a.Syncer.SyncNow()
				if err != nil {
					return "", err
				}
				return toJSON(st), nil
			},
		},
		{
			Name: "dingtalk_save_summary", Title: "保存每日总结",
			Description: "保存（覆盖）某天的总结，网页“日报”页面会展示。summary 用 Markdown；focus 为需要用户重点关注的事项列表。",
			Schema: obj(map[string]any{
				"date":    pDate,
				"summary": prop("string", "Markdown 格式的总结"),
				"focus":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "重点关注事项（每项一句话）"},
			}, "summary"),
			Run: func(ctx context.Context, x map[string]any) (string, error) {
				p, err := analysis.ParsePeriod(str(x, "date"), "", "")
				if err != nil {
					return "", err
				}
				var focus []string
				if l, ok := x["focus"].([]any); ok {
					for _, v := range l {
						focus = append(focus, fmt.Sprint(v))
					}
				}
				sum, err := a.Agent.SaveSummary(agentdb.Summary{DateKey: p.DateKey(), Summary: str(x, "summary"),
					Focus: focus, Author: "mcp"})
				if err != nil {
					return "", err
				}
				return "已保存 " + sum.DateKey + " 的总结", nil
			},
		},
		{
			Name: "dingtalk_get_summary", Title: "读取每日总结", ReadOnly: true,
			Description: "读取已保存的某天总结（可用于对比昨天的重点是否已处理）。",
			Schema:      obj(map[string]any{"date": pDate}),
			Run: func(ctx context.Context, x map[string]any) (string, error) {
				p, err := analysis.ParsePeriod(str(x, "date"), "", "")
				if err != nil {
					return "", err
				}
				sum, err := a.Agent.GetSummary(p.DateKey())
				if err != nil {
					return "", err
				}
				if sum == nil {
					return "该日期还没有保存的总结", nil
				}
				return toJSON(sum), nil
			},
		},
		{
			Name: "dingtalk_add_todos", Title: "添加待办/待跟踪",
			Description: "批量添加待办(todo)或待跟踪(follow_up)事项。按 dedupe_key（缺省为 类型+会话+标题）去重：" +
				"已存在的事项只更新内容、保留完成状态，所以每天重复运行是安全的。",
			Schema: obj(map[string]any{"items": map[string]any{"type": "array", "items": todoItem}}, "items"),
			Run: func(ctx context.Context, x map[string]any) (string, error) {
				raw, _ := json.Marshal(x["items"])
				var items []agentdb.Todo
				if err := json.Unmarshal(raw, &items); err != nil {
					return "", fmt.Errorf("items: %w", err)
				}
				res := a.Agent.UpsertTodos(items)
				created, updated, failed := 0, 0, 0
				var lines []string
				for _, r := range res {
					switch {
					case r.Error != "":
						failed++
						lines = append(lines, "失败: "+r.Todo.Title+" — "+r.Error)
					case r.Created:
						created++
						lines = append(lines, fmt.Sprintf("新建 #%d %s", r.Todo.ID, r.Todo.Title))
					default:
						updated++
						lines = append(lines, fmt.Sprintf("已存在 #%d %s（状态 %s）", r.Todo.ID, r.Todo.Title, r.Todo.Status))
					}
				}
				return fmt.Sprintf("新建 %d，已存在 %d，失败 %d\n%s", created, updated, failed, strings.Join(lines, "\n")), nil
			},
		},
		{
			Name: "dingtalk_list_todos", Title: "列出待办", ReadOnly: true,
			Description: "列出待办/待跟踪事项。默认只列未完成的。生成新待办前建议先调用，避免重复，并检查旧事项是否已在聊天中完成。",
			Schema: obj(map[string]any{
				"status":      map[string]any{"type": "string", "enum": []string{"open", "done", "dropped", "all"}},
				"kind":        map[string]any{"type": "string", "enum": []string{"todo", "follow_up"}},
				"source_date": prop("string", "只看某天生成的事项"),
			}),
			Run: func(ctx context.Context, x map[string]any) (string, error) {
				l, err := a.Agent.ListTodos(agentdb.TodoFilter{Status: str(x, "status"), Kind: str(x, "kind"),
					SourceDate: str(x, "source_date")})
				if err != nil {
					return "", err
				}
				return toJSON(map[string]any{"count": len(l), "todos": l}), nil
			},
		},
		{
			Name: "dingtalk_update_todo", Title: "更新待办",
			Description: "更新事项状态或内容，例如发现聊天中已完成时设为 done，不再需要时设为 dropped。",
			Schema: obj(map[string]any{
				"id":       prop("integer", "事项 id"),
				"status":   map[string]any{"type": "string", "enum": []string{"open", "done", "dropped"}},
				"title":    prop("string", ""),
				"detail":   prop("string", ""),
				"priority": map[string]any{"type": "string", "enum": []string{"high", "medium", "low"}},
				"due":      prop("string", "YYYY-MM-DD"),
				"owner":    prop("string", ""),
			}, "id"),
			Run: func(ctx context.Context, x map[string]any) (string, error) {
				var p agentdb.TodoPatch
				for k, dst := range map[string]**string{"status": &p.Status, "title": &p.Title, "detail": &p.Detail,
					"priority": &p.Priority, "due": &p.Due, "owner": &p.Owner} {
					if _, ok := x[k]; ok {
						v := str(x, k)
						*dst = &v
					}
				}
				t, err := a.Agent.UpdateTodo(int64(num(x, "id", 0)), p)
				if err != nil {
					return "", err
				}
				return toJSON(t), nil
			},
		},
	}
}

func cut(s string, a, b int) string {
	if len(s) < b {
		return s
	}
	return s[a:b]
}
