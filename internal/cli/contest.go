package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/alias-asso/iosu/internal/app"
	"github.com/alias-asso/iosu/internal/store/sqlc"
)

func contestCommands() []command {
	return []command{{
		name:  "contest create",
		short: "create a contest",
		setup: func(fs *flag.FlagSet) func(context.Context, *app.App) error {
			slug := fs.String("slug", "", "contest slug (lowercase, dashes)")
			name := fs.String("name", "", "contest name")
			start := fs.String("start-time", "", "start time ("+timeLayout+")")
			end := fs.String("end-time", "", "end time ("+timeLayout+")")
			return func(ctx context.Context, a *app.App) error {
				for f, v := range map[string]string{"slug": *slug, "name": *name, "start-time": *start, "end-time": *end} {
					if err := required(f, v); err != nil {
						return err
					}
				}
				startAt, err := parseTime("start-time", *start)
				if err != nil {
					return err
				}
				endAt, err := parseTime("end-time", *end)
				if err != nil {
					return err
				}
				c, err := a.CreateContest(ctx, app.CreateContestInput{
					Slug: *slug, Name: *name, StartTime: startAt, EndTime: endAt,
				})
				if err != nil {
					return err
				}
				fmt.Printf("created contest %q (id %d)\n", c.Slug, c.ID)
				return nil
			}
		},
	}, {
		name:  "contest update",
		short: "change an existing contest",
		setup: func(fs *flag.FlagSet) func(context.Context, *app.App) error {
			id := fs.Int64("id", 0, "contest ID (required)")
			slug := fs.String("slug", "", "new slug")
			name := fs.String("name", "", "new name")
			start := fs.String("start-time", "", "new start time ("+timeLayout+")")
			end := fs.String("end-time", "", "new end time ("+timeLayout+")")
			return func(ctx context.Context, a *app.App) error {
				if *id == 0 {
					return fmt.Errorf("-id is required")
				}
				startAt, err := optTime("start-time", *start)
				if err != nil {
					return err
				}
				endAt, err := optTime("end-time", *end)
				if err != nil {
					return err
				}
				if err := a.UpdateContest(ctx, sqlc.UpdateContestParams{
					ID:      *id,
					Slug:    optStr(*slug),
					Name:    optStr(*name),
					StartAt: startAt,
					EndAt:   endAt,
				}); err != nil {
					return err
				}
				fmt.Println("contest updated")
				return nil
			}
		},
	}, {
		name:  "contest list",
		short: "list contests",
		setup: func(fs *flag.FlagSet) func(context.Context, *app.App) error {
			return func(ctx context.Context, a *app.App) error {
				contests, err := a.Contests(ctx)
				if err != nil {
					return err
				}
				w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
				fmt.Fprintln(w, "ID\tSLUG\tNAME\tSTART\tEND")
				for _, c := range contests {
					fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n", c.ID, c.Slug, c.Name,
						time.Unix(c.StartAt, 0).Format(timeLayout),
						time.Unix(c.EndAt, 0).Format(timeLayout))
				}
				return w.Flush()
			}
		},
	}, {
		name:  "contest data",
		short: "import per-user inputs and outputs from a directory",
		setup: func(fs *flag.FlagSet) func(context.Context, *app.App) error {
			contest := fs.String("contest", "", "contest slug")
			dir := fs.String("directory", "", "directory holding <problem>/<user>/input.txt")
			return func(ctx context.Context, a *app.App) error {
				if err := required("contest", *contest); err != nil {
					return err
				}
				if err := required("directory", *dir); err != nil {
					return err
				}
				return importData(ctx, a, *contest, *dir)
			}
		},
	}}
}

// importData walks <dir>/<problem-slug>/<username>/{input.txt,outputN.txt} and
// stores what it finds. Problems and users it cannot resolve are skipped with a
// warning rather than aborting the run.
func importData(ctx context.Context, a *app.App, contestSlug, dir string) error {
	problemDirs, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("reading %s: %w", dir, err)
	}

	imported, skipped := 0, 0
	for _, pd := range problemDirs {
		if !pd.IsDir() {
			continue
		}
		problemSlug := pd.Name()

		problem, err := a.ProblemIn(ctx, contestSlug, problemSlug)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skipping %s: not a problem of contest %s\n", problemSlug, contestSlug)
			skipped++
			continue
		}

		userDirs, err := os.ReadDir(filepath.Join(dir, problemSlug))
		if err != nil {
			fmt.Fprintf(os.Stderr, "skipping %s: %v\n", problemSlug, err)
			skipped++
			continue
		}

		for _, ud := range userDirs {
			if !ud.IsDir() {
				continue
			}
			username := ud.Name()
			userPath := filepath.Join(dir, problemSlug, username)

			user, err := a.UserByUsername(ctx, username)
			if err != nil {
				fmt.Fprintf(os.Stderr, "skipping %s/%s: no such user\n", problemSlug, username)
				skipped++
				continue
			}

			input, err := os.ReadFile(filepath.Join(userPath, "input.txt"))
			if err != nil {
				fmt.Fprintf(os.Stderr, "skipping %s/%s: %v\n", problemSlug, username, err)
				skipped++
				continue
			}

			outputs := make([]string, 0, problem.Problem.Parts)
			for part := int64(1); part <= problem.Problem.Parts; part++ {
				b, err := os.ReadFile(filepath.Join(userPath, fmt.Sprintf("output%d.txt", part)))
				if err != nil {
					break
				}
				outputs = append(outputs, strings.TrimSpace(string(b)))
			}

			if err := a.SetProblemData(ctx, user.ID, problemSlug, strings.TrimSpace(string(input)), outputs); err != nil {
				fmt.Fprintf(os.Stderr, "skipping %s/%s: %v\n", problemSlug, username, err)
				skipped++
				continue
			}
			imported++
		}
	}

	fmt.Printf("imported %d, skipped %d\n", imported, skipped)
	return nil
}
