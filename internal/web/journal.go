package web

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/ndmik-dev/zaval/internal/store"
)

// journalDay is one row in the day list; the selected one fills the detail pane.
type journalDay struct {
	Key   string // YYYY-MM-DD
	Label string // short form for the list, the week header carries the month
	Full  string // full date for the detail pane
	N     int
	Today bool
	Sel   bool
	Href  string
}

type projectCount struct {
	store.Project
	N int
}

// journalWeek groups the day rows so a long list keeps some shape.
type journalWeek struct {
	Label string
	Days  []journalDay
}

type journalData struct {
	shell
	Weeks   []journalWeek
	Day     *journalDay
	Tasks   []taskRow
	Counts  []projectCount
	Total   int
	Current *store.Project // the project the filter is on, if any
}

// dailyData is the standup text, in the three blocks it is spoken in.
type dailyData struct {
	Project   *store.Project
	Yesterday string // label of the last day something was closed, when not literally yesterday
	Closed    []store.Task
	Today     []store.Task
	Blockers  []store.Task
}

// journalSpan is how far back the day list reaches.
const journalSpan = 45

func (s *Server) journal(w http.ResponseWriter, r *http.Request) {
	s.respondJournal(w, r)
}

// respondJournal reads its parameters from the query or the posted form, so
// actions taken in the detail pane land back on the same day and filter.
func (s *Server) respondJournal(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	now := time.Now()
	filter := r.FormValue("p")
	openID := taskParam(r)

	sh, err := s.shell("Журнал", "journal", filter)
	if err != nil {
		s.fail(w, "journal", err)
		return
	}
	sh.Ctx = map[string]string{"page": "journal", "p": filter, "d": r.FormValue("d")}
	sh.Open = s.openTask(openID, now)
	sh.Detail = openID != 0 || r.FormValue("d") != ""
	d := journalData{shell: sh}

	from := dayStart(now).AddDate(0, 0, -journalSpan)
	done, err := s.store.DoneBetween(utc(from), utc(dayStart(now).AddDate(0, 0, 1)))
	if err != nil {
		s.fail(w, "journal", err)
		return
	}
	byDay := map[string][]store.Task{}
	byProject := map[int64]int{}
	for _, t := range done {
		if !matchesFilter(t.Project, filter) {
			continue
		}
		key := localDay(t.DoneAt.String, now.Location()).Format("2006-01-02")
		byDay[key] = append(byDay[key], t)
		byProject[t.ProjectID]++
		d.Total++
	}
	for i := range sh.Projects {
		if sh.Projects[i].Slug == filter {
			d.Current = &sh.Projects[i].Project
		}
	}
	var days []journalDay
	want := r.FormValue("d")
	for i := 0; i <= journalSpan; i++ {
		day := dayStart(now).AddDate(0, 0, -i)
		key := day.Format("2006-01-02")
		tasks := byDay[key]
		if len(tasks) == 0 && !sameDay(day, now) {
			continue // quiet days are not worth a row
		}
		row := journalDay{
			Key:   key,
			Label: fmt.Sprintf("%s, %d", ukWeekdays[day.Weekday()], day.Day()),
			Full:  fmt.Sprintf("%s, %d %s", ukWeekdays[day.Weekday()], day.Day(), ukMonths[day.Month()-1]),
			N:     len(tasks),
			Today: sameDay(day, now),
			Href:  "/journal?" + journalQuery(filter, key, 0),
		}
		days = append(days, row)
	}
	// Selected day: the asked-for one, else the first with something in it.
	for i := range days {
		if days[i].Key == want || (want == "" && d.Day == nil && days[i].N > 0) {
			days[i].Sel = true
			d.Day = &days[i]
		}
	}
	if d.Day == nil && len(days) > 0 {
		days[0].Sel = true
		d.Day = &days[0]
	}
	d.Weeks = groupWeeks(days, now)
	if d.Day != nil {
		sh.Ctx["d"] = d.Day.Key
		d.Ctx = sh.Ctx
		for _, t := range byDay[d.Day.Key] {
			row := taskRow{Task: t, Href: "/journal?" + journalQuery(filter, d.Day.Key, t.ID), Sel: t.ID == openID, Tag: true}
			d.Tasks = append(d.Tasks, row)
		}
	}
	for _, p := range sh.Projects {
		if n := byProject[p.ID]; n > 0 {
			d.Counts = append(d.Counts, projectCount{p.Project, n})
		}
	}

	if r.Header.Get("HX-Request") != "" {
		s.renderPart(w, "journal", "app", d)
		return
	}
	s.render(w, "journal", d)
}

