package store

import (
	"fmt"
	"sort"
	"strings"
)

// ConversationList filters cached conversations; keyword matches title or cid.
func (s *Snapshot) ConversationList(keyword, typ string, limit, offset int) (int, []*Conversation) {
	kw := strings.ToLower(strings.TrimSpace(keyword))
	var hits []*Conversation
	for _, c := range s.Convs {
		if typ != "" && c.Type != typ {
			continue
		}
		if kw != "" && !strings.Contains(strings.ToLower(c.Title), kw) && !strings.Contains(c.CID, kw) {
			continue
		}
		hits = append(hits, c)
	}
	total := len(hits)
	if offset >= total {
		return total, []*Conversation{}
	}
	end := min(offset+limit, total)
	return total, hits[offset:end]
}

// Messages returns one page of a conversation, newest page first
// (offset=0 is the most recent `limit` messages), ordered oldest→newest.
func (s *Snapshot) Messages(cid string, limit, offset int, since, until int64) (int, []*Message, error) {
	table := s.cidTable[cid]
	if table == "" {
		return 0, []*Message{}, nil
	}
	where := `WHERE cid = ? AND recallStatus = 0`
	args := []any{cid}
	if since > 0 {
		where += ` AND createdAt > ?`
		args = append(args, since)
	}
	if until > 0 {
		where += ` AND createdAt <= ?`
		args = append(args, until)
	}
	var total int
	if err := s.DB.QueryRow(`SELECT COUNT(*) FROM "`+table+`" `+where, args...).Scan(&total); err != nil {
		return 0, nil, err
	}
	actual := max(0, total-limit-offset)
	n := limit
	if total-offset < limit {
		n = max(0, total-offset)
	}
	rows, err := s.DB.Query(`SELECT `+msgCols+` FROM "`+table+`" `+where+
		` ORDER BY createdAt ASC LIMIT ? OFFSET ?`, append(args, n, actual)...)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()
	msgs, err := s.scanMessages(rows, true)
	if msgs == nil {
		msgs = []*Message{}
	}
	return total, msgs, err
}

// RangeQuery selects messages across all tables.
type RangeQuery struct {
	From, To   int64    // ms, [From, To)
	CIDs       []string // optional filter
	Keyword    string   // optional LIKE filter on raw content
	SenderID   int64    // optional
	Desc       bool
	Limit      int // 0 = unlimited
	Offset     int
	WithImages bool
}

