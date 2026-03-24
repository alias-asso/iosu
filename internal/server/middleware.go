package server

import (
	"context"
	"net/http"

	"github.com/alias-asso/iosu/internal/service"
	"github.com/golang-jwt/jwt/v5"
)

func (s *Server) withAuth(giveAccess bool, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("token")
		if err != nil {
			if giveAccess {
				ctx := context.WithValue(r.Context(), "logged_in", false)
				next(w, r.WithContext(ctx))
				return
			}
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		token, err := jwt.ParseWithClaims(cookie.Value, &service.Claims{}, func(token *jwt.Token) (any, error) {
			return []byte(s.cfg.JwtKey), nil
		})

		if err != nil || !token.Valid {
			if giveAccess {
				ctx := context.WithValue(r.Context(), "logged_in", false)
				next(w, r.WithContext(ctx))
				return
			}
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		claims, ok := token.Claims.(*service.Claims)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		ctx := context.WithValue(r.Context(), "claims", claims)
		ctx = context.WithValue(ctx, "logged_in", true)
		next(w, r.WithContext(ctx))
	}
}

func (s *Server) withAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value("claims").(*service.Claims)
		if !claims.Admin {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		ctx := context.WithValue(r.Context(), "claims", &claims)
		next(w, r.WithContext(ctx))
	}
}
