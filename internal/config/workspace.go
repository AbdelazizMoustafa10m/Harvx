package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// WorkspaceConfig is the top-level type parsed from .harvx/workspace.toml.
// It wraps the WorkspaceManifest inside a [workspace] TOML table.
type WorkspaceConfig struct {
	Workspace WorkspaceManifest `toml:"workspace"`
}

// WorkspaceManifest holds the workspace name, description, and repo list.
type WorkspaceManifest struct {
	// Name is the human-readable workspace name. When empty, renderers
	// should default to "Workspace".
	Name string `toml:"name"`

	// Description is a short text explaining the purpose of the workspace.
	Description string `toml:"description"`

	// Repos is the ordered list of repositories in the workspace.
	Repos []WorkspaceRepo `toml:"repos"`
}

// WorkspaceRepo describes a single repository in the workspace.
type WorkspaceRepo struct {
	// Name is a short identifier for the repository (e.g. "api-gateway").
	Name string `toml:"name"`

	// Path is the local filesystem path to the repository. May use ~ or
	// relative paths which are expanded by ExpandPath.
	Path string `toml:"path"`

	// Description is a 1-3 line explanation of what the repository does.
	Description string `toml:"description"`

	// Entrypoints lists key files and directories that serve as starting
	// points for understanding the repository.
	Entrypoints []string `toml:"entrypoints"`

	// IntegratesWith lists the names of other workspace repositories that
	// this repository communicates with or depends on.
	IntegratesWith []string `toml:"integrates_with"`

	// SharedSchemas lists schema files (protobuf, OpenAPI, etc.) that are
	// shared across repositories.
	SharedSchemas []string `toml:"shared_schemas"`

	// Docs lists documentation files relevant to understanding this repository.
	Docs []string `toml:"docs"`
}

// LoadWorkspaceConfig reads and parses a workspace.toml file at path. It
// returns a fully decoded *WorkspaceConfig on success. Unknown TOML keys
// produce slog warnings (not errors) to maintain forward compatibility with
// future schema additions.
func LoadWorkspaceConfig(path string) (*WorkspaceConfig, error) {
	var cfg WorkspaceConfig
	meta, err := toml.DecodeFile(path, &cfg)
	if err != nil {
		return nil, fmt.Errorf("parse workspace config %s: %w", path, err)
	}

	warnUndecodedKeys(meta, path)

	return &cfg, nil
}

// DiscoverWorkspaceConfig searches for .harvx/workspace.toml starting from
// startDir, walking up parent directories. It stops at a .git boundary or
// the filesystem root, or after maxSearchDepth levels, whichever comes first.
// Returns an empty string if no workspace.toml is found.
func DiscoverWorkspaceConfig(startDir string) (string, error) {
	abs, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("abs path for %s: %w", startDir, err)
	}

	// Resolve symlinks to avoid loops and get the canonical path.
	if resolved, evalErr := filepath.EvalSymlinks(abs); evalErr == nil {
		abs = resolved
	} else {
		slog.Debug("symlink eval failed, using unresolved path",
			"dir", abs,
			"err", evalErr,
		)
	}

	dir := abs
	for depth := 0; depth < maxSearchDepth; depth++ {
		configPath := filepath.Join(dir, ".harvx", "workspace.toml")
		if _, statErr := os.Stat(configPath); statErr == nil {
			slog.Debug("discovered workspace config",
				"path", configPath,
				"depth", depth,
			)
			return configPath, nil
		}

		// Check for .git boundary: if .git exists here, we are at the repo
		// root. After checking for workspace.toml at this level (done above),
		// stop the search regardless.
		if _, statErr := os.Stat(filepath.Join(dir, ".git")); statErr == nil {
			slog.Debug("reached .git boundary, stopping workspace search",
				"dir", dir,
				"depth", depth,
			)
			return "", nil
		}

		// Move to parent directory.
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached the filesystem root.
			slog.Debug("reached filesystem root, no workspace.toml found")
			return "", nil
		}
		dir = parent
	}

	slog.Debug("reached max search depth without finding workspace.toml",
		"maxDepth", maxSearchDepth,
	)
	return "", nil
}

// ValidateWorkspace checks the workspace config for issues. It returns a
// list of warning strings describing problems found (missing repo paths,
// unknown integration targets, duplicate repo names). It does NOT return
// errors for these -- they are warnings only that allow the workspace to
// still be used.
func ValidateWorkspace(cfg *WorkspaceConfig) []string {
	if cfg == nil {
		return nil
	}

	var warnings []string

	repos := cfg.Workspace.Repos

	// Build a set of known repo names for integration edge validation.
	knownNames := make(map[string]bool, len(repos))
	for _, r := range repos {
		knownNames[r.Name] = true
	}

	// Check for duplicate repo names.
	seen := make(map[string]bool, len(repos))
	for _, r := range repos {
		if r.Name == "" {
			continue
		}
		if seen[r.Name] {
			warnings = append(warnings, fmt.Sprintf("duplicate repo name %q", r.Name))
		}
		seen[r.Name] = true
	}

	// Validate each repo.
	for _, r := range repos {
		// Check that the repo path exists on disk.
		if r.Path != "" {
			if _, statErr := os.Stat(r.Path); statErr != nil {
				warnings = append(warnings, fmt.Sprintf(
					"repo %q: path %q does not exist",
					r.Name, r.Path,
				))
			}
		}

		// Check that integration edges reference known repo names.
		for _, target := range r.IntegratesWith {
			if !knownNames[target] {
				warnings = append(warnings, fmt.Sprintf(
					"repo %q: integrates_with references unknown repo %q",
					r.Name, target,
				))
			}
		}
	}

	if len(warnings) > 0 {
		slog.Debug("workspace validation complete",
			"warnings", len(warnings),
		)
	}

	return warnings
}

// ExpandPath resolves ~ to $HOME and relative paths relative to the given
// base directory. Absolute paths are returned as-is.
func ExpandPath(path, baseDir string) string {
	if path == "" {
		return path
	}

	// Expand ~ prefix to $HOME.
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			slog.Warn("could not determine home directory for ~ expansion",
				"err", err,
			)
			return path
		}
		if path == "~" {
			return home
		}
		path = filepath.Join(home, path[2:])
	}

	// If already absolute, return as-is.
	if filepath.IsAbs(path) {
		return path
	}

	// Resolve relative path against baseDir.
	return filepath.Join(baseDir, path)
}

// GenerateWorkspaceInit returns a starter workspace.toml string with
// placeholder entries that the user can edit. The returned TOML is
// syntactically valid and can be written directly to a file.
func GenerateWorkspaceInit() string {
	return `# Harvx Workspace Manifest
# Describes related repositories and their relationships.
# See: harvx workspace --help

[workspace]
name = "My Workspace"
description = "Description of the workspace"

[[workspace.repos]]
name = "repo-name"
path = "~/work/repo-name"
description = "Short description of this repository"
entrypoints = ["src/main.ts"]
# integrates_with = ["other-repo"]
# shared_schemas = ["proto/schema.proto"]
# docs = ["docs/README.md"]
`
}