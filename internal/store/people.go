package store

import (
	"sort"
	"strconv"
	"strings"
)

// Person is a contact found by SearchPeople.
type Person struct {
	UID  int64  `json:"uid"`
	Name string `json:"name"`
	CID  string `json:"cid,omitempty"` // our single chat with them, if any
}

// SingleCIDFor returns the single-chat conversation with uid, or "".
func (s *Snapshot) SingleCIDFor(uid int64) string {
	id := strconv.FormatInt(uid, 10)
	for _, c := range s.Convs {
		if c.Type == "single" && c.OtherUID == id {
			return c.CID
		}
	}
	return ""
}

// SearchPeople finds real people (not bots, not me) whose name or nickname
// contains q. People we have a single chat with come first, then names that
// start with q.
func (s *Snapshot) SearchPeople(q string, limit int) []Person {
	return s.searchPeople(q, limit, false)
}

// SearchPeopleWithoutChat finds matching contacts without a single chat.
func (s *Snapshot) SearchPeopleWithoutChat(q string, limit int) []Person {
	return s.searchPeople(q, limit, true)
}

func (s *Snapshot) searchPeople(q string, limit int, withoutChat bool) []Person {
	q = strings.ToLower(strings.TrimSpace(q))
	if limit <= 0 {
		limit = 20
	}
	single := map[int64]*Conversation{}
	for _, c := range s.Convs {
		if c.Type == "single" {
			if id, err := strconv.ParseInt(c.OtherUID, 10, 64); err == nil {
				single[id] = c
			}
		}
	}
	type hit struct {
		p      Person
		chat   bool
		prefix bool
		last   int64
	}
	var hits []hit
	for uid, u := range s.Users {
		if uid == s.MyUID || u.IsBot() || (withoutChat && single[uid] != nil && single[uid].CID != "") {
			continue
		}
		name := u.Name()
		if name == "" {
			continue
		}
		ln, lnick := strings.ToLower(name), strings.ToLower(u.Nick)
		if q != "" && !strings.Contains(ln, q) && !strings.Contains(lnick, q) {
			continue
		}
		h := hit{p: Person{UID: uid, Name: name}, prefix: q != "" && strings.HasPrefix(ln, q)}
		if c := single[uid]; c != nil {
			h.p.CID, h.chat, h.last = c.CID, true, c.LastModify
		}
		hits = append(hits, h)
	}
	sort.Slice(hits, func(i, j int) bool {
		a, b := hits[i], hits[j]
		if a.chat != b.chat {
			return a.chat
		}
		if a.prefix != b.prefix {
			return a.prefix
		}
		if a.last != b.last {
			return a.last > b.last
		}
		if len([]rune(a.p.Name)) != len([]rune(b.p.Name)) {
			return len([]rune(a.p.Name)) < len([]rune(b.p.Name))
		}
		return a.p.Name < b.p.Name
	})
	out := make([]Person, 0, min(limit, len(hits)))
	for i := 0; i < len(hits) && i < limit; i++ {
		out = append(out, hits[i].p)
	}
	return out
}
