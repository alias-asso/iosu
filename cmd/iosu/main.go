// Command iosu administers the contest platform from the server's shell.
package main

import (
	"os"

	"github.com/alias-asso/iosu/internal/cli"
)

func main() {
	os.Exit(cli.Main(os.Args[1:]))
}
