// Package ai is the built-in assistant: it sends the day's DingTalk digest to
// an OpenAI-compatible chat model and stores the resulting daily / weekly
// report, focus items and to-dos in the agent database.
package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Message is one chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Usage is the token accounting reported by the API (may be zero when the
// server does not report it).
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Client talks to an OpenAI-compatible /chat/completions endpoint
// (DeepSeek, Qwen, Kimi, Zhipu, OpenAI, Ollama, ...).
type Client struct {
	BaseURL string
	APIKey  string
	Model   string
	HTTP    *http.Client
}

// Endpoint returns the chat completions URL for BaseURL. Both
// "https://host/v1" and a full ".../chat/completions" URL are accepted.
func (c *Client) Endpoint() string {
	u := strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	if strings.HasSuffix(u, "/chat/completions") {
		return u
	}
	return u + "/chat/completions"
}

type chatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Stream      bool      `json:"stream"`
	Temperature float64   `json:"temperature"`
}

type chatChoice struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
	Delta struct {
		Content          string `json:"content"`
		ReasoningContent string `json:"reasoning_content"`
	} `json:"delta"`
	FinishReason string `json:"finish_reason"`
}

type chatResponse struct {
	Choices []chatChoice `json:"choices"`
	Usage   *Usage       `json:"usage"`
	Error   *apiError    `json:"error"`
}

type apiError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    any    `json:"code"`
}

func (c *Client) postCompletion(ctx context.Context, body []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream, application/json")
	if k := strings.TrimSpace(c.APIKey); k != "" {
		req.Header.Set("Authorization", "Bearer "+k)
	}
	hc := c.HTTP
	if hc == nil {
		hc = &http.Client{Timeout: 10 * time.Minute}
	}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 AI 接口失败: %w", err)
	}
	return resp, nil
}

// completionStream reads single-line data events from compatible APIs.
type completionStream struct {
	*bufio.Scanner
}

func newCompletionStream(r io.Reader) completionStream {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 4<<20)
	return completionStream{sc}
}

func (sc completionStream) next() (string, bool) {
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			return "", false
		}
		return data, true
	}
	return "", false
}

// Chat sends messages and returns the full reply. progress, when non-nil, is
// called with the number of characters received so far (reasoning included)
// while the reply streams in.
func (c *Client) Chat(ctx context.Context, msgs []Message, progress func(chars int)) (string, Usage, error) {
	if strings.TrimSpace(c.BaseURL) == "" || strings.TrimSpace(c.Model) == "" {
		return "", Usage{}, errors.New("AI 接口地址和模型名不能为空")
	}
	body, _ := json.Marshal(chatRequest{Model: c.Model, Messages: msgs, Stream: true, Temperature: 0.3})
	resp, err := c.postCompletion(ctx, body)
	if err != nil {
		return "", Usage{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", Usage{}, fmt.Errorf("AI 接口返回 %d: %s", resp.StatusCode, errText(b))
	}
	if strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
		return readStream(resp.Body, progress)
	}
	// Some servers ignore stream=true and answer with one JSON document.
	b, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return "", Usage{}, err
	}
	var cr chatResponse
	if err := json.Unmarshal(b, &cr); err != nil {
		return "", Usage{}, fmt.Errorf("无法解析 AI 接口响应: %s", errText(b))
	}
	if cr.Error != nil {
		return "", Usage{}, fmt.Errorf("AI 接口错误: %s", cr.Error.Message)
	}
	if len(cr.Choices) == 0 {
		return "", Usage{}, errors.New("AI 接口没有返回内容")
	}
	var u Usage
	if cr.Usage != nil {
		u = *cr.Usage
	}
	out := cr.Choices[0].Message.Content
	if progress != nil {
		progress(len([]rune(out)))
	}
	return out, u, nil
}

// readStream parses an OpenAI-style server-sent event stream.
func readStream(r io.Reader, progress func(int)) (string, Usage, error) {
	var out strings.Builder
	var u Usage
	chars, finish := 0, ""
	sc := newCompletionStream(r)
	for data, ok := sc.next(); ok; data, ok = sc.next() {
		var cr chatResponse
		if err := json.Unmarshal([]byte(data), &cr); err != nil {
			continue
		}
		if cr.Error != nil {
			return "", u, fmt.Errorf("AI 接口错误: %s", cr.Error.Message)
		}
		if cr.Usage != nil {
			u = *cr.Usage
		}
		for _, ch := range cr.Choices {
			out.WriteString(ch.Delta.Content)
			chars += len([]rune(ch.Delta.Content)) + len([]rune(ch.Delta.ReasoningContent))
			if ch.FinishReason != "" {
				finish = ch.FinishReason
			}
		}
		if progress != nil {
			progress(chars)
		}
	}
	if err := sc.Err(); err != nil {
		return "", u, fmt.Errorf("读取 AI 接口响应失败: %w", err)
	}
	if finish == "length" {
		return out.String(), u, errors.New("AI 回复被截断（超出模型最大输出长度），可在设置中调小输入字数或换用更大的模型")
	}
	return out.String(), u, nil
}

func errText(b []byte) string {
	var cr chatResponse
	if json.Unmarshal(b, &cr) == nil && cr.Error != nil && cr.Error.Message != "" {
		return cr.Error.Message
	}
	s := strings.TrimSpace(string(b))
	if len([]rune(s)) > 300 {
		s = string([]rune(s)[:300]) + "…"
	}
	return s
}

// ExtractJSON pulls the first JSON object out of a model reply, tolerating
// markdown code fences and text around it.
func ExtractJSON(s string) (string, error) {
	s = strings.TrimSpace(s)
	// Reasoning models may emit <think>...</think> before the answer.
	if i := strings.Index(s, "</think>"); i >= 0 {
		s = s[i+len("</think>"):]
	}
	start := strings.Index(s, "{")
	if start < 0 {
		return "", errors.New("AI 回复中没有 JSON")
	}
	depth, inStr, esc := 0, false, false
	for i := start; i < len(s); i++ {
		ch := s[i]
		switch {
		case esc:
			esc = false
		case inStr && ch == '\\':
			esc = true
		case ch == '"':
			inStr = !inStr
		case inStr:
		case ch == '{':
			depth++
		case ch == '}':
			depth--
			if depth == 0 {
				return s[start : i+1], nil
			}
		}
	}
	return "", errors.New("AI 回复中的 JSON 不完整")
}
