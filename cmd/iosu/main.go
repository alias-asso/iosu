package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/alias-asso/iosu/internal/config"
	"github.com/alias-asso/iosu/internal/database"
	"github.com/alias-asso/iosu/internal/repository"
	"github.com/alias-asso/iosu/internal/service"
)

type SubCommand struct {
	Name  string
	Short string // one-line description shown in help
	Flags *flag.FlagSet
	Run   func(ctx context.Context, services *Services) error
}

type Command struct {
	Name  string
	Short string
	Subs  []SubCommand
}

type Services struct {
	authService    *service.AuthService
	contestService *service.ContestService
	problemService *service.ProblemService
	configService  *service.ConfigService
}

var configPath string

func defaultConfigPath() string {
	return filepath.Join(fmt.Sprintf("/etc/%s", config.PlateformName), "config.toml")
}

func attachConfigFlag(fs *flag.FlagSet) {
	fs.StringVar(&configPath, "config", defaultConfigPath(), "config file path")
}

func parseConfigFile() (*config.Config, *Services) {
	if _, err := os.Stat(configPath); err != nil {
		if os.IsNotExist(err) {
			fatal("Config file not found.")
		}
		fatal("Unable to read config file.")
	}

	cfg, err := config.ParseConfig(configPath)
	if err != nil {
		fatal("Unable to parse config: " + err.Error())
	}

	err, db := database.ConnectDb(cfg)
	if err != nil {
		fatal("Unable to connect to the database.")
	}

	contestRepo := repository.NewGormContestRepository(db)
	userRepo := repository.NewGormUserRepository(db)
	problemRepo := repository.NewGormProblemRepository(db)
	configRepo := repository.NewGormConfigRepository(db)

	contestService := service.NewConstestService(contestRepo, cfg.DataDirectory)
	authService := service.NewAuthService(userRepo, cfg.JwtKey, cfg.DefaultAdminPassword)
	problemService := service.NewProblemService(problemRepo, &contestService, &authService, cfg.DataDirectory)
	configService := service.NewConfigService(configRepo)

	return cfg, &Services{
		contestService: &contestService,
		authService:    &authService,
		problemService: &problemService,
		configService:  &configService,
	}
}

