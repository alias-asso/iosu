package server

import (
	"errors"
	"io"
	"math/rand"
	"net/http"
	"time"

	"github.com/alias-asso/iosu/internal/config"
	"github.com/alias-asso/iosu/internal/service"
	"github.com/golang-jwt/jwt/v5"
)

// route handler
func (s *Server) postLogin(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	input := service.LoginInput{
		Username: username,
		Password: password,
	}

	token, err := s.authService.Login(r.Context(), input)
	if err != nil {
		// TODO: important : handle error for real
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(time.Hour * 4),
	})

	// w.Header().Set("HX-Redirect", "/admin")
	// w.WriteHeader(http.StatusOK)
	// TODO: replace with ok template
	w.Write([]byte("logged in"))
}

// route handler
func (s *Server) postRegisterAccount(w http.ResponseWriter, r *http.Request) {
	// TODO
}

// route handler
func (s *Server) postBatchCreateAccounts(w http.ResponseWriter, r *http.Request) {
	file, _, err := r.FormFile("accounts")
	if err != nil {
		http.Error(w, "Error retrieving file.", http.StatusBadRequest)
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Error reading file.", http.StatusBadRequest)
		return
	}

	err = s.authService.BatchRegister(r.Context(), string(content))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCSV),
			errors.Is(err, service.ErrInvalidCSVHeader),
			errors.Is(err, service.ErrInvalidInput):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("ok"))
}
