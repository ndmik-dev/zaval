package web

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ndmik-dev/zaval/internal/store"
)

type releaseView struct {
	store.Release
	DateLabel string
	Before    []store.ReleaseItem
	After     []store.ReleaseItem
	OpenTasks int
	NextItem  string
}

type templateView struct {
	Project store.Project
	Before  []store.TemplateItem
	After   []store.TemplateItem
}

type releasesData struct {
	shell
	Upcoming []releaseView
	Past     []releaseView
	Selected *releaseView
	Tpl      *templateView
	Today    string
}

func viewRelease(r store.Release, now time.Time) releaseView {
	v := releaseView{Release: r}
	if r.Date.Valid {
		v.DateLabel = ukDateShort(r.Date.String, now)
	}
	for _, it := range r.Items {
		if it.Phase == "after" {
			v.After = append(v.After, it)
		} else {
			v.Before = append(v.Before, it)
		}
		if !it.Done && v.NextItem == "" {
			v.NextItem = it.Title
		}
	}
	for _, t := range r.Tasks {
		if t.State != "done" {
			v.OpenTasks++
		}
	}
	return v
}

// ukDateShort renders "пт, 5 вересня", or "сьогодні" / "завтра" when that is what it is.
func ukDateShort(ymd string, now time.Time) string {
	d, err := time.ParseInLocation("2006-01-02", ymd, now.Location())
	if err != nil {
		return ymd
	}
	switch {
	case sameDay(d, now):
		return "сьогодні"
	case sameDay(d, now.AddDate(0, 0, 1)):
		return "завтра"
	}
	return fmt.Sprintf("%s, %d %s", ukWeekdaysShort[d.Weekday()], d.Day(), ukMonths[d.Month()-1])
}

func (s *Server) releases(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	id, _ := strconv.ParseInt(q.Get("r"), 10, 64)
	s.respondReleases(w, r, id, q.Get("tpl"))
}

func (s *Server) respondReleases(w http.ResponseWriter, r *http.Request, selected int64, tplSlug string) {
	now := time.Now()
	sh, err := s.shell("Релізи", "releases", "")
	if err != nil {
		s.fail(w, "releases", err)
		return
	}
	d := releasesData{shell: sh, Today: now.Format("2006-01-02")}
	up, past, err := s.store.Releases()
	if err != nil {
		s.fail(w, "releases", err)
		return
	}
	for _, rel := range up {
		d.Upcoming = append(d.Upcoming, viewRelease(rel, now))
	}
	for _, rel := range past {
		d.Past = append(d.Past, viewRelease(rel, now))
	}
	if tplSlug != "" {
		p, err := s.store.ProjectBySlug(tplSlug)
		if err == nil {
			items, err := s.store.Templates(p.ID)
			if err != nil {
				s.fail(w, "templates", err)
				return
			}
			tv := &templateView{Project: p}
			for _, it := range items {
				if it.Phase == "after" {
					tv.After = append(tv.After, it)
				} else {
					tv.Before = append(tv.Before, it)
				}
			}
			d.Tpl = tv
		}
	} else {
		if selected == 0 && len(up) > 0 {
			selected = up[0].ID
		}
		if selected != 0 {
			if rel, err := s.store.Release(selected); err == nil {
				v := viewRelease(rel, now)
				d.Selected = &v
			}
		}
	}
	if r.Header.Get("HX-Request") != "" {
		s.renderPart(w, "releases", "app", d)
		return
	}
	s.render(w, "releases", d)
}

func (s *Server) createRelease(w http.ResponseWriter, r *http.Request) {
	projectID, _ := strconv.ParseInt(r.FormValue("project_id"), 10, 64)
	name := strings.TrimSpace(r.FormValue("name"))
	if projectID == 0 || name == "" {
		s.respondReleases(w, r, 0, "")
		return
	}
	rel, err := s.store.CreateRelease(projectID, name, r.FormValue("date"))
	if err != nil {
		s.fail(w, "create release", err)
		return
	}
	s.respondReleases(w, r, rel.ID, "")
}

