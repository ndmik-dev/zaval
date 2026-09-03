package web

import (
	"net/http"
	"strings"
	"time"

	"github.com/ndmik-dev/zaval/internal/store"
)

// A "release" is the project's checklist being ticked off: nothing to create,
// no versions or dates. Ticking starts it, «Зарелізено» records it and clears.
type checklistView struct {
	Project  store.Project
	Before   []store.ChecklistItem
	After    []store.ChecklistItem
	Done     int
	Total    int
	NextItem string
	LastAt   string // last release, "22 серпня"
	Edit     bool
}

type releasesData struct {
	shell
	Work    []projectItem
	Current *checklistView
}

func (s *Server) checklistView(p store.Project, edit bool, now time.Time) (*checklistView, error) {
	items, err := s.store.Checklist(p.ID)
	if err != nil {
		return nil, err
	}
	v := &checklistView{Project: p, Edit: edit, Total: len(items)}
	for _, it := range items {
		if it.Phase == "after" {
			v.After = append(v.After, it)
		} else {
			v.Before = append(v.Before, it)
		}
		if it.Done {
			v.Done++
		} else if v.NextItem == "" {
			v.NextItem = it.Title
		}
	}
	hist, err := s.store.ReleaseHistory(p.ID, 1)
	if err != nil {
		return nil, err
	}
	if len(hist) > 0 {
		d := localDay(hist[0].ReleasedAt, now.Location())
		v.LastAt = ukDateShort(d.Format("2006-01-02"), now)
	}
	return v, nil
}

func (s *Server) releases(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	s.respondReleases(w, r, q.Get("p"), q.Get("edit") != "")
}

func (s *Server) respondReleases(w http.ResponseWriter, r *http.Request, slug string, edit bool) {
	sh, err := s.shell("Релізи", "releases", "")
	if err != nil {
		s.fail(w, "releases", err)
		return
	}
	d := releasesData{shell: sh, Work: sh.work()}
	var current *store.Project
	for i := range d.Work {
		if d.Work[i].Slug == slug || (slug == "" && current == nil) {
			current = &d.Work[i].Project
		}
	}
	if current != nil {
		d.Current, err = s.checklistView(*current, edit, time.Now())
		if err != nil {
			s.fail(w, "checklist", err)
			return
		}
	}
	if r.Header.Get("HX-Request") != "" {
		s.renderPart(w, "releases", "app", d)
		return
	}
	s.render(w, "releases", d)
}

// checklistAction runs do for the project in the path and re-renders its checklist.
func (s *Server) checklistAction(w http.ResponseWriter, r *http.Request, do func(p store.Project) error) {
	p, err := s.store.Project(pathID(r))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := do(p); err != nil {
		s.fail(w, "checklist", err)
		return
	}
	s.respondReleases(w, r, p.Slug, r.FormValue("edit") != "")
}

func (s *Server) addChecklistItem(w http.ResponseWriter, r *http.Request) {
	s.checklistAction(w, r, func(p store.Project) error {
		title := strings.TrimSpace(r.FormValue("title"))
		if title == "" {
			return nil
		}
		_, err := s.store.AddChecklistItem(p.ID, phase(r), title)
		return err
	})
}

func (s *Server) markReleased(w http.ResponseWriter, r *http.Request) {
	s.checklistAction(w, r, func(p store.Project) error { return s.store.MarkReleased(p.ID) })
}

func (s *Server) resetChecklist(w http.ResponseWriter, r *http.Request) {
	s.checklistAction(w, r, func(p store.Project) error { return s.store.ResetChecklist(p.ID) })
}

// itemAction runs do for the checklist item in the path and re-renders its project.
func (s *Server) itemAction(w http.ResponseWriter, r *http.Request, do func(id int64) error) {
	id := pathID(r)
	projectID, err := s.store.ChecklistItemProject(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := do(id); err != nil {
		s.fail(w, "checklist item", err)
		return
	}
	p, err := s.store.Project(projectID)
	if err != nil {
		s.fail(w, "project", err)
		return
	}
	s.respondReleases(w, r, p.Slug, r.FormValue("edit") != "")
}

func (s *Server) toggleChecklistItem(w http.ResponseWriter, r *http.Request) {
	s.itemAction(w, r, s.store.ToggleChecklistItem)
}

func (s *Server) deleteChecklistItem(w http.ResponseWriter, r *http.Request) {
	s.itemAction(w, r, s.store.DeleteChecklistItem)
}

func (s *Server) updateChecklistItem(w http.ResponseWriter, r *http.Request) {
	s.itemAction(w, r, func(id int64) error {
		it := store.ChecklistItem{ID: id, Phase: phase(r), Title: strings.TrimSpace(r.FormValue("title")),
			Detail: strings.TrimSpace(r.FormValue("detail")), URL: strings.TrimSpace(r.FormValue("url")), Command: strings.TrimSpace(r.FormValue("command"))}
		if it.Title == "" {
			it.Title = "—"
		}
		return s.store.UpdateChecklistItem(it)
	})
}

func phase(r *http.Request) string {
	if r.FormValue("phase") == "after" {
		return "after"
	}
	return "before"
}
