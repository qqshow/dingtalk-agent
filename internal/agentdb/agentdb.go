// Package agentdb stores what agents write back: daily summaries and
// todo / follow-up items. It lives in data/agent.db, separate from the
// decrypted (read-only, regenerated) DingTalk snapshot.
package agentdb

import (
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/qqshow/dingtalk-agent/internal/sqlitex"
)

type DB struct{ db *sql.DB }

const schema = `
CREATE TABLE IF NOT EXISTS summaries (
	date_key   TEXT PRIMARY KEY,
	summary    TEXT NOT NULL,
	focus      TEXT NOT NULL DEFAULT '[]',
	author     TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS todos (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	kind        TEXT NOT NULL DEFAULT 'todo',
	title       TEXT NOT NULL,
	detail      TEXT NOT NULL DEFAULT '',
	priority    TEXT NOT NULL DEFAULT 'medium',
	status      TEXT NOT NULL DEFAULT 'open',
	due         TEXT NOT NULL DEFAULT '',
	owner       TEXT NOT NULL DEFAULT '',
	cid         TEXT NOT NULL DEFAULT '',
	conv_title  TEXT NOT NULL DEFAULT '',
	msg_id      INTEGER NOT NULL DEFAULT 0,
	source_date TEXT NOT NULL DEFAULT '',
	dedupe_key  TEXT NOT NULL UNIQUE,
	created_at  TEXT NOT NULL,
	updated_at  TEXT NOT NULL,
	done_at     TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_todos_status ON todos(status);
CREATE INDEX IF NOT EXISTS idx_todos_source ON todos(source_date);
`

func Open(path string) (*DB, error) {
	db, err := sqlitex.Open(path, "_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("init agent db: %w", err)
	}
	return &DB{db: db}, nil
}

func (d *DB) Close() error { return d.db.Close() }

func now() string { return time.Now().Format(time.DateTime) }

// ---------------------------------------------------------------- summaries

type Summary struct {
	DateKey   string   `json:"date"`
	Summary   string   `json:"summary"`
	Focus     []string `json:"focus"`
	Author    string   `json:"author"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
}

func (d *DB) SaveSummary(s Summary) (*Summary, error) {
	if strings.TrimSpace(s.DateKey) == "" || strings.TrimSpace(s.Summary) == "" {
		return nil, errors.New("date and summary are required")
	}
	if s.Focus == nil {
		s.Focus = []string{}
	}
	focus, _ := json.Marshal(s.Focus)
	t := now()
	_, err := d.db.Exec(`INSERT INTO summaries(date_key, summary, focus, author, created_at, updated_at)
		VALUES(?,?,?,?,?,?)
		ON CONFLICT(date_key) DO UPDATE SET summary=excluded.summary, focus=excluded.focus,
		author=excluded.author, updated_at=excluded.updated_at`,
		s.DateKey, s.Summary, string(focus), s.Author, t, t)
	if err != nil {
		return nil, err
	}
	return d.GetSummary(s.DateKey)
}

func (d *DB) GetSummary(dateKey string) (*Summary, error) {
	var s Summary
	var focus string
	err := d.db.QueryRow(`SELECT date_key, summary, focus, author, created_at, updated_at FROM summaries WHERE date_key=?`,
		dateKey).Scan(&s.DateKey, &s.Summary, &focus, &s.Author, &s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(focus), &s.Focus)
	if s.Focus == nil {
		s.Focus = []string{}
	}
	return &s, nil
}

// ListSummaries returns the most recent summaries (without full text if brief).
func (d *DB) ListSummaries(limit int) ([]Summary, error) {
	rows, err := d.db.Query(`SELECT date_key, summary, focus, author, created_at, updated_at
		FROM summaries ORDER BY date_key DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Summary{}
	for rows.Next() {
		var s Summary
		var focus string
		rows.Scan(&s.DateKey, &s.Summary, &focus, &s.Author, &s.CreatedAt, &s.UpdatedAt)
		json.Unmarshal([]byte(focus), &s.Focus)
		out = append(out, s)
	}
	return out, rows.Err()
}

// -------------------------------------------------------------------- todos

