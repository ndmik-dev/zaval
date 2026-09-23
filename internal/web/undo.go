package web

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/ndmik-dev/zaval/internal/store"
)

// undo is the toast under the page after a reversible action: one label, one
// inverse request. It lives only in the response that follows the action.
type undo struct {
	Label string
	Op    string // state | restore | move | toggle
	ID    int64
	Arg   string // the state or direction to go back to
}

// offerUndo attaches the toast to the page the handler is about to render.
func offerUndo(r *http.Request, label, op string, id int64, arg string) {
	// A state undo needs the old slot too; callers pass the task's state and
	// the helper appends the position they had at hand via undoState.
	r.ParseForm()
	r.Form.Set("undo_label", label)
	r.Form.Set("undo_op", op)
	r.Form.Set("undo_id", strconv.FormatInt(id, 10))
	r.Form.Set("undo_arg", arg)
}

func undoFrom(r *http.Request) *undo {
	if r.FormValue("undo_op") == "" {
		return nil
	}
	id, _ := strconv.ParseInt(r.FormValue("undo_id"), 10, 64)
	return &undo{Label: r.FormValue("undo_label"), Op: r.FormValue("undo_op"), ID: id, Arg: r.FormValue("undo_arg")}
}

// undoAction reverses the last action: the toast's button and ⌘Z post here.
func (s *Server) undoAction(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id, _ := strconv.ParseInt(r.FormValue("id"), 10, 64)
	var err error
	switch r.FormValue("op") {
	case "state":
		// arg is "state:position" — the page it was on and where on that page.
		state, pos, _ := strings.Cut(r.FormValue("arg"), ":")
		if err = s.store.SetTaskState(id, state); err == nil && pos != "" {
			if n, perr := strconv.Atoi(pos); perr == nil {
				err = s.store.SetPosition(id, n)
			}
		}
	case "restore":
		if t, ok := s.deleted[id]; ok {
			err = s.store.RestoreTask(t)
			delete(s.deleted, id)
		}
	case "move":
		r.Form.Set("dir", r.FormValue("arg"))
		s.moveLine(w, r, id)
		return
	case "toggle":
		err = s.store.ToggleChecklistItem(id)
	}
	if err != nil {
		s.fail(w, "undo", err)
		return
	}
	r.Form.Del("undo_op")
	s.respondNotebook(w, r)
}

// remember keeps a deleted task for the toast's lifetime.
func (s *Server) remember(t store.Task) {
	if s.deleted == nil {
		s.deleted = map[int64]store.Task{}
	}
	s.deleted[t.ID] = t
}

// undoState is the arg for a "state" undo: the page and the slot to return to.
func undoState(t store.Task) string {
	return t.State + ":" + strconv.Itoa(t.Position)
}
