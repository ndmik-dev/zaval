package web

import (
	"embed"
	"hash/fnv"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/ndmik-dev/zaval/internal/store"
)

//go:embed templates
var templateFS embed.FS

//go:embed static
var staticFS embed.FS

var funcs = template.FuncMap{
	"day":       ukDay,
	"dict":      dict,
	"percent":   func(a, b int) int { return a * 100 / b },
	"doneCount": doneCount,
	"hostOf":    hostOf,
	"add":       func(a, b int) int { return a + b },
	"colors":    func() []string { return projectColors },
	"closeURL":  closeURL,
	"without":   without,
}

// without copies a context map minus the given keys.
func without(m map[string]string, keys ...string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	for _, k := range keys {
		delete(out, k)
	}
	return out
}

// closeURL rebuilds the page URL the drawer sits on, without the task.
func closeURL(ctx map[string]string) string {
	path := "/"
	switch ctx["page"] {
	case "journal":
		path = "/journal"
	case "releases":
		path = "/releases"
	}
	q := url.Values{}
	for k, v := range ctx {
		if k != "page" && v != "" {
			q.Set(k, v)
		}
	}
	if len(q) == 0 {
		return path
	}
	return path + "?" + q.Encode()
}

// projectColors are the preset swatches offered in project settings.
var projectColors = []string{"#2F5D50", "#7C4A6B", "#8A6A30", "#4E6B8C", "#7A5C99", "#3E7D6E", "#A0522D", "#5C7A3E"}

func doneCount(items []store.ChecklistItem) int {
	n := 0
	for _, it := range items {
		if it.Done {
			n++
		}
	}
	return n
}

func hostOf(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return raw
	}
	return strings.TrimPrefix(u.Host, "www.")
}

// dict lets a template pass several named values to a sub-template.
func hashStatic() string {
	h := fnv.New64a()
	fs.WalkDir(staticFS, "static", func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			b, _ := staticFS.ReadFile(path)
			h.Write(b)
		}
		return nil
	})
	return strconv.FormatUint(h.Sum64(), 36)
}

func dict(kv ...any) map[string]any {
	m := make(map[string]any, len(kv)/2)
	for i := 0; i+1 < len(kv); i += 2 {
		m[kv[i].(string)] = kv[i+1]
	}
	return m
}

type Server struct {
	mux      *http.ServeMux
	pages    map[string]*template.Template
	store    *store.Store
	auth     *auth
	assetVer string // hash of the embedded static files; busts browser and service-worker caches
}

func New(st *store.Store, password string) *Server {
	s := &Server{mux: http.NewServeMux(), pages: map[string]*template.Template{}, store: st, auth: newAuth(password)}
	s.assetVer = hashStatic()
	s.parseTemplates()

	static, _ := fs.Sub(staticFS, "static")
	s.mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(static)))
	// The service worker must be served from the root to control the whole app.
	s.mux.HandleFunc("GET /sw.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		body, _ := staticFS.ReadFile("static/sw.js")
		w.Write(body)
	})
	s.mux.HandleFunc("GET /login", s.loginPage)
	s.mux.HandleFunc("POST /login", s.login)
	s.mux.HandleFunc("POST /logout", s.logout)
	s.mux.HandleFunc("GET /{$}", s.board)
	s.mux.HandleFunc("GET /journal", s.journal)
	s.mux.HandleFunc("GET /releases", s.releases)
	s.mux.HandleFunc("POST /checklist/{id}/items", s.addChecklistItem)
	s.mux.HandleFunc("POST /checklist/{id}/released", s.markReleased)
	s.mux.HandleFunc("POST /checklist-items/{id}", s.updateChecklistItem)
	s.mux.HandleFunc("POST /checklist-items/{id}/toggle", s.toggleChecklistItem)
	s.mux.HandleFunc("DELETE /checklist-items/{id}", s.deleteChecklistItem)
	s.mux.HandleFunc("POST /checklist-items/{id}/task", s.setChecklistItemTask)
	s.mux.HandleFunc("GET /tasks/options", s.taskOptions)
	s.mux.HandleFunc("GET /projects", s.projects)
	s.mux.HandleFunc("POST /projects", s.createProject)
	s.mux.HandleFunc("POST /projects/{id}", s.updateProject)
	s.mux.HandleFunc("DELETE /projects/{id}", s.deleteProject)
	s.mux.HandleFunc("POST /tasks/reorder", s.reorderTasks)
	s.mux.HandleFunc("GET /palette", s.palette)
	s.mux.HandleFunc("POST /palette", s.createQuick)
	s.mux.HandleFunc("POST /tasks/{id}/state", s.setTaskState)
	s.mux.HandleFunc("POST /tasks/{id}/waiting", s.setWaiting)
	s.mux.HandleFunc("DELETE /tasks/{id}", s.deleteTask)
	s.mux.HandleFunc("POST /tasks/{id}", s.updateTask)
	s.mux.HandleFunc("POST /tasks/{id}/links", s.addLink)
	s.mux.HandleFunc("DELETE /links/{id}", s.deleteLink)
	s.mux.HandleFunc("POST /tasks/{id}/steps", s.addStep)
	s.mux.HandleFunc("POST /steps/{id}/toggle", s.toggleStep)
	s.mux.HandleFunc("DELETE /steps/{id}", s.deleteStep)
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.auth.middleware(s.mux).ServeHTTP(w, r)
}

// Each page gets its own template set: layout + partials + the page file,
// so every page can define its own "content" block.
func (s *Server) parseTemplates() {
	pages, err := fs.Glob(templateFS, "templates/pages/*.html")
	if err != nil {
		log.Fatal(err)
	}
	for _, p := range pages {
		name := p[len("templates/pages/") : len(p)-len(".html")]
		fm := template.FuncMap{"static": func(name string) string { return "/static/" + name + "?v=" + s.assetVer }}
		s.pages[name] = template.Must(template.New("").Funcs(funcs).Funcs(fm).ParseFS(templateFS,
			"templates/layout.html", "templates/partials/*.html", p))
	}
}

func (s *Server) render(w http.ResponseWriter, page string, data any) {
	s.renderPart(w, page, "layout", data)
}

// renderPart executes one named template from a page's set — "layout" for a
// full page, "app" for htmx responses that morph the shell in place.
func (s *Server) renderPart(w http.ResponseWriter, page, name string, data any) {
	t, ok := s.pages[page]
	if !ok {
		http.Error(w, "no such page: "+page, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("render %s/%s: %v", page, name, err)
	}
}
