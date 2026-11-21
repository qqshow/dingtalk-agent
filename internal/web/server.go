// Package web serves the embedded UI, the REST API and the MCP endpoint.
package web

import (
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/qqshow/dingtalk-agent/internal/agentdb"
	"github.com/qqshow/dingtalk-agent/internal/ai"
	"github.com/qqshow/dingtalk-agent/internal/analysis"
	"github.com/qqshow/dingtalk-agent/internal/app"
	"github.com/qqshow/dingtalk-agent/internal/config"
	"github.com/qqshow/dingtalk-agent/internal/export"
	"github.com/qqshow/dingtalk-agent/internal/store"
	"github.com/qqshow/dingtalk-agent/internal/syncer"
	"github.com/qqshow/dingtalk-agent/internal/update"
)

//go:embed static
var staticFS embed.FS

type Server struct {
	a       *app.App
	ai      *ai.Runner
	upd     *update.Updater
	restart func() // restarts the process after an update is installed
	mux     *http.ServeMux
}

// New builds the HTTP handler. mcp is mounted at /mcp when non-nil.
func New(a *app.App, mcp http.Handler, runner *ai.Runner, upd *update.Updater, restart func()) http.Handler {
	s := &Server{a: a, ai: runner, upd: upd, restart: restart, mux: http.NewServeMux()}
	s.routes(mcp)
	return s.guard(s.logging(s.mux))
}

func (s *Server) routes(mcp http.Handler) {
	sub, _ := fs.Sub(staticFS, "static")
	files := http.FileServer(http.FS(sub))
	m := s.mux
	// no-cache: the browser revalidates, so a new build is picked up immediately.
	m.Handle("GET /static/", http.StripPrefix("/static/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		files.ServeHTTP(w, r)
	})))
	m.HandleFunc("GET /{$}", s.page("index.html"))
	m.HandleFunc("GET /report", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/#/report", http.StatusFound)
	})

	// Original browsing API (compatible with the original web UI).
	m.HandleFunc("GET /api/config", s.apiConfig)
	m.HandleFunc("GET /api/accounts", s.apiAccounts)
	m.HandleFunc("POST /api/accounts/select", s.apiSelectAccount)
	m.HandleFunc("GET /api/conversations", s.apiConversations)
	m.HandleFunc("GET /api/conversations/{cid}/messages", s.apiMessages)
	m.HandleFunc("GET /api/search", s.apiSearch)
	m.HandleFunc("GET /api/stats", s.apiStats)
	m.HandleFunc("GET /api/sync/status", s.apiSyncStatus)
	m.HandleFunc("GET /api/settings", s.apiGetSettings)
	m.HandleFunc("PUT /api/settings", s.apiPutSettings)
	m.HandleFunc("POST /api/sync/trigger", s.apiSyncTrigger)
	m.HandleFunc("POST /api/export/selected", s.apiExportSelected)
	m.HandleFunc("GET /api/exports", s.apiExports)
	m.HandleFunc("GET /api/exports/{name}/download", s.apiExportDownload)
	m.HandleFunc("GET /api/attachments/{path...}", s.apiAttachment)
	m.HandleFunc("GET /api/local-file", s.apiLocalFile)

	// Agent API.
	m.HandleFunc("GET /api/update/status", s.apiUpdateStatus)
	m.HandleFunc("POST /api/update/check", s.apiUpdateCheck)
	m.HandleFunc("POST /api/update/apply", s.apiUpdateApply)
	m.HandleFunc("GET /api/ai/settings", s.apiAIGetSettings)
	m.HandleFunc("PUT /api/ai/settings", s.apiAIPutSettings)
	m.HandleFunc("POST /api/ai/test", s.apiAITest)
	m.HandleFunc("POST /api/ai/run", s.apiAIRun)
	m.HandleFunc("GET /api/ai/status", s.apiAIStatus)

	m.HandleFunc("GET /api/agent", s.apiAgentIndex)
	m.HandleFunc("POST /api/agent/sync", s.apiAgentSync)
	m.HandleFunc("GET /api/agent/report", s.apiAgentReport)
	m.HandleFunc("GET /api/agent/digest", s.apiAgentDigest)
	m.HandleFunc("GET /api/agent/messages", s.apiAgentMessages)
	m.HandleFunc("GET /api/agent/calendar", s.apiAgentCalendar)
	m.HandleFunc("GET /api/agent/summary", s.apiGetSummary)
	m.HandleFunc("PUT /api/agent/summary", s.apiPutSummary)
	m.HandleFunc("POST /api/agent/summary", s.apiPutSummary)
	m.HandleFunc("GET /api/agent/summaries", s.apiListSummaries)
	m.HandleFunc("GET /api/agent/todos", s.apiListTodos)
	m.HandleFunc("POST /api/agent/todos", s.apiAddTodos)
	m.HandleFunc("PATCH /api/agent/todos/{id}", s.apiPatchTodo)
	m.HandleFunc("DELETE /api/agent/todos/{id}", s.apiDeleteTodo)

	if mcp != nil {
		m.Handle("/mcp", mcp)
	}
}

