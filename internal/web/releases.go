package web

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ndmik-dev/zaval/internal/store"
)

// A "release" is the project's checklist being ticked off: nothing to create,
// no versions or dates. «Зарелізено» archives the lines and empties the list.
type checklistView struct {
	Project store.Project
	Items   []store.ChecklistItem
	Done    int
	Total   int
	History []historyView
}

type historyView struct {
	store.ReleaseRecord
	When string
}

type releasesData struct {
	shell
	Work     []projectItem
	Current  *checklistView
	OpenItem *store.ChecklistItem // line shown in the side panel, if any
}

func (s *Server) checklistView(p store.Project, now time.Time) (*checklistView, error) {
	items, err := s.store.Checklist(p.ID)
	if err != nil {
		return nil, err
	}
	v := &checklistView{Project: p, Items: items, Total: len(items)}
	for _, it := range items {
		if it.Done {
			v.Done++
		}
	}
	hist, err := s.store.ReleaseHistory(p.ID, 10)
	if err != nil {
		return nil, err
	}
	for _, h := range hist {
		if len(h.Items) == 0 {
			continue // records from before lines were archived
		}
		d := localDay(h.ReleasedAt, now.Location())
		v.History = append(v.History, historyView{h, ukDateShort(d.Format("2006-01-02"), now)})
	}
	return v, nil
}

func (s *Server) releases(w http.ResponseWriter, r *http.Request) {
	s.respondReleases(w, r)
}

// respondReleases reads project, open line and open task from the query or the
// posted form, so every action lands back on the same view.
func (s *Server) respondReleases(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	slug := r.FormValue("p")
	openID, _ := strconv.ParseInt(r.FormValue("t"), 10, 64)
	itemID, _ := strconv.ParseInt(r.FormValue("i"), 10, 64)
	now := time.Now()
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
		slug = current.Slug
		d.Current, err = s.checklistView(*current, now)
		if err != nil {
			s.fail(w, "checklist", err)
			return
		}
	}
	// A task link inside the line panel opens the task instead of the line.
	if openID != 0 {
		d.Open = s.openTask(openID, now)
	}
	if d.Open == nil && itemID != 0 {
		if it, err := s.store.ChecklistItem(itemID); err == nil {
			d.OpenItem = &it
		}
	}
	itemVal := ""
	if d.OpenItem != nil {
		itemVal = strconv.FormatInt(d.OpenItem.ID, 10)
	}
	d.Ctx = map[string]string{"page": "releases", "p": slug, "i": itemVal}
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
	r.ParseForm()
	r.Form.Set("p", p.Slug)
	s.respondReleases(w, r)
}

func (s *Server) addChecklistItem(w http.ResponseWriter, r *http.Request) {
	s.checklistAction(w, r, func(p store.Project) error {
		title := strings.TrimSpace(r.FormValue("title"))
		if title == "" {
			return nil
		}
		_, err := s.store.AddChecklistItem(p.ID, "before", title)
		return err
	})
}

func (s *Server) markReleased(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	r.Form.Del("i")
	s.checklistAction(w, r, func(p store.Project) error { return s.store.MarkReleased(p.ID) })
}

// itemAction runs do for the checklist line in the path and re-renders its project.
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
	r.ParseForm()
	r.Form.Set("p", p.Slug)
	s.respondReleases(w, r)
}

func (s *Server) toggleChecklistItem(w http.ResponseWriter, r *http.Request) {
	s.itemAction(w, r, s.store.ToggleChecklistItem)
}

func (s *Server) deleteChecklistItem(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	r.Form.Del("i")
	s.itemAction(w, r, s.store.DeleteChecklistItem)
}

func (s *Server) updateChecklistItem(w http.ResponseWriter, r *http.Request) {
	s.itemAction(w, r, func(id int64) error {
		title := strings.TrimSpace(r.FormValue("title"))
		if title == "" {
			title = "—"
		}
		return s.store.UpdateChecklistItem(id, title, "before")
	})
}

func (s *Server) setChecklistItemTask(w http.ResponseWriter, r *http.Request) {
	taskID, _ := strconv.ParseInt(r.FormValue("task_id"), 10, 64)
	s.itemAction(w, r, func(id int64) error { return s.store.SetChecklistItemTask(id, taskID) })
}

// taskOptions renders the picker in the line panel: open tasks and recently
// closed ones that match the typed text.
func (s *Server) taskOptions(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("text"))
	itemID, _ := strconv.ParseInt(r.URL.Query().Get("item"), 10, 64)
	found, err := s.searchTasks(q, 8)
	if err != nil {
		s.fail(w, "search", err)
		return
	}
	s.renderPart(w, "releases", "task_options", map[string]any{"Tasks": found, "Item": itemID, "Query": q})
}
