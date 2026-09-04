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
	Label string
	N     int
	Today bool
	Sel   bool
	Href  string
}

type projectCount struct {
	store.Project
	N int
}

type journalData struct {
	shell
	Days   []journalDay
	Day    *journalDay
	Tasks  []taskRow
	Counts []projectCount
	Total  int
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
			Label: fmt.Sprintf("%s, %d %s", ukWeekdays[day.Weekday()], day.Day(), ukMonths[day.Month()-1]),
			N:     len(tasks),
			Today: sameDay(day, now),
			Href:  "/journal?" + journalQuery(filter, key, 0),
		}
		d.Days = append(d.Days, row)
	}
	// Selected day: the asked-for one, else the first with something in it.
	for i := range d.Days {
		if d.Days[i].Key == want || (want == "" && d.Day == nil && d.Days[i].N > 0) {
			d.Days[i].Sel = true
			d.Day = &d.Days[i]
		}
	}
	if d.Day == nil && len(d.Days) > 0 {
		d.Days[0].Sel = true
		d.Day = &d.Days[0]
	}
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