// ------------------------------------------------------------------ helpers

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.Encode(v)
}

func writeErr(w http.ResponseWriter, code int, err error) {
	if errors.Is(err, store.ErrNoData) {
		code = http.StatusServiceUnavailable
	}
	writeJSON(w, code, map[string]string{"detail": err.Error()})
}

func qInt(r *http.Request, key string, def, lo, hi int) int {
	v, err := strconv.Atoi(r.URL.Query().Get(key))
	if err != nil {
		return def
	}
	return max(lo, min(hi, v))
}

func qInt64(r *http.Request, key string) int64 {
	v, _ := strconv.ParseInt(r.URL.Query().Get(key), 10, 64)
	return v
}

func qBool(r *http.Request, key string) bool {
	v := strings.ToLower(r.URL.Query().Get(key))
	return v == "1" || v == "true" || v == "yes"
}

func decodeBody(r *http.Request, v any) error {
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 8<<20))
	return dec.Decode(v)
}

func (s *Server) snap(w http.ResponseWriter) *store.Snapshot {
	snap, err := s.a.Snapshot(false)
	if err != nil {
		writeErr(w, http.StatusServiceUnavailable, err)
		return nil
	}
	return snap
}

func (s *Server) page(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b, err := staticFS.ReadFile("static/" + name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Write(b)
	}
}

// guard blocks cross-site and DNS-rebinding access to the chat data: only
// loopback / configured hosts, and no foreign Origin.
func (s *Server) guard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
		host = strings.Trim(host, "[]")
		if !s.allowedHost(host) {
			http.Error(w, "forbidden host", http.StatusForbidden)
			return
		}
		if o := r.Header.Get("Origin"); o != "" {
			u, err := url.Parse(o)
			if err != nil || !s.allowedHost(u.Hostname()) {
				http.Error(w, "forbidden origin", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) allowedHost(h string) bool {
	h = strings.ToLower(h)
	if h == "localhost" || h == "127.0.0.1" || h == "::1" {
		return true
	}
	cfgHost := strings.ToLower(s.a.Cfg.Host)
	if cfgHost == "0.0.0.0" || cfgHost == "::" || cfgHost == "" {
		return true // user explicitly exposed the server on the network
	}
	return h == cfgHost
}

type statusRecorder struct {
	http.ResponseWriter
	code int
}

func (r *statusRecorder) WriteHeader(c int) { r.code = c; r.ResponseWriter.WriteHeader(c) }
func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (s *Server) logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, code: 200}
		next.ServeHTTP(rec, r)
		if strings.HasPrefix(r.URL.Path, "/static/") || r.URL.Path == "/api/sync/status" {
			return
		}
		lvl := slog.LevelInfo
		if rec.code >= 500 {
			lvl = slog.LevelError
		}
		slog.Log(r.Context(), lvl, "http", "method", r.Method, "path", r.URL.Path, "status", rec.code,
			"ms", time.Since(start).Milliseconds())
	})
}

// --------------------------------------------------------- browsing API

func (s *Server) apiConfig(w http.ResponseWriter, r *http.Request) {
	name := ""
	if snap, err := s.a.Store.Snapshot(); err == nil {
		name = snap.MyName()
	}
	writeJSON(w, 200, map[string]any{"user_uid": s.a.Cfg.UID(), "user_name": name, "version": s.a.Version})
}

func (s *Server) apiAccounts(w http.ResponseWriter, r *http.Request) {
	uid := s.a.Cfg.UID()
	name := ""
	if snap, err := s.a.Store.Snapshot(); err == nil {
		name = snap.MyName()
	}
	writeJSON(w, 200, map[string]any{
		"accounts": []map[string]any{{
			"uid": uid, "display_name": name, "version": config.AccountVersion(s.a.Cfg.DingTalkDir()), "active": true, "supported": true,
			"data_dir": s.a.Cfg.DingTalkDir(),
		}},
		"active_uid": uid,
	})
}

