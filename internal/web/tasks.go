package web

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/ndmik-dev/zaval/internal/links"
	"github.com/ndmik-dev/zaval/internal/store"
)

func (s *Server) createTask(w http.ResponseWriter, r *http.Request) {
	title, found := links.Parse(strings.TrimSpace(r.FormValue("title")))
	if title == "" {
		s.respondBoard(w, r)
		return
	}
	projectID, _ := strconv.ParseInt(r.FormValue("project_id"), 10, 64)
	state := r.FormValue("state")
	if state != "now" {
		state = "backlog"
	}
	task, err := s.store.CreateTask(projectID, title, state)
	if err != nil {
		s.fail(w, "create task", err)
		return
	}
	for _, l := range found {
		if _, err := s.store.AddLink(task.ID, l.URL, l.Kind, l.Label, l.Meta); err != nil {
			s.fail(w, "add link", err)
			return
		}
	}
	s.respondBoard(w, r)
}

func (s *Server) setTaskState(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err := s.store.SetTaskState(id, r.FormValue("state")); err != nil {
		if err == store.ErrNotFound {
			http.NotFound(w, r)
			return
		}
		s.fail(w, "set state", err)
		return
	}
	s.respondBoard(w, r)
}

func (s *Server) deleteTask(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err := s.store.DeleteTask(id); err != nil {
		s.fail(w, "delete task", err)
		return
	}
	s.respondBoard(w, r)
}

// respondBoard re-renders the whole app shell after a mutation; htmx morphs it
// in place. Plain form posts (no htmx) get a redirect back to the board.
func (s *Server) respondBoard(w http.ResponseWriter, r *http.Request) {
	filter := r.FormValue("p")
	if r.Header.Get("HX-Request") == "" {
		url := "/"
		if filter != "" {
			url += "?p=" + filter
		}
		http.Redirect(w, r, url, http.StatusSeeOther)
		return
	}
	data, err := s.boardData(filter)
	if err != nil {
		s.fail(w, "board", err)
		return
	}
	s.renderPart(w, "board", "app", data)
}

func (s *Server) fail(w http.ResponseWriter, what string, err error) {
	log.Printf("%s: %v", what, err)
	http.Error(w, "db error", http.StatusInternalServerError)
}
