package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/qqshow/dingtalk-agent/internal/agentdb"
	"github.com/qqshow/dingtalk-agent/internal/analysis"
	"github.com/qqshow/dingtalk-agent/internal/app"
	"github.com/qqshow/dingtalk-agent/internal/config"
	"github.com/qqshow/dingtalk-agent/internal/textutil"
)

// AuthorPrefix marks summaries written by the built-in assistant, so a
// scheduled run never overwrites a report an external agent saved.
const AuthorPrefix = "内置 AI"

const defaultMaxInputChars = 30000

var (
	ErrBusy     = errors.New("AI 正在生成中，请稍候")
	ErrNotReady = errors.New("AI 助手未启用或未配置（请在设置中填写接口地址和模型）")
	errSkipped  = errors.New("skipped")
)

// Result describes the last finished run.
type Result struct {
	Kind        string  `json:"kind"` // daily | weekly
	DateKey     string  `json:"date_key"`
	Trigger     string  `json:"trigger"` // manual | schedule
	OK          bool    `json:"ok"`
	Error       string  `json:"error,omitempty"`
	Note        string  `json:"note,omitempty"`
	FinishedAt  string  `json:"finished_at"`
	Seconds     float64 `json:"seconds"`
	Usage       Usage   `json:"usage"`
	NewItems    int     `json:"new_items"`
	Suggestions int     `json:"suggestions"`
	InputChars  int     `json:"input_chars"`
}

// Status is what the web UI polls.
type Status struct {
	Ready      bool    `json:"ready"`
	Running    bool    `json:"running"`
	Kind       string  `json:"kind,omitempty"`
	DateKey    string  `json:"date_key,omitempty"`
	Stage      string  `json:"stage,omitempty"`
	Chars      int     `json:"chars"`
	StartedAt  string  `json:"started_at,omitempty"`
	Last       *Result `json:"last,omitempty"`
	NextDaily  string  `json:"next_daily,omitempty"`
	NextWeekly string  `json:"next_weekly,omitempty"`
}

// state is persisted in data/ai_state.json so restarts do not re-run jobs.
type state struct {
	LastDaily  string    `json:"last_daily"`  // date key handled by the scheduler
	LastWeekly string    `json:"last_weekly"` // week key handled by the scheduler
	TryDate    string    `json:"try_date"`
	Tries      int       `json:"tries"`
	LastTry    time.Time `json:"last_try"`
	Last       *Result   `json:"last,omitempty"`
}

// Runner runs AI jobs one at a time, on demand or on schedule.
type Runner struct {
	a   *app.App
	now func() time.Time

	mu      sync.Mutex
	running bool
	cur     Status
	st      state
}

func New(a *app.App) *Runner {
	r := &Runner{a: a, now: time.Now}
	if b, err := os.ReadFile(r.statePath()); err == nil {
		json.Unmarshal(b, &r.st)
	}
	return r
}

func (r *Runner) statePath() string { return filepath.Join(r.a.Cfg.DataDir, "ai_state.json") }

func (r *Runner) saveState() {
	b, _ := json.MarshalIndent(r.st, "", "  ")
	tmp := r.statePath() + ".tmp"
	if os.WriteFile(tmp, b, 0o644) == nil {
		os.Rename(tmp, r.statePath())
	}
}

func (r *Runner) client(c config.AIConfig) *Client {
	return &Client{BaseURL: c.BaseURL, APIKey: c.APIKey, Model: c.Model}
}

// Status returns the current job state and schedule.
func (r *Runner) Status() Status {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := r.cur
	s.Running = r.running
	s.Last = r.st.Last
	c := r.a.Cfg.AI()
	s.Ready = c.Ready()
	if s.Ready {
		s.NextDaily, s.NextWeekly = r.nextRuns(c)
	}
	return s
}

