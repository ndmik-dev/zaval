package web

import (
	"embed"
	"html/template"
	"io/fs"
	"log"
	"net/http"

	"github.com/ndmik-dev/zaval/internal/store"
)

//go:embed templates
var templateFS embed.FS

//go:embed static
var staticFS embed.FS

var funcs = template.FuncMap{
	"day":  ukDay,
	"dict": dict,
}

// dict lets a template pass several named values to a sub-template.
func dict(kv ...any) map[string]any {
	m := make(map[string]any, len(kv)/2)
	for i := 0; i+1 < len(kv); i += 2 {
		m[kv[i].(string)] = kv[i+1]
	}
	return m
}

type Server struct {
	mux   *http.ServeMux
	pages map[string]*template.Template
	store *store.Store
}

func New(st *store.Store) *Server {
	s := &Server{mux: http.NewServeMux(), pages: map[string]*template.Template{}, store: st}
	s.parseTemplates()

	static, _ := fs.Sub(staticFS, "static")
	s.mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(static)))
	s.mux.HandleFunc("GET /{$}", s.board)
	s.mux.HandleFunc("POST /tasks", s.createTask)
	s.mux.HandleFunc("POST /tasks/{id}/state", s.setTaskState)
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
	s.mux.ServeHTTP(w, r)
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
		s.pages[name] = template.Must(template.New("").Funcs(funcs).ParseFS(templateFS,
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
