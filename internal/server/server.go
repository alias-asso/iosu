package server

import (
	"context"
	"log"
	"net/http"

	"github.com/alias-asso/iosu/internal/config"
	"github.com/alias-asso/iosu/internal/service"
)

type authService interface {
	Login(ctx context.Context, input service.LoginInput) (string, error)
	Register(ctx context.Context, input service.RegisterInput) error
	CreateDefaultAdmin(ctx context.Context)
	BatchRegister(ctx context.Context, csvContent string) error
}

type contestService interface {
	CreateContest(ctx context.Context, input service.CreateContestInput) error
	UpdateContest(ctx context.Context, input service.UpdateContestInput) error
}

type Server struct {
	contestService contestService
	authService    authService
	mux            *http.ServeMux
	cfg            *config.Config
}

func NewServer(contestService *service.ContestService, authService *service.AuthService, mux *http.ServeMux, cfg *config.Config) *Server {
	return &Server{
		contestService: contestService,
		authService:    authService,
		mux:            mux,
		cfg:            cfg,
	}
}

// Define a basic http server and connect to the database
// func NewServer(config config.Config) (Server, error) {
// 	mux := http.NewServeMux()

// 	err, db := database.ConnectDb(&config)
// 	if err != nil {
// 		log.Fatalln("Error connecting to the database")
// 	}

// 	return Server{
// 		mux: mux,
// 		cfg: &config,
// 	}, nil
// }

func (s *Server) SetupServer(config config.Config) error {
	registerRoutes(s)

	return nil
}

func (s *Server) Start(port string) {
	log.Printf("Listening on %s:%s", "localhost", port)
	err := http.ListenAndServe(":"+port, s.mux)
	if err != nil {
		log.Panicf("Error launching server : %s\n", err)
	}
}