// Test sends a tiny request to check the endpoint, key and model.
func (r *Runner) Test(ctx context.Context, c config.AIConfig) (string, time.Duration, error) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	out, _, err := r.client(c).Chat(ctx, []Message{{Role: "user", Content: "请只回复两个字：收到"}}, nil)
	return textutil.Snippet(out, 60), time.Since(start), err
}

// Start launches a job in the background. kind is "daily" or "weekly";
// date is any day of the target day / week ("" = today).
func (r *Runner) Start(kind, date, trigger string) error {
	if !r.a.Cfg.AI().Ready() {
		return ErrNotReady
	}
	if kind != "daily" && kind != "weekly" {
		return fmt.Errorf("unknown kind %q (daily|weekly)", kind)
	}
	day, err := analysis.ParseDay(date)
	if err != nil {
		return err
	}
	if !r.begin(kind, day) {
		return ErrBusy
	}
	go r.run(kind, day, trigger)
	return nil
}

func (r *Runner) begin(kind string, day time.Time) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.running {
		return false
	}
	r.running = true
	key := day.Format("2006-01-02")
	if kind == "weekly" {
		key = analysis.WeekKey(day)
	}
	r.cur = Status{Kind: kind, DateKey: key, Stage: "准备中", StartedAt: r.now().Format(time.DateTime)}
	return true
}

func (r *Runner) stage(s string) {
	r.mu.Lock()
	r.cur.Stage = s
	r.mu.Unlock()
}

func (r *Runner) progress(n int) {
	r.mu.Lock()
	r.cur.Chars = n
	r.mu.Unlock()
}

// run executes a job synchronously; begin must have succeeded.
func (r *Runner) run(kind string, day time.Time, trigger string) *Result {
	start := r.now()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	res := &Result{Kind: kind, Trigger: trigger}
	var err error
	if kind == "weekly" {
		err = r.weekly(ctx, day, res)
	} else {
		err = r.daily(ctx, day, trigger, res)
	}
	res.OK = err == nil
	if err != nil && !errors.Is(err, errSkipped) {
		res.Error = err.Error()
		slog.Error("AI job failed", "kind", kind, "date", res.DateKey, "err", err)
	} else if res.OK {
		slog.Info("AI job finished", "kind", kind, "date", res.DateKey, "new_items", res.NewItems,
			"suggestions", res.Suggestions, "tokens", res.Usage.TotalTokens)
	}
	res.FinishedAt = r.now().Format(time.DateTime)
	res.Seconds = r.now().Sub(start).Seconds()

	r.mu.Lock()
	r.running = false
	r.cur = Status{}
	if !errors.Is(err, errSkipped) {
		r.st.Last = res
		r.saveState()
	}
	r.mu.Unlock()
	if errors.Is(err, errSkipped) {
		res.OK, res.Error = true, ""
	}
	return res
}

// chatJSON asks the model and decodes its JSON answer into v, retrying once
// when the reply is not valid JSON.
func (r *Runner) chatJSON(ctx context.Context, c config.AIConfig, msgs []Message, v any, res *Result) error {
	r.stage("AI 生成中")
	cl := r.client(c)
	for attempt := 0; ; attempt++ {
		out, u, err := cl.Chat(ctx, msgs, r.progress)
		res.Usage.PromptTokens += u.PromptTokens
		res.Usage.CompletionTokens += u.CompletionTokens
		res.Usage.TotalTokens += u.TotalTokens
		if err != nil {
			return err
		}
		js, err := ExtractJSON(out)
		if err == nil {
			if err = json.Unmarshal([]byte(js), v); err == nil {
				return nil
			}
		}
		if attempt >= 1 {
			return fmt.Errorf("AI 返回的内容不是有效的 JSON: %w", err)
		}
		msgs = append(msgs, Message{Role: "assistant", Content: out},
			Message{Role: "user", Content: "你的回复不是有效的 JSON。请只输出符合要求格式的 JSON 对象，不要输出其他任何文字。"})
	}
}

// ------------------------------------------------------------------ daily

