package output

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Format constants define the supported output format identifiers.
const (
	// FormatMarkdown selects Markdown rendering.
	FormatMarkdown = "markdown"

	// FormatXML selects XML rendering.
	FormatXML = "xml"
)

// Default output filename constants.
const (
	// DefaultOutputBase is the base filename (without extension) for the default output file.
	DefaultOutputBase = "harvx-output"

	// ExtensionMarkdown is the file extension for Markdown output.
	ExtensionMarkdown = ".md"

	// ExtensionXML is the file extension for XML output.
	ExtensionXML = ".xml"
)

// NewRenderer returns a Renderer for the given format string. It returns a
// *MarkdownRenderer for FormatMarkdown and a *XMLRenderer for FormatXML.
// An error is returned for unknown format values.
func NewRenderer(format string) (Renderer, error) {
	switch strings.ToLower(format) {
	case FormatMarkdown:
		return NewMarkdownRenderer(), nil
	case FormatXML:
		return NewXMLRenderer(), nil
	default:
		return nil, fmt.Errorf("unknown output format: %q", format)
	}
}

// ExtensionForFormat returns the file extension for the given format string.
// It returns ".xml" for FormatXML and ".md" for everything else (including
// FormatMarkdown and unknown formats).
func ExtensionForFormat(format string) string {
	switch strings.ToLower(format) {
	case FormatXML:
		return ExtensionXML
	default:
		return ExtensionMarkdown
	}
}

// DefaultOutputPath returns the default output file path for the given format.
// For example, "markdown" yields "harvx-output.md" and "xml" yields "harvx-output.xml".
func DefaultOutputPath(format string) string {
	return DefaultOutputBase + ExtensionForFormat(format)
}

// ResolveOutputPath determines the final output file path using the following
// precedence (highest to lowest):
//
//  1. outputFlag -- the CLI --output / -o flag value
//  2. profileOutput -- the output path from the TOML profile config
//  3. DefaultOutputPath(format) -- the default based on format
//
// If the resolved path has no file extension, the correct extension for the
// format is appended (.md or .xml).
func ResolveOutputPath(outputFlag, profileOutput, format string) string {
	resolved := outputFlag
	if resolved == "" {
		resolved = profileOutput
	}
	if resolved == "" {
		return DefaultOutputPath(format)
	}

	// Append extension if the resolved path has none.
	if filepath.Ext(resolved) == "" {
		resolved += ExtensionForFormat(format)
	}

	return resolved
}
