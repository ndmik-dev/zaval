package web

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ndmik-dev/zaval/internal/store"
)

type journalDay struct {
	Label string
	Today bool
	Tasks []taskRow
}

type projectCount struct {
	store.Project
	N int
}

type journalData struct {
	shell
	WeekLabel string
	PrevWeek  string
	NextWeek  string
	IsCurrent bool
	Days      []journalDay
	Counts    []projectCount
	Total     int
	Daily     dailyData
}

type dailyData struct {
	Project   *store.Project
	Scope     string // "day" or "week"
	Text      string
	Yesterday string
}

func (s *Server) journal(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	now := time.Now()
	week := weekStart(now)
	if v, err := time.ParseInLocation("2006-01-02", q.Get("w"), now.Location()); err == nil {
		week = weekStart(v)
	}
	filter := q.Get("p")

	sh, err := s.shell("Журнал", "journal", filter)
	if err != nil {
		s.fail(w, "journal", err)
		return
	}
	d := journalData{
		shell:     sh,
		WeekLabel: weekLabel(week),
		PrevWeek:  week.AddDate(0, 0, -7).Format("2006-01-02"),
		NextWeek:  week.AddDate(0, 0, 7).Format("2006-01-02"),
		IsCurrent: week.Equal(weekStart(now)),
	}

	done, err := s.store.DoneBetween(utc(week), utc(week.AddDate(0, 0, 7)))
	if err != nil {
		s.fail(w, "journal", err)
		return
	}
	byProject := map[int64]int{}
	var days []journalDay
	var lastDay string
	for _, t := range done {
		if !matchesFilter(t.Project, filter) {
			continue
		}
		byProject[t.ProjectID]++
		d.Total++
		day := localDay(t.DoneAt.String, now.Location())
		key := day.Format("2006-01-02")
		if key != lastDay {
			days = append(days, journalDay{Label: fmt.Sprintf("%s, %d", ukWeekdays[day.Weekday()], day.Day()), Today: sameDay(day, now)})
			lastDay = key
		}
		days[len(days)-1].Tasks = append(days[len(days)-1].Tasks, taskRow{Task: t})
	}
	d.Days = days
	for _, p := range sh.Projects {
		if n := byProject[p.ID]; n > 0 {
			d.Counts = append(d.Counts, projectCount{p.Project, n})
		}
	}

	d.Daily, err = s.daily(sh, q.Get("d"), q.Get("scope"), now)
	if err != nil {
		s.fail(w, "daily", err)
		return
	}
	if r.Header.Get("HX-Request") != "" {
		s.renderPart(w, "journal", "app", d)
		return
	}
	s.render(w, "journal", d)
}

// daily builds the standup text: what was closed for the project on the last
// working day (or this week), and what is in "Зараз" for it now.
func (s *Server) daily(sh shell, slug, scope string, now time.Time) (dailyData, error) {
	d := dailyData{Scope: "day"}
	if scope == "week" {
		d.Scope = "week"
	}
	for i := range sh.Projects {
		if sh.Projects[i].Slug == slug || ((slug == "" || slug == "-") && sh.Projects[i].Kind == "work") {
			d.Project = &sh.Projects[i].Project
			break
		}
	}
	if d.Project == nil {
		return d, nil
	}
	today := dayStart(now)
	from, to := today.AddDate(0, 0, -7), today
	if d.Scope == "week" {
		from = weekStart(now)
		to = today.AddDate(0, 0, 1)
	}
	done, err := s.store.DoneBetween(utc(from), utc(to))
	if err != nil {
		return d, err
	}
	var closed []store.Task
	var lastDay time.Time
	for _, t := range done { // newest first
		if t.ProjectID != d.Project.ID {
			continue
		}
		day := localDay(t.DoneAt.String, now.Location())
		if d.Scope == "day" {
			if lastDay.IsZero() {
				lastDay = day
			}
			if !sameDay(day, lastDay) {
				break
			}
		}
		closed = append(closed, t)
	}
	nowTasks, err := s.store.TasksByState("now")
	if err != nil {
		return d, err
	}

	var b strings.Builder
	head := "Вчора"
	if d.Scope == "week" {
		head = "Цього тижня"
	} else if !lastDay.IsZero() && !sameDay(lastDay, today.AddDate(0, 0, -1)) {
		head = fmt.Sprintf("%s, %d", ukWeekdays[lastDay.Weekday()], lastDay.Day())
		d.Yesterday = head
	}
	b.WriteString(head + ":\n")
	if len(closed) == 0 {
		b.WriteString("- —\n")
	}
	for i := len(closed) - 1; i >= 0; i-- { // chronological
		b.WriteString("- " + taskLine(closed[i]) + "\n")
	}
	b.WriteString("Сьогодні:\n")
	n := 0
	for _, t := range nowTasks {
		if t.ProjectID == d.Project.ID && t.Waiting == "" {
			b.WriteString("- " + taskLine(t) + "\n")
			n++
		}
	}
	if n == 0 {
		b.WriteString("- —\n")
	}
	waiting, err := s.store.WaitingTasks()
	if err != nil {
		return d, err
	}
	blockers := 0
	for _, t := range waiting {
		if t.ProjectID == d.Project.ID {
			if blockers == 0 {
				b.WriteString("Блокери:\n")
			}
			b.WriteString("- " + taskLine(t) + " — чекаю: " + t.Waiting + "\n")
			blockers++
		}
	}
	if blockers == 0 {
		b.WriteString("Блокери: нема")
	}
	d.Text = strings.TrimRight(b.String(), "\n")
	return d, nil
}

func taskLine(t store.Task) string {
	line := t.Title
	for _, l := range t.Links {
		if l.Kind == "jira" || l.Kind == "github" || l.Kind == "gitlab" {
			ref := l.Label
			if l.Meta != "" && l.Kind != "jira" {
				ref = l.Meta
			}
			return line + " (" + ref + ")"
		}
	}
	return line
}

func weekStart(t time.Time) time.Time {
	t = dayStart(t)
	wd := int(t.Weekday())
	if wd == 0 {
		wd = 7
	}
	return t.AddDate(0, 0, 1-wd)
}

func dayStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func sameDay(a, b time.Time) bool {
	return a.Year() == b.Year() && a.YearDay() == b.YearDay()
}

func localDay(stored string, loc *time.Location) time.Time {
	t, _ := time.Parse(store.TimeLayout, stored)
	return dayStart(t.In(loc))
}

func utc(t time.Time) string {
	return t.UTC().Format(store.TimeLayout)
}

func weekLabel(monday time.Time) string {
	sunday := monday.AddDate(0, 0, 6)
	if monday.Month() == sunday.Month() {
		return fmt.Sprintf("%d–%d %s", monday.Day(), sunday.Day(), ukMonths[monday.Month()-1])
	}
	return fmt.Sprintf("%d %s – %d %s", monday.Day(), ukMonths[monday.Month()-1], sunday.Day(), ukMonths[sunday.Month()-1])
}