func buildCommands() []Command {

	ccFlags := flag.NewFlagSet("create", flag.ExitOnError)
	attachConfigFlag(ccFlags)
	ccName := ccFlags.String("name", "", "contest name")
	ccSlug := ccFlags.String("slug", "", "contest slug")
	ccStartTime := ccFlags.String("start-time", "", "contest start time (yyyy-mm-dd hh:mm:ss)")
	ccEndTime := ccFlags.String("end-time", "", "contest end time (yyyy-mm-dd hh:mm:ss)")

	cuContestFlags := flag.NewFlagSet("update", flag.ExitOnError)
	attachConfigFlag(cuContestFlags)
	cuContestID := cuContestFlags.Uint("id", 0, "contest ID (required)")
	cuContestName := cuContestFlags.String("name", "", "new contest name")
	cuContestSlug := cuContestFlags.String("slug", "", "new contest slug")
	cuContestStartTime := cuContestFlags.String("start-time", "", "new start time (yyyy-mm-dd hh:mm:ss)")
	cuContestEndTime := cuContestFlags.String("end-time", "", "new end time (yyyy-mm-dd hh:mm:ss)")

	cdFlags := flag.NewFlagSet("data", flag.ExitOnError)
	attachConfigFlag(cdFlags)
	cdSlug := cdFlags.String("contest", "", "contest slug")
	cdDir := cdFlags.String("directory", "", "path to data directory")

	dcFlags := flag.NewFlagSet("create", flag.ExitOnError)
	attachConfigFlag(dcFlags)
	dcName := dcFlags.String("name", "", "difficulty name")
	dcPoints := dcFlags.Uint("points", 0, "difficulty points")

	pcFlags := flag.NewFlagSet("create", flag.ExitOnError)
	attachConfigFlag(pcFlags)
	pcContest := pcFlags.String("contest", "", "problem contest")
	pcName := pcFlags.String("name", "", "problem name")
	pcDiff := pcFlags.String("difficulty", "", "problem difficulty")
	pcAuth := pcFlags.String("author", "", "problem author")
	pcSlug := pcFlags.String("slug", "", "problem slug")
	pcMult := pcFlags.Float64("multiplier", 1.0, "points multiplier")
	pcAdder := pcFlags.Uint("adder", 0, "points to add")
	pcParts := pcFlags.Uint("parts", 1, "number of parts")

	puFlags := flag.NewFlagSet("update", flag.ExitOnError)
	attachConfigFlag(puFlags)
	puID := puFlags.Uint("id", 0, "problem ID (required)")
	puSlug := puFlags.String("slug", "", "new problem slug")
	puName := puFlags.String("name", "", "new problem name")
	puAuthor := puFlags.String("author", "", "new problem author")
	puMult := puFlags.Float64("multiplier", 0, "new points multiplier")
	puAdder := puFlags.Uint("adder", 0, "new points adder")
	puParts := puFlags.Uint("parts", 0, "new number of parts")

	cuFlags := flag.NewFlagSet("update", flag.ExitOnError)
	attachConfigFlag(cuFlags)
	cuTitle := cuFlags.String("site-title", "", "site title")
	cuMain := cuFlags.String("main-text", "", "main text")
	cuSec := cuFlags.String("secondary-text", "", "secondary text")
	cuContest := cuFlags.String("current-contest", "", "current contest")

	cihFlags := flag.NewFlagSet("import-help", flag.ExitOnError)
	attachConfigFlag(cihFlags)
	cihFile := cihFlags.String("i", "", "path to markdown file")

	cirFlags := flag.NewFlagSet("import-rules", flag.ExitOnError)
	attachConfigFlag(cirFlags)
	cirFile := cirFlags.String("i", "", "path to markdown file")

	cilFlags := flag.NewFlagSet("import-legal", flag.ExitOnError)
	attachConfigFlag(cilFlags)
	cilFile := cilFlags.String("i", "", "path to markdown file")

	cicFlags := flag.NewFlagSet("import-credits", flag.ExitOnError)
	attachConfigFlag(cicFlags)
	cicFile := cicFlags.String("i", "", "path to markdown file")

	abFlags := flag.NewFlagSet("batch-create", flag.ExitOnError)
	attachConfigFlag(abFlags)
	abFile := abFlags.String("i", "", "path to CSV file")

	acFlags := flag.NewFlagSet("unactivated", flag.ExitOnError)
	attachConfigFlag(acFlags)
	acUrl := acFlags.String("url", "url", "url to use")

	return []Command{
		{
			Name:  "contest",
			Short: "manage contests",
			Subs: []SubCommand{
				{
					Name:  "create",
					Short: "create a new contest",
					Flags: ccFlags,
					Run: func(ctx context.Context, svc *Services) error {
						start, err := time.Parse("2006-01-02 15:04:05", *ccStartTime)
						if err != nil {
							return fmt.Errorf("unable to parse start time")
						}
						end, err := time.Parse("2006-01-02 15:04:05", *ccEndTime)
						if err != nil {
							return fmt.Errorf("unable to parse end time")
						}
						err = svc.contestService.CreateContest(ctx, service.CreateContestInput{
							Name:      *ccName,
							Slug:      *ccSlug,
							StartTime: start,
							EndTime:   end,
						})
						if err != nil {
							return err
						}
						fmt.Println("Contest created successfully.")
						return nil
					},
				},
				{
					Name:  "update",
					Short: "update an existing contest",
					Flags: cuContestFlags,
					Run: func(ctx context.Context, svc *Services) error {
						if *cuContestID == 0 {
							return fmt.Errorf("-id flag is required")
						}

						input := service.UpdateContestInput{ID: *cuContestID}

						if *cuContestName != "" {
							input.Name = cuContestName
						}
						if *cuContestSlug != "" {
							input.Slug = cuContestSlug
						}
						if *cuContestStartTime != "" {
							t, err := time.Parse("2006-01-02 15:04:05", *cuContestStartTime)
							if err != nil {
								return fmt.Errorf("unable to parse start time")
							}
							input.StartTime = &t
						}
						if *cuContestEndTime != "" {
							t, err := time.Parse("2006-01-02 15:04:05", *cuContestEndTime)
							if err != nil {
								return fmt.Errorf("unable to parse end time")
							}
							input.EndTime = &t
						}

						if err := svc.contestService.UpdateContest(ctx, input); err != nil {
							return err
						}
						fmt.Println("Contest updated successfully.")
						return nil
					},
				},
				{
					Name:  "data",
					Short: "import per-user problem data from a directory tree",
					Flags: cdFlags,
					Run: func(ctx context.Context, svc *Services) error {
						if *cdSlug == "" {
							return fmt.Errorf("-contest flag is required")
						}
						if *cdDir == "" {
							return fmt.Errorf("-directory flag is required")
						}
						runContestData(ctx, svc, *cdSlug, *cdDir)
						fmt.Println("Contest data import completed.")
						return nil
					},
				},
			},
		},
		{
			Name:  "difficulty",
			Short: "manage difficulty levels",
			Subs: []SubCommand{
				{
					Name:  "create",
					Short: "create a new difficulty level",
					Flags: dcFlags,
					Run: func(ctx context.Context, svc *Services) error {
						err := svc.problemService.CreateDifficulty(ctx, service.CreateDifficultyInput{
							DifficultyName: *dcName,
							Points:         *dcPoints,
						})
						if err != nil {
							return err
						}
						fmt.Println("Difficulty created successfully.")
						return nil
					},
				},
			},
		},
		{
			Name:  "problem",
			Short: "manage problems",
			Subs: []SubCommand{
				{
					Name:  "create",
					Short: "create a new problem",
					Flags: pcFlags,
					Run: func(ctx context.Context, svc *Services) error {
						err := svc.problemService.CreateProblem(ctx, service.CreateProblemInput{
							ContestName:      *pcContest,
							DifficultyName:   *pcDiff,
							Name:             *pcName,
							Slug:             *pcSlug,
							Author:           *pcAuth,
							PointsMultiplier: pcMult,
							PointsAdder:      pcAdder,
							Parts:            pcParts,
						})
						if err != nil {
							return err
						}
						fmt.Println("Problem created successfully.")
						return nil
					},
				},
				{
					Name:  "update",
					Short: "update an existing problem",
					Flags: puFlags,
					Run: func(ctx context.Context, svc *Services) error {
						if *puID == 0 {
							return fmt.Errorf("-id flag is required")
						}

						input := service.UpdateProblemInput{ID: *puID}

						if *puSlug != "" {
							input.Slug = puSlug
						}
						if *puName != "" {
							input.Name = puName
						}
						if *puAuthor != "" {
							input.Author = puAuthor
						}
						if *puMult != 0 {
							input.PointsMultiplier = puMult
						}
						if *puAdder != 0 {
							input.PointsAdder = puAdder
						}
						if *puParts != 0 {
							input.Parts = puParts
						}

						if err := svc.problemService.UpdateProblem(ctx, input); err != nil {
							return err
						}
						fmt.Println("Problem updated successfully.")
						return nil
					},
				},
			},
		},
		{
			Name:  "config",
			Short: "manage site configuration",
			Subs: []SubCommand{
				{
					Name:  "update",
					Short: "update one or more config fields",
					Flags: cuFlags,
					Run: func(ctx context.Context, svc *Services) error {
						err := svc.configService.UpdateConfig(ctx, service.UpdateConfigInput{
							SiteTitle:      cuTitle,
							MainText:       cuMain,
							SecondaryText:  cuSec,
							CurrentContest: cuContest,
						})
						if err != nil {
							return err
						}
						fmt.Println("Config updated successfully.")
						return nil
					},
				},
				{
					Name:  "import-help",
					Short: "set help page content from a markdown file",
					Flags: cihFlags,
					Run: func(ctx context.Context, svc *Services) error {
						if *cihFile == "" {
							return fmt.Errorf("-i flag is required")
						}
						content, err := os.ReadFile(*cihFile)
						if err != nil {
							return fmt.Errorf("unable to read file: %w", err)
						}
						s := string(content)
						return svc.configService.UpdateConfig(ctx, service.UpdateConfigInput{
							HelpContent: &s,
						})
					},
				},
				{
					Name:  "import-rules",
					Short: "set rules page content from a markdown file",
					Flags: cirFlags,
					Run: func(ctx context.Context, svc *Services) error {
						if *cirFile == "" {
							return fmt.Errorf("-i flag is required")
						}
						content, err := os.ReadFile(*cirFile)
						if err != nil {
							return fmt.Errorf("unable to read file: %w", err)
						}
						s := string(content)
						return svc.configService.UpdateConfig(ctx, service.UpdateConfigInput{
							RulesContent: &s,
						})
					},
				},
				{
					Name:  "import-legal",
					Short: "set legal page content from a markdown file",
					Flags: cilFlags,
					Run: func(ctx context.Context, svc *Services) error {
						if *cilFile == "" {
							return fmt.Errorf("-i flag is required")
						}
						content, err := os.ReadFile(*cilFile)
						if err != nil {
							return fmt.Errorf("unable to read file: %w", err)
						}
						s := string(content)
						return svc.configService.UpdateConfig(ctx, service.UpdateConfigInput{
							LegalContent: &s,
						})
					},
				},
				{
					Name:  "import-credits",
					Short: "set credits page content from a markdown file",
					Flags: cicFlags,
					Run: func(ctx context.Context, svc *Services) error {
						if *cicFile == "" {
							return fmt.Errorf("-i flag is required")
						}
						content, err := os.ReadFile(*cicFile)
						if err != nil {
							return fmt.Errorf("unable to read file: %w", err)
						}
						s := string(content)
						return svc.configService.UpdateConfig(ctx, service.UpdateConfigInput{
							CreditsContent: &s,
						})
					},
				},
			},
		},
		{
			Name:  "auth",
			Short: "manage user accounts",
			Subs: []SubCommand{
				{
					Name:  "batch-create",
					Short: "create accounts in bulk from a CSV file",
					Flags: abFlags,
					Run: func(ctx context.Context, svc *Services) error {
						if *abFile == "" {
							return fmt.Errorf("-i flag is required")
						}
						content, err := os.ReadFile(*abFile)
						if err != nil {
							return fmt.Errorf("unable to read file: %w", err)
						}
						if err := svc.authService.BatchRegister(ctx, string(content)); err != nil {
							return err
						}
						fmt.Println("Accounts created successfully.")
						return nil
					},
				},
				{

					Name:  "unactivated",
					Short: "show all unactivated accounts and their activation links",
					Flags: acFlags,
					Run: func(ctx context.Context, svc *Services) error {
						authService := svc.authService
						ac, err := authService.GetActivationCodes(ctx)
						if err != nil {
							return err
						}
						for _, a := range ac {
							fmt.Printf("%s : %s : https://%s/activate/%s\n", a.User.Username, a.User.Email, *acUrl, a.Code)
						}
						return nil
					},
				},
			},
		},
	}
}

