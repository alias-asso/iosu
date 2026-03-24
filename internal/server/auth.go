package server

import (
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/alias-asso/iosu/internal/service"
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
		MaxAge:   int(time.Hour * 24),
	})

	http.Redirect(w, r, "/", http.StatusFound)
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

func (s *Server) getLogin(w http.ResponseWriter, r *http.Request) {
	s.render(w, r.Context(), "pages/login.gohtml", nil)
}

func (s *Server) postLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:   "token",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
	w.Header().Set("HX-Redirect", "/")
	w.WriteHeader(http.StatusOK)
}

func (s *Server) getActivate(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")

	if len(code) != 32 {
		s.render(w, r.Context(), "pages/error.gohtml", "Invalid or expired activation code.")
		return
	}

	input := service.GetActivationCodeInput{
		ActivationCode: code,
	}

	ac, err := s.authService.GetActivationCode(r.Context(), input)
	if err != nil {
		s.render(w, r.Context(), "pages/error.gohtml", "Invalid or expired activation code.")
		return
	}
	s.render(w, r.Context(), "pages/activate-account.gohtml", struct {
		ActivationCode string
	}{
		ActivationCode: ac.Code})
}

func (s *Server) postActivate(w http.ResponseWriter, r *http.Request) {
	code := r.FormValue("activation-code")
	password := r.FormValue("password")

	if len(code) != 32 {
		http.Error(w, "Invalid or expired activation code", http.StatusBadRequest)
		return
	}

	input := service.ActivateInput{
		ActivationCode: code,
		Password:       password,
	}

	err := s.authService.Activate(r.Context(), input)
	if err != nil && errors.Is(err, service.ErrActivationCodeExpired) {
		http.Error(w, "Invalid ore expired activation code", http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/login", http.StatusFound)
}
