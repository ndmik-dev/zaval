package web

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/ndmik-dev/zaval/internal/store"
)

func (s *Server) setTaskState(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if r.FormValue("state") == "waiting" {
		// "Чекаю" keeps now/backlog; only the note changes.
		task, err := s.store.Task(id)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if task.Waiting == "" {
			if err := s.store.SetWaiting(id, "чекаю"); err != nil {
				s.fail(w, "waiting", err)
				return
			}
		}
		s.respondBoard(w, r)
		return
	}
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

// respondBoard re-renders the page the request came from (every page puts its
// name in the context), so a task edited from the journal stays on the journal.
// Plain form posts (no htmx) get a redirect back to the board.
func (s *Server) respondBoard(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	openID := taskParam(r)
	if r.Header.Get("HX-Request") == "" {
		http.Redirect(w, r, boardURL(openID), http.StatusSeeOther)
		return
	}
	switch r.FormValue("page") {
	case "journal":
		s.respondJournal(w, r)
		return
	case "releases":
		s.respondReleases(w, r)
		return
	}
	data, err := s.boardData(group(r), openID, r.FormValue("daily"))
	if err != nil {
		s.fail(w, "board", err)
		return
	}
	s.renderPart(w, "board", "app", data)
}

func boardURL(openID int64) string {
	if openID == 0 {
		return "/"
	}
	return "/?t=" + strconv.FormatInt(openID, 10)
}

func (s *Server) fail(w http.ResponseWriter, what string, err error) {
	log.Printf("%s: %v", what, err)
	http.Error(w, "db error", http.StatusInternalServerError)
}

func (s *Server) reorderTasks(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	if err := s.store.Reorder(ids(r.Form["now"]), ids(r.Form["backlog"]), ids(r.Form["waiting"])); err != nil {
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

func (s *Server) setWaiting(w http.ResponseWriter, r *http.Request) {
	id := pathID(r)
	if err := s.store.SetWaiting(id, strings.TrimSpace(r.FormValue("waiting"))); err != nil {
		s.fail(w, "waiting", err)
		return
	}
	s.respondTask(w, r, id)
}
