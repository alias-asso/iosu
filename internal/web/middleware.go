package web

import (
	"context"
	"log"
	"net/http"
	"runtime/debug"

	"github.com/alias-asso/iosu/internal/app"
)

type ctxKey int

const userKey ctxKey = iota

const maxBodyBytes = 1 << 20

func userFrom(r *http.Request) (app.User, bool) {
	u, ok := r.Context().Value(userKey).(app.User)
	return u, ok
}

func withUser(r *http.Request, u app.User) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), userKey, u))
}

func (s *Server) optionalAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if u, ok := s.authenticate(r); ok {
			r = withUser(r, u)
		}
		next(w, r)
	}
}

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

func (s *Server) contestAccess(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("contest")
		if slug == "" {
			slug = r.PathValue("slug")
		}
		contest, err := s.app.Contest(r.Context(), slug)
		if err != nil {
			s.renderError(w, r, err)
			return
		}
		if user, ok := s.authenticate(r); ok {
			next(w, withUser(r, user))
			return
		}
		if contest.Mode == app.ContestModeFreePlay {
			next(w, r)
			return
		}
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}

func (s *Server) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, ok := s.authenticate(r)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		if !u.Admin {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		next(w, withUser(r, u))
	}
}

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
		// Everything is served from this origin, htmx is vendored under /static.
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
