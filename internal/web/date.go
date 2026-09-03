package web

import (
	"fmt"
	"time"
)

var ukWeekdays = [...]string{"Неділя", "Понеділок", "Вівторок", "Середа", "Четвер", "Пʼятниця", "Субота"}
var ukWeekdaysShort = [...]string{"нд", "пн", "вт", "ср", "чт", "пт", "сб"}
var ukMonths = [...]string{"січня", "лютого", "березня", "квітня", "травня", "червня", "липня", "серпня", "вересня", "жовтня", "листопада", "грудня"}

// ukDate renders "Середа, 3 вересня".
func ukDate(t time.Time) string {
	return fmt.Sprintf("%s, %d %s", ukWeekdays[t.Weekday()], t.Day(), ukMonths[t.Month()-1])
}

// ageDays counts calendar days since a stored UTC timestamp, in local time.
// 1 means "today"; the age badge is shown from 3.
func ageDays(stored string, now time.Time) int {
	t, err := time.Parse("2006-01-02 15:04:05", stored)
	if err != nil {
		return 0
	}
	local := t.In(now.Location())
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, now.Location())
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return int(today.Sub(start).Hours()/24) + 1
}

// ukDay renders the ordinal day badge: "3-й день".
func ukDay(n int) string {
	return fmt.Sprintf("%d-й день", n)
}
