package web

import (
	"errors"
	"net/http"
	"time"

	"github.com/alias-asso/iosu/internal/app"
	"github.com/golang-jwt/jwt/v5"
)

const (
	cookieName = "token"
	tokenTTL   = 24 * time.Hour
)

type claims struct {
	UserID int64 `json:"uid"`
	jwt.RegisteredClaims
}

func (s *Server) issueToken(u app.User) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		UserID: u.ID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   u.Username,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	})
	return token.SignedString([]byte(s.cfg.JWTKey))
}

// parseToken verifies a token, pinning the signing algorithm rather than
// trusting the one named in the header.
func (s *Server) parseToken(raw string) (*claims, error) {
	token, err := jwt.ParseWithClaims(
		raw,
		&claims{},
		func(*jwt.Token) (any, error) { return []byte(s.cfg.JWTKey), nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
	if err != nil {
		return nil, err
	}
	c, ok := token.Claims.(*claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return c, nil
}

func (s *Server) setTokenCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   !s.cfg.DevMode,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(tokenTTL.Seconds()),
	})
}

func (s *Server) clearTokenCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   !s.cfg.DevMode,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
}
