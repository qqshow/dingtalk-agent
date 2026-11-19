package store

import (
	"encoding/json"
	"log/slog"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/qqshow/dingtalk-agent/internal/sqlitex"
)

// Event is one calendar entry (meeting, approval todo, AI minutes, task).
type Event struct {
	EventID      int64  `json:"event_id"`
	UniqueID     string `json:"unique_id"`
	Kind         string `json:"kind"`     // meeting | approval_todo | ai_minutes | task | leave
	Calendar     string `json:"calendar"` // folder name, e.g. 我的日历 / 我的待办
	Subject      string `json:"subject"`
	Start        int64  `json:"start"`
	End          int64  `json:"end"`
	StartStr     string `json:"start_str"`
	EndStr       string `json:"end_str"`
	AllDay       bool   `json:"all_day"`
	Minutes      int    `json:"duration_minutes"`
	Location     string `json:"location,omitempty"`
	MeetingRoom  string `json:"meeting_room,omitempty"`
	HasVideo     bool   `json:"has_video_conference"`
	Comment      string `json:"comment,omitempty"` // description; sometimes an AI meeting summary
	Organizer    string `json:"organizer"`
	OrganizerUID int64  `json:"organizer_uid"`
	IOrganized   bool   `json:"i_organized"`
	MyResponse   string `json:"my_response"` // accepted | no_response | other
	Status       string `json:"status"`      // normal | cancelled | done | in_progress | open
	URL          string `json:"url,omitempty"`
}

func (s *Snapshot) openCalendar(path string) {
	if _, err := os.Stat(path); err != nil {
		return
	}
	db, err := sqlitex.Open(path, "mode=ro&immutable=1")
	if err != nil {
		slog.Warn("open calendar snapshot", "err", err)
		return
	}
	db.SetMaxOpenConns(4)
	s.folders = map[string]string{}
	rows, err := db.Query(`SELECT folder_id, folder_name FROM calendar_folder`)
	if err != nil {
		slog.Warn("calendar snapshot unreadable", "err", err)
		db.Close()
		return
	}
	for rows.Next() {
		var id, name any
		rows.Scan(&id, &name)
		s.folders[sqlitex.Str(id)] = sqlitex.Str(name)
	}
	rows.Close()
	s.Cal = db
}

// EventQuery selects calendar entries overlapping [From, To).
type EventQuery struct {
	From, To      int64
	Kinds         []string // empty = all
	IncludeClosed bool     // include cancelled meetings and finished todos
}

func eventKind(folderID string) string {
	switch {
	case strings.HasSuffix(folderID, "_task"):
		return "approval_todo"
	case strings.HasSuffix(folderID, "_shanji"):
		return "ai_minutes"
	case strings.HasSuffix(folderID, "_tb"):
		return "task"
	}
	return "meeting"
}

const (
	dayMS       = 24 * 60 * 60 * 1000
	sourceLeave = 113 // attendance / leave entries
)

