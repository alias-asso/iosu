package web

import (
	"context"
	"log"
	"net/http"
	"runtime/debug"

	"github.com/alias-asso/iosu/internal/app"
)

// ctxKey is unexported so no other package can collide with these keys.
type ctxKey int

const userKey ctxKey = iota

// maxBodyBytes caps request bodies. Answers and login forms are tiny; the CSV
// upload path lives in the CLI.
const maxBodyBytes = 1 << 20

// userFrom returns the authenticated user, if any.
func userFrom(r *http.Request) (app.User, bool) {
	u, ok := r.Context().Value(userKey).(app.User)
	return u, ok
}

func withUser(r *http.Request, u app.User) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), userKey, u))
}

// optionalAuth attaches the user when there is a valid token, and otherwise
// lets the request through anonymously.
func (s *Server) optionalAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if u, ok := s.authenticate(r); ok {
			r = withUser(r, u)
		}
		next(w, r)
	}
}

// requireAuth sends anonymous visitors to the login page.
func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, ok := s.authenticate(r)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next(w, withUser(r, u))
	}
}

// authenticate resolves the request's cookie to a user. The user is re-read
// from the database so a deactivated or deleted account stops working at once.
func (s *Server) authenticate(r *http.Request) (app.User, bool) {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return app.User{}, false
	}
	claims, err := s.parseToken(cookie.Value)
	if err != nil {
		return app.User{}, false
	}
	user, err := s.app.User(r.Context(), claims.UserID)
	if err != nil || !user.Activated {
		return app.User{}, false
	}
	return user, true
}

// recoverPanic keeps one bad request from taking down the process.
func recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				log.Printf("panic serving %s %s: %v\n%s", r.Method, r.URL.Path, v, debug.Stack())
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "same-origin")
		// Everything is served from this origin; htmx is vendored under /static.
		h.Set("Content-Security-Policy",
			"default-src 'self'; img-src 'self' data:; style-src 'self'; script-src 'self'; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}

func limitBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		next.ServeHTTP(w, r)
	})
}