type newItem struct {
	Kind      string `json:"kind"`
	Title     string `json:"title"`
	Detail    string `json:"detail"`
	Owner     string `json:"owner"`
	Due       string `json:"due"`
	Priority  string `json:"priority"`
	ConvTitle string `json:"conv_title"`
}

// flexID accepts 12, "12" or "#12".
type flexID int64

func (f *flexID) UnmarshalJSON(b []byte) error {
	s := strings.Trim(strings.TrimSpace(string(b)), `"#`)
	n, err := strconv.ParseInt(strings.TrimPrefix(s, "#"), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid todo id %s", b)
	}
	*f = flexID(n)
	return nil
}

type todoUpdate struct {
	ID      flexID `json:"id"`
	Suggest string `json:"suggest"`
	Reason  string `json:"reason"`
}

type dailyResult struct {
	Summary     string       `json:"summary"`
	Focus       []string     `json:"focus"`
	NewItems    []newItem    `json:"new_items"`
	TodoUpdates []todoUpdate `json:"todo_updates"`
}

func (r *Runner) daily(ctx context.Context, day time.Time, trigger string, res *Result) error {
	c := r.a.Cfg.AI()
	date := day.Format("2006-01-02")
	res.DateKey = date
	r.stage("同步并整理数据")
	maxChars := c.MaxInputChars
	if maxChars <= 0 {
		maxChars = defaultMaxInputChars
	}
	// Privacy: only conversations that involve me (single chats, groups I
	// spoke in or was @-ed in, project groups); other groups appear only as
	// statistics.
	dg, err := r.a.Digest(date, "", "", analysis.DigestOptions{Scope: "involved", SkipBots: true, MaxChars: maxChars}, true)
	if err != nil {
		return err
	}
	if trigger == "schedule" && dg.Report.Totals.Messages == 0 && dg.Report.Calendar.MeetingCount == 0 {
		res.Note = "当天没有消息和会议，跳过"
		return errSkipped
	}
	open, err := r.a.Agent.ListTodos(agentdb.TodoFilter{Status: "open", Limit: 100})
	if err != nil {
		return err
	}
	user := dailyUserPrompt(date, open, dg.Markdown)
	res.InputChars = len([]rune(user))

	var out dailyResult
	if err := r.chatJSON(ctx, c, []Message{{Role: "system", Content: dailySystem}, {Role: "user", Content: user}}, &out, res); err != nil {
		return err
	}
	r.stage("保存结果")
	cids := map[string]string{}
	for _, cv := range dg.Convs {
		cids[cv.Title] = cv.CID
	}
	for _, g := range dg.Report.Groups.List {
		if _, ok := cids[g.Title]; !ok {
			cids[g.Title] = g.CID
		}
	}
	openIDs := map[int64]bool{}
	for _, t := range open {
		openIDs[t.ID] = true
	}
	n, s, err := applyDaily(r.a.Agent, date, AuthorPrefix+" · "+c.Model, out, cids, openIDs)
	res.NewItems, res.Suggestions = n, s
	return err
}

