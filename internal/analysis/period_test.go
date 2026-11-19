package analysis

import (
	"testing"
	"time"
)

func TestParsePeriod(t *testing.T) {
	today := dayStart(time.Now())
	cases := []struct {
		date, from, to string
		wantFrom       time.Time
		wantDays       int
		wantKey        string
	}{
		{"", "", "", today, 1, today.Format("2006-01-02")},
		{"yesterday", "", "", today.AddDate(0, 0, -1), 1, today.AddDate(0, 0, -1).Format("2006-01-02")},
		{"-3", "", "", today.AddDate(0, 0, -3), 1, ""},
		{"2026-11-02", "", "", time.Date(2026, 11, 2, 0, 0, 0, 0, time.Local), 1, "2026-11-02"},
		{"", "2026-11-10", "2026-11-16", time.Date(2026, 11, 10, 0, 0, 0, 0, time.Local), 7, "2026-11-10~2026-11-16"},
		{"", "2026-11-16", "2026-11-10", time.Date(2026, 11, 10, 0, 0, 0, 0, time.Local), 7, ""}, // swapped
	}
	for _, c := range cases {
		p, err := ParsePeriod(c.date, c.from, c.to)
		if err != nil {
			t.Fatalf("%+v: %v", c, err)
		}
		if !p.From.Equal(c.wantFrom) {
			t.Errorf("%+v: from = %v, want %v", c, p.From, c.wantFrom)
		}
		if days := p.To.Sub(p.From).Round(time.Hour) / (24 * time.Hour); int(days) != c.wantDays {
			t.Errorf("%+v: days = %d, want %d", c, days, c.wantDays)
		}
		if c.wantKey != "" && p.DateKey() != c.wantKey {
			t.Errorf("%+v: key = %q, want %q", c, p.DateKey(), c.wantKey)
		}
	}
	if _, err := ParsePeriod("2026-13-01", "", ""); err == nil {
		t.Error("expected error for invalid date")
	}
}
