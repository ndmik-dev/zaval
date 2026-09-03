package web

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"strings"
	"time"
)

// Single-user auth: one password from the environment. The session cookie is
// an HMAC of a fixed message keyed by the password, so restarts keep sessions
// and changing the password logs every device out.
type auth struct {
	password string
	token    string
}

func newAuth(password string) *auth {
	a := &auth{password: password}
	if password != "" {
		mac := hmac.New(sha256.New, []byte(password))
		mac.Write([]byte("dayboard-session"))
		a.token = hex.EncodeToString(mac.Sum(nil))
	}
	return a
}

const sessionCookie = "dayboard"

func (a *auth) enabled() bool { return a.password != "" }

func (a *auth) loggedIn(r *http.Request) bool {
	if !a.enabled() {
		return true
	}
	c, err := r.Cookie(sessionCookie)
	return err == nil && subtle.ConstantTimeCompare([]byte(c.Value), []byte(a.token)) == 1
}

func (a *auth) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if !a.enabled() || a.loggedIn(r) || p == "/login" || p == "/sw.js" || strings.HasPrefix(p, "/static/") {
			next.ServeHTTP(w, r)
			return
		}
		if r.Header.Get("HX-Request") != "" {
			w.Header().Set("HX-Redirect", "/login")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	})
}

func (s *Server) loginPage(w http.ResponseWriter, r *http.Request) {
	if s.auth.loggedIn(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	s.renderPart(w, "login", "login", map[string]any{"Title": "Вхід", "Error": false})
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if subtle.ConstantTimeCompare([]byte(r.FormValue("password")), []byte(s.auth.password)) != 1 {
		time.Sleep(400 * time.Millisecond)
		w.WriteHeader(http.StatusUnauthorized)
		s.renderPart(w, "login", "login", map[string]any{"Title": "Вхід", "Error": true})
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: s.auth.token, Path: "/", HttpOnly: true,
		Secure: secure(r), SameSite: http.SameSiteLaxMode, MaxAge: 180 * 24 * 3600,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: secure(r)})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func secure(r *http.Request) bool {
	return r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
}
