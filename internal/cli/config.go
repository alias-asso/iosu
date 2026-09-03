package cli

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"

	"github.com/alias-asso/iosu/internal/app"
	"github.com/alias-asso/iosu/internal/store/sqlc"
)

// markdownPages are the four editable pages, each imported from a file by
// "iosu config import".
var markdownPages = []struct {
	flag string
	set  func(*sqlc.UpdateSiteConfigParams, sql.NullString)
}{
	{"help", func(p *sqlc.UpdateSiteConfigParams, v sql.NullString) { p.HelpContent = v }},
	{"rules", func(p *sqlc.UpdateSiteConfigParams, v sql.NullString) { p.RulesContent = v }},
	{"legal", func(p *sqlc.UpdateSiteConfigParams, v sql.NullString) { p.LegalContent = v }},
	{"credits", func(p *sqlc.UpdateSiteConfigParams, v sql.NullString) { p.CreditsContent = v }},
}

func configCommands() []command {
	return []command{{
		name:  "config update",
		short: "change the site texts",
		setup: func(fs *flag.FlagSet) func(context.Context, *app.App) error {
			title := fs.String("site-title", "", "site title")
			main := fs.String("main-text", "", "home page main text")
			secondary := fs.String("secondary-text", "", "home page secondary text")
			contest := fs.String("current-contest", "", "slug of the contest the navigation links to; empty disables it")
			return func(ctx context.Context, a *app.App) error {
				set := wasSet(fs)
				if err := a.UpdateSiteConfig(ctx, sqlc.UpdateSiteConfigParams{
					SiteTitle:      setStr(*title, set["site-title"]),
					MainText:       setStr(*main, set["main-text"]),
					SecondaryText:  setStr(*secondary, set["secondary-text"]),
					CurrentContest: setStr(*contest, set["current-contest"]),
				}); err != nil {
					return err
				}
				fmt.Println("site config updated")
				return nil
			}
		},
	}, {
		name:  "config import",
		short: "load the help, rules, legal or credits page from a markdown file",
		setup: func(fs *flag.FlagSet) func(context.Context, *app.App) error {
			files := make(map[string]*string, len(markdownPages))
			for _, page := range markdownPages {
				files[page.flag] = fs.String(page.flag, "", "path to the "+page.flag+" markdown file")
			}
			return func(ctx context.Context, a *app.App) error {
				var params sqlc.UpdateSiteConfigParams
				var loaded []string
				for _, page := range markdownPages {
					path := *files[page.flag]
					if path == "" {
						continue
					}
					body, err := os.ReadFile(path)
					if err != nil {
						return fmt.Errorf("reading %s: %w", path, err)
					}
					page.set(&params, sql.NullString{String: string(body), Valid: true})
					loaded = append(loaded, page.flag)
				}
				if len(loaded) == 0 {
					return fmt.Errorf("give at least one of -help, -rules, -legal or -credits")
				}
				if err := a.UpdateSiteConfig(ctx, params); err != nil {
					return err
				}
				fmt.Printf("imported: %v\n", loaded)
				return nil
			}
		},
	}}
}