func printGlobalHelp(cmds []Command) {
	fmt.Fprintf(os.Stderr, "Usage:\n  %s <command> <subcommand> [flags]\n\nCommands:\n", os.Args[0])
	w := tabwriter.NewWriter(os.Stderr, 0, 0, 3, ' ', 0)
	for _, c := range cmds {
		for _, s := range c.Subs {
			fmt.Fprintf(w, "  %s %s\t%s\n", c.Name, s.Name, s.Short)
		}
	}
	w.Flush()
	fmt.Fprintf(os.Stderr, "\nRun '%s <command> help' for subcommand list.\n", os.Args[0])
}

func printCommandHelp(cmd Command) {
	fmt.Fprintf(os.Stderr, "Usage:\n  %s %s <subcommand> [flags]\n\nSubcommands:\n", os.Args[0], cmd.Name)
	w := tabwriter.NewWriter(os.Stderr, 0, 0, 3, ' ', 0)
	for _, s := range cmd.Subs {
		fmt.Fprintf(w, "  %-16s\t%s\n", s.Name, s.Short)
	}
	w.Flush()
}

func printSubCommandHelp(cmd Command, sub SubCommand) {
	fmt.Fprintf(os.Stderr, "Usage:\n  %s %s %s [flags]\n\n%s\n\nFlags:\n",
		os.Args[0], cmd.Name, sub.Name, sub.Short)
	sub.Flags.PrintDefaults()
}

