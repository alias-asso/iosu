package server

import (
	"context"
	"html/template"
	"log"
	"net/http"

	"github.com/alias-asso/iosu/internal/config"
	"github.com/alias-asso/iosu/internal/database"
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

type problemService interface {
	GetProblemPartsHtml(ctx context.Context, input service.GetProblemPartHtmlInput) ([]template.HTML, error)
	CreateProblemData(ctx context.Context, input service.CreateProblemDataInput) error
	Submit(ctx context.Context, input service.SubmitInput) (bool, error)
	GetProblems(ctx context.Context, input service.GetProblemsInput) ([]database.Problem, error)
	CreateDifficulty(ctx context.Context, input service.CreateDifficultyInput) error
	GetProblem(ctx context.Context, input service.GetProblemInput) (database.Problem, error)
}

type Server struct {
	contestService contestService
	authService    authService
	problemService problemService
	mux            *http.ServeMux
	cfg            *config.Config
}

func NewServer(contestService *service.ContestService, authService *service.AuthService, problemService *service.ProblemService, mux *http.ServeMux, cfg *config.Config) *Server {
	return &Server{
		contestService: contestService,
		authService:    authService,
		problemService: problemService,
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

func (s *Server) SetupServer(config *config.Config) error {
	registerRoutes(s)
	log.Println("Registered routes.")
	return nil
}

func (s *Server) Start(port string) {
	log.Printf("Listening on %s:%s", "localhost", port)
	err := http.ListenAndServe(":"+port, s.mux)
	if err != nil {
		log.Panicf("Error launching server : %s\n", err)
	}
}
