package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestConverseStreamsToolCalls(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req toolRequest
		json.NewDecoder(r.Body).Decode(&req)
		if len(req.Tools) != 1 || req.Tools[0].Function.Name != "lookup" || req.Tools[0].Type != "function" {
			t.Errorf("tools = %+v", req.Tools)
		}
		// The second message replays a previous tool call and its result.
		if len(req.Messages) != 3 || req.Messages[1].ToolCalls[0].Function.Name != "lookup" || req.Messages[2].ToolCallID != "c0" {
			t.Errorf("messages = %+v", req.Messages)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		chunks := []string{
			`{"choices":[{"delta":{"content":"让我查一下"}}]}`,
			`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_a","type":"function","function":{"name":"lookup","arguments":""}}]}}]}`,
			`{"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"q\":"}}]}}]}`,
			`{"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"张三\"}"}}]}}]}`,
			`{"choices":[{"delta":{"tool_calls":[{"index":1,"id":"call_b","function":{"name":"lookup","arguments":"{}"}}]},"finish_reason":"tool_calls"}]}`,
		}
		for _, c := range chunks {
			fmt.Fprintf(w, "data: %s\n\n", c)
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()
	var streamed strings.Builder
	c := &Client{BaseURL: srv.URL, Model: "m"}
	msgs := []ChatMessage{
		{Role: "user", Content: "hi"},
		{Role: "assistant", ToolCalls: []ToolCall{{ID: "c0", Name: "lookup", Arguments: "{}"}}},
		{Role: "tool", ToolCallID: "c0", Content: "result"},
	}
	out, _, err := c.Converse(context.Background(), msgs, []ToolSpec{{Name: "lookup", Parameters: map[string]any{"type": "object"}}},
		func(s string) { streamed.WriteString(s) })
	if err != nil {
		t.Fatal(err)
	}
	if out.Content != "让我查一下" || streamed.String() != out.Content {
		t.Fatalf("content = %q streamed = %q", out.Content, streamed.String())
	}
	if len(out.ToolCalls) != 2 || out.ToolCalls[0].ID != "call_a" || out.ToolCalls[0].Arguments != `{"q":"张三"}` ||
		out.ToolCalls[1].ID != "call_b" {
		t.Fatalf("tool calls = %+v", out.ToolCalls)
	}
}

func TestConverseToolsUnsupported(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req toolRequest
		json.NewDecoder(r.Body).Decode(&req)
		if len(req.Tools) > 0 {
			w.WriteHeader(400)
			fmt.Fprint(w, `{"error":{"message":"this model does not support tools"}}`)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"content":"好的"}}]}`)
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, Model: "m"}
	_, _, err := c.Converse(context.Background(), []ChatMessage{{Role: "user", Content: "hi"}}, []ToolSpec{{Name: "x"}}, nil)
	if !errors.Is(err, ErrToolsUnsupported) {
		t.Fatalf("err = %v, want ErrToolsUnsupported", err)
	}
	out, _, err := c.Converse(context.Background(), []ChatMessage{{Role: "user", Content: "hi"}}, nil, nil)
	if err != nil || out.Content != "好的" {
		t.Fatalf("without tools: %+v %v", out, err)
	}
}

func TestConverseSendsImagesAsContentParts(t *testing.T) {
	var got []map[string]json.RawMessage
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Messages []map[string]json.RawMessage `json:"messages"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		got = req.Messages
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"content":"ok"}}]}`)
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, Model: "m"}
	msgs := []ChatMessage{
		{Role: "system", Content: "sys"},
		{Role: "user", Content: "看图", Images: []string{"data:image/png;base64,AAAA"}},
	}
	if _, _, err := c.Converse(context.Background(), msgs, nil, nil); err != nil {
		t.Fatal(err)
	}
	if string(got[0]["content"]) != `"sys"` {
		t.Errorf("text-only message must stay a string, got %s", got[0]["content"])
	}
	want := `[{"type":"text","text":"看图"},{"type":"image_url","image_url":{"url":"data:image/png;base64,AAAA"}}]`
	if string(got[1]["content"]) != want {
		t.Errorf("image message content = %s\nwant %s", got[1]["content"], want)
	}
}

func TestIsRequestRejected(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{&StatusError{Code: 400}, true},
		{&StatusError{Code: 422}, true},
		{fmt.Errorf("%w: %w", ErrToolsUnsupported, &StatusError{Code: 400}), true},
		{&StatusError{Code: 401}, false},
		{&StatusError{Code: 403}, false},
		{&StatusError{Code: 429}, false},
		{&StatusError{Code: 500}, false},
		{errors.New("network down"), false},
	}
	for _, c := range cases {
		if got := IsRequestRejected(c.err); got != c.want {
			t.Errorf("IsRequestRejected(%v) = %v, want %v", c.err, got, c.want)
		}
	}
}