func (s *Snapshot) tablesFor(cids []string) []string {
	if len(cids) == 0 {
		return s.tables
	}
	seen := map[string]bool{}
	var out []string
	for _, c := range cids {
		if t := s.cidTable[c]; t != "" && !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	sort.Strings(out)
	return out
}

// Range runs q and returns matching non-recalled messages.
func (s *Snapshot) Range(q RangeQuery) ([]*Message, error) {
	tables := s.tablesFor(q.CIDs)
	if len(tables) == 0 {
		return []*Message{}, nil
	}
	var conds []string
	var condArgs []any
	conds = append(conds, "recallStatus = 0")
	if q.From > 0 {
		conds = append(conds, "createdAt >= ?")
		condArgs = append(condArgs, q.From)
	}
	if q.To > 0 {
		conds = append(conds, "createdAt < ?")
		condArgs = append(condArgs, q.To)
	}
	if len(q.CIDs) > 0 {
		conds = append(conds, "cid IN ("+strings.TrimSuffix(strings.Repeat("?,", len(q.CIDs)), ",")+")")
		for _, c := range q.CIDs {
			condArgs = append(condArgs, c)
		}
	}
	if q.Keyword != "" {
		conds = append(conds, "content LIKE ?")
		condArgs = append(condArgs, "%"+q.Keyword+"%")
	}
	if q.SenderID != 0 {
		conds = append(conds, "senderId = ?")
		condArgs = append(condArgs, q.SenderID)
	}
	where := " WHERE " + strings.Join(conds, " AND ")

	var parts []string
	var args []any
	for _, t := range tables {
		parts = append(parts, `SELECT `+msgCols+` FROM "`+t+`"`+where)
		args = append(args, condArgs...)
	}
	sqlText := strings.Join(parts, " UNION ALL ")
	order := "ASC"
	if q.Desc {
		order = "DESC"
	}
	sqlText = "SELECT * FROM (" + sqlText + ") ORDER BY createdAt " + order
	if q.Limit > 0 {
		sqlText += fmt.Sprintf(" LIMIT %d OFFSET %d", q.Limit, q.Offset)
	}
	rows, err := s.DB.Query(sqlText, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	msgs, err := s.scanMessages(rows, q.WithImages)
	if msgs == nil {
		msgs = []*Message{}
	}
	return msgs, err
}

// LastMineAfter returns, per cid, the time of my latest message at/after `after`.
func (s *Snapshot) LastMineAfter(cids []string, after int64) map[string]int64 {
	out := map[string]int64{}
	for _, cid := range cids {
		t := s.cidTable[cid]
		if t == "" {
			continue
		}
		var ts any
		s.DB.QueryRow(`SELECT MAX(createdAt) FROM "`+t+`" WHERE cid = ? AND senderId = ? AND createdAt >= ?`,
			cid, s.MyUID, after).Scan(&ts)
		if v, ok := ts.(int64); ok && v > 0 {
			out[cid] = v
		}
	}
	return out
}

// LastMessage returns the latest non-recalled message of a conversation.
func (s *Snapshot) LastMessage(cid string) *Message {
	t := s.cidTable[cid]
	if t == "" {
		return nil
	}
	rows, err := s.DB.Query(`SELECT `+msgCols+` FROM "`+t+`" WHERE cid = ? AND recallStatus = 0 ORDER BY createdAt DESC LIMIT 1`, cid)
	if err != nil {
		return nil
	}
	defer rows.Close()
	msgs, _ := s.scanMessages(rows, false)
	if len(msgs) == 0 {
		return nil
	}
	return msgs[0]
}

type Stats struct {
	TotalConversations int    `json:"total_conversations"`
	TotalMessages      int64  `json:"total_messages"`
	SingleChats        int    `json:"single_chats"`
	GroupChats         int    `json:"group_chats"`
	ProjectGroups      int    `json:"project_groups"`
	TotalUsers         int    `json:"total_users"`
	LatestMessageTime  int64  `json:"latest_message_time"`
	LatestMessageStr   string `json:"latest_message_str"`
}

// LatestMessageTime returns the newest timestamp without counting the messages.
func (s *Snapshot) LatestMessageTime() int64 {
	var latest int64
	for _, t := range s.tables {
		var mx any
		s.DB.QueryRow(`SELECT MAX(createdAt) FROM "` + t + `"`).Scan(&mx)
		if v, ok := mx.(int64); ok && v > latest {
			latest = v
		}
	}
	return latest
}

func (s *Snapshot) Stats() Stats {
	st := Stats{TotalConversations: len(s.Convs), TotalUsers: len(s.Users)}
	for _, c := range s.Convs {
		if c.Type == "single" {
			st.SingleChats++
		} else {
			st.GroupChats++
			if c.IsProject {
				st.ProjectGroups++
			}
		}
	}
	for _, t := range s.tables {
		var n, mx any
		s.DB.QueryRow(`SELECT COUNT(*), MAX(createdAt) FROM "`+t+`"`).Scan(&n, &mx)
		if v, ok := n.(int64); ok {
			st.TotalMessages += v
		}
		if v, ok := mx.(int64); ok && v > st.LatestMessageTime {
			st.LatestMessageTime = v
		}
	}
	st.LatestMessageStr = FormatTS(st.LatestMessageTime)
	return st
}

// LastMsg is a compact preview of a conversation's latest message.
type LastMsg struct {
	Sender string `json:"sender"`
	Text   string `json:"text"`
	Time   int64  `json:"time"`
	Mine   bool   `json:"mine"`
}

// LastMessages returns the latest message preview of every conversation,
// computed once per snapshot (one grouped query per message table).
func (s *Snapshot) LastMessages() map[string]LastMsg {
	s.lastOnce.Do(func() {
		s.lastMsgs = map[string]LastMsg{}
		for _, t := range s.tables {
			rows, err := s.DB.Query(`SELECT ` + msgCols + ` FROM "` + t + `" m WHERE recallStatus = 0 AND createdAt = (
				SELECT MAX(createdAt) FROM "` + t + `" WHERE cid = m.cid AND recallStatus = 0)`)
			if err != nil {
				continue
			}
			msgs, _ := s.scanMessages(rows, false)
			rows.Close()
			for _, m := range msgs {
				if old, ok := s.lastMsgs[m.CID]; ok && old.Time >= m.CreatedAt {
					continue
				}
				text := m.AgentText()
				if r := []rune(text); len(r) > 80 {
					text = string(r[:80])
				}
				s.lastMsgs[m.CID] = LastMsg{Sender: m.SenderName, Text: strings.Join(strings.Fields(text), " "),
					Time: m.CreatedAt, Mine: m.IsMine}
			}
		}
	})
	return s.lastMsgs
}
