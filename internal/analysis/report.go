package analysis

import (
	"sort"
	"strconv"
	"time"

	"github.com/qqshow/dingtalk-agent/internal/store"
	"github.com/qqshow/dingtalk-agent/internal/textutil"
)

type Report struct {
	Period      PeriodInfo      `json:"period"`
	Me          Person          `json:"me"`
	DataAsOf    string          `json:"data_as_of"`
	Totals      Totals          `json:"totals"`
	People      PeopleSection   `json:"people"`
	Groups      GroupSection    `json:"groups"`
	Projects    GroupSection    `json:"projects"`
	MentionsMe  []MsgRef        `json:"mentions_of_me"`
	Awaiting    []Pending       `json:"awaiting_my_reply"`
	MyHourly    [24]int         `json:"my_hourly_activity"`
	Calendar    CalendarSection `json:"calendar"`
	Definitions Definitions     `json:"definitions"`
}

type PeriodInfo struct {
	Label   string `json:"label"`
	DateKey string `json:"date_key"`
	From    string `json:"from"`
	To      string `json:"to"`
}

type Person struct {
	UID  int64  `json:"uid"`
	Name string `json:"name"`
}

type Totals struct {
	Messages            int `json:"messages"`
	SentByMe            int `json:"sent_by_me"`
	Received            int `json:"received"`
	ActiveConversations int `json:"active_conversations"`
	ActiveSingleChats   int `json:"active_single_chats"`
	ActiveGroups        int `json:"active_groups"`
}

type PersonStat struct {
	UID            int64    `json:"uid"`
	Name           string   `json:"name"`
	DirectSent     int      `json:"direct_sent_by_me"`
	DirectReceived int      `json:"direct_received"`
	MentionedMe    int      `json:"mentioned_me"`
	MentionedByMe  int      `json:"mentioned_by_me"`
	Score          int      `json:"score"`
	SingleCID      string   `json:"single_cid,omitempty"`
	Groups         []string `json:"groups,omitempty"`
}

type PeopleSection struct {
	Count int          `json:"count"`
	List  []PersonStat `json:"list"`
}

type GroupStat struct {
	CID          string   `json:"cid"`
	Title        string   `json:"title"`
	IsProject    bool     `json:"is_project"`
	Messages     int      `json:"messages"`
	MyMessages   int      `json:"my_messages"`
	MentionsMe   int      `json:"mentions_me"`
	Participants int      `json:"participants"`
	TopSpeakers  []string `json:"top_speakers"`
	Involved     bool     `json:"involved"`
	LastAt       string   `json:"last_at"`
}

type GroupSection struct {
	ActiveCount   int         `json:"active_count"`
	InvolvedCount int         `json:"involved_count"`
	List          []GroupStat `json:"list"`
}

type MsgRef struct {
	CID       string `json:"cid"`
	ConvTitle string `json:"conv_title"`
	MsgID     int64  `json:"msg_id"`
	Sender    string `json:"sender"`
	Time      string `json:"time"`
	Text      string `json:"text"`
	Replied   bool   `json:"replied"`
	RepliedAt string `json:"replied_at,omitempty"`
}

type Pending struct {
	CID        string `json:"cid"`
	ConvTitle  string `json:"conv_title"`
	Type       string `json:"type"`
	Reason     string `json:"reason"`
	LastSender string `json:"last_sender"`
	LastAt     string `json:"last_at"`
	LastText   string `json:"last_text"`
	Unanswered int    `json:"unanswered_count"`
	MsgID      int64  `json:"msg_id"`
}

type Definitions struct {
	People   string `json:"people"`
	Involved string `json:"involved_group"`
	Project  string `json:"project_group"`
	Awaiting string `json:"awaiting_my_reply"`
	Calendar string `json:"calendar"`
}

var defs = Definitions{
	People:   "单聊中有往来的人 + 群聊中@我或被我@的人（不含机器人/系统账号）",
	Involved: "我在群里发过言，或被单独@（不含@所有人）",
	Project:  "钉钉标记为项目群的群（sceneGroupCode/tagCodes=project）",
	Awaiting: "单聊最后一条消息来自对方且我之后未回复；或群里单独@我后我未在该群发言",
	Calendar: "来自钉钉本地日历缓存：meetings=未取消的会议；next_day=下一天的日程；pending_approvals=近30天未完成的审批/流程待办。本地不含完整参会人名单",
}

