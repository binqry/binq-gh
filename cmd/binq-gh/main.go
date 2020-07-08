package main

import (
	"os"

	"github.com/progrhyme/binq-gh/internal/cli"
)

func main() {
	os.Exit(cli.NewCLI(os.Stdout, os.Stderr, os.Stdin).Run(os.Args))
}