func dispatch(cmds []Command) {
	args := os.Args[1:]

	if len(args) == 0 || args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		printGlobalHelp(cmds)
		os.Exit(0)
	}

	cmdName := args[0]
	for _, cmd := range cmds {
		if cmd.Name != cmdName {
			continue
		}

		if len(args) < 2 || args[1] == "help" || args[1] == "-h" || args[1] == "--help" {
			printCommandHelp(cmd)
			os.Exit(0)
		}

		subName := args[1]
		for _, sub := range cmd.Subs {
			if sub.Name != subName {
				continue
			}

			remaining := args[2:]
			if len(remaining) > 0 && (remaining[0] == "help" || remaining[0] == "-h" || remaining[0] == "--help") {
				printSubCommandHelp(cmd, sub)
				os.Exit(0)
			}

			sub.Flags.Parse(remaining)

			_, services := parseConfigFile()
			if err := sub.Run(context.Background(), services); err != nil {
				fmt.Fprintf(os.Stderr, "[Error] %s.\n", err.Error())
				os.Exit(1)
			}
			return
		}

		fmt.Fprintf(os.Stderr, "[Error] Unknown subcommand '%s' for '%s'.\n\n", subName, cmdName)
		printCommandHelp(cmd)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "[Error] Unknown command '%s'.\n\n", cmdName)
	printGlobalHelp(cmds)
	os.Exit(1)
}