// groupWeeks splits the day rows into weeks, newest first.
func groupWeeks(days []journalDay, now time.Time) []journalWeek {
	thisWeek := weekStart(now)
	var out []journalWeek
	var lastKey string
	for _, day := range days {
		t, err := time.ParseInLocation("2006-01-02", day.Key, now.Location())
		if err != nil {
			continue
		}
		ws := weekStart(t)
		if key := ws.Format("2006-01-02"); key != lastKey {
			out = append(out, journalWeek{Label: weekLabel(ws, thisWeek)})
			lastKey = key
		}
		out[len(out)-1].Days = append(out[len(out)-1].Days, day)
	}
	return out
}

func weekStart(t time.Time) time.Time {
	t = dayStart(t)
	wd := int(t.Weekday())
	if wd == 0 {
		wd = 7 // Sunday closes the week here, it does not open it
	}
	return t.AddDate(0, 0, 1-wd)
}

func weekLabel(ws, thisWeek time.Time) string {
	switch {
	case ws.Equal(thisWeek):
		return "Цей тиждень"
	case ws.Equal(thisWeek.AddDate(0, 0, -7)):
		return "Минулий тиждень"
	}
	end := ws.AddDate(0, 0, 6)
	if ws.Month() == end.Month() {
		return fmt.Sprintf("%d–%d %s", ws.Day(), end.Day(), ukMonths[ws.Month()-1])
	}
	return fmt.Sprintf("%d %s – %d %s", ws.Day(), ukMonths[ws.Month()-1], end.Day(), ukMonths[end.Month()-1])
}

func journalQuery(filter, day string, taskID int64) string {
	q := url.Values{}
	if filter != "" {
		q.Set("p", filter)
	}
	if day != "" {
		q.Set("d", day)
	}
	if taskID != 0 {
		q.Set("t", strconv.FormatInt(taskID, 10))
	}
	return q.Encode()
}

// daily collects the standup for a work project: what was closed on the last
// day anything was closed, what is in «Зараз» now, and what it waits on.
func (s *Server) daily(sh shell, slug string, now time.Time) (dailyData, error) {
	var d dailyData
	for i := range sh.Projects {
		if sh.Projects[i].Slug == slug || (slug == "" && sh.Projects[i].Kind == "work") {
			d.Project = &sh.Projects[i].Project
			break
		}
	}
	if d.Project == nil {
		return d, nil
	}
	today := dayStart(now)
	done, err := s.store.DoneBetween(utc(today.AddDate(0, 0, -14)), utc(today.AddDate(0, 0, 1)))
	if err != nil {
		return d, err
	}
	var lastDay time.Time
	for _, t := range done { // newest first
		if t.ProjectID != d.Project.ID {
			continue
		}
		day := localDay(t.DoneAt.String, now.Location())
		if sameDay(day, today) {
			continue // today's work belongs under "Сьогодні"
		}
		if lastDay.IsZero() {
			lastDay = day
		}
		if !sameDay(day, lastDay) {
			break
		}
		d.Closed = append([]store.Task{t}, d.Closed...) // chronological
	}
	if !lastDay.IsZero() && !sameDay(lastDay, today.AddDate(0, 0, -1)) {
		d.Yesterday = fmt.Sprintf("%s, %d %s", ukWeekdays[lastDay.Weekday()], lastDay.Day(), ukMonths[lastDay.Month()-1])
	}
	nowTasks, err := s.store.TasksByState("now")
	if err != nil {
		return d, err
	}
	for _, t := range nowTasks {
		if t.ProjectID == d.Project.ID && t.Waiting == "" {
			d.Today = append(d.Today, t)
		}
	}
	for _, t := range done {
		if t.ProjectID == d.Project.ID && sameDay(localDay(t.DoneAt.String, now.Location()), today) {
			d.Today = append(d.Today, t)
		}
	}
	waiting, err := s.store.WaitingTasks()
	if err != nil {
		return d, err
	}
	for _, t := range waiting {
		if t.ProjectID == d.Project.ID {
			d.Blockers = append(d.Blockers, t)
		}
	}
	return d, nil
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