// Events returns calendar entries ordered by start time. It returns an empty
// list (not an error) when the calendar is unavailable.
func (s *Snapshot) Events(q EventQuery) ([]Event, error) {
	out := []Event{}
	if s.Cal == nil {
		return out, nil
	}
	rows, err := s.Cal.Query(`SELECT e.event_id, e.unique_id, e.folder_id, e.subject, e.start_time, e.end_time,
			e.all_day_event, e.location, e.locationModel, e.meetingRoom, e.meetingRooms, e.has_videoConference,
			e.n_comment, e.url, c.organizers, c.sender_id, c.c_role, c.c_self_status, c.c_status, c.source
		FROM calendar_event e JOIN calendar c ON c.calendar_id = e.calendar_id
		WHERE c.c_is_deleted = 0 AND e.start_time < ? AND e.end_time > ?
		ORDER BY e.start_time, e.end_time`, q.To+dayMS, q.From-dayMS)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	kinds := map[string]bool{}
	for _, k := range q.Kinds {
		kinds[k] = true
	}
	for rows.Next() {
		var eid, uid, folder, subj, st, en, allDay, loc, locModel, room, rooms, video, comment, url, orgs, sender, role, self, status, source any
		if err := rows.Scan(&eid, &uid, &folder, &subj, &st, &en, &allDay, &loc, &locModel, &room, &rooms, &video,
			&comment, &url, &orgs, &sender, &role, &self, &status, &source); err != nil {
			return nil, err
		}
		fid := sqlitex.Str(folder)
		e := Event{
			EventID: sqlitex.Int(eid), UniqueID: sqlitex.Str(uid), Kind: eventKind(fid), Calendar: s.folders[fid],
			Subject: cleanText(sqlitex.Str(subj)), Start: sqlitex.Int(st), End: sqlitex.Int(en),
			AllDay: sqlitex.Int(allDay) != 0, HasVideo: sqlitex.Int(video) != 0,
			Comment: cleanText(strings.TrimSpace(sqlitex.Str(comment))), URL: sqlitex.Str(url),
		}
		if sqlitex.Int(source) == sourceLeave {
			e.Kind = "leave" // attendance status ("请假中"), not a meeting
		}
		if len(kinds) > 0 && !kinds[e.Kind] {
			continue
		}
		if e.AllDay {
			// All-day entries are stored as UTC midnights; map to local dates.
			days := max(1, (e.End-e.Start+dayMS/2)/dayMS)
			y, m, d := time.UnixMilli(e.Start).UTC().Date()
			start := time.Date(y, m, d, 0, 0, 0, 0, time.Local)
			e.Start, e.End = start.UnixMilli(), start.AddDate(0, 0, int(days)).UnixMilli()
		}
		if e.Start >= q.To || e.End <= q.From {
			continue
		}
		e.StartStr, e.EndStr = FormatTS(e.Start), FormatTS(e.End)
		if !e.AllDay && e.End > e.Start {
			e.Minutes = int((e.End - e.Start) / 60000)
		}

		e.Location = strings.TrimSpace(sqlitex.Str(loc))
		if e.Location == "" {
			e.Location = sqlitex.Str(asMap(sqlitex.Str(locModel))["place"])
		}
		var roomList []map[string]any
		if json.Unmarshal([]byte(sqlitex.Str(rooms)), &roomList) == nil {
			var names []string
			for _, r := range roomList {
				if n := sqlitex.Str(r["roomName"]); n != "" {
					names = append(names, n)
				}
			}
			e.MeetingRoom = strings.Join(names, "、")
		}
		if e.MeetingRoom == "" {
			e.MeetingRoom = sqlitex.Str(asMap(sqlitex.Str(room))["roomName"])
		}
		if e.Comment == e.MeetingRoom {
			e.Comment = ""
		}

		var orgIDs []string
		json.Unmarshal([]byte(sqlitex.Str(orgs)), &orgIDs)
		if len(orgIDs) > 0 {
			e.OrganizerUID, _ = strconv.ParseInt(orgIDs[0], 10, 64)
		} else {
			e.OrganizerUID = sqlitex.Int(sender)
		}
		e.Organizer = s.UserName("", e.OrganizerUID)
		e.IOrganized = e.OrganizerUID == s.MyUID && s.MyUID != 0

		switch sqlitex.Int(self) {
		case 2:
			e.MyResponse = "accepted"
		case 1:
			e.MyResponse = "no_response"
		default:
			e.MyResponse = "other"
		}
		cs := sqlitex.Int(status)
		if e.Kind == "meeting" || e.Kind == "leave" {
			e.Status = "normal"
			if cs == -1 {
				e.Status = "cancelled"
			}
		} else {
			switch cs {
			case 2:
				e.Status = "done"
			case 1:
				e.Status = "in_progress"
			default:
				e.Status = "open"
			}
		}
		if !q.IncludeClosed && (e.Status == "cancelled" || e.Status == "done") {
			continue
		}
		out = append(out, e)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Start < out[j].Start })
	return out, rows.Err()
}

// TimeRange formats an event's time for display ("10:00-12:00" / "全天").
func (e Event) TimeRange(withDate bool) string {
	if e.AllDay {
		if withDate {
			return time.UnixMilli(e.Start).Format("01-02") + " 全天"
		}
		return "全天"
	}
	f := "15:04"
	if withDate {
		f = "01-02 15:04"
	}
	return time.UnixMilli(e.Start).Format(f) + "-" + time.UnixMilli(e.End).Format("15:04")
}
