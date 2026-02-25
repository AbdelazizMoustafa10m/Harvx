// Package workflows implements high-level workflow commands for the harvx CLI.
// Each workflow (brief, slice, review-slice) composes pipeline stages with
// workflow-specific logic for file selection, rendering, and output routing.
package workflows

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ModuleMapEntry describes a single top-level directory in the repository
// with a human-readable one-line description of its purpose.
type ModuleMapEntry struct {
	// Name is the directory name (e.g., "cmd", "internal", "docs").
	Name string

	// Description is a short one-line purpose description.
	Description string
}

// knownDirectories maps well-known directory names to their conventional
// descriptions. This lookup is used by GenerateModuleMap to provide meaningful
// descriptions for standard project layout directories.
var knownDirectories = map[string]string{
	"cmd":           "CLI entry points",
	"internal":      "Private packages (not importable externally)",
	"pkg":           "Public library packages",
	"lib":           "Shared libraries",
	"src":           "Source code",
	"app":           "Application code",
	"api":           "API definitions and handlers",
	"docs":          "Documentation",
	"doc":           "Documentation",
	"test":          "Test files and fixtures",
	"tests":         "Test files and fixtures",
	"testdata":      "Test fixture data",
	"scripts":       "Automation and build scripts",
	"tools":         "Development tools and utilities",
	"build":         "Build configuration and packaging",
	"deploy":        "Deployment configuration",
	"deployments":   "Deployment configuration",
	"config":        "Configuration files",
	"configs":       "Configuration files",
	"migrations":    "Database migrations",
	"vendor":        "Vendored dependencies",
	"node_modules":  "Node.js dependencies (vendored)",
	"assets":        "Static assets (images, fonts, etc.)",
	"static":        "Static files served directly",
	"public":        "Publicly served files",
	"templates":     "Template files",
	"views":         "View templates",
	"components":    "UI components",
	"pages":         "Page components or routes",
	"routes":        "Route definitions",
	"middleware":    "HTTP/gRPC middleware",
	"services":     "Service layer implementations",
	"models":       "Data models and schemas",
	"types":        "Type definitions",
	"utils":        "Utility functions",
	"helpers":      "Helper functions",
	"hooks":        "React hooks or git hooks",
	"grammars":     "Parser grammars (e.g., tree-sitter WASM)",
	"proto":        "Protocol Buffer definitions",
	"protos":       "Protocol Buffer definitions",
	"web":          "Web frontend code",
	"frontend":     "Frontend application code",
	"backend":      "Backend application code",
	"server":       "Server implementation",
	"client":       "Client implementation",
	"examples":     "Example code and usage",
	"fixtures":     "Test fixtures",
	"mocks":        "Mock implementations for testing",
	"e2e":          "End-to-end tests",
	"integration":  "Integration tests",
	"benchmarks":   "Performance benchmarks",
	"bin":          "Compiled binaries",
	"dist":         "Distribution/build output",
	"out":          "Build output",
	"coverage":     "Code coverage reports",
	".github":      "GitHub workflows and configuration",
	".gitlab":      "GitLab CI configuration",
	".circleci":    "CircleCI configuration",
	".vscode":      "VS Code workspace settings",
	"prisma":       "Prisma ORM schema and migrations",
	"resources":    "Application resources",
	"locales":      "Internationalization/localization files",
	"i18n":         "Internationalization files",
	"plugins":      "Plugin implementations",
	"extensions":   "Extension modules",
	"certs":        "TLS/SSL certificates",
	"data":         "Data files",
}

// GenerateModuleMap scans the top-level directories in rootDir and returns a
// sorted list of ModuleMapEntry values. Each entry includes the directory name
// and a short description inferred from well-known conventions or directory
// content analysis.
//
// Hidden directories (starting with ".") other than well-known ones like
// .github are excluded. Vendor and node_modules directories are noted but
// not deeply described.
func GenerateModuleMap(rootDir string) ([]ModuleMapEntry, error) {
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		return nil, fmt.Errorf("reading directory %s: %w", rootDir, err)
	}

	var modules []ModuleMapEntry

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()

		// Skip most hidden directories, but allow well-known ones.
		if strings.HasPrefix(name, ".") {
			if _, known := knownDirectories[name]; !known {
				continue
			}
		}

		desc := describeDirectory(rootDir, name)
		modules = append(modules, ModuleMapEntry{
			Name:        name,
			Description: desc,
		})
	}

	// Sort by name for deterministic output.
	sort.Slice(modules, func(i, j int) bool {
		return modules[i].Name < modules[j].Name
	})

	return modules, nil
}

// describeDirectory returns a one-line description for a directory. It first
// checks the knownDirectories map for conventional descriptions, then falls
// back to content-based inference.
func describeDirectory(rootDir, name string) string {
	if desc, ok := knownDirectories[name]; ok {
		return desc
	}

	// Try to infer from directory content.
	dirPath := filepath.Join(rootDir, name)
	dirEntries, err := os.ReadDir(dirPath)
	if err != nil {
		return "Project directory"
	}

	return inferDescription(dirEntries)
}

// inferDescription attempts to determine a directory's purpose by examining
// its contents. Returns a generic description if no specific purpose can be
// determined.
func inferDescription(entries []os.DirEntry) string {
	hasGo := false
	hasTS := false
	hasPy := false
	hasRust := false
	hasMD := false

	for _, e := range entries {
		name := e.Name()
		switch {
		case strings.HasSuffix(name, ".go"):
			hasGo = true
		case strings.HasSuffix(name, ".ts") || strings.HasSuffix(name, ".tsx"):
			hasTS = true
		case strings.HasSuffix(name, ".py"):
			hasPy = true
		case strings.HasSuffix(name, ".rs"):
			hasRust = true
		case strings.HasSuffix(name, ".md"):
			hasMD = true
		}
	}

	switch {
	case hasGo:
		return "Go source code"
	case hasTS:
		return "TypeScript source code"
	case hasPy:
		return "Python source code"
	case hasRust:
		return "Rust source code"
	case hasMD:
		return "Documentation"
	default:
		return "Project directory"
	}
}

// RenderModuleMap formats a list of ModuleMapEntry values as a Markdown list
// suitable for inclusion in the brief output.
func RenderModuleMap(entries []ModuleMapEntry) string {
	if len(entries) == 0 {
		return ""
	}

	var b strings.Builder
	for _, e := range entries {
		fmt.Fprintf(&b, "- `%s/` -- %s\n", e.Name, e.Description)
	}
	return b.String()
}
