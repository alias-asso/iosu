package server

func registerRoutes(s *Server) {
	s.mux.HandleFunc("/", s.getNotFound)

	// Login routes
	s.mux.HandleFunc("GET /login", s.getLogin)
	s.mux.HandleFunc("POST /login", s.postLogin)

	// Register routes
	s.mux.HandleFunc("POST /register", s.withAuth(s.withAdmin(s.postRegisterAccount)))
	s.mux.HandleFunc("POST /register/batch", s.withAuth(s.withAdmin(s.postBatchCreateAccounts)))

	// Contest routes
	s.mux.HandleFunc("POST /contest", s.withAuth(s.withAdmin(s.postCreateContest)))
}
