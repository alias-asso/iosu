package cli

import (
	"context"
	"flag"
	"fmt"

	"github.com/alias-asso/iosu/internal/app"
	"github.com/alias-asso/iosu/internal/store/sqlc"
)

func problemCommands() []command {
	return []command{{
		name:  "difficulty create",
		short: "create a difficulty tier",
		setup: func(fs *flag.FlagSet) func(context.Context, *app.App) error {
			name := fs.String("name", "", "difficulty name")
			points := fs.Int64("points", 0, "points awarded per solved part")
			return func(ctx context.Context, a *app.App) error {
				if err := required("name", *name); err != nil {
					return err
				}
				if err := a.CreateDifficulty(ctx, *name, *points); err != nil {
					return err
				}
				fmt.Printf("created difficulty %q\n", *name)
				return nil
			}
		},
	}, {
		name:  "problem create",
		short: "create a problem",
		setup: func(fs *flag.FlagSet) func(context.Context, *app.App) error {
			contest := fs.String("contest", "", "contest slug")
			slug := fs.String("slug", "", "problem slug (lowercase, dashes)")
			name := fs.String("name", "", "problem name")
			difficulty := fs.String("difficulty", "", "difficulty name")
			author := fs.String("author", "", "problem author")
			parts := fs.Int64("parts", 1, "number of parts")
			multiplier := fs.Float64("multiplier", 1.0, "points multiplier")
			adder := fs.Int64("adder", 0, "extra points")
			return func(ctx context.Context, a *app.App) error {
				for f, v := range map[string]string{"contest": *contest, "slug": *slug, "name": *name, "difficulty": *difficulty} {
					if err := required(f, v); err != nil {
						return err
					}
				}
				p, err := a.CreateProblem(ctx, app.CreateProblemInput{
					ContestSlug:      *contest,
					DifficultyName:   *difficulty,
					Slug:             *slug,
					Name:             *name,
					Author:           *author,
					Parts:            *parts,
					PointsMultiplier: *multiplier,
					PointsAdder:      *adder,
				})
				if err != nil {
					return err
				}
				fmt.Printf("created problem %q (id %d)\n", p.Slug, p.ID)
				return nil
			}
		},
	}, {
		name:  "problem update",
		short: "change an existing problem",
		setup: func(fs *flag.FlagSet) func(context.Context, *app.App) error {
			id := fs.Int64("id", 0, "problem ID (required)")
			slug := fs.String("slug", "", "new slug")
			name := fs.String("name", "", "new name")
			author := fs.String("author", "", "new author")
			parts := fs.Int64("parts", 0, "new number of parts")
			multiplier := fs.Float64("multiplier", 0, "new points multiplier")
			adder := fs.Int64("adder", 0, "new extra points")
			return func(ctx context.Context, a *app.App) error {
				if *id == 0 {
					return fmt.Errorf("-id is required")
				}
				if err := a.UpdateProblem(ctx, sqlc.UpdateProblemParams{
					ID:               *id,
					Slug:             optStr(*slug),
					Name:             optStr(*name),
					Author:           optStr(*author),
					Parts:            optInt(*parts),
					PointsMultiplier: optFloat(*multiplier),
					PointsAdder:      optInt(*adder),
				}); err != nil {
					return err
				}
				fmt.Println("problem updated")
				return nil
			}
		},
	}}
}
