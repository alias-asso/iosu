package server

import "net/http"

func registerRoutes(s *Server) {
	s.mux.HandleFunc("GET /", s.withAuth(true, s.getIndex))

	// static assets
	fileServer := http.FileServer(http.Dir("./static/"))
	s.mux.Handle("GET /static/", http.StripPrefix("/static", fileServer))

	// Login routes
	s.mux.HandleFunc("GET /login", s.withAuth(true, s.getLogin))
	s.mux.HandleFunc("POST /login", s.postLogin)

	// Register routes
	s.mux.HandleFunc("POST /register", s.withAuth(false, s.withAdmin(s.postRegisterAccount)))
	s.mux.HandleFunc("POST /register/batch", s.withAuth(false, s.withAdmin(s.postBatchCreateAccounts)))

	// Contest routes
	s.mux.HandleFunc("GET /contest/{name}", s.withAuth(false, s.getContest))
	s.mux.HandleFunc("POST /contest", s.withAuth(false, s.withAdmin(s.postCreateContest)))

	// Problem routes
	s.mux.HandleFunc("GET /contest/{name}/{slug}", s.withAuth(false, s.getProblem))
	s.mux.HandleFunc("GET /contest/{name}/{slug}/img/{img}", s.withAuth(false, s.getProblemImages))
}
