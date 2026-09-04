package web

import (
	"testing"
	"time"
)

func TestUkDateShort(t *testing.T) {
	now, _ := time.ParseInLocation("2006-01-02", "2026-09-03", time.UTC)
	cases := map[string]string{"2026-09-03": "сьогодні", "2026-09-04": "завтра", "2026-09-05": "сб, 5 вересня", "2026-09-07": "пн, 7 вересня"}
	for in, want := range cases {
		if got := ukDateShort(in, now); got != want {
			t.Errorf("%s: got %q want %q", in, got, want)
		}
	}
}
