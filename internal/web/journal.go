package web

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/ndmik-dev/zaval/internal/store"
)

// journalDay is one heading in the feed, with everything closed that day.
type journalDay struct {
	Key   string // YYYY-MM-DD
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
	Feed    []journalDay
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

// journalSpan is how far back the feed reaches.
const journalSpan = 45

func (s *Server) journal(w http.ResponseWriter, r *http.Request) {
	s.respondJournal(w, r)
}

// respondJournal reads its parameters from the query or the posted form, so
// actions taken in the detail pane land back on the same filter.
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
	sh.Ctx = map[string]string{"page": "journal", "p": filter}
	sh.Open = s.openTask(openID, now)
	d := journalData{shell: sh}
	for i := range sh.Projects {
		if sh.Projects[i].Slug == filter {
			d.Current = &sh.Projects[i].Project
		}
	}

	from := dayStart(now).AddDate(0, 0, -journalSpan)
	done, err := s.store.DoneBetween(utc(from), utc(dayStart(now).AddDate(0, 0, 1)))
	if err != nil {
		s.fail(w, "journal", err)
		return
	}
	byProject := map[int64]int{}
	for _, t := range done { // newest first, so the feed is already in order
		if !matchesFilter(t.Project, filter) {
			continue
		}
		byProject[t.ProjectID]++
		d.Total++
		day := localDay(t.DoneAt.String, now.Location())
		key := day.Format("2006-01-02")
		if len(d.Feed) == 0 || d.Feed[len(d.Feed)-1].Key != key {
			d.Feed = append(d.Feed, journalDay{
				Key:   key,
				Label: fmt.Sprintf("%s, %d %s", ukWeekdays[day.Weekday()], day.Day(), ukMonths[day.Month()-1]),
				Today: sameDay(day, now),
			})
		}
		row := taskRow{Task: t, Href: "/journal?" + journalQuery(filter, t.ID), Sel: t.ID == openID, Tag: true}
		d.Feed[len(d.Feed)-1].Tasks = append(d.Feed[len(d.Feed)-1].Tasks, row)
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

func journalQuery(filter string, taskID int64) string {
	q := url.Values{}
	if filter != "" {
		q.Set("p", filter)
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