type Todo struct {
	ID         int64  `json:"id"`
	Kind       string `json:"kind"` // todo | follow_up
	Title      string `json:"title"`
	Detail     string `json:"detail"`
	Priority   string `json:"priority"` // high | medium | low
	Status     string `json:"status"`   // open | done | dropped
	Due        string `json:"due"`
	Owner      string `json:"owner"` // who is responsible / who to chase
	CID        string `json:"cid"`
	ConvTitle  string `json:"conv_title"`
	MsgID      int64  `json:"msg_id"`
	SourceDate string `json:"source_date"`
	DedupeKey  string `json:"dedupe_key"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
	DoneAt     string `json:"done_at"`
}

var (
	validKind     = map[string]bool{"todo": true, "follow_up": true}
	validPriority = map[string]bool{"high": true, "medium": true, "low": true}
	validStatus   = map[string]bool{"open": true, "done": true, "dropped": true}
)

func (t *Todo) normalize() error {
	t.Title = strings.TrimSpace(t.Title)
	if t.Title == "" {
		return errors.New("todo title is required")
	}
	t.Kind = strings.ToLower(strings.TrimSpace(t.Kind))
	if t.Kind == "" || t.Kind == "followup" || t.Kind == "follow-up" {
		if t.Kind == "" {
			t.Kind = "todo"
		} else {
			t.Kind = "follow_up"
		}
	}
	if !validKind[t.Kind] {
		return fmt.Errorf("invalid kind %q (todo|follow_up)", t.Kind)
	}
	t.Priority = strings.ToLower(strings.TrimSpace(t.Priority))
	if t.Priority == "" {
		t.Priority = "medium"
	}
	if !validPriority[t.Priority] {
		return fmt.Errorf("invalid priority %q (high|medium|low)", t.Priority)
	}
	if t.DedupeKey == "" {
		h := sha1.Sum([]byte(t.Kind + "|" + t.CID + "|" + strings.ToLower(t.Title)))
		t.DedupeKey = hex.EncodeToString(h[:8])
	}
	return nil
}

// UpsertResult reports what happened for each submitted todo.
type UpsertResult struct {
	Todo    Todo   `json:"todo"`
	Created bool   `json:"created"`
	Error   string `json:"error,omitempty"`
}

// UpsertTodos inserts new items; items whose dedupe_key already exists get
// their content refreshed but keep status (so re-running a daily job does
// not resurrect finished items or create duplicates).
func (d *DB) UpsertTodos(items []Todo) []UpsertResult {
	var out []UpsertResult
	for _, t := range items {
		if err := t.normalize(); err != nil {
			out = append(out, UpsertResult{Todo: t, Error: err.Error()})
			continue
		}
		ts := now()
		var existing int64
		d.db.QueryRow(`SELECT id FROM todos WHERE dedupe_key=?`, t.DedupeKey).Scan(&existing)
		if existing > 0 {
			_, err := d.db.Exec(`UPDATE todos SET title=?, detail=CASE WHEN ?<>'' THEN ? ELSE detail END,
				priority=?, due=CASE WHEN ?<>'' THEN ? ELSE due END, owner=CASE WHEN ?<>'' THEN ? ELSE owner END,
				updated_at=? WHERE id=?`,
				t.Title, t.Detail, t.Detail, t.Priority, t.Due, t.Due, t.Owner, t.Owner, ts, existing)
			r := UpsertResult{}
			if err != nil {
				r.Error = err.Error()
			}
			if got, _ := d.GetTodo(existing); got != nil {
				r.Todo = *got
			}
			out = append(out, r)
			continue
		}
		res, err := d.db.Exec(`INSERT INTO todos(kind,title,detail,priority,status,due,owner,cid,conv_title,msg_id,
			source_date,dedupe_key,created_at,updated_at) VALUES(?,?,?,?, 'open', ?,?,?,?,?,?,?,?,?)`,
			t.Kind, t.Title, t.Detail, t.Priority, t.Due, t.Owner, t.CID, t.ConvTitle, t.MsgID,
			t.SourceDate, t.DedupeKey, ts, ts)
		if err != nil {
			out = append(out, UpsertResult{Todo: t, Error: err.Error()})
			continue
		}
		id, _ := res.LastInsertId()
		got, _ := d.GetTodo(id)
		out = append(out, UpsertResult{Todo: *got, Created: true})
	}
	return out
}

const todoCols = `id,kind,title,detail,priority,status,due,owner,cid,conv_title,msg_id,source_date,dedupe_key,created_at,updated_at,done_at`

func scanTodo(r interface{ Scan(...any) error }) (*Todo, error) {
	var t Todo
	err := r.Scan(&t.ID, &t.Kind, &t.Title, &t.Detail, &t.Priority, &t.Status, &t.Due, &t.Owner, &t.CID,
		&t.ConvTitle, &t.MsgID, &t.SourceDate, &t.DedupeKey, &t.CreatedAt, &t.UpdatedAt, &t.DoneAt)
	return &t, err
}

func (d *DB) GetTodo(id int64) (*Todo, error) {
	t, err := scanTodo(d.db.QueryRow(`SELECT `+todoCols+` FROM todos WHERE id=?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return t, err
}

