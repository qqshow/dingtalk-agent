package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
)

// ToolSpec describes a function the model may call.
type ToolSpec struct {
	Name        string
	Description string
	Parameters  map[string]any // JSON schema
}

// ToolCall is a function call requested by the model.
type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON text
}

// ChatMessage is a message in a tool-using conversation.
type ChatMessage struct {
	Role       string     // system | user | assistant | tool
	Content    string     //
	Images     []string   // user only: data URLs (data:image/...;base64,...)
	ToolCalls  []ToolCall // assistant only
	ToolCallID string     // tool only
}

// StatusError is a non-2xx answer from the AI endpoint.
type StatusError struct {
	Code int
	Msg  string
}

func (e *StatusError) Error() string { return fmt.Sprintf("AI 接口返回 %d: %s", e.Code, e.Msg) }

// IsRequestRejected reports whether err is a 4xx answer other than an
// authentication, permission or rate-limit error: the endpoint refused what
// was sent (for example tools or images the model does not support).
func IsRequestRejected(err error) bool {
	var se *StatusError
	if !errors.As(err, &se) || se.Code/100 != 4 {
		return false
	}
	return se.Code != 401 && se.Code != 403 && se.Code != 429
}

// ErrToolsUnsupported means the endpoint rejected the request because of the
// tools field; the caller may retry without tools.
var ErrToolsUnsupported = errors.New("model does not support tool calls")

type wireFunc struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type wireToolCall struct {
	Index    *int     `json:"index,omitempty"`
	ID       string   `json:"id,omitempty"`
	Type     string   `json:"type,omitempty"`
	Function wireFunc `json:"function"`
}

