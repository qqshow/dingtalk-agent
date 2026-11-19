package store

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/qqshow/dingtalk-agent/internal/sqlitex"
)

func TestEventsAllDayAndStatus(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cal.db")
	db, err := sqlitex.Open(path, "")
	if err != nil {
		t.Fatal(err)
	}
	day := time.Date(2026, 11, 18, 0, 0, 0, 0, time.Local)
	utcMidnight := time.Date(2026, 11, 18, 0, 0, 0, 0, time.UTC).UnixMilli()
	at := func(h, m int) int64 {
		return day.Add(time.Duration(h)*time.Hour + time.Duration(m)*time.Minute).UnixMilli()
	}
	stmts := []string{
		`CREATE TABLE calendar_folder (folder_id TEXT, folder_name TEXT)`,
		`CREATE TABLE calendar (calendar_id INTEGER, organizers TEXT, sender_id INTEGER, c_role INTEGER,
			c_self_status INTEGER, c_status INTEGER, c_is_deleted INTEGER, source INTEGER)`,
		`CREATE TABLE calendar_event (event_id INTEGER, calendar_id INTEGER, unique_id TEXT, folder_id TEXT, subject TEXT,
			start_time INTEGER, end_time INTEGER, all_day_event INTEGER, location TEXT, locationModel TEXT, meetingRoom TEXT,
			meetingRooms TEXT, has_videoConference INTEGER, n_comment TEXT, url TEXT)`,
		`INSERT INTO calendar_folder VALUES ('1','我的日历'),('1_task','我的待办')`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatal(err)
		}
	}
	ins := func(id int, folder, subj string, st, en int64, allDay, self, status int, rooms string) {
		src := 88
		if subj == "请假中" {
			src = sourceLeave
		}
		db.Exec(`INSERT INTO calendar VALUES (?, '["7"]', 7, 1, ?, ?, 0, ?)`, id, self, status, src)
		db.Exec(`INSERT INTO calendar_event VALUES (?,?,?,?,?,?,?,?, '', '{"place":""}', '', ?, 0, '', '')`,
			id, id, "u", folder, subj, st, en, allDay, rooms)
	}
	ins(1, "1", "评审会", at(10, 0), at(11, 30), 0, 2, 0, `[{"roomName":"908会议室"}]`)
	ins(2, "1", "已取消的会", at(14, 0), at(15, 0), 0, 1, -1, `[]`)
	ins(3, "1_task", "报销待办", utcMidnight, utcMidnight+dayMS, 1, 2, 0, `[]`)
	ins(4, "1_task", "已完成待办", utcMidnight, utcMidnight+dayMS, 1, 2, 2, `[]`)
	ins(5, "1", "请假中", at(16, 0), at(18, 0), 0, 2, 0, `[]`)
	db.Close()

	s := &Snapshot{MyUID: 7, Users: map[int64]User{7: {UID: 7, RealName: "我"}}}
	s.openCalendar(path)
	if s.Cal == nil {
		t.Fatal("calendar not opened")
	}
	defer s.Cal.Close()

	evs, err := s.Events(EventQuery{From: day.UnixMilli(), To: day.AddDate(0, 0, 1).UnixMilli()})
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) != 3 {
		t.Fatalf("want 3 open events, got %d: %+v", len(evs), evs)
	}
	if evs[2].Kind != "leave" {
		t.Errorf("leave entry kind = %q", evs[2].Kind)
	}
	todo, meeting := evs[0], evs[1] // all-day starts at local midnight → first
	if todo.Kind != "approval_todo" || !todo.AllDay || todo.Start != day.UnixMilli() || todo.Status != "open" {
		t.Errorf("all-day todo mapped wrong: %+v", todo)
	}
	if meeting.Subject != "评审会" || meeting.Minutes != 90 || meeting.MeetingRoom != "908会议室" ||
		!meeting.IOrganized || meeting.MyResponse != "accepted" || meeting.Calendar != "我的日历" {
		t.Errorf("meeting parsed wrong: %+v", meeting)
	}

	// The all-day entry (stored 08:00→08:00 next day in UTC+8) must not leak into the next local day.
	next, _ := s.Events(EventQuery{From: day.AddDate(0, 0, 1).UnixMilli(), To: day.AddDate(0, 0, 2).UnixMilli()})
	if len(next) != 0 {
		t.Errorf("all-day event leaked into next day: %+v", next)
	}

	all, _ := s.Events(EventQuery{From: day.UnixMilli(), To: day.AddDate(0, 0, 1).UnixMilli(), IncludeClosed: true})
	if len(all) != 5 {
		t.Errorf("IncludeClosed: want 5, got %d", len(all))
	}
}
