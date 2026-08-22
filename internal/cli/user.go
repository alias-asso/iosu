package cli

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/alias-asso/iosu/internal/app"
)

func userCommands() []command {
	return []command{{
		name:  "user batch-create",
		short: "create accounts from a CSV with username and email columns",
		setup: func(fs *flag.FlagSet) func(context.Context, *app.App) error {
			path := fs.String("i", "", "path to the CSV file")
			return func(ctx context.Context, a *app.App) error {
				if err := required("i", *path); err != nil {
					return err
				}
				body, err := os.ReadFile(*path)
				if err != nil {
					return fmt.Errorf("reading %s: %w", *path, err)
				}
				n, err := a.BatchRegister(ctx, string(body))
				if err != nil {
					return err
				}
				fmt.Printf("created %d accounts; list their activation links with: iosu user pending\n", n)
				return nil
			}
		},
	}, {
		name:  "user pending",
		short: "list accounts that have not been activated, with their links",
		setup: func(fs *flag.FlagSet) func(context.Context, *app.App) error {
			base := fs.String("url", "https://example.org", "site base URL used to build the links")
			return func(ctx context.Context, a *app.App) error {
				pending, err := a.PendingActivations(ctx)
				if err != nil {
					return err
				}
				w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
				fmt.Fprintln(w, "USERNAME\tEMAIL\tEXPIRES\tLINK")
				for _, p := range pending {
					fmt.Fprintf(w, "%s\t%s\t%s\t%s/activate/%s\n",
						p.User.Username, p.User.Email,
						time.Unix(p.ActivationCode.ExpiresAt, 0).Format(timeLayout),
						strings.TrimRight(*base, "/"), p.ActivationCode.Code)
				}
				return w.Flush()
			}
		},
	}, {
		name:  "user passwd",
		short: "set a user's password, read from stdin",
		setup: func(fs *flag.FlagSet) func(context.Context, *app.App) error {
			username := fs.String("username", "", "account to change")
			return func(ctx context.Context, a *app.App) error {
				if err := required("username", *username); err != nil {
					return err
				}
				fmt.Fprint(os.Stderr, "new password (input is echoed): ")
				line, err := bufio.NewReader(os.Stdin).ReadString('\n')
				if err != nil && line == "" {
					return fmt.Errorf("reading password: %w", err)
				}
				if err := a.SetPassword(ctx, *username, strings.TrimRight(line, "\r\n")); err != nil {
					return err
				}
				fmt.Printf("password updated for %s\n", *username)
				return nil
			}
		},
	}, {
		name:  "user admin",
		short: "grant or revoke admin rights",
		setup: func(fs *flag.FlagSet) func(context.Context, *app.App) error {
			username := fs.String("username", "", "account to change")
			revoke := fs.Bool("revoke", false, "revoke instead of granting")
			return func(ctx context.Context, a *app.App) error {
				if err := required("username", *username); err != nil {
					return err
				}
				if err := a.SetAdmin(ctx, *username, !*revoke); err != nil {
					return err
				}
				verb := "granted to"
				if *revoke {
					verb = "revoked from"
				}
				fmt.Printf("admin rights %s %s\n", verb, *username)
				return nil
			}
		},
	}}
}