// applyDaily stores the daily result: summary, new items and suggestions.
func applyDaily(db *agentdb.DB, date, author string, out dailyResult, cids map[string]string, openIDs map[int64]bool) (int, int, error) {
	if strings.TrimSpace(out.Summary) == "" {
		return 0, 0, errors.New("AI 没有生成日报内容")
	}
	focus := []string{}
	for _, f := range out.Focus {
		if f = strings.TrimSpace(f); f != "" {
			focus = append(focus, f)
		}
	}
	if _, err := db.SaveSummary(agentdb.Summary{DateKey: date, Summary: out.Summary, Focus: focus, Author: author}); err != nil {
		return 0, 0, err
	}
	var items []agentdb.Todo
	for _, it := range out.NewItems {
		if strings.TrimSpace(it.Title) == "" {
			continue
		}
		due := strings.TrimSpace(it.Due)
		if _, err := time.Parse("2006-01-02", due); err != nil {
			due = ""
		}
		prio := strings.ToLower(strings.TrimSpace(it.Priority))
		if prio != "high" && prio != "low" {
			prio = "medium"
		}
		kind := "todo"
		if k := strings.ToLower(strings.TrimSpace(it.Kind)); strings.HasPrefix(k, "follow") {
			kind = "follow_up"
		}
		title := strings.TrimSpace(it.ConvTitle)
		items = append(items, agentdb.Todo{Kind: kind, Title: it.Title, Detail: it.Detail, Owner: it.Owner,
			Due: due, Priority: prio, CID: cids[title], ConvTitle: title, SourceDate: date})
	}
	created := 0
	for _, r := range db.UpsertTodos(items) {
		if r.Created {
			created++
		}
	}
	suggested := 0
	for _, u := range out.TodoUpdates {
		id := int64(u.ID)
		s := strings.ToLower(strings.TrimSpace(u.Suggest))
		if !openIDs[id] || (s != "done" && s != "dropped") {
			continue
		}
		if db.SuggestTodo(id, s, u.Reason) == nil {
			suggested++
		}
	}
	return created, suggested, nil
}

// ----------------------------------------------------------------- weekly

type weeklyResult struct {
	Summary string   `json:"summary"`
	Focus   []string `json:"focus"`
}

func (r *Runner) weekly(ctx context.Context, day time.Time, res *Result) error {
	c := r.a.Cfg.AI()
	mon, sun := analysis.WeekRange(day)
	res.DateKey = analysis.WeekKey(day)
	last := sun
	if today := dayOf(r.now()); today.Before(sun) {
		last = today
	}
	if last.Before(mon) {
		return errors.New("这一周还没有开始")
	}
	from, to := mon.Format("2006-01-02"), last.Format("2006-01-02")
	r.stage("同步并整理数据")
	rep, err := r.a.Report("", from, to, true)
	if err != nil {
		return err
	}
	w := weeklyInput{From: from, To: to, Report: rep.Report, Summaries: map[string]*agentdb.Summary{}}
	for d := mon; !d.After(last); d = d.AddDate(0, 0, 1) {
		k := d.Format("2006-01-02")
		s, _ := r.a.Agent.GetSummary(k)
		if s == nil && (d.Weekday() == time.Saturday || d.Weekday() == time.Sunday) {
			continue // quiet weekends are not worth a line
		}
		w.Days = append(w.Days, k)
		w.Summaries[k] = s
	}
	if w.Todos, err = r.a.Agent.TodosChangedBetween(from, to); err != nil {
		return err
	}
	nmon, nsun := mon.AddDate(0, 0, 7), sun.AddDate(0, 0, 7)
	w.NextLabel = nmon.Format("2006-01-02") + " ~ " + nsun.Format("2006-01-02")
	if cal, err := r.a.Calendar("", nmon.Format("2006-01-02"), nsun.Format("2006-01-02"), false, false); err == nil {
		for _, e := range cal.Events {
			if e.Kind != "approval_todo" {
				w.NextWeek = append(w.NextWeek, e)
			}
		}
	}
	user := weeklyUserPrompt(w)
	res.InputChars = len([]rune(user))

	var out weeklyResult
	if err := r.chatJSON(ctx, c, []Message{{Role: "system", Content: weeklySystem}, {Role: "user", Content: user}}, &out, res); err != nil {
		return err
	}
	if strings.TrimSpace(out.Summary) == "" {
		return errors.New("AI 没有生成周报内容")
	}
	r.stage("保存结果")
	_, err = r.a.Agent.SaveSummary(agentdb.Summary{DateKey: res.DateKey, Summary: out.Summary, Focus: out.Focus,
		Author: AuthorPrefix + " · " + c.Model})
	return err
}

// --------------------------------------------------------------- schedule

func dayOf(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.Local)
}

