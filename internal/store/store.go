// Package store gives read access to the decrypted DingTalk database.
//
// Each sync produces a brand-new immutable snapshot file; data/decrypted/current
// names the active one. Store watches that pointer (so a separate `mcp` stdio
// process sees syncs done by the web server) and swaps snapshots atomically.
package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/qqshow/dingtalk-agent/internal/config"
	"github.com/qqshow/dingtalk-agent/internal/sqlitex"
)

// ErrNoData is returned before the first successful sync.
var ErrNoData = errors.New("no decrypted database yet: run a sync first")

const msgTableCount = 128

type Store struct {
	cfg *config.Config

	mu        sync.Mutex // serializes reloads
	cur       atomic.Pointer[Snapshot]
	lastCheck atomic.Int64 // unix ms of last pointer check
}

func New(cfg *config.Config) *Store { return &Store{cfg: cfg} }

func (s *Store) pointerPath() string { return filepath.Join(s.cfg.DecryptedDir(), "current") }

// CurrentPath returns the active snapshot file path ("" if none).
func (s *Store) CurrentPath() string {
	b, err := os.ReadFile(s.pointerPath())
	if err != nil {
		return ""
	}
	name := strings.TrimSpace(string(b))
	if name == "" {
		return ""
	}
	return filepath.Join(s.cfg.DecryptedDir(), filepath.Base(name))
}

// Publish makes path the active snapshot and loads it.
func (s *Store) Publish(path string) error {
	tmp := s.pointerPath() + ".tmp"
	if err := os.WriteFile(tmp, []byte(filepath.Base(path)), 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, s.pointerPath()); err != nil {
		return err
	}
	s.lastCheck.Store(0)
	_, err := s.Snapshot()
	s.cleanup()
	return err
}

// cleanup removes snapshot files that are no longer current. Files still open
// (by this or another process) fail to delete on Windows; they are retried on
// the next sync.
func (s *Store) cleanup() {
	cur := filepath.Base(s.CurrentPath())
	curCal := filepath.Base(calendarPathFor(s.CurrentPath()))
	files, _ := filepath.Glob(filepath.Join(s.cfg.DecryptedDir(), "dingtalk_*.db*"))
	cals, _ := filepath.Glob(filepath.Join(s.cfg.DecryptedDir(), "calendar_*.db*"))
	for _, f := range append(files, cals...) {
		base := filepath.Base(f)
		if strings.HasPrefix(base, cur) || strings.HasPrefix(base, curCal) {
			continue
		}
		if st, err := os.Stat(f); err == nil && time.Since(st.ModTime()) < 10*time.Second {
			continue // possibly a snapshot being built right now
		}
		os.Remove(f)
	}
}