// Input bundles the messages already loaded for a period so Report and
// Digest can share one query.
type Input struct {
	Snap   *store.Snapshot
	Period Period
	Msgs   []*store.Message
}

func Load(s *store.Snapshot, p Period) (*Input, error) {
	msgs, err := s.Range(store.RangeQuery{From: p.FromMS(), To: p.ToMS()})
	if err != nil {
		return nil, err
	}
	return &Input{Snap: s, Period: p, Msgs: msgs}, nil
}

func isSelfChat(c *store.Conversation, me int64) bool {
	return c != nil && c.Type == "single" && c.OtherUID == strconv.FormatInt(me, 10)
}

// BuildReport computes the structured daily report.
func BuildReport(in *Input) *Report {
	s, me := in.Snap, in.Snap.MyUID
	r := &Report{
		Period: PeriodInfo{Label: in.Period.Label, DateKey: in.Period.DateKey(),
			From: in.Period.From.Format(time.DateTime), To: in.Period.To.Format(time.DateTime)},
		Me:          Person{UID: me, Name: s.MyName()},
		DataAsOf:    s.Created.Format(time.DateTime),
		Definitions: defs,
		MentionsMe:  []MsgRef{},
		Awaiting:    []Pending{},
	}

	people := map[int64]*PersonStat{}
	person := func(uid int64) *PersonStat {
		p := people[uid]
		if p == nil {
			p = &PersonStat{UID: uid, Name: s.UserName("", uid)}
			people[uid] = p
		}
		return p
	}
	type gacc struct {
		stat     *GroupStat
		speakers map[int64]int
	}
	groups := map[string]*gacc{}
	singles := map[string][]*store.Message{}
	active := map[string]bool{}
	var mentionMsgs []*store.Message

	for _, m := range in.Msgs {
		c := s.ConvByID[m.CID]
		r.Totals.Messages++
		active[m.CID] = true
		if m.IsSystem {
			continue
		}
		if m.IsMine {
			r.Totals.SentByMe++
			r.MyHourly[time.UnixMilli(m.CreatedAt).Hour()]++
		} else {
			r.Totals.Received++
		}
		if c == nil {
			continue
		}
		if c.Type == "single" {
			if isSelfChat(c, me) {
				continue
			}
			singles[m.CID] = append(singles[m.CID], m)
			continue
		}
		g := groups[m.CID]
		if g == nil {
			g = &gacc{stat: &GroupStat{CID: c.CID, Title: c.Title, IsProject: c.IsProject}, speakers: map[int64]int{}}
			groups[m.CID] = g
		}
		g.stat.Messages++
		g.stat.LastAt = m.CreatedAtStr
		if m.CreatorType != 2 && m.SenderID != 0 {
			g.speakers[m.SenderID]++
		}
		if m.IsMine {
			g.stat.MyMessages++
			g.stat.Involved = true
			for uidStr := range m.AtIDs {
				if uid, err := strconv.ParseInt(uidStr, 10, 64); err == nil && uid != me && !s.IsBot(uid) {
					p := person(uid)
					p.MentionedByMe++
					p.Groups = appendUniq(p.Groups, c.Title)
				}
			}
		} else if m.MentionsMe {
			g.stat.MentionsMe++
			g.stat.Involved = true
			mentionMsgs = append(mentionMsgs, m)
			if !s.IsBot(m.SenderID) && m.SenderID != 0 {
				p := person(m.SenderID)
				p.MentionedMe++
				p.Groups = appendUniq(p.Groups, c.Title)
			}
		}
	}

	// Single chats → direct interactions + awaiting reply.
	for cid, msgs := range singles {
		c := s.ConvByID[cid]
		other, _ := strconv.ParseInt(c.OtherUID, 10, 64)
		bot := s.IsBot(other) || c.IsNotification
		if !bot {
			p := person(other)
			p.SingleCID = cid
			for _, m := range msgs {
				if m.IsMine {
					p.DirectSent++
				} else {
					p.DirectReceived++
				}
			}
		}
		if bot {
			continue
		}
		last := s.LastMessage(cid)
		if last != nil && !last.IsMine && last.CreatedAt >= in.Period.FromMS() {
			unanswered := 0
			for i := len(msgs) - 1; i >= 0 && !msgs[i].IsMine; i-- {
				unanswered++
			}
			r.Awaiting = append(r.Awaiting, Pending{
				CID: cid, ConvTitle: c.Title, Type: "single", Reason: "单聊未回复",
				LastSender: last.SenderName, LastAt: last.CreatedAtStr, LastText: textutil.Snippet(last.AgentText(), 200),
				Unanswered: max(unanswered, 1), MsgID: last.ID,
			})
		}
	}

	// Mentions of me and whether I replied in that group afterwards.
	var mcids []string
	for _, m := range mentionMsgs {
		mcids = append(mcids, m.CID)
	}
	lastMine := s.LastMineAfter(uniq(mcids), in.Period.FromMS())
	pendingGroup := map[string]*Pending{}
	for _, m := range mentionMsgs {
		ref := MsgRef{CID: m.CID, ConvTitle: s.ConvTitle(m.CID), MsgID: m.ID, Sender: m.SenderName,
			Time: m.CreatedAtStr, Text: textutil.Snippet(m.AgentText(), 300)}
		if t, ok := lastMine[m.CID]; ok && t > m.CreatedAt {
			ref.Replied, ref.RepliedAt = true, store.FormatTS(t)
		} else {
			pg := pendingGroup[m.CID]
			if pg == nil {
				pg = &Pending{CID: m.CID, ConvTitle: ref.ConvTitle, Type: "group", Reason: "群里@我未回复"}
				pendingGroup[m.CID] = pg
			}
			pg.Unanswered++
			pg.LastSender, pg.LastAt, pg.LastText, pg.MsgID = m.SenderName, m.CreatedAtStr, ref.Text, m.ID
		}
		r.MentionsMe = append(r.MentionsMe, ref)
	}
	for _, pg := range pendingGroup {
		r.Awaiting = append(r.Awaiting, *pg)
	}
	sort.Slice(r.Awaiting, func(i, j int) bool { return r.Awaiting[i].LastAt > r.Awaiting[j].LastAt })

	// Totals.
	r.Totals.ActiveConversations = len(active)
	for cid := range active {
		if c := s.ConvByID[cid]; c != nil {
			if c.Type == "single" {
				r.Totals.ActiveSingleChats++
			} else {
				r.Totals.ActiveGroups++
			}
		}
	}

	// People.
	for uid, p := range people {
		if uid == me || uid == 0 {
			continue
		}
		p.Score = p.DirectSent + p.DirectReceived + 3*(p.MentionedMe+p.MentionedByMe)
		r.People.List = append(r.People.List, *p)
	}
	sort.Slice(r.People.List, func(i, j int) bool { return r.People.List[i].Score > r.People.List[j].Score })
	r.People.Count = len(r.People.List)
	if r.People.List == nil {
		r.People.List = []PersonStat{}
	}

	// Groups / projects.
	r.Groups.List, r.Projects.List = []GroupStat{}, []GroupStat{}
	for _, g := range groups {
		g.stat.Participants = len(g.speakers)
		g.stat.TopSpeakers = topSpeakers(s, g.stat.CID, g.speakers, 3)
		r.Groups.List = append(r.Groups.List, *g.stat)
	}
	sort.Slice(r.Groups.List, func(i, j int) bool {
		a, b := r.Groups.List[i], r.Groups.List[j]
		if a.Involved != b.Involved {
			return a.Involved
		}
		return a.Messages > b.Messages
	})
	for _, g := range r.Groups.List {
		r.Groups.ActiveCount++
		if g.Involved {
			r.Groups.InvolvedCount++
		}
		if g.IsProject {
			r.Projects.List = append(r.Projects.List, g)
			r.Projects.ActiveCount++
			if g.Involved {
				r.Projects.InvolvedCount++
			}
		}
	}
	r.Calendar = BuildCalendar(s, in.Period)
	return r
}

func topSpeakers(s *store.Snapshot, cid string, sp map[int64]int, n int) []string {
	type kv struct {
		uid int64
		c   int
	}
	var l []kv
	for u, c := range sp {
		l = append(l, kv{u, c})
	}
	sort.Slice(l, func(i, j int) bool { return l[i].c > l[j].c })
	out := []string{}
	for i := 0; i < len(l) && i < n; i++ {
		out = append(out, s.UserName(cid, l[i].uid)+"("+strconv.Itoa(l[i].c)+")")
	}
	return out
}

func appendUniq(l []string, v string) []string {
	for _, x := range l {
		if x == v {
			return l
		}
	}
	return append(l, v)
}

func uniq(l []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, v := range l {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}