// at returns day at "HH:MM" (falls back to 18:30).
func at(day time.Time, hhmm string) time.Time {
	t, err := time.Parse("15:04", strings.TrimSpace(hhmm))
	if err != nil {
		t, _ = time.Parse("15:04", "18:30")
	}
	return day.Add(time.Duration(t.Hour())*time.Hour + time.Duration(t.Minute())*time.Minute)
}

// nextRuns reports when the scheduler will next produce each report.
// Caller holds r.mu.
func (r *Runner) nextRuns(c config.AIConfig) (daily, weekly string) {
	now := r.now()
	today := dayOf(now)
	if c.DailyEnabled {
		d := at(today, c.DailyTime)
		if r.st.LastDaily == today.Format("2006-01-02") {
			d = at(today.AddDate(0, 0, 1), c.DailyTime)
		} else if d.Before(now) {
			d = now
		}
		daily = d.Format("2006-01-02 15:04")
	}
	if c.WeeklyEnabled {
		for i := 0; i < 8; i++ {
			d := today.AddDate(0, 0, i)
			if int(d.Weekday()) != c.WeeklyDay || (i == 0 && r.st.LastWeekly == analysis.WeekKey(today)) {
				continue
			}
			t := at(d, c.DailyTime)
			if t.Before(now) {
				t = now
			}
			weekly = t.Format("2006-01-02 15:04")
			break
		}
	}
	return daily, weekly
}

// Schedule runs the daily report every day at DailyTime and the weekly
// report on WeeklyDay, catching up later that day if the program was not
// running at that time. Failed runs are retried up to 3 times, 30 min apart.
func (r *Runner) Schedule(ctx context.Context) {
	go func() {
		t := time.NewTicker(30 * time.Second)
		defer t.Stop()
		for {
			r.tick()
			select {
			case <-ctx.Done():
				return
			case <-t.C:
			}
		}
	}()
}

func (r *Runner) tick() {
	c := r.a.Cfg.AI()
	if !c.Ready() {
		return
	}
	now := r.now()
	today := dayOf(now)
	key := today.Format("2006-01-02")
	if now.Before(at(today, c.DailyTime)) {
		return
	}
	r.mu.Lock()
	if r.st.TryDate != key {
		r.st.TryDate, r.st.Tries = key, 0
	}
	needDaily := c.DailyEnabled && r.st.LastDaily != key
	needWeekly := c.WeeklyEnabled && int(today.Weekday()) == c.WeeklyDay && r.st.LastWeekly != analysis.WeekKey(today)
	canTry := r.st.Tries < 3 && now.Sub(r.st.LastTry) >= 30*time.Minute
	r.mu.Unlock()
	if !(needDaily || needWeekly) || !canTry {
		return
	}

	if needDaily {
		if s, _ := r.a.Agent.GetSummary(key); s != nil && !strings.HasPrefix(s.Author, AuthorPrefix) {
			slog.Info("AI daily skipped: a report from another agent already exists", "date", key, "author", s.Author)
			r.mark(func(st *state) { st.LastDaily = key })
			needDaily = false
		}
	}
	if needDaily {
		if !r.begin("daily", today) {
			return // a manual job is running; try again next tick
		}
		res := r.run("daily", today, "schedule")
		r.mark(func(st *state) {
			if res.OK {
				st.LastDaily = key
			} else {
				st.Tries++
				st.LastTry = r.now()
			}
		})
		if !res.OK {
			return
		}
	}
	if needWeekly {
		if !r.begin("weekly", today) {
			return
		}
		res := r.run("weekly", today, "schedule")
		r.mark(func(st *state) {
			if res.OK {
				st.LastWeekly = analysis.WeekKey(today)
			} else {
				st.Tries++
				st.LastTry = r.now()
			}
		})
	}
}

func (r *Runner) mark(f func(*state)) {
	r.mu.Lock()
	f(&r.st)
	r.saveState()
	r.mu.Unlock()
}
