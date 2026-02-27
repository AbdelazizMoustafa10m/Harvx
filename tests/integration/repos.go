//go:build integration

package integration

import (
	"path/filepath"
	"runtime"
)

// projectRoot returns the absolute path to the project root.
func projectRoot() string {
	// Navigate from tests/integration/ up to project root.
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..")
}

// TestRepo describes a test repository used for integration testing.
type TestRepo struct {
	// Name is a short identifier for the repo.
	Name string

	// Description describes the repo type.
	Description string

	// Dir is the absolute path to the repo directory.
	Dir string

	// Language is the primary language of the repo.
	Language string

	// ExpectedFileCountMin is the minimum number of files Harvx should discover.
	ExpectedFileCountMin int

	// HasGoFiles indicates whether the repo contains Go source files.
	HasGoFiles bool

	// HasTSFiles indicates whether the repo contains TypeScript files.
	HasTSFiles bool

	// HasPythonFiles indicates whether the repo contains Python files.
	HasPythonFiles bool
}

// testRepos returns all test repositories defined in testdata/.
// Uses projectRoot() from setup_test.go.
func testRepos() []TestRepo {
	root := projectRoot()
	return []TestRepo{
		{
			Name:                 "go-cli",
			Description:          "Go CLI project with cobra-style commands",
			Dir:                  filepath.Join(root, "testdata", "oss-go-cli"),
			Language:             "go",
			ExpectedFileCountMin: 10,
			HasGoFiles:           true,
		},
		{
			Name:                 "ts-nextjs",
			Description:          "TypeScript/Next.js web application",
			Dir:                  filepath.Join(root, "testdata", "oss-ts-nextjs"),
			Language:             "typescript",
			ExpectedFileCountMin: 10,
			HasTSFiles:           true,
		},
		{
			Name:                 "python-fastapi",
			Description:          "Python FastAPI REST API",
			Dir:                  filepath.Join(root, "testdata", "oss-python-fastapi"),
			Language:             "python",
			ExpectedFileCountMin: 10,
			HasPythonFiles:       true,
		},
		{
			Name:                 "monorepo",
			Description:          "Multi-package monorepo with JS/TS and Go",
			Dir:                  filepath.Join(root, "testdata", "oss-monorepo"),
			Language:             "mixed",
			ExpectedFileCountMin: 15,
			HasGoFiles:           true,
			HasTSFiles:           true,
		},
	}
}

// repoByName returns the TestRepo with the given name, or panics.
func repoByName(name string) TestRepo {
	for _, r := range testRepos() {
		if r.Name == name {
			return r
		}
	}
	panic("unknown test repo: " + name)
}