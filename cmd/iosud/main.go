package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/alias-asso/iosu/internal/config"
	"github.com/alias-asso/iosu/internal/database"
	"github.com/alias-asso/iosu/internal/repository"
	"github.com/alias-asso/iosu/internal/server"
	"github.com/alias-asso/iosu/internal/service"
)

var configDirPath string = fmt.Sprintf("/etc/%s", config.PlateformName)
var configPath = flag.String("c", filepath.Join(configDirPath, "config.toml"), "Config file path.")

func main() {
	flag.Parse()

	// Check if config file exist and is readable
	if _, err := os.Stat(*configPath); err != nil {
		if os.IsNotExist(err) {
			log.Fatalln("Error : config file not found.")
		} else {
			log.Fatalln("Error : unable to read config file.")
		}
	}

	config, err := config.ParseConfig(*configPath)
	if err != nil {
		log.Fatalln("Error parsing config : " + err.Error())
	}

	err, db := database.ConnectDb(config)
	if err != nil {
		log.Fatalln("Error connecting to the database")
	}

	contestRepo := repository.NewGormContestRepository(db)

	contestService := &service.ContestService{
		Repo:    contestRepo,
		DataDir: config.DataDirectory,
	}

	mux := http.NewServeMux()
	server := &server.Server{
		ContestService: contestService,
		Mux:            mux,
		Cfg:            config,
	}

	err = database.Migrate(db)
	if err != nil {
		return err
	}

	createDefaultAdmin(db, &config, context.Background())

	server.Start(config.ServerPort)
}
