package web

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/qqshow/dingtalk-agent/internal/ai"
	"github.com/qqshow/dingtalk-agent/internal/config"
)

// aiSettingsView is config.AIConfig with the key masked.
type aiSettingsView struct {
	config.AIConfig
	APIKey     string `json:"api_key"` // masked
	HasKey     bool   `json:"has_key"`
	KeyFromEnv bool   `json:"key_from_env"`
	Ready      bool   `json:"ready"`
}

func maskKey(k string) string {
	k = strings.TrimSpace(k)
	if k == "" {
		return ""
	}
	r := []rune(k)
	if len(r) <= 8 {
		return "••••"
	}
	return string(r[:3]) + "••••" + string(r[len(r)-4:])
}

func (s *Server) aiView() aiSettingsView {
	c := s.a.Cfg.AI()
	return aiSettingsView{AIConfig: c, APIKey: maskKey(c.APIKey), HasKey: c.APIKey != "",
		KeyFromEnv: config.AIKeyFromEnv(), Ready: c.Ready()}
}

// aiInput is what the settings form sends. An empty or masked api_key keeps
// the stored key; clear_key removes it.
type aiInput struct {
	Enabled       *bool   `json:"enabled"`
	BaseURL       *string `json:"base_url"`
	Model         *string `json:"model"`
	APIKey        *string `json:"api_key"`
	ClearKey      bool    `json:"clear_key"`
	DailyEnabled  *bool   `json:"daily_enabled"`
	DailyTime     *string `json:"daily_time"`
	WeeklyEnabled *bool   `json:"weekly_enabled"`
	WeeklyDay     *int    `json:"weekly_day"`
	MaxInputChars *int    `json:"max_input_chars"`
}

// merge applies the form on top of the stored settings (stored key, not the
// env override, so an env key is never written to disk).
func (in aiInput) merge(c config.AIConfig) (config.AIConfig, error) {
	if in.Enabled != nil {
		c.Enabled = *in.Enabled
	}
	if in.BaseURL != nil {
		u := strings.TrimSpace(*in.BaseURL)
		if u != "" && !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
			return c, errors.New("接口地址需要以 http:// 或 https:// 开头")
		}
		c.BaseURL = u
	}
	if in.Model != nil {
		c.Model = strings.TrimSpace(*in.Model)
	}
	if in.APIKey != nil {
		if k := strings.TrimSpace(*in.APIKey); k != "" && !strings.Contains(k, "••••") {
			c.APIKey = k
		}
	}
	if in.ClearKey {
		c.APIKey = ""
	}
	if in.DailyEnabled != nil {
		c.DailyEnabled = *in.DailyEnabled
	}
	if in.DailyTime != nil {
		if _, err := time.Parse("15:04", strings.TrimSpace(*in.DailyTime)); err != nil {
			return c, errors.New("生成时间格式应为 HH:MM")
		}
		c.DailyTime = strings.TrimSpace(*in.DailyTime)
	}
	if in.WeeklyEnabled != nil {
		c.WeeklyEnabled = *in.WeeklyEnabled
	}
	if in.WeeklyDay != nil {
		if *in.WeeklyDay < 0 || *in.WeeklyDay > 6 {
			return c, errors.New("weekly_day 应为 0（周日）到 6（周六）")
		}
		c.WeeklyDay = *in.WeeklyDay
	}
	if in.MaxInputChars != nil {
		if *in.MaxInputChars != 0 && (*in.MaxInputChars < 5000 || *in.MaxInputChars > 500000) {
			return c, errors.New("输入字数上限应在 5000 到 500000 之间")
		}
		c.MaxInputChars = *in.MaxInputChars
	}
	return c, nil
}

func (s *Server) apiAIGetSettings(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"settings": s.aiView(), "status": s.ai.Status()})
}

func (s *Server) apiAIPutSettings(w http.ResponseWriter, r *http.Request) {
	var in aiInput
	if err := decodeBody(r, &in); err != nil {
		writeErr(w, 400, err)
		return
	}
	c, err := in.merge(s.a.Cfg.StoredAI()) // never persist an env-provided key
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	if err := s.a.Cfg.SetAI(c); err != nil {
		writeErr(w, 500, err)
		return
	}
	writeJSON(w, 200, map[string]any{"settings": s.aiView(), "status": s.ai.Status()})
}

// apiAITest checks the connection with the posted (unsaved) form values.
func (s *Server) apiAITest(w http.ResponseWriter, r *http.Request) {
	var in aiInput
	if err := decodeBody(r, &in); err != nil {
		writeErr(w, 400, err)
		return
	}
	c, err := in.merge(s.a.Cfg.AI())
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	reply, d, err := s.ai.Test(r.Context(), c)
	if err != nil {
		writeJSON(w, 200, map[string]any{"ok": false, "error": err.Error(), "ms": d.Milliseconds()})
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true, "reply": reply, "ms": d.Milliseconds()})
}

func (s *Server) apiAIRun(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Kind string `json:"kind"`
		Date string `json:"date"`
	}
	if err := decodeBody(r, &body); err != nil {
		writeErr(w, 400, err)
		return
	}
	if err := s.ai.Start(body.Kind, body.Date, "manual"); err != nil {
		code := 400
		if errors.Is(err, ai.ErrBusy) {
			code = http.StatusConflict
		}
		writeErr(w, code, err)
		return
	}
	writeJSON(w, 202, s.ai.Status())
}

func (s *Server) apiAIStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, s.ai.Status())
}
