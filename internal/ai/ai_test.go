package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qqshow/dingtalk-agent/internal/agentdb"
)

func TestEndpoint(t *testing.T) {
	for in, want := range map[string]string{
		"https://api.deepseek.com/v1":                "https://api.deepseek.com/v1/chat/completions",
		"https://api.deepseek.com/v1/":               "https://api.deepseek.com/v1/chat/completions",
		"http://localhost:11434/v1/chat/completions": "http://localhost:11434/v1/chat/completions",
		" https://open.bigmodel.cn/api/paas/v4 ":     "https://open.bigmodel.cn/api/paas/v4/chat/completions",
	} {
		if got := (&Client{BaseURL: in}).Endpoint(); got != want {
			t.Errorf("Endpoint(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestChatStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sk-test" {
			t.Errorf("auth header = %q", r.Header.Get("Authorization"))
		}
		var req chatRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.Model != "m1" || !req.Stream || len(req.Messages) != 1 {
			t.Errorf("request = %+v", req)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		for _, part := range []string{`{"sum`, `mary":`, `"你好"}`} {
			b, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"delta": map[string]string{"content": part}}}})
			fmt.Fprintf(w, "data: %s\n\n", b)
		}
		fmt.Fprint(w, `data: {"choices":[],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`+"\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()
	var last int
	out, u, err := (&Client{BaseURL: srv.URL, APIKey: "sk-test", Model: "m1"}).Chat(context.Background(),
		[]Message{{Role: "user", Content: "hi"}}, func(n int) { last = n })
	if err != nil {
		t.Fatal(err)
	}
	if out != `{"summary":"你好"}` || u.TotalTokens != 15 || last != len([]rune(out)) {
		t.Fatalf("out=%q usage=%+v progress=%d", out, u, last)
	}
}

func TestChatPlainJSONAndErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok/chat/completions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"choices":[{"message":{"content":"收到"}}],"usage":{"total_tokens":3}}`)
		case "/bad/chat/completions":
			w.WriteHeader(401)
			fmt.Fprint(w, `{"error":{"message":"Invalid API key"}}`)
		case "/cut/chat/completions":
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprint(w, `data: {"choices":[{"delta":{"content":"{\"a\""},"finish_reason":"length"}]}`+"\n\ndata: [DONE]\n\n")
		}
	}))
	defer srv.Close()
	c := &Client{Model: "m"}
	c.BaseURL = srv.URL + "/ok"
	if out, u, err := c.Chat(context.Background(), nil, nil); err != nil || out != "收到" || u.TotalTokens != 3 {
		t.Fatalf("plain: %q %+v %v", out, u, err)
	}
	c.BaseURL = srv.URL + "/bad"
	if _, _, err := c.Chat(context.Background(), nil, nil); err == nil || !strings.Contains(err.Error(), "Invalid API key") {
		t.Fatalf("401 error = %v", err)
	}
	c.BaseURL = srv.URL + "/cut"
	if _, _, err := c.Chat(context.Background(), nil, nil); err == nil || !strings.Contains(err.Error(), "截断") {
		t.Fatalf("truncation error = %v", err)
	}
}

func TestExtractJSON(t *testing.T) {
	cases := map[string]string{
		`{"a":1}`:                               `{"a":1}`,
		"```json\n{\"a\": {\"b\": \"}\"}}\n```": `{"a": {"b": "}"}}`,
		"好的，以下是结果：\n{\"s\":\"x\\\"y\"} 完毕":      `{"s":"x\"y"}`,
		"<think>{draft}</think>\n{\"ok\":true}": `{"ok":true}`,
	}
	for in, want := range cases {
		got, err := ExtractJSON(in)
		if err != nil || got != want {
			t.Errorf("ExtractJSON(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	if _, err := ExtractJSON(`{"a": [1, 2`); err == nil {
		t.Error("incomplete JSON accepted")
	}
}

func TestApplyDaily(t *testing.T) {
	db, err := agentdb.Open(filepath.Join(t.TempDir(), "agent.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	existing := db.UpsertTodos([]agentdb.Todo{{Title: "发送报价单给客户"}, {Title: "整理周会纪要"}})
	id1, id2 := existing[0].Todo.ID, existing[1].Todo.ID

	var out dailyResult
	raw := fmt.Sprintf(`{
	  "summary": "## 今日概览\n- 消息 10 条",
	  "focus": ["回复张三", " "],
	  "new_items": [
	    {"kind": "follow_up", "title": "跟进李四的测试报告", "owner": "李四", "due": "下周", "priority": "HIGH", "conv_title": "项目群"},
	    {"kind": "todo", "title": "准备明天的评审材料", "due": "2026-11-22", "priority": "urgent", "conv_title": "不存在的会话"},
	    {"title": "  "}
	  ],
	  "todo_updates": [
	    {"id": "#%d", "suggest": "done", "reason": "客户回复已收到报价单"},
	    {"id": %d, "suggest": "maybe", "reason": "无效建议"},
	    {"id": 9999, "suggest": "done", "reason": "不存在的事项"}
	  ]
	}`, id1, id2)
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatal(err)
	}
	n, s, err := applyDaily(db, "2026-11-21", AuthorPrefix+" · m", out,
		map[string]string{"项目群": "cid-1"}, map[int64]bool{id1: true, id2: true})
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 || s != 1 {
		t.Fatalf("created=%d suggested=%d, want 2 and 1", n, s)
	}
	sum, _ := db.GetSummary("2026-11-21")
	if sum == nil || len(sum.Focus) != 1 || !strings.HasPrefix(sum.Author, AuthorPrefix) {
		t.Fatalf("summary = %+v", sum)
	}
	todos, _ := db.ListTodos(agentdb.TodoFilter{Status: "open"})
	byTitle := map[string]agentdb.Todo{}
	for _, td := range todos {
		byTitle[td.Title] = td
	}
	f := byTitle["跟进李四的测试报告"]
	if f.Kind != "follow_up" || f.Priority != "high" || f.Due != "" || f.CID != "cid-1" || f.SourceDate != "2026-11-21" {
		t.Errorf("follow-up = %+v", f)
	}
	if p := byTitle["准备明天的评审材料"]; p.Priority != "medium" || p.Due != "2026-11-22" || p.CID != "" {
		t.Errorf("todo = %+v", p)
	}
	// The suggestion is recorded but the status is left for the user.
	got, _ := db.GetTodo(id1)
	if got.Status != "open" || got.AISuggest != "done" || got.AINote == "" {
		t.Fatalf("suggested todo = %+v", got)
	}
	done := "done"
	got, _ = db.UpdateTodo(id1, agentdb.TodoPatch{Status: &done})
	if got.Status != "done" || got.AISuggest != "" {
		t.Fatalf("confirming should clear the suggestion: %+v", got)
	}
}
