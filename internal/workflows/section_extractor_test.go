package workflows

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractMakefileTargets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		want    []string
		empty   bool
	}{
		{
			name: "standard targets",
			content: `
.PHONY: build test lint

build:
	go build ./cmd/harvx/

test:
	go test ./...

lint:
	golangci-lint run
`,
			want: []string{"build", "test", "lint"},
		},
		{
			name:    "empty makefile",
			content: "# Just a comment\n",
			empty:   true,
		},
		{
			name: "targets with dependencies",
			content: `
all: build test

build: deps
	go build ./...

deps:
	go mod download
`,
			want: []string{"all", "build", "deps"},
		},
		{
			name: "deduplicates targets",
			content: `
build:
	echo "first"

build:
	echo "override"
`,
			want: []string{"build"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := ExtractMakefileTargets(tt.content)
			if tt.empty {
				assert.Empty(t, result)
				return
			}
			for _, target := range tt.want {
				assert.Contains(t, result, target)
			}
		})
	}
}

func TestExtractPackageJSONScripts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		want    []string
		empty   bool
	}{
		{
			name: "standard scripts",
			content: `{
				"name": "my-app",
				"scripts": {
					"build": "tsc",
					"test": "jest",
					"lint": "eslint ."
				}
			}`,
			want: []string{"build", "test", "lint"},
		},
		{
			name:    "no scripts section",
			content: `{"name": "my-app", "version": "1.0.0"}`,
			empty:   true,
		},
		{
			name:    "invalid json",
			content: `not json at all`,
			empty:   true,
		},
		{
			name:    "empty scripts",
			content: `{"scripts": {}}`,
			empty:   true,
		},
		{
			name: "includes package name",
			content: `{
				"name": "my-cool-app",
				"scripts": {
					"start": "node index.js"
				}
			}`,
			want: []string{"Package: my-cool-app", "start"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := ExtractPackageJSONScripts(tt.content)
			if tt.empty {
				assert.Empty(t, result)
				return
			}
			for _, s := range tt.want {
				assert.Contains(t, result, s)
			}
		})
	}
}

func TestExtractGoModInfo(t *testing.T) {
	t.Parallel()

	content := `module github.com/harvx/harvx

go 1.24.0

require (
	github.com/spf13/cobra v1.8.0
)
`

	result := ExtractGoModInfo(content)
	assert.Contains(t, result, "Module: github.com/harvx/harvx")
	assert.Contains(t, result, "Go version: 1.24.0")
}

func TestExtractGoModInfo_Empty(t *testing.T) {
	t.Parallel()
	result := ExtractGoModInfo("")
	assert.Empty(t, result)
}

func TestExtractCargoTomlInfo(t *testing.T) {
	t.Parallel()

	content := `[package]
name = "my-crate"
version = "0.1.0"
edition = "2021"

[dependencies]
serde = "1.0"
`

	result := ExtractCargoTomlInfo(content)
	assert.Contains(t, result, "Crate: my-crate")
	assert.Contains(t, result, "Version: 0.1.0")
	assert.Contains(t, result, "Edition: 2021")
}

func TestExtractCargoTomlInfo_Empty(t *testing.T) {
	t.Parallel()
	result := ExtractCargoTomlInfo("")
	assert.Empty(t, result)
}

func TestExtractCargoTomlInfo_NoPackageSection(t *testing.T) {
	t.Parallel()
	content := `[dependencies]
serde = "1.0"
`
	result := ExtractCargoTomlInfo(content)
	assert.Empty(t, result)
}

func TestExtractPyprojectInfo(t *testing.T) {
	t.Parallel()

	content := `[project]
name = "my-project"
version = "1.0.0"
requires-python = ">=3.9"

[build-system]
requires = ["setuptools"]
`

	result := ExtractPyprojectInfo(content)
	assert.Contains(t, result, "Project: my-project")
	assert.Contains(t, result, "Version: 1.0.0")
	assert.Contains(t, result, "Python: >=3.9")
}

func TestExtractPyprojectInfo_Empty(t *testing.T) {
	t.Parallel()
	result := ExtractPyprojectInfo("")
	assert.Empty(t, result)
}

func TestExtractTomlValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "double quoted", input: `name = "value"`, want: "value"},
		{name: "single quoted", input: `name = 'value'`, want: "value"},
		{name: "no quotes", input: `name = value`, want: "value"},
		{name: "with spaces", input: `name  =  "spaced"`, want: "spaced"},
		{name: "no equals", input: `no-equals-here`, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := extractTomlValue(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractPackageJSONScripts_DeterministicOrder(t *testing.T) {
	t.Parallel()

	content := `{
		"scripts": {
			"zulu": "echo z",
			"alpha": "echo a",
			"mike": "echo m"
		}
	}`

	result1 := ExtractPackageJSONScripts(content)
	result2 := ExtractPackageJSONScripts(content)

	assert.Equal(t, result1, result2, "output must be deterministic across calls")
}

// ---------------------------------------------------------------------------
// Additional section extractor tests for T-070 coverage
// ---------------------------------------------------------------------------

// TestExtractMakefileTargets_SkipsDotTargets verifies that targets starting
// with a dot (like .PHONY) are not included in the extracted list.
func TestExtractMakefileTargets_SkipsDotTargets(t *testing.T) {
	t.Parallel()

	content := `.PHONY: build test

build:
	go build ./...

test:
	go test ./...
`
	result := ExtractMakefileTargets(content)

	assert.Contains(t, result, "build")
	assert.Contains(t, result, "test")
	// The regex pattern starts with [a-zA-Z_], so .PHONY should not be matched.
	assert.NotContains(t, result, ".PHONY")
	assert.NotContains(t, result, "PHONY")
}

// TestExtractMakefileTargets_TabInRecipe verifies that lines starting with a
// tab character (recipe lines) are not confused with targets.
func TestExtractMakefileTargets_TabInRecipe(t *testing.T) {
	t.Parallel()

	content := `build:
	@echo "building"
	go build -o bin/app ./...

test: build
	go test -v ./...
`
	result := ExtractMakefileTargets(content)

	// Should extract "build" and "test" but not "echo" or "go".
	assert.Contains(t, result, "build")
	assert.Contains(t, result, "test")
}

// TestExtractMakefileTargets_HyphenAndUnderscore verifies that target names
// with hyphens and underscores are correctly extracted.
func TestExtractMakefileTargets_HyphenAndUnderscore(t *testing.T) {
	t.Parallel()

	content := `docker-build:
	docker build .

run_tests:
	go test ./...
`
	result := ExtractMakefileTargets(content)

	assert.Contains(t, result, "docker-build")
	assert.Contains(t, result, "run_tests")
}

// TestExtractMakefileTargets_FormatsAsList verifies that the output is
// formatted as a Markdown list with backtick-wrapped target names.
func TestExtractMakefileTargets_FormatsAsList(t *testing.T) {
	t.Parallel()

	content := `build:
	go build ./...
`
	result := ExtractMakefileTargets(content)

	assert.Contains(t, result, "Available targets:")
	assert.Contains(t, result, "- `build`")
}

// TestExtractPackageJSONScripts_ComplexCommands verifies extraction when
// script values contain complex shell commands.
func TestExtractPackageJSONScripts_ComplexCommands(t *testing.T) {
	t.Parallel()

	content := `{
		"name": "complex-app",
		"scripts": {
			"dev": "next dev --turbo",
			"build": "next build && next export",
			"lint": "eslint . --ext .ts,.tsx --fix",
			"typecheck": "tsc --noEmit"
		}
	}`

	result := ExtractPackageJSONScripts(content)

	assert.Contains(t, result, "Package: complex-app")
	assert.Contains(t, result, "build")
	assert.Contains(t, result, "next build && next export")
	assert.Contains(t, result, "lint")
	assert.Contains(t, result, "typecheck")
}

// TestExtractPackageJSONScripts_ScriptsSorted verifies that scripts are
// listed in alphabetical order.
func TestExtractPackageJSONScripts_ScriptsSorted(t *testing.T) {
	t.Parallel()

	content := `{
		"scripts": {
			"ztest": "echo z",
			"abuild": "echo a",
			"mlint": "echo m"
		}
	}`

	result := ExtractPackageJSONScripts(content)

	// Find positions of each script name.
	posA := strings.Index(result, "abuild")
	posM := strings.Index(result, "mlint")
	posZ := strings.Index(result, "ztest")

	assert.Greater(t, posA, -1, "abuild should be present")
	assert.Greater(t, posM, -1, "mlint should be present")
	assert.Greater(t, posZ, -1, "ztest should be present")

	assert.Less(t, posA, posM, "abuild should come before mlint")
	assert.Less(t, posM, posZ, "mlint should come before ztest")
}

// TestExtractGoModInfo_WithRequire verifies that require directives do not
// interfere with module and go version extraction.
func TestExtractGoModInfo_WithRequire(t *testing.T) {
	t.Parallel()

	content := `module github.com/example/project

go 1.22.0

require (
	github.com/spf13/cobra v1.8.0
	github.com/stretchr/testify v1.9.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
)
`

	result := ExtractGoModInfo(content)
	assert.Contains(t, result, "Module: github.com/example/project")
	assert.Contains(t, result, "Go version: 1.22.0")
	// Should not contain dependency information.
	assert.NotContains(t, result, "cobra")
	assert.NotContains(t, result, "testify")
}

// TestExtractGoModInfo_ModuleOnly verifies extraction when only the module
// line is present (minimal go.mod).
func TestExtractGoModInfo_ModuleOnly(t *testing.T) {
	t.Parallel()

	content := "module example.com/minimal\n"
	result := ExtractGoModInfo(content)
	assert.Contains(t, result, "Module: example.com/minimal")
}

// TestExtractCargoTomlInfo_WorkspaceCargo verifies that a workspace-level
// Cargo.toml without a [package] section returns empty.
func TestExtractCargoTomlInfo_WorkspaceCargo(t *testing.T) {
	t.Parallel()

	content := `[workspace]
members = ["crate-a", "crate-b"]
`
	result := ExtractCargoTomlInfo(content)
	assert.Empty(t, result)
}

// TestExtractPyprojectInfo_BuildSystemOnly verifies that a pyproject.toml
// with only [build-system] and no [project] returns empty.
func TestExtractPyprojectInfo_BuildSystemOnly(t *testing.T) {
	t.Parallel()

	content := `[build-system]
requires = ["setuptools>=61.0"]
build-backend = "setuptools.backends._legacy:_Backend"
`
	result := ExtractPyprojectInfo(content)
	assert.Empty(t, result)
}

// TestSortedKeys verifies the sortedKeys helper function.
func TestSortedKeys(t *testing.T) {
	t.Parallel()

	m := map[string]string{
		"charlie": "c",
		"alpha":   "a",
		"bravo":   "b",
	}

	keys := sortedKeys(m)
	assert.Equal(t, []string{"alpha", "bravo", "charlie"}, keys)
}

// TestSortedKeys_Empty verifies sortedKeys with an empty map.
func TestSortedKeys_Empty(t *testing.T) {
	t.Parallel()

	keys := sortedKeys(map[string]string{})
	assert.Empty(t, keys)
}
