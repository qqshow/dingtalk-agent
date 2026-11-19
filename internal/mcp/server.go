// Package mcp implements a small Model Context Protocol server (tools and
// prompts) over stdio and streamable HTTP (JSON responses, no SSE).
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/qqshow/dingtalk-agent/internal/app"
)

var supportedVersions = []string{"2025-06-18", "2025-03-26", "2024-11-05"}

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type Server struct {
	a       *app.App
	tools   []Tool
	byName  map[string]Tool
	prompts []Prompt
}

func New(a *app.App) *Server {
	s := &Server{a: a, byName: map[string]Tool{}}
	s.tools = s.buildTools()
	for _, t := range s.tools {
		s.byName[t.Name] = t
	}
	s.prompts = buildPrompts()
	return s
}

// handle processes one message; nil response for notifications.
func (s *Server) handle(ctx context.Context, req request) *response {
	isNotification := len(req.ID) == 0 || string(req.ID) == "null"
	result, err := s.dispatch(ctx, req)
	if isNotification {
		return nil
	}
	resp := &response{JSONRPC: "2.0", ID: req.ID}
	if err != nil {
		resp.Error = err
	} else {
		resp.Result = result
	}
	return resp
}

func (s *Server) dispatch(ctx context.Context, req request) (any, *rpcError) {
	switch req.Method {
	case "initialize":
		var p struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		json.Unmarshal(req.Params, &p)
		ver := supportedVersions[0]
		for _, v := range supportedVersions {
			if v == p.ProtocolVersion {
				ver = v
			}
		}
		return map[string]any{
			"protocolVersion": ver,
			"capabilities": map[string]any{
				"tools":   map[string]any{"listChanged": false},
				"prompts": map[string]any{"listChanged": false},
			},
			"serverInfo": map[string]any{"name": "dingtalk-agent", "title": "钉钉聊天记录", "version": s.a.Version},
			"instructions": "访问用户本机钉钉（DingTalk）聊天记录。做每日总结时：先调用 dingtalk_daily_report 获取结构化统计，" +
				"再用 dingtalk_daily_digest 读取按会话分组的消息全文（含当天和次日日程，日程详情可用 dingtalk_calendar），总结后用 dingtalk_save_summary 保存，" +
				"用 dingtalk_add_todos 写入待办(todo)和待跟踪(follow_up)事项。数据默认每5分钟内会自动刷新。",
		}, nil
	case "ping":
		return map[string]any{}, nil
	case "notifications/initialized", "notifications/cancelled", "initialized":
		return nil, nil
	case "tools/list":
		list := make([]map[string]any, 0, len(s.tools))
		for _, t := range s.tools {
			list = append(list, map[string]any{"name": t.Name, "title": t.Title, "description": t.Description,
				"inputSchema": t.Schema, "annotations": map[string]any{"readOnlyHint": t.ReadOnly}})
		}
		return map[string]any{"tools": list}, nil
	case "tools/call":
		var p struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return nil, &rpcError{Code: -32602, Message: "invalid params"}
		}
		t, ok := s.byName[p.Name]
		if !ok {
			return nil, &rpcError{Code: -32602, Message: "unknown tool: " + p.Name}
		}
		args := map[string]any{}
		if len(p.Arguments) > 0 && string(p.Arguments) != "null" {
			if err := json.Unmarshal(p.Arguments, &args); err != nil {
				return toolError("arguments must be an object"), nil
			}
		}
		out, err := safeCall(t, ctx, args)
		if err != nil {
			return toolError(err.Error()), nil
		}
		return map[string]any{"content": []map[string]any{{"type": "text", "text": out}}, "isError": false}, nil
	case "prompts/list":
		list := make([]map[string]any, 0, len(s.prompts))
		for _, p := range s.prompts {
			list = append(list, map[string]any{"name": p.Name, "title": p.Title, "description": p.Description,
				"arguments": p.Arguments})
		}
		return map[string]any{"prompts": list}, nil
	case "prompts/get":
		var p struct {
			Name      string            `json:"name"`
			Arguments map[string]string `json:"arguments"`
		}
		json.Unmarshal(req.Params, &p)
		for _, pr := range s.prompts {
			if pr.Name == p.Name {
				return map[string]any{"description": pr.Description, "messages": []map[string]any{{
					"role": "user", "content": map[string]any{"type": "text", "text": pr.Render(p.Arguments)},
				}}}, nil
			}
		}
		return nil, &rpcError{Code: -32602, Message: "unknown prompt: " + p.Name}
	case "resources/list":
		return map[string]any{"resources": []any{}}, nil
	}
	return nil, &rpcError{Code: -32601, Message: "method not found: " + req.Method}
}

func safeCall(t Tool, ctx context.Context, args map[string]any) (out string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("internal error: %v", r)
			slog.Error("tool panic", "tool", t.Name, "panic", r)
		}
	}()
	return t.Run(ctx, args)
}

func toolError(msg string) map[string]any {
	return map[string]any{"content": []map[string]any{{"type": "text", "text": "Error: " + msg}}, "isError": true}
}

// handleRaw processes a single message or a batch.
func (s *Server) handleRaw(ctx context.Context, raw []byte) []byte {
	raw = []byte(strings.TrimSpace(string(raw)))
	if len(raw) == 0 {
		return nil
	}
	if raw[0] == '[' {
		var reqs []request
		if err := json.Unmarshal(raw, &reqs); err != nil {
			return mustJSON(response{JSONRPC: "2.0", ID: json.RawMessage("null"), Error: &rpcError{-32700, "parse error"}})
		}
		var out []*response
		for _, r := range reqs {
			if resp := s.handle(ctx, r); resp != nil {
				out = append(out, resp)
			}
		}
		if len(out) == 0 {
			return nil
		}
		return mustJSON(out)
	}
	var req request
	if err := json.Unmarshal(raw, &req); err != nil {
		return mustJSON(response{JSONRPC: "2.0", ID: json.RawMessage("null"), Error: &rpcError{-32700, "parse error"}})
	}
	if resp := s.handle(ctx, req); resp != nil {
		return mustJSON(resp)
	}
	return nil
}

func mustJSON(v any) []byte {
	var b strings.Builder
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.Encode(v)
	return []byte(strings.TrimRight(b.String(), "\n"))
}

// ServeStdio speaks newline-delimited JSON-RPC on r/w until EOF.
func (s *Server) ServeStdio(ctx context.Context, r io.Reader, w io.Writer) error {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 64<<20)
	var mu sync.Mutex
	var wg sync.WaitGroup
	for sc.Scan() {
		line := append([]byte(nil), sc.Bytes()...)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if out := s.handleRaw(ctx, line); out != nil {
				mu.Lock()
				w.Write(append(out, '\n'))
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return sc.Err()
}

// ServeHTTP implements the streamable HTTP transport with plain JSON replies.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
	case http.MethodGet:
		w.Header().Set("Allow", "POST")
		http.Error(w, "SSE stream not supported; use POST", http.StatusMethodNotAllowed)
		return
	case http.MethodDelete:
		w.WriteHeader(http.StatusNoContent)
		return
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 16<<20))
	if err != nil {
		http.Error(w, "read error", http.StatusBadRequest)
		return
	}
	out := s.handleRaw(r.Context(), body)
	if out == nil {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(out)
}
