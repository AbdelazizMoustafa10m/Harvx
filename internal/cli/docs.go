// Package cli implements the Cobra command hierarchy for the harvx CLI tool.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

// docsCmd is a hidden parent command for documentation generation subcommands.
// It is used during the build process to produce man pages and other reference
// documentation from the Cobra command tree.
var docsCmd = &cobra.Command{
	Use:    "docs",
	Short:  "Generate documentation",
	Hidden: true,
}

// docsManCmd generates man pages for all registered commands. The pages are
// written to the directory specified by --output-dir (default: ./man).
var docsManCmd = &cobra.Command{
	Use:   "man",
	Short: "Generate man pages for all commands",
	RunE:  runDocsMan,
}

func init() {
	docsManCmd.Flags().String("output-dir", "./man", "directory to write man pages to")
	docsCmd.AddCommand(docsManCmd)
	rootCmd.AddCommand(docsCmd)
}

// runDocsMan generates man pages for the entire command tree and writes them
// to the configured output directory. It reports the number of pages generated.
func runDocsMan(cmd *cobra.Command, _ []string) error {
	outputDir, _ := cmd.Flags().GetString("output-dir")

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("creating output directory %s: %w", outputDir, err)
	}

	header := &doc.GenManHeader{
		Title:   "HARVX",
		Section: "1",
		Source:  "Harvx",
		Manual:  "Harvx Manual",
	}

	if err := doc.GenManTree(cmd.Root(), header, outputDir); err != nil {
		return fmt.Errorf("generating man pages: %w", err)
	}

	// Count generated files.
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return fmt.Errorf("reading output directory: %w", err)
	}

	count := 0
	for _, e := range entries {
		if !e.IsDir() {
			count++
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Generated %d man pages in %s\n", count, outputDir)
	return nil
}