type wireMessage struct {
	Role       string         `json:"role"`
	Content    any            `json:"content"` // string, or []wirePart when images are attached
	ToolCalls  []wireToolCall `json:"tool_calls,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
}

// wirePart is one item of an OpenAI-style multimodal content array.
type wirePart struct {
	Type     string   `json:"type"` // text | image_url
	Text     string   `json:"text,omitempty"`
	ImageURL *wireURL `json:"image_url,omitempty"`
}

type wireURL struct {
	URL string `json:"url"`
}

// wireContent keeps plain-text messages as a string, so endpoints without
// vision support see exactly what they saw before images existed.
func wireContent(m ChatMessage) any {
	if len(m.Images) == 0 {
		return m.Content
	}
	parts := make([]wirePart, 0, len(m.Images)+1)
	if m.Content != "" {
		parts = append(parts, wirePart{Type: "text", Text: m.Content})
	}
	for _, u := range m.Images {
		parts = append(parts, wirePart{Type: "image_url", ImageURL: &wireURL{URL: u}})
	}
	return parts
}

type wireTool struct {
	Type     string `json:"type"`
	Function struct {
		Name        string         `json:"name"`
		Description string         `json:"description"`
		Parameters  map[string]any `json:"parameters"`
	} `json:"function"`
}

type toolRequest struct {
	Model       string        `json:"model"`
	Messages    []wireMessage `json:"messages"`
	Tools       []wireTool    `json:"tools,omitempty"`
	Stream      bool          `json:"stream"`
	Temperature float64       `json:"temperature"`
}

type toolChoice struct {
	Message struct {
		Content   string         `json:"content"`
		ToolCalls []wireToolCall `json:"tool_calls"`
	} `json:"message"`
	Delta struct {
		Content   string         `json:"content"`
		ToolCalls []wireToolCall `json:"tool_calls"`
	} `json:"delta"`
	FinishReason string `json:"finish_reason"`
}

type toolResponse struct {
	Choices []toolChoice `json:"choices"`
	Usage   *Usage       `json:"usage"`
	Error   *apiError    `json:"error"`
}

// Converse sends a conversation with optional tools. Text is streamed to
// onText as it arrives; the returned message holds the full text and any
// tool calls the model requested.
func (c *Client) Converse(ctx context.Context, msgs []ChatMessage, tools []ToolSpec, onText func(string)) (*ChatMessage, Usage, error) {
	if strings.TrimSpace(c.BaseURL) == "" || strings.TrimSpace(c.Model) == "" {
		return nil, Usage{}, errors.New("AI 接口地址和模型名不能为空")
	}
	req := toolRequest{Model: c.Model, Stream: true, Temperature: 0.4}
	for _, m := range msgs {
		wm := wireMessage{Role: m.Role, Content: wireContent(m), ToolCallID: m.ToolCallID}
		for _, tc := range m.ToolCalls {
			wm.ToolCalls = append(wm.ToolCalls, wireToolCall{ID: tc.ID, Type: "function",
				Function: wireFunc{Name: tc.Name, Arguments: tc.Arguments}})
		}
		req.Messages = append(req.Messages, wm)
	}
	for _, t := range tools {
		var wt wireTool
		wt.Type = "function"
		wt.Function.Name, wt.Function.Description, wt.Function.Parameters = t.Name, t.Description, t.Parameters
		req.Tools = append(req.Tools, wt)
	}
	body, _ := json.Marshal(req)
	resp, err := c.postCompletion(ctx, body)
	if err != nil {
		return nil, Usage{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		se := &StatusError{Code: resp.StatusCode, Msg: errText(b)}
		if len(tools) > 0 && IsRequestRejected(se) {
			return nil, Usage{}, fmt.Errorf("%w: %w", ErrToolsUnsupported, se)
		}
		return nil, Usage{}, se
	}
	if strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
		return readToolStream(resp.Body, onText)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return nil, Usage{}, err
	}
	var tr toolResponse
	if err := json.Unmarshal(b, &tr); err != nil {
		return nil, Usage{}, fmt.Errorf("无法解析 AI 接口响应: %s", errText(b))
	}
	if tr.Error != nil {
		return nil, Usage{}, fmt.Errorf("AI 接口错误: %s", tr.Error.Message)
	}
	if len(tr.Choices) == 0 {
		return nil, Usage{}, errors.New("AI 接口没有返回内容")
	}
	ch := tr.Choices[0].Message
	out := &ChatMessage{Role: "assistant", Content: ch.Content}
	for i, tc := range ch.ToolCalls {
		id := tc.ID
		if id == "" {
			id = fmt.Sprintf("call_%d", i)
		}
		out.ToolCalls = append(out.ToolCalls, ToolCall{ID: id, Name: tc.Function.Name, Arguments: tc.Function.Arguments})
	}
	if onText != nil && out.Content != "" {
		onText(out.Content)
	}
	var u Usage
	if tr.Usage != nil {
		u = *tr.Usage
	}
	return out, u, nil
}

// readToolStream assembles streamed text and tool-call fragments.
func readToolStream(r io.Reader, onText func(string)) (*ChatMessage, Usage, error) {
	var text strings.Builder
	var u Usage
	type partial struct {
		id, name string
		args     strings.Builder
	}
	calls := map[int]*partial{}
	sc := newCompletionStream(r)
	for data, ok := sc.next(); ok; data, ok = sc.next() {
		var tr toolResponse
		if json.Unmarshal([]byte(data), &tr) != nil {
			continue
		}
		if tr.Error != nil {
			return nil, u, fmt.Errorf("AI 接口错误: %s", tr.Error.Message)
		}
		if tr.Usage != nil {
			u = *tr.Usage
		}
		for _, ch := range tr.Choices {
			if d := ch.Delta.Content; d != "" {
				text.WriteString(d)
				if onText != nil {
					onText(d)
				}
			}
			for j, tc := range ch.Delta.ToolCalls {
				idx := j
				if tc.Index != nil {
					idx = *tc.Index
				}
				p := calls[idx]
				if p == nil {
					p = &partial{}
					calls[idx] = p
				}
				if tc.ID != "" {
					p.id = tc.ID
				}
				if tc.Function.Name != "" {
					p.name += tc.Function.Name
				}
				p.args.WriteString(tc.Function.Arguments)
			}
		}
	}
	if err := sc.Err(); err != nil {
		return nil, u, fmt.Errorf("读取 AI 接口响应失败: %w", err)
	}
	out := &ChatMessage{Role: "assistant", Content: text.String()}
	idxs := make([]int, 0, len(calls))
	for i := range calls {
		idxs = append(idxs, i)
	}
	sort.Ints(idxs)
	for _, i := range idxs {
		p := calls[i]
		if p.name == "" {
			continue
		}
		id := p.id
		if id == "" {
			id = fmt.Sprintf("call_%d", i)
		}
		out.ToolCalls = append(out.ToolCalls, ToolCall{ID: id, Name: p.name, Arguments: p.args.String()})
	}
	return out, u, nil
}
