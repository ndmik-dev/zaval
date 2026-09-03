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
	Age int
}

type boardData struct {
	shell
	Date      string
	Now       []taskRow
	Backlog   []taskRow
	DoneToday []taskRow
	Release   *checklistView // a release in progress, for the banner
	AddSlug   string         // project preselected in the add bar
}

func (s *Server) board(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	openID, _ := strconv.ParseInt(q.Get("t"), 10, 64)
	data, err := s.boardData(q.Get("p"), openID)
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

func (s *Server) boardData(filter string, openID int64) (boardData, error) {
	now := time.Now()
	sh, err := s.shell("Дошка", "board", filter)
	if err != nil {
		return boardData{}, err
	}
	d := boardData{shell: sh, Date: ukDate(now)}
	for _, p := range sh.Projects {
		if p.Slug == filter || d.AddSlug == "" {
			d.AddSlug = p.Slug
		}
	}
	if openID != 0 {
		t, err := s.store.Task(openID)
		if err == nil {
			row := taskRow{Task: t}
			if t.State == "now" && t.NowSince.Valid {
				row.Age = ageDays(t.NowSince.String, now)
			}
			d.Open = &row
		}
	}

	// The filter narrows the backlog only: "Зараз" and "Готово" are short and always shown whole.
	load := func(ts []store.Task, err error, filtered bool) ([]taskRow, error) {
		if err != nil {
			return nil, err
		}
		var rows []taskRow
		for _, t := range ts {
			if !t.Project.OnBoard || (filtered && !matchesFilter(t.Project, filter)) {
				continue
			}
			row := taskRow{Task: t}
			if t.State == "now" && t.NowSince.Valid {
				row.Age = ageDays(t.NowSince.String, now)
			}
			rows = append(rows, row)
		}
		return rows, nil
	}
	nowTasks, err := s.store.TasksByState("now")
	if d.Now, err = load(nowTasks, err, false); err != nil {
		return d, err
	}
	backlog, err := s.store.TasksByState("backlog")
	if d.Backlog, err = load(backlog, err, true); err != nil {
		return d, err
	}
	doneToday, err := s.store.DoneToday()
	if d.DoneToday, err = load(doneToday, err, false); err != nil {
		return d, err
	}

	// Banner: the filtered project's release, else the first project with a tick in its checklist.
	progress, err := s.store.ChecklistProgress()
	if err != nil {
		return d, err
	}
	for _, p := range sh.Projects {
		pr := progress[p.ID]
		if pr.Done == 0 || (filter != "" && p.Slug != filter) {
			continue
		}
		if d.Release, err = s.checklistView(p.Project, false, now); err != nil {
			return d, err
		}
		break
	}
	return d, nil
}

func matchesFilter(p store.Project, filter string) bool {
	switch filter {
	case "":
		return true
	case "pet":
		return p.Kind == "pet"
	default:
		return p.Slug == filter
	}
}
