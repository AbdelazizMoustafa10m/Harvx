// Package main is the entry point for the gosync CLI tool.
package main

import (
	"fmt"
	"os"

	"github.com/example/gosync/cmd"
)

var version = "dev"

func main() {
	cmd.SetVersion(version)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}