func (s *Server) apiSelectAccount(w http.ResponseWriter, r *http.Request) {
	writeErr(w, http.StatusBadRequest, errors.New("only the active account is supported; set uid/dingtalk_dir in data/config.json to change it"))
}

func (s *Server) apiConversations(w http.ResponseWriter, r *http.Request) {
	snap := s.snap(w)
	if snap == nil {
		return
	}
	total, list := snap.ConversationList(r.URL.Query().Get("keyword"), r.URL.Query().Get("type"),
		qInt(r, "limit", 50, 1, 5000), qInt(r, "offset", 0, 0, 1<<30))
	type convOut struct {
		*store.Conversation
		Last *store.LastMsg `json:"last_message,omitempty"`
	}
	last := snap.LastMessages()
	out := make([]convOut, len(list))
	for i, c := range list {
		out[i] = convOut{Conversation: c}
		if lm, ok := last[c.CID]; ok {
			out[i].Last = &lm
		}
	}
	writeJSON(w, 200, map[string]any{"total": total, "conversations": out})
}

func (s *Server) apiMessages(w http.ResponseWriter, r *http.Request) {
	snap := s.snap(w)
	if snap == nil {
		return
	}
	total, msgs, err := snap.Messages(r.PathValue("cid"), qInt(r, "limit", 50, 1, 500),
		qInt(r, "offset", 0, 0, 1<<30), qInt64(r, "since"), qInt64(r, "until"))
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	writeJSON(w, 200, map[string]any{"total": total, "messages": msgs})
}

func (s *Server) apiSearch(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		writeErr(w, 400, errors.New("missing q"))
		return
	}
	snap := s.snap(w)
	if snap == nil {
		return
	}
	rq := store.RangeQuery{Keyword: q, Desc: true, Limit: qInt(r, "limit", 50, 1, 500),
		Offset: qInt(r, "offset", 0, 0, 1<<30), WithImages: true}
	if cid := r.URL.Query().Get("cid"); cid != "" {
		rq.CIDs = []string{cid}
	}
	if f, t := r.URL.Query().Get("from"), r.URL.Query().Get("to"); f != "" || t != "" {
		p, err := analysis.ParsePeriod("", f, t)
		if err != nil {
			writeErr(w, 400, err)
			return
		}
		rq.From, rq.To = p.FromMS(), p.ToMS()
	}
	msgs, err := snap.Range(rq)
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	writeJSON(w, 200, map[string]any{"query": q, "total": len(msgs), "messages": withTitles(snap, msgs)})
}

type msgWithConv struct {
	*store.Message
	ConvTitle string `json:"conv_title"`
}

func withTitles(snap *store.Snapshot, msgs []*store.Message) []msgWithConv {
	out := make([]msgWithConv, len(msgs))
	for i, m := range msgs {
		out[i] = msgWithConv{m, snap.ConvTitle(m.CID)}
	}
	return out
}

func (s *Server) apiStats(w http.ResponseWriter, r *http.Request) {
	snap := s.snap(w)
	if snap == nil {
		return
	}
	writeJSON(w, 200, snap.Stats())
}

func (s *Server) apiSyncStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, s.a.Syncer.State())
}

func (s *Server) settings(r *http.Request) map[string]any {
	cfg := s.a.Cfg
	base := "http://" + r.Host
	return map[string]any{
		"sync_interval_minutes": int(cfg.Interval() / time.Minute),
		"auto_update_check":     cfg.AutoUpdateCheck(),
		"sync":                  s.a.Syncer.State(),
		"uid":                   cfg.UID(),
		"dingtalk_dir":          cfg.DingTalkDir(),
		"data_dir":              cfg.DataDir,
		"version":               s.a.Version,
		"mcp_url":               base + "/mcp",
		"api_url":               base + "/api/agent",
	}
}

func (s *Server) apiGetSettings(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, s.settings(r))
}

