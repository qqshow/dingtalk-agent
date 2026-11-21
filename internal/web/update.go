package web

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

func (s *Server) apiUpdateStatus(w http.ResponseWriter, r *http.Request) {
	st := s.upd.Status()
	writeJSON(w, 200, map[string]any{"status": st, "auto_check": s.a.Cfg.AutoUpdateCheck()})
}

func (s *Server) apiUpdateCheck(w http.ResponseWriter, r *http.Request) {
	st, err := s.upd.Check(r.Context())
	if err != nil {
		slog.Warn("update check failed", "err", err)
	}
	writeJSON(w, 200, map[string]any{"status": st, "auto_check": s.a.Cfg.AutoUpdateCheck()})
}

// apiUpdateApply downloads and installs the new version in the background,
// then restarts the program. The page polls /api/update/status and
// /api/config to follow progress and notice the new version.
func (s *Server) apiUpdateApply(w http.ResponseWriter, r *http.Request) {
	st := s.upd.Status()
	switch {
	case !st.Supported:
		writeErr(w, 400, errText(st.Reason))
		return
	case !st.Available:
		writeErr(w, 400, errText("没有可用的新版本，请先检查更新"))
		return
	case st.Applying || st.Restarting:
		writeErr(w, http.StatusConflict, errText("正在升级中"))
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
		defer cancel()
		if err := s.upd.Apply(ctx); err != nil {
			slog.Error("update failed", "err", err)
			return
		}
		slog.Info("update installed, restarting", "version", s.upd.Status().Latest.Version())
		time.Sleep(800 * time.Millisecond) // let the page see "restarting"
		if s.restart != nil {
			s.restart()
		}
	}()
	writeJSON(w, 202, map[string]any{"status": s.upd.Status()})
}

type errText string

func (e errText) Error() string { return string(e) }
