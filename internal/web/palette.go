package web

import (
	"net/http"
	"strings"

	"github.com/ndmik-dev/zaval/internal/store"
)

type paletteData struct {
	Query   string
	Parsed  quickInput
	Matches []taskRow
	NowN    int
}

// palette renders the live preview under the ⌘K input.
func (s *Server) palette(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("text"))
	projects, err := s.store.Projects()
	if err != nil {
		s.fail(w, "palette", err)
		return
	}
	d := paletteData{Query: q, Parsed: parseQuick(q, projects)}
	if d.Parsed.Project == nil {
		d.Parsed.Project = s.defaultProject(projects, r.URL.Query().Get("p"))
	}
	if len([]rune(d.Parsed.Title)) >= 2 {
		all, err := s.store.SearchTasks()
		if err != nil {
			s.fail(w, "search", err)
			return
		}
		for _, t := range all {
			if matchQuery(t.Title, d.Parsed.Title) {
				d.Matches = append(d.Matches, taskRow{Task: t})
				if len(d.Matches) == 6 {
					break
				}
			}
		}
	}
	if now, err := s.store.TasksByState("now"); err == nil {
		d.NowN = len(now)
	}
	s.renderPart(w, "board", "palette_results", d)
}

// createQuick creates a task from the raw ⌘K line.
func (s *Server) createQuick(w http.ResponseWriter, r *http.Request) {
	projects, err := s.store.Projects()
	if err != nil {
		s.fail(w, "palette", err)
		return
	}
	q := parseQuick(strings.TrimSpace(r.FormValue("text")), projects)
	if q.Title == "" {
		s.respondBoard(w, r)
		return
	}
	if q.Project == nil {
		q.Project = s.defaultProject(projects, r.FormValue("p"))
	}
	state := "backlog"
	if q.Now || r.FormValue("force_now") != "" {
		state = "now"
	}
	task, err := s.store.CreateTask(q.Project.ID, q.Title, state)
	if err != nil {
		s.fail(w, "create task", err)
		return
	}
	for _, l := range q.Links {
		if _, err := s.store.AddLink(task.ID, l.URL, l.Kind, l.Label, l.Meta); err != nil {
			s.fail(w, "add link", err)
			return
		}
	}
	// Created from another page: send the browser to the board instead of morphing it in place.
	if from := r.FormValue("from"); from != "" && from != "board" {
		w.Header().Set("HX-Redirect", "/")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	r.Form.Del("t")
	s.respondBoard(w, r)
}

// defaultProject is the filtered project when the board is filtered,
// otherwise the first project shown on the board.
func (s *Server) defaultProject(projects []store.Project, filter string) *store.Project {
	for i := range projects {
		if projects[i].Slug == filter {
			return &projects[i]
		}
	}
	for i := range projects {
		if projects[i].OnBoard {
			return &projects[i]
		}
	}
	if len(projects) > 0 {
		return &projects[0]
	}
	return nil
}
