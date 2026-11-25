package web

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/qqshow/dingtalk-agent/internal/chat"
	"github.com/qqshow/dingtalk-agent/internal/store"
)

func pathID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid id %q", r.PathValue("id"))
	}
	return id, nil
}

func (s *Server) apiChatSessions(w http.ResponseWriter, r *http.Request) {
	list, err := s.a.Agent.ListChatSessions(200)
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	writeJSON(w, 200, map[string]any{"sessions": list})
}

func (s *Server) apiChatSession(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	sess, err := s.a.Agent.GetChatSession(id)
	if err != nil {
		writeErr(w, 404, err)
		return
	}
	msgs, err := s.a.Agent.ChatMessages(id)
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	writeJSON(w, 200, map[string]any{"session": sess, "messages": msgs})
}

func (s *Server) apiChatRename(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	var body struct {
		Title string `json:"title"`
	}
	if err := decodeBody(r, &body); err != nil {
		writeErr(w, 400, err)
		return
	}
	if err := s.a.Agent.RenameChatSession(id, body.Title); err != nil {
		writeErr(w, 400, err)
		return
	}
	sess, _ := s.a.Agent.GetChatSession(id)
	writeJSON(w, 200, map[string]any{"session": sess})
}

func (s *Server) apiChatDelete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	if err := s.a.Agent.DeleteChatSession(id); err != nil {
		writeErr(w, 404, err)
		return
	}
	if err := os.RemoveAll(chat.SessionImageDir(s.a.Cfg.DataDir, id)); err != nil {
		slog.Warn("remove chat images", "session", id, "err", err)
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

// apiChatImage serves an image attached to a chat message.
func (s *Server) apiChatImage(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	p, err := chat.ImagePath(s.a.Cfg.DataDir, id, r.PathValue("name"))
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeFile(w, r, p)
}

// apiChatPeople searches contacts for @mentions, including people we only
// know from groups (no single chat).
func (s *Server) apiChatPeople(w http.ResponseWriter, r *http.Request) {
	snap := s.snap(w)
	if snap == nil {
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	q := r.URL.Query().Get("q")
	if r.URL.Query().Get("no_chat") != "1" {
		writeJSON(w, 200, map[string]any{"people": snap.SearchPeople(q, limit)})
		return
	}
	// Only people we have no single chat with: the page already lists
	// single chats from the conversation list.
	out := []store.Person{}
	for _, p := range snap.SearchPeople(q, 100000) {
		if p.CID == "" {
			out = append(out, p)
			if len(out) == limit {
				break
			}
		}
	}
	writeJSON(w, 200, map[string]any{"people": out})
}

// apiChatSend answers a message as a server-sent event stream. Closing the
// connection (the page's stop button) cancels the answer.
func (s *Server) apiChatSend(w http.ResponseWriter, r *http.Request) {
	var req chat.Request
	if err := decodeBodyLimit(r, &req, chat.MaxBodyBytes); err != nil {
		writeErr(w, 400, err)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeErr(w, 500, fmt.Errorf("streaming not supported"))
		return
	}
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(200)
	flusher.Flush()

	var mu sync.Mutex
	emit := func(e chat.Event) {
		b, _ := json.Marshal(e)
		mu.Lock()
		defer mu.Unlock()
		fmt.Fprintf(w, "data: %s\n\n", b)
		flusher.Flush()
	}
	if err := s.chat.Send(r.Context(), req, emit); err != nil {
		emit(chat.Event{Type: "error", Error: err.Error()})
	}
}
