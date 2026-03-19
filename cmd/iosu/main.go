package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

	contestDataCmd         = flag.NewFlagSet("data", flag.ExitOnError)
	contestDataContestSlug = contestDataCmd.String("contest", "", "contest slug")
	contestDataDirectory   = contestDataCmd.String("directory", "", "path to data directory")

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

	configUpdateCmd            = flag.NewFlagSet("update", flag.ExitOnError)
	configUpdateSiteTitle      = configUpdateCmd.String("site-title", "", "site title")
	configUpdateMainText       = configUpdateCmd.String("main-text", "", "main text")
	configUpdateSecondaryText  = configUpdateCmd.String("secondary-text", "", "secondary text")
	configUpdateCurrentContest = configUpdateCmd.String("current-contest", "", "current contest")
)

type Services struct {
	authService    *service.AuthService
	contestService *service.ContestService
	problemService *service.ProblemService
	configService  *service.ConfigService
}

func setupCommonFlags() {
	for _, fs := range []*flag.FlagSet{contestCreateCmd, contestDataCmd, difficultyCreateCmd, problemCreateCmd, configUpdateCmd} {
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
	configRepo := repository.NewGormConfigRepository(db)

	contestService := service.NewConstestService(contestRepo, config.DataDirectory)
	authService := service.NewAuthService(userRepo, config.JwtKey, config.DefaultAdminPassword)
	problemService := service.NewProblemService(problemRepo, &contestService, &authService, config.DataDirectory)
	configService := service.NewConfigService(configRepo)

	return config, &Services{
		contestService: &contestService,
		authService:    &authService,
		problemService: &problemService,
		configService:  &configService,
	}
}

func runContestData(ctx context.Context, services *Services, contestSlug, directory string) {
	dirInfo, err := os.Stat(directory)
	if err != nil || !dirInfo.IsDir() {
		fmt.Println("[Error] Directory not found or is not a directory: " + directory)
		os.Exit(1)
	}

	problemDirs, err := os.ReadDir(directory)
	if err != nil {
		fmt.Println("[Error] Unable to read directory: " + err.Error())
		os.Exit(1)
	}

	for _, problemDir := range problemDirs {
		if !problemDir.IsDir() {
			continue
		}

		problemSlug := problemDir.Name()
		problemPath := filepath.Join(directory, problemSlug)

		// Verify the problem exists in the contest.
		input := service.GetProblemInput{
			Slug: problemSlug,
		}
		problem, err := services.problemService.GetProblem(ctx, input)
		if err != nil || problem.Contest.Slug != contestSlug {
			fmt.Printf("[Warning] Problem '%s' not found in contest '%s', skipping directory.\n", problemSlug, contestSlug)
			continue
		}

		userDirs, err := os.ReadDir(problemPath)
		if err != nil {
			fmt.Printf("[Warning] Unable to read problem directory '%s': %s, skipping.\n", problemSlug, err.Error())
			continue
		}

		for _, userDir := range userDirs {
			if !userDir.IsDir() {
				continue
			}

			username := userDir.Name()
			userPath := filepath.Join(problemPath, username)

			// Verify the user exists.
			input := service.GetUserInput{
				Username: username,
			}
			user, err := services.authService.GetUser(ctx, input)
			if err != nil {
				fmt.Printf("[Warning] User '%s' not found, skipping directory.\n", username)
				continue
			}

			// Read input.txt.
			inputPath := filepath.Join(userPath, "input.txt")
			inputBytes, err := os.ReadFile(inputPath)
			if err != nil {
				fmt.Printf("[Warning] No input.txt for user '%s' / problem '%s', skipping.\n", username, problemSlug)
				continue
			}
			inputValue := strings.TrimSpace(string(inputBytes))

			// Read all output files (output1.txt, output2.txt, ...).
			var outputValues []string
			for part := 1; ; part++ {
				outputPath := filepath.Join(userPath, fmt.Sprintf("output%d.txt", part))
				outputBytes, err := os.ReadFile(outputPath)
				if err != nil {
					break // No more parts.
				}
				outputValues = append(outputValues, strings.TrimSpace(string(outputBytes)))
			}

			createDataInput := service.CreateProblemDataInput{
				UserID:       user.ID,
				Slug:         problemSlug,
				InputValue:   inputValue,
				OutputValues: outputValues,
			}

			if err := services.problemService.CreateProblemData(ctx, createDataInput); err != nil {
				fmt.Printf("[Error] Failed to save data for user '%s' / problem '%s': %s\n", username, problemSlug, err.Error())
				continue
			}

			fmt.Printf("Data saved for user '%s' / problem '%s'.\n", username, problemSlug)
		}
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

		case "data":
			contestDataCmd.Parse(os.Args[3:])

			if *contestDataContestSlug == "" {
				fmt.Println("[Error] -contest flag is required.")
				os.Exit(1)
			}
			if *contestDataDirectory == "" {
				fmt.Println("[Error] -directory flag is required.")
				os.Exit(1)
			}

			_, services := parseConfigFile()
			runContestData(context.Background(), services, *contestDataContestSlug, *contestDataDirectory)
			fmt.Println("Contest data import completed.")
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
	case "config":
		if len(os.Args) < 3 {
			fmt.Println("[Error] Expected a subcommand.")
			os.Exit(1)
		}
		switch os.Args[2] {
		case "update":
			configUpdateCmd.Parse(os.Args[3:])
			_, services := parseConfigFile()
			input := service.UpdateConfigInput{
				SiteTitle:      configUpdateSiteTitle,
				MainText:       configUpdateMainText,
				SecondaryText:  configUpdateSecondaryText,
				CurrentContest: configUpdateCurrentContest,
			}
			err := services.configService.UpdateConfig(context.Background(), input)

			if err != nil {
				fmt.Println("[Error] " + err.Error() + ".")
				os.Exit(1)
			}
			fmt.Println("Config updated successfully.")
		}
	default:
		fmt.Println("[Error] Unknown subcommand.")
		os.Exit(1)
	}
}
