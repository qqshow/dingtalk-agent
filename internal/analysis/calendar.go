package analysis

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/qqshow/dingtalk-agent/internal/store"
)

// CalendarSection is the calendar part of the daily report.
type CalendarSection struct {
	Available        bool          `json:"available"`
	Meetings         []store.Event `json:"meetings"` // meetings in the period (not cancelled)
	MeetingCount     int           `json:"meeting_count"`
	MeetingMinutes   int           `json:"meeting_minutes"`
	Cancelled        []store.Event `json:"cancelled_meetings"`
	OtherEntries     []store.Event `json:"other_entries"` // AI minutes, Teambition tasks in the period
	NextDayLabel     string        `json:"next_day_label,omitempty"`
	NextDay          []store.Event `json:"next_day"`          // upcoming entries on the next day
	PendingApprovals []store.Event `json:"pending_approvals"` // open approval todos (last 30 days)
}

// BuildCalendar loads the calendar facts for the period.
func BuildCalendar(s *store.Snapshot, p Period) CalendarSection {
	c := CalendarSection{Available: s.Cal != nil, Meetings: []store.Event{}, Cancelled: []store.Event{},
		OtherEntries: []store.Event{}, NextDay: []store.Event{}, PendingApprovals: []store.Event{}}
	if s.Cal == nil {
		return c
	}
	evs, _ := s.Events(store.EventQuery{From: p.FromMS(), To: p.ToMS(), IncludeClosed: true})
	for _, e := range evs {
		switch {
		case e.Kind == "meeting" && e.Status == "cancelled":
			c.Cancelled = append(c.Cancelled, e)
		case e.Kind == "meeting":
			c.Meetings = append(c.Meetings, e)
			c.MeetingMinutes += e.Minutes
		case e.Kind == "approval_todo":
			// handled by PendingApprovals below
		default:
			c.OtherEntries = append(c.OtherEntries, e)
		}
	}
	c.MeetingCount = len(c.Meetings)

	// Next day (only meaningful for single-day periods).
	if p.To.Sub(p.From) <= 25*time.Hour {
		nd := p.To
		c.NextDayLabel = nd.Format("2006-01-02") + " " + weekdays[nd.Weekday()]
		c.NextDay, _ = s.Events(store.EventQuery{From: nd.UnixMilli(), To: nd.AddDate(0, 0, 1).UnixMilli()})
	}

	// Open approval todos created in the 30 days up to the period end.
	c.PendingApprovals, _ = s.Events(store.EventQuery{From: p.To.AddDate(0, 0, -30).UnixMilli(), To: p.ToMS(),
		Kinds: []string{"approval_todo"}})
	sort.Slice(c.PendingApprovals, func(i, j int) bool { return c.PendingApprovals[i].Start > c.PendingApprovals[j].Start })
	return c
}

func eventLine(e store.Event, withDate bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "- %s %s", e.TimeRange(withDate), e.Subject)
	var meta []string
	if e.MeetingRoom != "" {
		meta = append(meta, e.MeetingRoom)
	} else if e.Location != "" {
		meta = append(meta, e.Location)
	}
	if e.HasVideo {
		meta = append(meta, "视频会议")
	}
	switch e.Kind {
	case "meeting":
		if e.IOrganized {
			meta = append(meta, "我组织")
		} else if e.Organizer != "" {
			meta = append(meta, "组织者 "+e.Organizer)
		}
		if e.MyResponse == "no_response" && !e.IOrganized {
			meta = append(meta, "我未回复邀请")
		}
	case "leave":
		meta = append(meta, "请假")
	case "ai_minutes":
		meta = append(meta, "AI听记")
	case "task":
		meta = append(meta, "任务")
	}
	if len(meta) > 0 {
		b.WriteString("（" + strings.Join(meta, "，") + "）")
	}
	if e.Comment != "" {
		b.WriteString("\n  备注：" + snippet(e.Comment, 300))
	}
	b.WriteString("\n")
	return b.String()
}

func calendarMarkdown(c CalendarSection, multiDay bool) string {
	if !c.Available {
		return ""
	}
	var b strings.Builder
	title := "今日日程"
	if multiDay {
		title = "期间日程"
	}
	fmt.Fprintf(&b, "\n## %s（会议 %d 个，共 %d 分钟）\n", title, c.MeetingCount, c.MeetingMinutes)
	if len(c.Meetings) == 0 && len(c.OtherEntries) == 0 {
		b.WriteString("- 无\n")
	}
	entries := append(append([]store.Event{}, c.Meetings...), c.OtherEntries...)
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].Start < entries[j].Start })
	for _, e := range entries {
		b.WriteString(eventLine(e, multiDay))
	}
	for _, e := range c.Cancelled {
		fmt.Fprintf(&b, "- ~~%s %s~~（已取消）\n", e.TimeRange(multiDay), e.Subject)
	}
	if c.NextDayLabel != "" {
		fmt.Fprintf(&b, "\n## 明日日程 %s\n", c.NextDayLabel)
		if len(c.NextDay) == 0 {
			b.WriteString("- 无\n")
		}
		for _, e := range c.NextDay {
			b.WriteString(eventLine(e, false))
		}
	}
	if len(c.PendingApprovals) > 0 {
		b.WriteString("\n## 未处理的审批/流程待办\n")
		for _, e := range c.PendingApprovals {
			st := ""
			if e.Status == "in_progress" {
				st = "（处理中）"
			}
			fmt.Fprintf(&b, "- %s %s%s\n", time.UnixMilli(e.Start).Format("01-02"), e.Subject, st)
		}
	}
	return b.String()
}
