package web

import (
	"testing"
	"time"
)

func TestWeekStart(t *testing.T) {
	loc := time.UTC
	cases := map[string]string{
		"2026-09-03": "2026-08-31", // Thursday
		"2026-08-31": "2026-08-31", // Monday
		"2026-09-06": "2026-08-31", // Sunday belongs to the week before
	}
	for in, want := range cases {
		d, _ := time.ParseInLocation("2006-01-02", in, loc)
		if got := weekStart(d).Format("2006-01-02"); got != want {
			t.Errorf("%s: got %s want %s", in, got, want)
		}
	}
	m, _ := time.ParseInLocation("2006-01-02", "2026-08-31", loc)
	if got := weekLabel(m); got != "31 серпня – 6 вересня" {
		t.Errorf("label: %s", got)
	}
	m, _ = time.ParseInLocation("2006-01-02", "2026-09-07", loc)
	if got := weekLabel(m); got != "7–13 вересня" {
		t.Errorf("label: %s", got)
	}
}