// Snapshot returns the active snapshot, reloading if the pointer changed.
func (s *Store) Snapshot() (*Snapshot, error) {
	now := time.Now().UnixMilli()
	cur := s.cur.Load()
	if cur != nil && now-s.lastCheck.Load() < 2000 {
		return cur, nil
	}
	s.lastCheck.Store(now)
	path := s.CurrentPath()
	if path == "" {
		if cur != nil {
			return cur, nil
		}
		return nil, ErrNoData
	}
	if cur != nil && cur.Path == path {
		return cur, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if cur = s.cur.Load(); cur != nil && cur.Path == path {
		return cur, nil
	}
	snap, err := openSnapshot(path, s.cfg.UID())
	if err != nil {
		if cur != nil {
			slog.Error("loading new snapshot failed, keeping old one", "err", err)
			return cur, nil
		}
		return nil, err
	}
	old := s.cur.Swap(snap)
	if old != nil {
		// Let in-flight requests finish before closing the old pool.
		db, cal, path := old.DB, old.Cal, old.Path
		time.AfterFunc(30*time.Second, func() {
			db.Close()
			if cal != nil {
				cal.Close()
			}
			os.Remove(path)
			os.Remove(calendarPathFor(path))
		})
	}
	slog.Info("snapshot loaded", "path", filepath.Base(path), "conversations", len(snap.Convs),
		"users", len(snap.Users), "calendar", snap.Cal != nil)
	return snap, nil
}

// Snapshot is an immutable, fully loaded view of one decrypted database.
type Snapshot struct {
	Path     string
	DB       *sql.DB
	MyUID    int64
	Created  time.Time
	Users    map[int64]User
	nicks    map[string]map[int64]string // cid -> uid -> group nickname
	Convs    []*Conversation             // top first, then most recent
	ConvByID map[string]*Conversation
	cidTable map[string]string
	tables   []string

	Cal     *sql.DB           // decrypted calendar_v2.db; nil when unavailable
	folders map[string]string // calendar folder_id -> name

	lastOnce sync.Once
	lastMsgs map[string]LastMsg
}

// calendarPathFor maps dingtalk_<ts>.db to its calendar_<ts>.db sibling.
func calendarPathFor(msgPath string) string {
	if msgPath == "" {
		return ""
	}
	dir, base := filepath.Split(msgPath)
	return filepath.Join(dir, "calendar_"+strings.TrimPrefix(base, "dingtalk_"))
}

type User struct {
	UID      int64  `json:"uid"`
	Nick     string `json:"nick"`
	RealName string `json:"real_name"`
	UserType int64  `json:"user_type"`
}

// IsBot reports whether the profile is a robot / service account.
// Real people have userType 0; system accounts (钉钉云盘 = 1) and robots
// (2, 3, 5) are non-zero.
func (u User) IsBot() bool { return u.UserType != 0 }

// IsBot reports whether uid belongs to a robot / service account.
func (s *Snapshot) IsBot(uid int64) bool {
	u, ok := s.Users[uid]
	return ok && u.IsBot()
}

func (u User) Name() string {
	if u.RealName != "" {
		return u.RealName
	}
	return u.Nick
}

type Conversation struct {
	CID            string `json:"cid"`
	Type           string `json:"type"` // single | group
	Title          string `json:"title"`
	MemberCount    int64  `json:"member_count"`
	CreateAt       int64  `json:"create_at"`
	LastModify     int64  `json:"last_modify"`
	UnreadCount    int64  `json:"unread_count"`
	IsTop          bool   `json:"is_top"`
	OwnerID        int64  `json:"owner_id"`
	OtherUID       string `json:"other_uid,omitempty"`
	IsProject      bool   `json:"is_project"`
	IsNotification bool   `json:"is_notification"`
	IsBot          bool   `json:"is_bot"` // single chat with a robot / service account
	top            int64
}

func openSnapshot(path, myUID string) (*Snapshot, error) {
	db, err := sqlitex.Open(path, "mode=ro&immutable=1")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(8)
	s := &Snapshot{Path: path, DB: db}
	s.MyUID, _ = strconv.ParseInt(myUID, 10, 64)
	if st, err := os.Stat(path); err == nil {
		s.Created = st.ModTime()
	}
	if err := s.load(); err != nil {
		db.Close()
		return nil, fmt.Errorf("load snapshot %s: %w", filepath.Base(path), err)
	}
	s.openCalendar(calendarPathFor(path))
	return s, nil
}

func (s *Snapshot) load() error {
	// Message tables present in this DB.
	rows, err := s.DB.Query(`SELECT name FROM sqlite_master WHERE type='table' AND name LIKE 'tbmsg\_%' ESCAPE '\'`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var n string
		rows.Scan(&n)
		if len(n) == len("tbmsg_000") {
			s.tables = append(s.tables, n)
		}
	}
	rows.Close()
	sort.Strings(s.tables)
	if len(s.tables) == 0 {
		return errors.New("no tbmsg_* tables")
	}

	// Users.
	s.Users = map[int64]User{}
	rows, err = s.DB.Query(`SELECT uid, nick, realName, userType FROM tbuser_profile_v2`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var uid, nick, real, ut any
		rows.Scan(&uid, &nick, &real, &ut)
		u := User{UID: sqlitex.Int(uid), Nick: cleanText(sqlitex.Str(nick)), RealName: cleanText(sqlitex.Str(real)),
			UserType: sqlitex.Int(ut)}
		s.Users[u.UID] = u
	}
	rows.Close()

	// Group nicknames.
	s.nicks = map[string]map[int64]string{}
	if rows, err = s.DB.Query(`SELECT cid, uid, groupNick FROM tbuser_group_nick`); err == nil {
		for rows.Next() {
			var cid, uid, nick any
			rows.Scan(&cid, &uid, &nick)
			c := sqlitex.Str(cid)
			if s.nicks[c] == nil {
				s.nicks[c] = map[int64]string{}
			}
			s.nicks[c][sqlitex.Int(uid)] = sqlitex.Str(nick)
		}
		rows.Close()
	}

	// Conversations.
	s.ConvByID = map[string]*Conversation{}
	rows, err = s.DB.Query(`SELECT cid, type, title, memberCount, createAt, lastModify,
		unreadCount, top, ownerId, isNotification, extension FROM tbconversation WHERE status = 1`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var cid, typ, title, mc, ca, lm, uc, top, owner, notif, ext any
		if err := rows.Scan(&cid, &typ, &title, &mc, &ca, &lm, &uc, &top, &owner, &notif, &ext); err != nil {
			rows.Close()
			return err
		}
		c := &Conversation{
			CID: sqlitex.Str(cid), Title: cleanText(sqlitex.Str(title)),
			MemberCount: sqlitex.Int(mc), CreateAt: sqlitex.Int(ca), LastModify: sqlitex.Int(lm),
			UnreadCount: sqlitex.Int(uc), top: sqlitex.Int(top), OwnerID: sqlitex.Int(owner),
			IsNotification: sqlitex.Int(notif) != 0,
		}
		c.IsTop = c.top > 0
		if sqlitex.Int(typ) == 1 {
			c.Type = "single"
			if a, b, ok := strings.Cut(c.CID, ":"); ok {
				other := b
				if b == strconv.FormatInt(s.MyUID, 10) {
					other = a
				}
				c.OtherUID = other
				if id, err := strconv.ParseInt(other, 10, 64); err == nil {
					if u, ok := s.Users[id]; ok {
						if u.Name() != "" {
							c.Title = u.Name()
						}
						c.IsBot = u.IsBot()
					}
				}
			}
		} else {
			c.Type = "group"
			c.IsProject = isProjectGroup(sqlitex.Str(ext))
		}
		s.Convs = append(s.Convs, c)
		s.ConvByID[c.CID] = c
	}
	rows.Close()
	sort.SliceStable(s.Convs, func(i, j int) bool {
		if s.Convs[i].top != s.Convs[j].top {
			return s.Convs[i].top > s.Convs[j].top
		}
		return s.Convs[i].LastModify > s.Convs[j].LastModify
	})

	// cid -> message table.
	s.cidTable = map[string]string{}
	for _, t := range s.tables {
		rows, err := s.DB.Query(`SELECT DISTINCT cid FROM "` + t + `"`)
		if err != nil {
			continue
		}
		for rows.Next() {
			var cid any
			rows.Scan(&cid)
			s.cidTable[sqlitex.Str(cid)] = t
		}
		rows.Close()
	}
	return nil
}

// isProjectGroup detects DingTalk "project" scene groups from the extension.
func isProjectGroup(ext string) bool {
	if ext == "" || !strings.Contains(ext, "project") {
		return false
	}
	var m map[string]any
	if json.Unmarshal([]byte(ext), &m) != nil {
		return false
	}
	if sqlitex.Str(m["sceneGroupCode"]) == "project" || sqlitex.Str(m["groupTagConfig"]) == "project" {
		return true
	}
	for _, t := range strings.Split(sqlitex.Str(m["tagCodes"]), ",") {
		if strings.TrimSpace(t) == "project" {
			return true
		}
	}
	return false
}

// UserName resolves a display name, preferring the group nickname in cid.
func (s *Snapshot) UserName(cid string, uid int64) string {
	if n := s.nicks[cid][uid]; n != "" {
		return n
	}
	if u, ok := s.Users[uid]; ok && u.Name() != "" {
		return u.Name()
	}
	if uid == 0 {
		return ""
	}
	return strconv.FormatInt(uid, 10)
}

// MyName is the account owner's display name.
func (s *Snapshot) MyName() string { return s.UserName("", s.MyUID) }

// ConvTitle returns a conversation's title, or the cid when unknown.
func (s *Snapshot) ConvTitle(cid string) string {
	if c := s.ConvByID[cid]; c != nil && c.Title != "" {
		return c.Title
	}
	return cid
}
