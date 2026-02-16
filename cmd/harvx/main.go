// Package main is the entry point for the harvx CLI tool.
package main

import (
	"os"

	"github.com/harvx/harvx/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
