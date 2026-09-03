package web

import (
	"log"
	"net/http"
	"net/url"
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
	r.Form.Del("t")
	s.respondBoard(w, r)
}

// respondBoard re-renders the whole app shell after a mutation; htmx morphs it
// in place. Plain form posts (no htmx) get a redirect back to the board.
func (s *Server) respondBoard(w http.ResponseWriter, r *http.Request) {
	filter := r.FormValue("p")
	openID, _ := strconv.ParseInt(r.FormValue("t"), 10, 64)
	if r.Header.Get("HX-Request") == "" {
		http.Redirect(w, r, boardURL(filter, openID), http.StatusSeeOther)
		return
	}
	data, err := s.boardData(filter, openID)
	if err != nil {
		s.fail(w, "board", err)
		return
	}
	s.renderPart(w, "board", "app", data)
}

func boardURL(filter string, openID int64) string {
	q := url.Values{}
	if filter != "" {
		q.Set("p", filter)
	}
	if openID != 0 {
		q.Set("t", strconv.FormatInt(openID, 10))
	}
	if len(q) == 0 {
		return "/"
	}
	return "/?" + q.Encode()
}

func (s *Server) fail(w http.ResponseWriter, what string, err error) {
	log.Printf("%s: %v", what, err)
	http.Error(w, "db error", http.StatusInternalServerError)
}

func (s *Server) reorderTasks(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	if err := s.store.Reorder(ids(r.Form["now"]), ids(r.Form["backlog"])); err != nil {
		s.fail(w, "reorder", err)
		return
	}
	s.respondBoard(w, r)
}

func ids(raw []string) []int64 {
	out := make([]int64, 0, len(raw))
	for _, v := range raw {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			out = append(out, id)
		}
	}
	return out
}
