package web

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/ndmik-dev/zaval/internal/store"
)

type projectItem struct {
	store.Project
	store.Counts
}

type taskRow struct {
	store.Task
	Age  int
	Href string // where a click on the row leads; each page builds its own
	Sel  bool   // shown in the detail pane right now
	Tag  bool   // show the project tag (off when the list is already grouped by project)
}

// band is one project's open tasks when the board is grouped by project.
type band struct {
	store.Project
	Rows []taskRow
}

type boardData struct {
	shell
	Date      string
	Group     string // "state" or "project"
	Now       []taskRow
	Waiting   []taskRow
	Backlog   []taskRow
	DoneToday []taskRow
	Bands     []band
	Quiet     []store.Project // projects with nothing open, listed as chips
	Work      []projectItem
	Daily     *dailyData // filled when no task is selected
	DailySlug string
}

func (s *Server) board(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	if g := r.FormValue("g"); g == "state" || g == "project" {
		http.SetCookie(w, &http.Cookie{Name: "group", Value: g, Path: "/", MaxAge: 86400 * 365, SameSite: http.SameSiteLaxMode})
	}
	data, err := s.boardData(group(r), taskParam(r), r.FormValue("daily"))
	if err != nil {
		log.Println("board:", err)
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if r.Header.Get("HX-Request") != "" {
		s.renderPart(w, "board", "app", data)
		return
	}
	s.render(w, "board", data)
}

func taskParam(r *http.Request) int64 {
	id, _ := strconv.ParseInt(r.FormValue("t"), 10, 64)
	return id
}

// group is the board's grouping: the request wins, then the remembered choice.
func group(r *http.Request) string {
	if g := r.FormValue("g"); g == "state" || g == "project" {
		return g
	}
	if c, err := r.Cookie("group"); err == nil && c.Value == "project" {
		return "project"
	}
	return "state"
}

func (s *Server) boardData(grouping string, openID int64, daily string) (boardData, error) {
	now := time.Now()
	sh, err := s.shell("Дошка", "board", "")
	if err != nil {
		return boardData{}, err
	}
	d := boardData{shell: sh, Date: ukDate(now), Group: grouping, Work: sh.work()}
	d.Ctx = map[string]string{"page": "board", "g": grouping, "daily": daily}
	d.Open = s.openTask(openID, now)
	d.Detail = d.Open != nil

	rows := func(ts []store.Task, err error) ([]taskRow, error) {
		if err != nil {
			return nil, err
		}
		var out []taskRow
		for _, t := range ts {
			if t.Waiting != "" && t.State != "done" {
				continue // listed under "Чекаю" instead
			}
			out = append(out, s.row(t, openID, now))
		}
		return out, nil
	}
	if d.Now, err = rows(s.store.TasksByState("now")); err != nil {
		return d, err
	}
	if d.Backlog, err = rows(s.store.TasksByState("backlog")); err != nil {
		return d, err
	}
	if d.DoneToday, err = rows(s.store.DoneToday()); err != nil {
		return d, err
	}
	waiting, err := s.store.WaitingTasks()
	if err != nil {
		return d, err
	}
	for _, t := range waiting {
		d.Waiting = append(d.Waiting, s.row(t, openID, now))
	}

	if grouping == "project" {
		d.Bands, d.Quiet = bands(sh.Projects, d.Now, d.Waiting, d.Backlog)
	}

	// The detail pane shows the standup text whenever no task is selected.
	if d.Open == nil {
		dd, err := s.daily(sh, daily, now)
		if err != nil {
			return d, err
		}
		if dd.Project != nil {
			d.Daily = &dd
			d.DailySlug = dd.Project.Slug
		}
	}
	return d, nil
}

// row builds a board row: age badge, link back to the board, selected flag.
func (s *Server) row(t store.Task, openID int64, now time.Time) taskRow {
	r := taskRow{Task: t, Href: "/?t=" + strconv.FormatInt(t.ID, 10), Sel: t.ID == openID, Tag: true}
	switch {
	case t.Waiting != "" && t.WaitingSince.Valid:
		r.Age = ageDays(t.WaitingSince.String, now)
	case t.State == "now" && t.NowSince.Valid:
		r.Age = ageDays(t.NowSince.String, now)
	}
	return r
}

// bands regroups the open rows by project, keeping the project order from the
// rail; projects with nothing open are returned separately as "quiet".
func bands(projects []projectItem, lists ...[]taskRow) ([]band, []store.Project) {
	byProject := map[int64][]taskRow{}
	for _, list := range lists {
		for _, r := range list {
			r.Tag = false
			byProject[r.ProjectID] = append(byProject[r.ProjectID], r)
		}
	}
	var out []band
	var quiet []store.Project
	for _, p := range projects {
		if rows := byProject[p.ID]; len(rows) > 0 {
			out = append(out, band{Project: p.Project, Rows: rows})
		} else {
			quiet = append(quiet, p.Project)
		}
	}
	return out, quiet
}

func matchesFilter(p store.Project, filter string) bool {
	return filter == "" || p.Slug == filter
}