func (s *Server) apiPutSettings(w http.ResponseWriter, r *http.Request) {
	var body struct {
		SyncIntervalMinutes *int  `json:"sync_interval_minutes"`
		AutoUpdateCheck     *bool `json:"auto_update_check"`
	}
	if err := decodeBody(r, &body); err != nil {
		writeErr(w, 400, err)
		return
	}
	if body.AutoUpdateCheck != nil {
		if err := s.a.Cfg.SetAutoUpdateCheck(*body.AutoUpdateCheck); err != nil {
			writeErr(w, 500, err)
			return
		}
	}
	if body.SyncIntervalMinutes != nil {
		m := *body.SyncIntervalMinutes
		if m < 0 || m > 24*60 || (m > 0 && m < 1) {
			writeErr(w, 400, errors.New("sync_interval_minutes must be 0 (off) or 1..1440"))
			return
		}
		if err := s.a.Syncer.SetInterval(time.Duration(m) * time.Minute); err != nil {
			writeErr(w, 500, err)
			return
		}
		time.Sleep(50 * time.Millisecond) // let the scheduler publish the new next_sync_time
	}
	writeJSON(w, 200, s.settings(r))
}

func (s *Server) apiSyncTrigger(w http.ResponseWriter, r *http.Request) {
	full := qBool(r, "full")
	op := "manual_sync"
	if full {
		op = "full_sync"
	}
	if !s.a.Syncer.Begin(op) {
		writeErr(w, http.StatusConflict, syncer.ErrBusy)
		return
	}
	go func() {
		err := s.a.Syncer.RunReserved()
		if err != nil || !full {
			return
		}
		// Full export after a successful sync (original "全量导出").
		if !s.a.Syncer.Begin("full_sync") {
			return
		}
		var path string
		snap, err := s.a.Store.Snapshot()
		if err == nil {
			path, err = s.a.Exporter.Export(snap, s.a.Cfg.UID(), nil, 0)
		}
		s.a.Syncer.Finish(err, path)
	}()
	writeJSON(w, 200, map[string]any{"status": "started", "full": full})
}

func (s *Server) apiExportSelected(w http.ResponseWriter, r *http.Request) {
	var body struct {
		CIDs      []string `json:"cids"`
		SinceTime *float64 `json:"since_time"`
	}
	if err := decodeBody(r, &body); err != nil || len(body.CIDs) == 0 {
		writeErr(w, 400, errors.New("no conversations selected"))
		return
	}
	snap := s.snap(w)
	if snap == nil {
		return
	}
	if !s.a.Syncer.Begin("export") {
		writeErr(w, http.StatusConflict, syncer.ErrBusy)
		return
	}
	var since int64
	if body.SinceTime != nil {
		since = int64(*body.SinceTime)
	}
	go func() {
		path, err := s.a.Exporter.Export(snap, s.a.Cfg.UID(), body.CIDs, since)
		s.a.Syncer.Finish(err, path)
	}()
	writeJSON(w, 200, map[string]any{"status": "started", "selected_count": len(body.CIDs)})
}

func (s *Server) apiExports(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"exports": s.a.Exporter.List()})
}

func (s *Server) apiExportDownload(w http.ResponseWriter, r *http.Request) {
	dir, ok := s.a.Exporter.Resolve(r.PathValue("name"))
	if !ok {
		writeErr(w, 404, errors.New("export not found"))
		return
	}
	name := filepath.Base(dir) + ".zip"
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	if err := export.WriteZip(w, dir); err != nil {
		slog.Error("zip export failed", "err", err)
	}
}

var allowedAttachmentDirs = map[string]bool{"ImageFiles": true, "AudioFiles": true, "VideoFiles": true,
	"resource_cache": true, "DTIMChatImageFiles": true, "GifEmotionFiles": true}

func (s *Server) apiAttachment(w http.ResponseWriter, r *http.Request) {
	rel := filepath.Clean(filepath.FromSlash(r.PathValue("path")))
	first := strings.SplitN(filepath.ToSlash(rel), "/", 2)[0]
	if !allowedAttachmentDirs[first] || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		writeErr(w, 403, errors.New("access denied"))
		return
	}
	base := filepath.Clean(s.a.Cfg.DingTalkDir())
	full := filepath.Join(base, rel)
	if !strings.HasPrefix(full, base+string(filepath.Separator)) {
		writeErr(w, 403, errors.New("access denied"))
		return
	}
	http.ServeFile(w, r, full)
}

var allowedLocalExt = map[string]bool{".xlsx": true, ".xls": true, ".docx": true, ".doc": true, ".pptx": true,
	".ppt": true, ".pdf": true, ".txt": true, ".csv": true, ".json": true, ".xml": true, ".html": true, ".htm": true,
	".zip": true, ".rar": true, ".7z": true, ".gz": true, ".tar": true, ".png": true, ".jpg": true, ".jpeg": true,
	".gif": true, ".bmp": true, ".webp": true, ".mp4": true, ".mp3": true, ".wav": true, ".avi": true, ".m4a": true,
	".md": true}

