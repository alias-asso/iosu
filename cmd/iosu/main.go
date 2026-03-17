package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/alias-asso/iosu/internal/config"
	"github.com/alias-asso/iosu/internal/database"
	"github.com/alias-asso/iosu/internal/repository"
	"github.com/alias-asso/iosu/internal/service"
)

var (
	configPath string

	contestCreateCmd       = flag.NewFlagSet("create", flag.ExitOnError)
	contestCreateName      = contestCreateCmd.String("name", "", "contest name")
	contestCreateSlug      = contestCreateCmd.String("slug", "", "contest slug")
	contestCreateStartTime = contestCreateCmd.String("start-time", "", "contest start time (yyyy-mm-dd hh:mm:ss)")
	contestCreateEndTime   = contestCreateCmd.String("end-time", "", "contest end time (yyyy-mm-dd hh:mm:ss)")

	difficultyCreateCmd    = flag.NewFlagSet("create", flag.ExitOnError)
	difficultyCreateName   = difficultyCreateCmd.String("name", "", "difficulty name")
	difficultyCreatePoints = difficultyCreateCmd.Uint("points", 0, "difficulty points")

	problemCreateCmd         = flag.NewFlagSet("create", flag.ExitOnError)
	problemCreateContestName = problemCreateCmd.String("contest", "", "problem contest")
	problemCreateName        = problemCreateCmd.String("name", "", "problem name")
	problemCreateDifficulty  = problemCreateCmd.String("difficulty", "", "problem name")
	problemCreateSlug        = problemCreateCmd.String("slug", "", "problem slug")
	problemCreatePointsMult  = problemCreateCmd.Float64("multiplier", 1.0, "points multiplier")
	problemCreatePointsAdd   = problemCreateCmd.Uint("adder", 0, "how many points to add")
	problemCreateParts       = problemCreateCmd.Uint("parts", 1, "number of parts")
)

type Services struct {
	authService    *service.AuthService
	contestService *service.ContestService
	problemService *service.ProblemService
}

func setupCommonFlags() {
	for _, fs := range []*flag.FlagSet{contestCreateCmd, difficultyCreateCmd, problemCreateCmd} {
		fs.StringVar(
			&configPath,
			"config",
			filepath.Join(fmt.Sprintf("/etc/%s", config.PlateformName), "config.toml"),
			"config file path",
		)
	}
}

func parseConfigFile() (*config.Config, *Services) {
	if _, err := os.Stat(configPath); err != nil {
		if os.IsNotExist(err) {
			fmt.Println("[Error] Config file not found.")
			os.Exit(1)
		} else {
			fmt.Println("[Error] Unable to read config file.")
			os.Exit(1)
		}
	}

	config, err := config.ParseConfig(configPath)
	if err != nil {
		fmt.Println("[Error] Unable to parse config : " + err.Error())
		os.Exit(1)
	}

	err, db := database.ConnectDb(config)
	if err != nil {
		fmt.Println("[Error] Unable to connect to the database.")
	}

	contestRepo := repository.NewGormContestRepository(db)
	userRepo := repository.NewGormUserRepository(db)
	problemRepo := repository.NewGormProblemRepository(db)

	contestService := service.NewConstestService(contestRepo, config.DataDirectory)
	authService := service.NewAuthService(userRepo, config.JwtKey, config.DefaultAdminPassword)
	problemService := service.NewProblemService(problemRepo, &contestService, &authService, config.DataDirectory)

	return config, &Services{
		contestService: &contestService,
		authService:    &authService,
		problemService: &problemService,
	}
}

func main() {
	setupCommonFlags()

	if len(os.Args) < 2 {
		fmt.Println("[Error] Expected a subcommand.")
		os.Exit(1)
	}
	switch os.Args[1] {
	case "contest":

		if len(os.Args) < 3 {
			fmt.Println("[Error] Expected a subcommand.")
			os.Exit(1)
		}
		switch os.Args[2] {
		case "create":
			contestCreateCmd.Parse(os.Args[3:])

			_, services := parseConfigFile()

			contestName := *contestCreateName
			contestSlug := *contestCreateSlug
			contestStartTime, err := time.Parse("2006-01-02 15:04:05", *contestCreateStartTime)
			if err != nil {
				fmt.Println("[Error] Unable to parse start time.")
				os.Exit(1)
			}
			contestEndTime, err := time.Parse("2006-01-02 15:04:05", *contestCreateEndTime)
			if err != nil {
				fmt.Println("[Error] Unable to parse end time.")
				os.Exit(1)
			}
			input := service.CreateContestInput{
				Name:      contestName,
				Slug:      contestSlug,
				StartTime: contestStartTime,
				EndTime:   contestEndTime,
			}
			err = services.contestService.CreateContest(context.Background(), input)
			if err != nil {
				fmt.Println("[Error] " + err.Error() + ".")
				os.Exit(1)
			}
			fmt.Println("Contest created successfully.")
		}
	case "difficulty":
		if len(os.Args) < 3 {
			fmt.Println("[Error] Expected a subcommand.")
			os.Exit(1)
		}
		switch os.Args[2] {
		case "create":
			difficultyCreateCmd.Parse(os.Args[3:])
			_, services := parseConfigFile()
			input := service.CreateDifficultyInput{
				DifficultyName: *difficultyCreateName,
				Points:         *difficultyCreatePoints,
			}
			err := services.problemService.CreateDifficulty(context.Background(), input)

			if err != nil {
				fmt.Println("[Error] " + err.Error() + ".")
				os.Exit(1)
			}
			fmt.Println("Difficulty created successfully.")
		}
	case "problem":
		if len(os.Args) < 3 {
			fmt.Println("[Error] Expected a subcommand.")
			os.Exit(1)
		}
		switch os.Args[2] {
		case "create":
			problemCreateCmd.Parse(os.Args[3:])
			_, services := parseConfigFile()
			input := service.CreateProblemInput{
				ContestName:      *problemCreateContestName,
				DifficultyName:   *problemCreateDifficulty,
				Name:             *problemCreateName,
				Slug:             *problemCreateSlug,
				PointsMultiplier: problemCreatePointsMult,
				PointsAdder:      problemCreatePointsAdd,
				Parts:            problemCreateParts,
			}
			err := services.problemService.CreateProblem(context.Background(), input)

			if err != nil {
				fmt.Println("[Error] " + err.Error() + ".")
				os.Exit(1)
			}
			fmt.Println("Problem created successfully.")
		}
	default:
		fmt.Println("[Error] Unknown subcommand.")
		os.Exit(1)
	}
}
