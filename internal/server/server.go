package server

import (
	"context"
	"log"
	"net/http"

	"github.com/alias-asso/iosu/internal/config"
	"github.com/alias-asso/iosu/internal/database"
	"gorm.io/gorm"
)

type Server struct {
	db  *gorm.DB
	mux *http.ServeMux
	cfg *config.Config
}

// Define a basic http server and connect to the database
func NewServer(config config.Config) (Server, error) {
	mux := http.NewServeMux()

	err, db := database.ConnectDb(&config)
	if err != nil {
		log.Fatalln("Error connecting to the database")
	}

	createDefaultAdmin(db, &config, context.Background())

	return Server{
		mux: mux,
		db:  db,
		cfg: &config,
	}, nil
}

func (s *Server) SetupServer(config config.Config) error {
	registerRoutes(s)

	err := database.Migrate(s.db)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) Start(port string) {
	log.Printf("Listening on %s:%s", "localhost", port)
	http.ListenAndServe(":"+port, s.mux)
}
