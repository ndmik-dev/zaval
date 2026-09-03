package web

import (
	"embed"
	"html/template"
	"io/fs"
	"log"
	"net/http"
)

//go:embed templates
var templateFS embed.FS

//go:embed static
var staticFS embed.FS

type Server struct {
	mux   *http.ServeMux
	pages map[string]*template.Template
}

func New() *Server {
	s := &Server{mux: http.NewServeMux(), pages: map[string]*template.Template{}}
	s.parseTemplates()

	static, _ := fs.Sub(staticFS, "static")
	s.mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(static)))
	s.mux.HandleFunc("GET /{$}", s.board)
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
		s.pages[name] = template.Must(template.ParseFS(templateFS,
			"templates/layout.html", "templates/partials/*.html", p))
	}
}

func (s *Server) render(w http.ResponseWriter, page string, data any) {
	t, ok := s.pages[page]
	if !ok {
		http.Error(w, "no such page: "+page, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, "layout", data); err != nil {
		log.Printf("render %s: %v", page, err)
	}
}

func (s *Server) board(w http.ResponseWriter, r *http.Request) {
	s.render(w, "board", map[string]any{"Title": "Дошка"})
}
