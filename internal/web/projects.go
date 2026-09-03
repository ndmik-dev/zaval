package web

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/ndmik-dev/zaval/internal/store"
)

type projectsData struct {
	shell
	All      []projectItem // including hidden ones
	Expanded int64
	Error    string
}

func (s *Server) projects(w http.ResponseWriter, r *http.Request) {
	s.respondProjects(w, r, r.URL.Query().Get("e"), "")
}

func (s *Server) respondProjects(w http.ResponseWriter, r *http.Request, expandSlug, errMsg string) {
	sh, err := s.shell("Проєкти", "projects", "")
	if err != nil {
		s.fail(w, "projects", err)
		return
	}
	d := projectsData{shell: sh, Error: errMsg}
	all, err := s.store.Projects()
	if err != nil {
		s.fail(w, "projects", err)
		return
	}
	counts, err := s.store.ProjectCounts()
	if err != nil {
		s.fail(w, "projects", err)
		return
	}
	for _, p := range all {
		d.All = append(d.All, projectItem{p, counts[p.ID]})
		if p.Slug == expandSlug {
			d.Expanded = p.ID
		}
	}
	if r.Header.Get("HX-Request") != "" {
		s.renderPart(w, "projects", "app", d)
		return
	}
	s.render(w, "projects", d)
}

func (s *Server) createProject(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.FormValue("name"))
	kind := r.FormValue("kind")
	if kind != "work" {
		kind = "pet"
	}
	color := r.FormValue("color")
	if color == "" {
		color = map[string]string{"work": "#2F5D50", "pet": "#8A6A30"}[kind]
	}
	if name == "" {
		s.respondProjects(w, r, "", "")
		return
	}
	p, err := s.store.CreateProject(name, slugify(name), kind, color)
	if err != nil {
		s.respondProjects(w, r, "", "Проєкт із такою назвою вже є")
		return
	}
	s.respondProjects(w, r, p.Slug, "")
}

func (s *Server) updateProject(w http.ResponseWriter, r *http.Request) {
	p, err := s.store.Project(pathID(r))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if v := strings.TrimSpace(r.FormValue("name")); v != "" {
		p.Name = v
	}
	if v := slugify(r.FormValue("slug")); v != "" {
		p.Slug = v
	}
	if v := r.FormValue("color"); colorRe.MatchString(v) {
		p.Color = v
	}
	if v := r.FormValue("kind"); v == "work" || v == "pet" {
		p.Kind = v
	}
	p.JiraKey = strings.ToUpper(strings.TrimSpace(r.FormValue("jira_key")))
	p.JiraHost = strings.TrimSpace(r.FormValue("jira_host"))
	p.Repos = strings.TrimSpace(r.FormValue("repos"))
	p.Channels = strings.TrimSpace(r.FormValue("channels"))
	p.OnBoard = r.FormValue("on_board") != ""
	if err := s.store.UpdateProject(p); err != nil {
		s.respondProjects(w, r, p.Slug, "Не збереглося: назва або slug уже зайняті")
		return
	}
	s.respondProjects(w, r, p.Slug, "")
}

func (s *Server) deleteProject(w http.ResponseWriter, r *http.Request) {
	err := s.store.DeleteProject(pathID(r))
	if err == store.ErrHasTasks {
		s.respondProjects(w, r, "", "У проєкті є задачі — спершу перенеси або видали їх")
		return
	}
	if err != nil {
		s.fail(w, "delete project", err)
		return
	}
	s.respondProjects(w, r, "", "")
}

var (
	colorRe   = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
	nonSlugRe = regexp.MustCompile(`[^a-z0-9]+`)
)

// slugify makes the #tag used in ⌘K: ascii letters and digits only.
func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = nonSlugRe.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}