func (s *Server) updateRelease(w http.ResponseWriter, r *http.Request) {
	id := pathID(r)
	rel, err := s.store.Release(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		name = rel.Name
	}
	if err := s.store.UpdateRelease(id, name, r.FormValue("date")); err != nil {
		s.fail(w, "update release", err)
		return
	}
	s.respondReleases(w, r, id, "")
}

func (s *Server) setReleased(w http.ResponseWriter, r *http.Request) {
	id := pathID(r)
	if err := s.store.SetReleased(id, r.FormValue("released") != "0"); err != nil {
		s.fail(w, "set released", err)
		return
	}
	s.respondReleases(w, r, id, "")
}

func (s *Server) deleteRelease(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteRelease(pathID(r)); err != nil {
		s.fail(w, "delete release", err)
		return
	}
	s.respondReleases(w, r, 0, "")
}

func (s *Server) addReleaseItem(w http.ResponseWriter, r *http.Request) {
	id := pathID(r)
	if title := strings.TrimSpace(r.FormValue("title")); title != "" {
		if _, err := s.store.AddReleaseItem(id, phase(r), title, "", "", ""); err != nil {
			s.fail(w, "add item", err)
			return
		}
	}
	s.respondReleases(w, r, id, "")
}

func (s *Server) toggleReleaseItem(w http.ResponseWriter, r *http.Request) {
	s.releaseItemAction(w, r, s.store.ToggleReleaseItem)
}

func (s *Server) deleteReleaseItem(w http.ResponseWriter, r *http.Request) {
	s.releaseItemAction(w, r, s.store.DeleteReleaseItem)
}

func (s *Server) releaseItemAction(w http.ResponseWriter, r *http.Request, do func(int64) error) {
	id := pathID(r)
	relID, err := s.store.ReleaseItemRelease(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := do(id); err != nil {
		s.fail(w, "release item", err)
		return
	}
	s.respondReleases(w, r, relID, "")
}

func (s *Server) addTemplateItem(w http.ResponseWriter, r *http.Request) {
	p, err := s.store.Project(pathID(r))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if title := strings.TrimSpace(r.FormValue("title")); title != "" {
		if _, err := s.store.AddTemplateItem(p.ID, phase(r), title, "", "", ""); err != nil {
			s.fail(w, "add template item", err)
			return
		}
	}
	s.respondReleases(w, r, 0, p.Slug)
}

func (s *Server) updateTemplateItem(w http.ResponseWriter, r *http.Request) {
	id := pathID(r)
	projectID, err := s.store.TemplateItemProject(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	it := store.TemplateItem{ID: id, Phase: phase(r), Title: strings.TrimSpace(r.FormValue("title")),
		Detail: strings.TrimSpace(r.FormValue("detail")), URL: strings.TrimSpace(r.FormValue("url")), Command: strings.TrimSpace(r.FormValue("command"))}
	if it.Title == "" {
		it.Title = "—"
	}
	if err := s.store.UpdateTemplateItem(it); err != nil {
		s.fail(w, "update template item", err)
		return
	}
	p, _ := s.store.Project(projectID)
	s.respondReleases(w, r, 0, p.Slug)
}

func (s *Server) deleteTemplateItem(w http.ResponseWriter, r *http.Request) {
	id := pathID(r)
	projectID, err := s.store.TemplateItemProject(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := s.store.DeleteTemplateItem(id); err != nil {
		s.fail(w, "delete template item", err)
		return
	}
	p, _ := s.store.Project(projectID)
	s.respondReleases(w, r, 0, p.Slug)
}

func (s *Server) setTaskRelease(w http.ResponseWriter, r *http.Request) {
	id := pathID(r)
	relID, _ := strconv.ParseInt(r.FormValue("release_id"), 10, 64)
	if err := s.store.SetTaskRelease(id, relID); err != nil {
		s.fail(w, "task release", err)
		return
	}
	s.respondTask(w, r, id)
}

func phase(r *http.Request) string {
	if r.FormValue("phase") == "after" {
		return "after"
	}
	return "before"
}
