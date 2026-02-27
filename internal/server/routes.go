package server

func registerRoutes(s *Server) {
	s.Mux.HandleFunc("/", s.getNotFound)

	// Login routes
	s.Mux.HandleFunc("GET /login", s.getLogin)
	s.Mux.HandleFunc("POST /login", s.postLogin)

	// Register routes
	s.Mux.HandleFunc("POST /register", s.withAuth(s.withAdmin(s.postRegisterAccount)))
	s.Mux.HandleFunc("POST /register/batch", s.withAuth(s.withAdmin(s.postBatchCreateAccounts)))

	// Contest routes
	s.Mux.HandleFunc("POST /contest", s.withAuth(s.withAdmin(s.postCreateContest)))
}
