package server

import (
	"io/fs"
	"net/http"
)

func registerRoutes(s *Server) {
	s.mux.HandleFunc("GET /", s.withAuth(true, s.getIndex))

	// static assets
	sub, _ := fs.Sub(content, "static")
	fileServer := http.FileServer(http.FS(sub))
	s.mux.Handle("GET /static/", http.StripPrefix("/static", fileServer))

	// Login routes
	s.mux.HandleFunc("GET /login", s.withAuth(true, s.getLogin))
	s.mux.HandleFunc("POST /login", s.postLogin)
	s.mux.HandleFunc("POST /logout", s.postLogout)

	// Activate registerRoutes
	s.mux.HandleFunc("POST /activate", s.postActivate)
	s.mux.HandleFunc("GET /activate/{code}", s.getActivate)

	// Register routes
	// s.mux.HandleFunc("POST /register", s.withAuth(false, s.withAdmin(s.postRegisterAccount)))
	// s.mux.HandleFunc("POST /register/batch", s.withAuth(false, s.withAdmin(s.postBatchCreateAccounts)))

	// Contest routes
	s.mux.HandleFunc("GET /contest/{slug}/", s.withAuth(false, s.getContest))
	s.mux.HandleFunc("GET /contest/{slug}/leaderboard", s.withAuth(false, s.getLeaderboard))
	// s.mux.HandleFunc("POST /contest", s.withAuth(false, s.withAdmin(s.postCreateContest)))

	// Problem routes
	s.mux.HandleFunc("GET /contest/{contest_slug}/{problem_slug}/", s.withAuth(false, s.getProblem))
	s.mux.HandleFunc("GET /contest/{contest_slug}/{problem_slug}/img/{img}", s.withAuth(false, s.getProblemImages))
	s.mux.HandleFunc("POST /contest/{contest_slug}/{problem_slug}/submit/{part}", s.withAuth(false, s.postSubmit))
	s.mux.HandleFunc("GET /contest/{contest_slug}/{problem_slug}/input/", s.withAuth(false, s.getInput))

	// Public routes
	s.mux.HandleFunc("GET /help", s.withAuth(true, s.getHelp))
	s.mux.HandleFunc("GET /rules", s.withAuth(true, s.getRules))
	s.mux.HandleFunc("GET /legal", s.withAuth(true, s.getLegal))
	s.mux.HandleFunc("GET /credits", s.withAuth(true, s.getCredits))
}
