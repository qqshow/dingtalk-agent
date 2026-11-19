// Package analysis turns raw messages into the structured daily facts an
// agent needs: activity totals, who I interacted with, which groups/projects
// I was involved in, mentions of me and conversations awaiting my reply.
package analysis

import (
	"fmt"
	"strings"
	"time"
)

// Period is a half-open local-time interval [From, To).
type Period struct {
	From  time.Time
	To    time.Time
	Label string
}

func (p Period) FromMS() int64 { return p.From.UnixMilli() }
func (p Period) ToMS() int64   { return p.To.UnixMilli() }

var weekdays = []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}

func dayStart(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.Local)
}

// ParseDay accepts "", "today", "yesterday", "-N" (N days ago) or YYYY-MM-DD.
func ParseDay(s string) (time.Time, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	now := time.Now()
	switch s {
	case "", "today", "今天":
		return dayStart(now), nil
	case "yesterday", "昨天":
		return dayStart(now).AddDate(0, 0, -1), nil
	}
	var n int
	if _, err := fmt.Sscanf(s, "-%d", &n); err == nil && n >= 0 && strings.HasPrefix(s, "-") {
		return dayStart(now).AddDate(0, 0, -n), nil
	}
	t, err := time.ParseInLocation("2006-01-02", s, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date %q (use YYYY-MM-DD, today, yesterday)", s)
	}
	return t, nil
}

// ParsePeriod builds a period from either a single date, or from/to dates
// (inclusive). Empty input means today.
func ParsePeriod(date, from, to string) (Period, error) {
	if from != "" || to != "" {
		f, err := ParseDay(from)
		if err != nil {
			return Period{}, err
		}
		t := dayStart(time.Now())
		if to != "" {
			if t, err = ParseDay(to); err != nil {
				return Period{}, err
			}
		}
		if t.Before(f) {
			f, t = t, f
		}
		return Period{From: f, To: t.AddDate(0, 0, 1),
			Label: f.Format("2006-01-02") + " ~ " + t.Format("2006-01-02")}, nil
	}
	d, err := ParseDay(date)
	if err != nil {
		return Period{}, err
	}
	return Period{From: d, To: d.AddDate(0, 0, 1),
		Label: d.Format("2006-01-02") + " " + weekdays[d.Weekday()]}, nil
}

// DateKey is the canonical key for per-day records (summary/todos).
func (p Period) DateKey() string {
	if p.To.Sub(p.From) <= 25*time.Hour {
		return p.From.Format("2006-01-02")
	}
	return p.From.Format("2006-01-02") + "~" + p.To.AddDate(0, 0, -1).Format("2006-01-02")
}
