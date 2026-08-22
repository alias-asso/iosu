// Package cli implements the iosu administration command.
package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/alias-asso/iosu/internal/app"
	"github.com/alias-asso/iosu/internal/config"
	"github.com/alias-asso/iosu/internal/store"
)

// command is one "iosu <group> <action>". setup registers the command's flags
// on fs and returns the function that runs it, closing over those flags.
type command struct {
	name  string
	short string
	setup func(fs *flag.FlagSet) func(context.Context, *app.App) error
}

func commands() []command {
	var cmds []command
	cmds = append(cmds, contestCommands()...)
	cmds = append(cmds, problemCommands()...)
	cmds = append(cmds, configCommands()...)
	cmds = append(cmds, userCommands()...)
	return cmds
}

// Main runs the CLI and returns the process exit code.
func Main(args []string) int {
	cmds := commands()

	cmd, rest, ok := match(cmds, args)
	if !ok {
		if len(args) > 0 && args[0] != "help" && args[0] != "-h" && args[0] != "--help" {
			fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", strings.Join(args, " "))
			usage(cmds)
			return 1
		}
		usage(cmds)
		return 0
	}

	fs := flag.NewFlagSet("iosu "+cmd.name, flag.ExitOnError)
	configPath := fs.String("config", config.DefaultPath, "config file path")
	run := cmd.setup(fs)
	if err := fs.Parse(rest); err != nil {
		return 2
	}

	cfg, err := config.Parse(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		return 1
	}
	db, err := store.Open(cfg.DBPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "database: %v\n", err)
		return 1
	}
	defer db.Close()

	if err := run(context.Background(), app.New(db, cfg.DataDir)); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	return 0
}

// match finds the command whose two-word name starts args.
func match(cmds []command, args []string) (command, []string, bool) {
	if len(args) < 2 {
		return command{}, nil, false
	}
	name := args[0] + " " + args[1]
	for _, c := range cmds {
		if c.name == name {
			return c, args[2:], true
		}
	}
	return command{}, nil, false
}

func usage(cmds []command) {
	fmt.Fprintln(os.Stderr, "usage: iosu <command> [flags]\n\ncommands:")
	w := tabwriter.NewWriter(os.Stderr, 0, 0, 2, ' ', 0)
	for _, c := range cmds {
		fmt.Fprintf(w, "  %s\t%s\n", c.name, c.short)
	}
	w.Flush()
	fmt.Fprintln(os.Stderr, "\nrun 'iosu <command> -h' for a command's flags")
}
