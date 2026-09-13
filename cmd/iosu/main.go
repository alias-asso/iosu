package main

import (
	"os"

	"github.com/alias-asso/iosu/internal/cli"
)

func main() {
	os.Exit(cli.Main(os.Args[1:]))
}
