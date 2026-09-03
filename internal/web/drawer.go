package web

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/ndmik-dev/zaval/internal/links"
)

func pathID(r *http.Request) int64 {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	return id
}

// respondTask re-renders the board with the drawer open on the given task.
func (s *Server) respondTask(w http.ResponseWriter, r *http.Request, taskID int64) {
	r.ParseForm()
	r.Form.Set("t", strconv.FormatInt(taskID, 10))
	s.respondBoard(w, r)
}

func (s *Server) updateTask(w http.ResponseWriter, r *http.Request) {
	id := pathID(r)
	task, err := s.store.Task(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		title = task.Title
	}
	notes := r.FormValue("notes")
	if !r.Form.Has("notes") {
		notes = task.Notes
	}
	if err := s.store.UpdateTask(id, title, notes); err != nil {
		s.fail(w, "update task", err)
		return
	}
	s.respondTask(w, r, id)
}

func (s *Server) addLink(w http.ResponseWriter, r *http.Request) {
	id := pathID(r)
	raw := strings.TrimSpace(r.FormValue("url"))
	if raw != "" {
		l := links.Classify(raw)
		if _, err := s.store.AddLink(id, l.URL, l.Kind, l.Label, l.Meta); err != nil {
			s.fail(w, "add link", err)
			return
		}
	}
	s.respondTask(w, r, id)
}

func (s *Server) deleteLink(w http.ResponseWriter, r *http.Request) {
	id := pathID(r)
	taskID, err := s.store.LinkTask(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := s.store.DeleteLink(id); err != nil {
		s.fail(w, "delete link", err)
		return
	}
	s.respondTask(w, r, taskID)
}

func (s *Server) addStep(w http.ResponseWriter, r *http.Request) {
	id := pathID(r)
	if title := strings.TrimSpace(r.FormValue("title")); title != "" {
		if _, err := s.store.AddStep(id, title); err != nil {
			s.fail(w, "add step", err)
			return
		}
	}
	s.respondTask(w, r, id)
}

func (s *Server) toggleStep(w http.ResponseWriter, r *http.Request) {
	s.stepAction(w, r, s.store.ToggleStep)
}

func (s *Server) deleteStep(w http.ResponseWriter, r *http.Request) {
	s.stepAction(w, r, s.store.DeleteStep)
}

func (s *Server) stepAction(w http.ResponseWriter, r *http.Request, do func(int64) error) {
	id := pathID(r)
	taskID, err := s.store.StepTask(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := do(id); err != nil {
		s.fail(w, "step", err)
		return
	}
	s.respondTask(w, r, taskID)
}