func (s *Server) apiLocalFile(w http.ResponseWriter, r *http.Request) {
	p := filepath.Clean(r.URL.Query().Get("path"))
	if p == "" || strings.HasPrefix(p, `\\`) || !filepath.IsAbs(p) {
		writeErr(w, 403, errors.New("invalid path"))
		return
	}
	if !allowedLocalExt[strings.ToLower(filepath.Ext(p))] {
		writeErr(w, 403, errors.New("file type not allowed"))
		return
	}
	if _, err := os.Stat(p); err != nil {
		writeErr(w, 404, errors.New("file not found"))
		return
	}
	w.Header().Set("Content-Disposition", `attachment; filename*=UTF-8''`+url.PathEscape(filepath.Base(p)))
	http.ServeFile(w, r, p)
}

// --------------------------------------------------------------- agent API

func (s *Server) apiAgentIndex(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{
		"description": "DingTalk chat data for agents. Dates: YYYY-MM-DD | today | yesterday | -N. " +
			"Add refresh=1 to sync first when data is older than 5 minutes.",
		"endpoints": []map[string]string{
			{"method": "POST", "path": "/api/agent/sync", "desc": "decrypt latest DingTalk data now (synchronous)"},
			{"method": "GET", "path": "/api/agent/report?date=&from=&to=&refresh=", "desc": "structured stats: totals, people interacted, groups/projects involved, mentions of me, awaiting my reply"},
			{"method": "GET", "path": "/api/agent/digest?date=&from=&to=&scope=all|involved&max_chars=60000&format=markdown|json&refresh=", "desc": "conversation-grouped message digest for LLM summarization"},
			{"method": "GET", "path": "/api/agent/messages?cid=&date=&from=&to=&limit=", "desc": "full messages of one conversation in a period"},
			{"method": "GET", "path": "/api/agent/calendar?date=&from=&to=&include_closed=", "desc": "calendar entries: meetings (room, organizer, my response), approval todos, AI minutes"},
			{"method": "GET", "path": "/api/search?q=&from=&to=&cid=&limit=", "desc": "keyword search"},
			{"method": "GET", "path": "/api/conversations?keyword=&type=single|group", "desc": "list conversations"},
			{"method": "GET|PUT", "path": "/api/agent/summary?date=", "desc": "read / save the daily summary {date, summary(markdown), focus[]}"},
			{"method": "GET", "path": "/api/agent/summaries?limit=", "desc": "recent saved summaries"},
			{"method": "GET", "path": "/api/agent/todos?status=open|done|dropped|all&kind=todo|follow_up&source_date=", "desc": "list todos / follow-ups"},
			{"method": "POST", "path": "/api/agent/todos", "desc": "add todos: {items:[{kind,title,detail,priority,due,owner,cid,conv_title,msg_id,source_date,dedupe_key}]} (deduplicated)"},
			{"method": "PATCH", "path": "/api/agent/todos/{id}", "desc": "update {status,title,detail,priority,due,owner,kind}"},
			{"method": "DELETE", "path": "/api/agent/todos/{id}", "desc": "delete todo"},
			{"method": "POST", "path": "/mcp", "desc": "MCP (streamable HTTP) endpoint exposing the same operations as tools"},
		},
	})
}

func (s *Server) apiAgentSync(w http.ResponseWriter, r *http.Request) {
	st, err := s.a.Syncer.SyncNow()
	if err != nil {
		code := 500
		if errors.Is(err, syncer.ErrBusy) {
			code = http.StatusConflict
		}
		writeErr(w, code, err)
		return
	}
	writeJSON(w, 200, st)
}

func (s *Server) apiAgentReport(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	rep, err := s.a.Report(q.Get("date"), q.Get("from"), q.Get("to"), qBool(r, "refresh"))
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	writeJSON(w, 200, rep)
}

func (s *Server) apiAgentDigest(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	o := analysis.DigestOptions{Scope: q.Get("scope"), MaxChars: qInt(r, "max_chars", 60000, 2000, 2000000),
		MaxMsgChars: qInt(r, "max_msg_chars", 300, 50, 5000), SkipBots: q.Get("include_bots") == ""}
	d, err := s.a.Digest(q.Get("date"), q.Get("from"), q.Get("to"), o, qBool(r, "refresh"))
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	if q.Get("format") == "json" {
		writeJSON(w, 200, map[string]any{"digest": d, "report": d.Report})
		return
	}
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Write([]byte(d.Markdown))
}