type TodoFilter struct {
	Status     string // open|done|dropped|all ("" = open)
	Kind       string
	SourceDate string
	Limit      int
}

func (d *DB) ListTodos(f TodoFilter) ([]Todo, error) {
	var conds []string
	var args []any
	switch f.Status {
	case "", "open":
		conds = append(conds, "status='open'")
	case "all":
	default:
		conds = append(conds, "status=?")
		args = append(args, f.Status)
	}
	if f.Kind != "" {
		conds = append(conds, "kind=?")
		args = append(args, f.Kind)
	}
	if f.SourceDate != "" {
		conds = append(conds, "source_date=?")
		args = append(args, f.SourceDate)
	}
	q := `SELECT ` + todoCols + ` FROM todos`
	if len(conds) > 0 {
		q += " WHERE " + strings.Join(conds, " AND ")
	}
	q += ` ORDER BY CASE status WHEN 'open' THEN 0 ELSE 1 END,
		CASE priority WHEN 'high' THEN 0 WHEN 'medium' THEN 1 ELSE 2 END,
		CASE WHEN due='' THEN 1 ELSE 0 END, due, id DESC`
	if f.Limit <= 0 {
		f.Limit = 200
	}
	q += fmt.Sprintf(" LIMIT %d", f.Limit)
	rows, err := d.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Todo{}
	for rows.Next() {
		t, err := scanTodo(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

// TodoPatch updates selected fields; nil means unchanged.
type TodoPatch struct {
	Status   *string `json:"status"`
	Title    *string `json:"title"`
	Detail   *string `json:"detail"`
	Priority *string `json:"priority"`
	Due      *string `json:"due"`
	Owner    *string `json:"owner"`
	Kind     *string `json:"kind"`
}

func (d *DB) UpdateTodo(id int64, p TodoPatch) (*Todo, error) {
	t, err := d.GetTodo(id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, fmt.Errorf("todo %d not found", id)
	}
	if p.Status != nil {
		if !validStatus[*p.Status] {
			return nil, fmt.Errorf("invalid status %q (open|done|dropped)", *p.Status)
		}
		if *p.Status != "open" && t.Status == "open" {
			t.DoneAt = now()
		}
		if *p.Status == "open" {
			t.DoneAt = ""
		}
		t.Status = *p.Status
	}
	if p.Title != nil && strings.TrimSpace(*p.Title) != "" {
		t.Title = strings.TrimSpace(*p.Title)
	}
	if p.Detail != nil {
		t.Detail = *p.Detail
	}
	if p.Priority != nil {
		if !validPriority[*p.Priority] {
			return nil, fmt.Errorf("invalid priority %q", *p.Priority)
		}
		t.Priority = *p.Priority
	}
	if p.Kind != nil {
		if !validKind[*p.Kind] {
			return nil, fmt.Errorf("invalid kind %q", *p.Kind)
		}
		t.Kind = *p.Kind
	}
	if p.Due != nil {
		t.Due = *p.Due
	}
	if p.Owner != nil {
		t.Owner = *p.Owner
	}
	_, err = d.db.Exec(`UPDATE todos SET status=?,title=?,detail=?,priority=?,kind=?,due=?,owner=?,done_at=?,updated_at=? WHERE id=?`,
		t.Status, t.Title, t.Detail, t.Priority, t.Kind, t.Due, t.Owner, t.DoneAt, now(), id)
	if err != nil {
		return nil, err
	}
	return d.GetTodo(id)
}

func (d *DB) DeleteTodo(id int64) error {
	res, err := d.db.Exec(`DELETE FROM todos WHERE id=?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("todo %d not found", id)
	}
	return nil
}