func runContestData(ctx context.Context, services *Services, contestSlug, directory string) {
	dirInfo, err := os.Stat(directory)
	if err != nil || !dirInfo.IsDir() {
		fatal("Directory not found or is not a directory: " + directory)
	}

	problemDirs, err := os.ReadDir(directory)
	if err != nil {
		fatal("Unable to read directory: " + err.Error())
	}

	for _, problemDir := range problemDirs {
		if !problemDir.IsDir() {
			continue
		}

		problemSlug := problemDir.Name()
		problemPath := filepath.Join(directory, problemSlug)

		problem, err := services.problemService.GetProblem(ctx, service.GetProblemInput{Slug: problemSlug})
		if err != nil || problem.Contest.Slug != contestSlug {
			fmt.Printf("[Warning] Problem '%s' not found in contest '%s', skipping.\n", problemSlug, contestSlug)
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

			user, err := services.authService.GetUser(ctx, service.GetUserInput{Username: username})
			if err != nil {
				fmt.Printf("[Warning] User '%s' not found, skipping.\n", username)
				continue
			}

			inputBytes, err := os.ReadFile(filepath.Join(userPath, "input.txt"))
			if err != nil {
				fmt.Printf("[Warning] No input.txt for user '%s' / problem '%s', skipping.\n", username, problemSlug)
				continue
			}

			var outputValues []string
			for part := 1; ; part++ {
				b, err := os.ReadFile(filepath.Join(userPath, fmt.Sprintf("output%d.txt", part)))
				if err != nil {
					break
				}
				outputValues = append(outputValues, strings.TrimSpace(string(b)))
			}

			err = services.problemService.CreateProblemData(ctx, service.CreateProblemDataInput{
				UserID:       user.ID,
				Slug:         problemSlug,
				InputValue:   strings.TrimSpace(string(inputBytes)),
				OutputValues: outputValues,
			})
			if err != nil {
				fmt.Printf("[Error] Failed to save data for user '%s' / problem '%s': %s\n", username, problemSlug, err.Error())
				continue
			}
			fmt.Printf("Data saved for user '%s' / problem '%s'.\n", username, problemSlug)
		}
	}
}

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, "[Error]", msg)
	os.Exit(1)
}

func main() {
	dispatch(buildCommands())
}