func (s *Server) apiAgentMessages(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	cid := q.Get("cid")
	if cid == "" {
		writeErr(w, 400, errors.New("missing cid"))
		return
	}
	p, err := analysis.ParsePeriod(q.Get("date"), q.Get("from"), q.Get("to"))
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	snap, err := s.a.Snapshot(qBool(r, "refresh"))
	if err != nil {
		writeErr(w, 503, err)
		return
	}
	msgs, err := snap.Range(store.RangeQuery{From: p.FromMS(), To: p.ToMS(), CIDs: []string{cid},
		Limit: qInt(r, "limit", 1000, 1, 10000)})
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	writeJSON(w, 200, map[string]any{"cid": cid, "title": snap.ConvTitle(cid), "period": p.Label,
		"total": len(msgs), "messages": msgs})
}

func (s *Server) apiAgentCalendar(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	res, err := s.a.Calendar(q.Get("date"), q.Get("from"), q.Get("to"), qBool(r, "include_closed"), qBool(r, "refresh"))
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	writeJSON(w, 200, res)
}

func dateKeyFrom(r *http.Request, date string) (string, error) {
	if strings.Contains(date, "~") {
		return date, nil
	}
	p, err := analysis.ParsePeriod(date, "", "")
	if err != nil {
		return "", err
	}
	return p.DateKey(), nil
}

func (s *Server) apiGetSummary(w http.ResponseWriter, r *http.Request) {
	key, err := dateKeyFrom(r, r.URL.Query().Get("date"))
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	sum, err := s.a.Agent.GetSummary(key)
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	writeJSON(w, 200, map[string]any{"date": key, "summary": sum})
}

func (s *Server) apiPutSummary(w http.ResponseWriter, r *http.Request) {
	var body agentdb.Summary
	if err := decodeBody(r, &body); err != nil {
		writeErr(w, 400, err)
		return
	}
	if body.DateKey == "" {
		body.DateKey = r.URL.Query().Get("date")
	}
	key, err := dateKeyFrom(r, body.DateKey)
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	body.DateKey = key
	if body.Author == "" {
		body.Author = "agent"
	}
	sum, err := s.a.Agent.SaveSummary(body)
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	writeJSON(w, 200, sum)
}

func (s *Server) apiListSummaries(w http.ResponseWriter, r *http.Request) {
	l, err := s.a.Agent.ListSummaries(qInt(r, "limit", 30, 1, 365))
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	writeJSON(w, 200, map[string]any{"summaries": l})
}

func (s *Server) apiListTodos(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	l, err := s.a.Agent.ListTodos(agentdb.TodoFilter{Status: q.Get("status"), Kind: q.Get("kind"),
		SourceDate: q.Get("source_date"), Limit: qInt(r, "limit", 200, 1, 2000)})
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	writeJSON(w, 200, map[string]any{"todos": l})
}

func (s *Server) apiAddTodos(w http.ResponseWriter, r *http.Request) {
	var raw json.RawMessage
	if err := decodeBody(r, &raw); err != nil {
		writeErr(w, 400, err)
		return
	}
	var items []agentdb.Todo
	var wrapped struct {
		Items []agentdb.Todo `json:"items"`
	}
	if json.Unmarshal(raw, &items) != nil {
		if err := json.Unmarshal(raw, &wrapped); err != nil || len(wrapped.Items) == 0 {
			var one agentdb.Todo
			if err := json.Unmarshal(raw, &one); err != nil || one.Title == "" {
				writeErr(w, 400, errors.New("expected {items:[...]}, an array, or a single todo"))
				return
			}
			items = []agentdb.Todo{one}
		} else {
			items = wrapped.Items
		}
	}
	writeJSON(w, 200, map[string]any{"results": s.a.Agent.UpsertTodos(items)})
}

func (s *Server) apiPatchTodo(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	var p agentdb.TodoPatch
	if err := decodeBody(r, &p); err != nil {
		writeErr(w, 400, err)
		return
	}
	t, err := s.a.Agent.UpdateTodo(id, p)
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	writeJSON(w, 200, t)
}

func (s *Server) apiDeleteTodo(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err := s.a.Agent.DeleteTodo(id); err != nil {
		writeErr(w, 404, err)
		return
	}
	writeJSON(w, 200, map[string]any{"deleted": id})
}
