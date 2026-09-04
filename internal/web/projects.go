package web

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ndmik-dev/zaval/internal/store"
)

type projectsData struct {
	shell
	All      []projectItem // including hidden ones
	Expanded *store.Project
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
	for i := range all {
		d.All = append(d.All, projectItem{all[i], counts[all[i].ID]})
		if all[i].Slug == expandSlug {
			d.Expanded = &all[i]
		}
	}
	d.Detail = d.Expanded != nil
	d.Ctx = map[string]string{"page": "projects"}
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
	slug := slugify(name)
	if slug == "" {
		slug = "p" + strconv.FormatInt(time.Now().Unix()%100000, 10)
	}
	p, err := s.store.CreateProject(name, slug, kind, color)
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
	if err := s.store.UpdateProject(p); err != nil {
		s.respondProjects(w, r, p.Slug, "Не збереглося: назва або slug уже зайняті")
		return
	}
	s.respondProjects(w, r, p.Slug, "") // the pane keeps the project it just saved
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

// slugify makes the #tag used in ⌘K: ascii letters and digits only,
// Cyrillic transliterated so «Проєкт» becomes #proiekt.
func slugify(s string) string {
	s = translit.Replace(strings.ToLower(strings.TrimSpace(s)))
	s = nonSlugRe.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

var translit = strings.NewReplacer(
	"а", "a", "б", "b", "в", "v", "г", "h", "ґ", "g", "д", "d", "е", "e", "є", "ie", "ж", "zh", "з", "z",
	"и", "y", "і", "i", "ї", "i", "й", "i", "к", "k", "л", "l", "м", "m", "н", "n", "о", "o", "п", "p",
	"р", "r", "с", "s", "т", "t", "у", "u", "ф", "f", "х", "kh", "ц", "ts", "ч", "ch", "ш", "sh", "щ", "shch",
	"ь", "", "ю", "iu", "я", "ia", "ы", "y", "э", "e", "ё", "e", "ъ", "", "ʼ", "", "'", "",
)
