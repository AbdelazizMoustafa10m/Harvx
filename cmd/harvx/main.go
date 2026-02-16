// Package main is the entry point for the harvx CLI tool.
package main

import "fmt"

// Build-time metadata injected via ldflags.
// These will move to internal/buildinfo in T-006.
var (
	version   = "dev"
	commit    = "none"
	date      = "unknown"
	goVersion = "unknown"
)

func main() {
	fmt.Println("harvx")
}
