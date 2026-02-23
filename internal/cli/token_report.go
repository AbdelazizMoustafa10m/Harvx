// Package cli implements the Cobra command hierarchy for the harvx CLI tool.
// This file provides helper functions for writing token count reports to an
// io.Writer (typically os.Stderr when operating in report-only mode).
package cli

import (
	"io"

	"github.com/harvx/harvx/internal/pipeline"
	"github.com/harvx/harvx/internal/tokenizer"
)

// PrintTokenReport writes a formatted token report to w.
// This is the handler for the --token-count flag behavior.
// In report-only mode, callers should pass os.Stderr as w.
func PrintTokenReport(w io.Writer, files []*pipeline.FileDescriptor, tokenizerName string, budget int) {
	report := tokenizer.NewTokenReport(files, tokenizerName, budget)
	_, _ = io.WriteString(w, report.Format())
}

// PrintTopFiles writes a formatted top-N files report to w.
// In report-only mode, callers should pass os.Stderr as w.
// n controls the maximum number of files to display; 0 shows all files.
func PrintTopFiles(w io.Writer, files []*pipeline.FileDescriptor, n int) {
	report := tokenizer.NewTopFilesReport(files, n)
	_, _ = io.WriteString(w, report.Format())
}
