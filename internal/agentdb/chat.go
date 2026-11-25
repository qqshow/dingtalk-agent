package agentdb

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const chatSchema = `
CREATE TABLE IF NOT EXISTS chat_sessions (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	title      TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS chat_messages (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	session_id INTEGER NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
	role       TEXT NOT NULL,             -- user | assistant
	content    TEXT NOT NULL DEFAULT '',
	context    TEXT NOT NULL DEFAULT '',  -- data injected for @mentions (user messages)
	meta       TEXT NOT NULL DEFAULT '{}',
	created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_chat_messages_session ON chat_messages(session_id, id);
`

// ChatSession is one AI conversation.
type ChatSession struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	Messages  int    `json:"message_count"`
}

// Mention is a person or group referenced with @ in a chat message.
type Mention struct {
	CID   string `json:"cid"`
	Title string `json:"title"`
	Kind  string `json:"kind"` // person | group
}

// ToolUse records a tool call made while answering, for display.
type ToolUse struct {
	Name    string `json:"name"`
	Title   string `json:"title"`
	Args    string `json:"args"`
	Summary string `json:"summary"`
	Error   string `json:"error,omitempty"`
}

// ChatMeta is extra information stored with a message.
type ChatMeta struct {
	Mentions []Mention `json:"mentions,omitempty"`
	Days     int       `json:"days,omitempty"`
	Images   []string  `json:"images,omitempty"` // file names under the session's image folder
	Tools    []ToolUse `json:"tools,omitempty"`
	Model    string    `json:"model,omitempty"`
	Error    string    `json:"error,omitempty"`
	Note     string    `json:"note,omitempty"`
}

// ChatMessage is a stored message.
type ChatMessage struct {
	ID        int64    `json:"id"`
	SessionID int64    `json:"session_id"`
	Role      string   `json:"role"`
	Content   string   `json:"content"`
	Context   string   `json:"-"`
	Meta      ChatMeta `json:"meta"`
	CreatedAt string   `json:"created_at"`
}

func (d *DB) initChat() error {
	_, err := d.db.Exec(chatSchema)
	return err
}

func (d *DB) CreateChatSession(title string) (*ChatSession, error) {
	t := now()
	res, err := d.db.Exec(`INSERT INTO chat_sessions(title, created_at, updated_at) VALUES(?,?,?)`, title, t, t)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return d.GetChatSession(id)
}

func (d *DB) GetChatSession(id int64) (*ChatSession, error) {
	var s ChatSession
	err := d.db.QueryRow(`SELECT id, title, created_at, updated_at,
		(SELECT COUNT(*) FROM chat_messages WHERE session_id=chat_sessions.id) FROM chat_sessions WHERE id=?`, id).
		Scan(&s.ID, &s.Title, &s.CreatedAt, &s.UpdatedAt, &s.Messages)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("对话 %d 不存在", id)
	}
	return &s, err
}

func (d *DB) ListChatSessions(limit int) ([]ChatSession, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := d.db.Query(`SELECT id, title, created_at, updated_at,
		(SELECT COUNT(*) FROM chat_messages WHERE session_id=chat_sessions.id)
		FROM chat_sessions ORDER BY updated_at DESC, id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ChatSession{}
	for rows.Next() {
		var s ChatSession
		if err := rows.Scan(&s.ID, &s.Title, &s.CreatedAt, &s.UpdatedAt, &s.Messages); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (d *DB) RenameChatSession(id int64, title string) error {
	title = strings.TrimSpace(title)
	if title == "" {
		return errors.New("标题不能为空")
	}
	_, err := d.db.Exec(`UPDATE chat_sessions SET title=? WHERE id=?`, title, id)
	return err
}

func (d *DB) DeleteChatSession(id int64) error {
	if _, err := d.db.Exec(`DELETE FROM chat_messages WHERE session_id=?`, id); err != nil {
		return err
	}
	res, err := d.db.Exec(`DELETE FROM chat_sessions WHERE id=?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("对话 %d 不存在", id)
	}
	return nil
}

func (d *DB) AddChatMessage(m ChatMessage) (*ChatMessage, error) {
	meta, _ := json.Marshal(m.Meta)
	t := now()
	res, err := d.db.Exec(`INSERT INTO chat_messages(session_id, role, content, context, meta, created_at) VALUES(?,?,?,?,?,?)`,
		m.SessionID, m.Role, m.Content, m.Context, string(meta), t)
	if err != nil {
		return nil, err
	}
	d.db.Exec(`UPDATE chat_sessions SET updated_at=? WHERE id=?`, t, m.SessionID)
	m.ID, _ = res.LastInsertId()
	m.CreatedAt = t
	return &m, nil
}

// ChatMessages returns a session's messages in order.
func (d *DB) ChatMessages(sessionID int64) ([]ChatMessage, error) {
	rows, err := d.db.Query(`SELECT id, session_id, role, content, context, meta, created_at
		FROM chat_messages WHERE session_id=? ORDER BY id`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ChatMessage{}
	for rows.Next() {
		var m ChatMessage
		var meta string
		if err := rows.Scan(&m.ID, &m.SessionID, &m.Role, &m.Content, &m.Context, &meta, &m.CreatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(meta), &m.Meta)
		out = append(out, m)
	}
	return out, rows.Err()
}
