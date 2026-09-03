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

	load := func(ts []store.Task, err error) ([]taskRow, error) {
		if err != nil {
			return nil, err
		}
		var rows []taskRow
		for _, t := range ts {
			if !t.Project.OnBoard || !matchesFilter(t.Project, filter) {
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
	if d.Now, err = load(s.store.TasksByState("now")); err != nil {
		return d, err
	}
	if d.Backlog, err = load(s.store.TasksByState("backlog")); err != nil {
		return d, err
	}
	if d.DoneToday, err = load(s.store.DoneToday()); err != nil {
		return d, err
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
