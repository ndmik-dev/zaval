package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/ndmik-dev/zaval/internal/store"
)

func testServer(t *testing.T, password string) *Server {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	st.Seed()
	return New(st, password)
}

func get(s *Server, path string, cookie *http.Cookie) *httptest.ResponseRecorder {
	r := httptest.NewRequest("GET", path, nil)
	if cookie != nil {
		r.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	return w
}

func TestAuth(t *testing.T) {
	s := testServer(t, "secret")
	if w := get(s, "/", nil); w.Code != http.StatusSeeOther || w.Header().Get("Location") != "/login" {
		t.Fatalf("anonymous board: %d %s", w.Code, w.Header().Get("Location"))
	}
	if w := get(s, "/static/app.css", nil); w.Code != http.StatusOK {
		t.Errorf("static must be public: %d", w.Code)
	}
	if w := get(s, "/login", nil); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "password") {
		t.Errorf("login page: %d", w.Code)
	}

	r := httptest.NewRequest("POST", "/login", strings.NewReader(url.Values{"password": {"wrong"}}.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized || len(w.Result().Cookies()) != 0 {
		t.Fatalf("wrong password: %d", w.Code)
	}

	r = httptest.NewRequest("POST", "/login", strings.NewReader(url.Values{"password": {"secret"}}.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()
	s.ServeHTTP(w, r)
	cookies := w.Result().Cookies()
	if w.Code != http.StatusSeeOther || len(cookies) != 1 || !cookies[0].HttpOnly {
		t.Fatalf("login: %d cookies=%d", w.Code, len(cookies))
	}
	if w := get(s, "/", cookies[0]); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "Беклог") {
		body := w.Body.String()
		if len(body) > 300 {
			body = body[len(body)-300:]
		}
		t.Errorf("board with session: %d ...%s", w.Code, body)
	}
	forged := &http.Cookie{Name: sessionCookie, Value: "nope"}
	if w := get(s, "/", forged); w.Code != http.StatusSeeOther {
		t.Errorf("forged cookie accepted: %d", w.Code)
	}
	r = httptest.NewRequest("GET", "/", nil)
	r.Header.Set("HX-Request", "true")
	w = httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized || w.Header().Get("HX-Redirect") != "/login" {
		t.Errorf("htmx anonymous: %d %q", w.Code, w.Header().Get("HX-Redirect"))
	}
}

func TestNoPassword(t *testing.T) {
	s := testServer(t, "")
	if w := get(s, "/", nil); w.Code != http.StatusOK {
		t.Fatalf("open mode: %d", w.Code)
	}
}

// Every page must render for an anonymous-free server, with and without a drawer.
func TestPagesRender(t *testing.T) {
	s := testServer(t, "")
	st := s.store
	atl, _ := st.ProjectBySlug("atl")
	st.CreateTask(atl.ID, "x", "now")
	item, _ := st.AddChecklistItem(atl.ID, "before", "line")
	st.MarkReleased(atl.ID)
	item, _ = st.AddChecklistItem(atl.ID, "before", "line 2")
	for _, path := range []string{"/", "/backlog", "/releases", "/releases?p=atl&i=" + itoa(item.ID), "/projects", "/projects?e=atl"} {
		w := get(s, path, nil)
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "</html>") {
			t.Errorf("%s: %d, body ends %q", path, w.Code, tail(w.Body.String()))
		}
	}
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

func tail(s string) string {
	if len(s) > 120 {
		return s[len(s)-120:]
	}
	return s
}
