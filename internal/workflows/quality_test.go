package workflows

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeGoldenQuestionsTOML is a test helper that writes a golden questions
// TOML file at the given path. It creates any intermediate directories.
func writeGoldenQuestionsTOML(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

// createRepoFile is a test helper that creates a file with content in the
// given root directory. Intermediate directories are created automatically.
func createRepoFile(t *testing.T, root, relPath, content string) {
	t.Helper()
	absPath := filepath.Join(root, relPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(absPath), 0o755))
	require.NoError(t, os.WriteFile(absPath, []byte(content), 0o644))
}

func TestEvaluateQuality_AllCriticalFilesFound(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	// Create all files referenced by the golden questions.
	createRepoFile(t, root, "internal/auth/token.go", "package auth\n")
	createRepoFile(t, root, "internal/db/config.go", "package db\n")
	createRepoFile(t, root, "middleware/auth.go", "package middleware\n")
	createRepoFile(t, root, "README.md", "# Project\n")

	tomlContent := `
[[questions]]
id = "auth-flow"
question = "Where is authentication performed?"
expected_answer = "middleware/auth.go"
category = "architecture"
critical_files = ["internal/auth/token.go", "middleware/auth.go"]

[[questions]]
id = "db-config"
question = "Where is the database configured?"
expected_answer = "internal/db/config.go"
category = "configuration"
critical_files = ["internal/db/config.go"]

[[questions]]
id = "readme"
question = "Does the project have documentation?"
expected_answer = "README.md"
category = "conventions"
critical_files = ["README.md"]
`

	questionsPath := filepath.Join(root, ".harvx", "golden-questions.toml")
	writeGoldenQuestionsTOML(t, questionsPath, tomlContent)

	result, err := EvaluateQuality(QualityOptions{
		RootDir: root,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 3, result.TotalQuestions)
	assert.Equal(t, 3, result.CoveredCount)
	assert.Equal(t, 0, result.UncoveredCount)
	assert.InDelta(t, 100.0, result.CoveragePercent, 0.01)

	// Every question should be covered with no missing files.
	for _, q := range result.Questions {
		assert.True(t, q.Covered, "question %q should be covered", q.ID)
		assert.Empty(t, q.MissingFiles, "question %q should have no missing files", q.ID)
		assert.NotEmpty(t, q.FoundFiles, "question %q should have found files", q.ID)
	}
}

func TestEvaluateQuality_PartialCoverage(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	// Only create some of the referenced files.
	createRepoFile(t, root, "internal/auth/token.go", "package auth\n")
	// middleware/auth.go is intentionally missing.
	// internal/db/config.go is intentionally missing.
	createRepoFile(t, root, "README.md", "# Project\n")

	tomlContent := `
[[questions]]
id = "auth-flow"
question = "Where is authentication performed?"
expected_answer = "middleware/auth.go and internal/auth/token.go"
category = "architecture"
critical_files = ["internal/auth/token.go", "middleware/auth.go"]

[[questions]]
id = "db-config"
question = "Where is the database configured?"
expected_answer = "internal/db/config.go"
category = "configuration"
critical_files = ["internal/db/config.go"]

[[questions]]
id = "readme"
question = "Does the project have documentation?"
expected_answer = "README.md"
category = "conventions"
critical_files = ["README.md"]
`

	questionsPath := filepath.Join(root, ".harvx", "golden-questions.toml")
	writeGoldenQuestionsTOML(t, questionsPath, tomlContent)

	result, err := EvaluateQuality(QualityOptions{
		RootDir: root,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 3, result.TotalQuestions)
	assert.Equal(t, 1, result.CoveredCount)   // only "readme"
	assert.Equal(t, 2, result.UncoveredCount) // "auth-flow" and "db-config"

	// Verify the coverage percentage is approximately 33.33%.
	assert.InDelta(t, 100.0/3.0, result.CoveragePercent, 0.01)

	// Check per-question details.
	questionMap := make(map[string]QuestionResult, len(result.Questions))
	for _, q := range result.Questions {
		questionMap[q.ID] = q
	}

	// auth-flow: token.go found, middleware/auth.go missing.
	authQ := questionMap["auth-flow"]
	assert.False(t, authQ.Covered)
	assert.Contains(t, authQ.FoundFiles, "internal/auth/token.go")
	assert.Contains(t, authQ.MissingFiles, "middleware/auth.go")

	// db-config: config.go missing.
	dbQ := questionMap["db-config"]
	assert.False(t, dbQ.Covered)
	assert.Contains(t, dbQ.MissingFiles, "internal/db/config.go")

	// readme: README.md found.
	readmeQ := questionMap["readme"]
	assert.True(t, readmeQ.Covered)
	assert.Contains(t, readmeQ.FoundFiles, "README.md")
	assert.Empty(t, readmeQ.MissingFiles)
}

func TestEvaluateQuality_NoCriticalFilesAlwaysCovered(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	tomlContent := `
[[questions]]
id = "general-question"
question = "What framework does this project use?"
expected_answer = "Go with Cobra"
category = "architecture"

[[questions]]
id = "no-files-question"
question = "What is the project license?"
expected_answer = "MIT"
category = "conventions"
critical_files = []
`

	questionsPath := filepath.Join(root, ".harvx", "golden-questions.toml")
	writeGoldenQuestionsTOML(t, questionsPath, tomlContent)

	result, err := EvaluateQuality(QualityOptions{
		RootDir: root,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 2, result.TotalQuestions)
	assert.Equal(t, 2, result.CoveredCount)
	assert.Equal(t, 0, result.UncoveredCount)
	assert.InDelta(t, 100.0, result.CoveragePercent, 0.01)

	// Both questions should be covered since neither has critical files.
	for _, q := range result.Questions {
		assert.True(t, q.Covered, "question %q with no critical files should be covered", q.ID)
		assert.Empty(t, q.MissingFiles)
		assert.Empty(t, q.FoundFiles)
	}
}

func TestEvaluateQuality_GlobPatterns(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	// Create files matching glob patterns.
	createRepoFile(t, root, "internal/auth/token.go", "package auth\n")
	createRepoFile(t, root, "internal/auth/session.go", "package auth\n")
	createRepoFile(t, root, "internal/db/config.go", "package db\n")
	createRepoFile(t, root, "lib/utils/helpers.go", "package utils\n")

	tomlContent := `
[[questions]]
id = "auth-glob"
question = "Where is authentication handled?"
expected_answer = "internal/auth/ directory"
category = "architecture"
critical_files = ["internal/auth/**/*.go"]

[[questions]]
id = "lib-glob"
question = "Where are utility functions?"
expected_answer = "lib/ directory"
category = "architecture"
critical_files = ["lib/**"]

[[questions]]
id = "missing-glob"
question = "Where are the tests?"
expected_answer = "tests/ directory"
category = "conventions"
critical_files = ["tests/**/*.go"]

[[questions]]
id = "star-ext"
question = "Are there any Go files in internal/db?"
expected_answer = "Yes, config.go"
category = "architecture"
critical_files = ["internal/db/*.go"]
`

	questionsPath := filepath.Join(root, ".harvx", "golden-questions.toml")
	writeGoldenQuestionsTOML(t, questionsPath, tomlContent)

	result, err := EvaluateQuality(QualityOptions{
		RootDir: root,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 4, result.TotalQuestions)

	questionMap := make(map[string]QuestionResult, len(result.Questions))
	for _, q := range result.Questions {
		questionMap[q.ID] = q
	}

	// auth-glob: should match internal/auth/token.go and session.go.
	assert.True(t, questionMap["auth-glob"].Covered, "internal/auth/**/*.go should match")
	assert.Contains(t, questionMap["auth-glob"].FoundFiles, "internal/auth/**/*.go")

	// lib-glob: should match lib/utils/helpers.go.
	assert.True(t, questionMap["lib-glob"].Covered, "lib/** should match")

	// missing-glob: tests/ does not exist, should not be covered.
	assert.False(t, questionMap["missing-glob"].Covered, "tests/**/*.go should not match")
	assert.Contains(t, questionMap["missing-glob"].MissingFiles, "tests/**/*.go")

	// star-ext: should match internal/db/config.go.
	assert.True(t, questionMap["star-ext"].Covered, "internal/db/*.go should match")
}

func TestEvaluateQuality_MissingQuestionsFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	// Do not create any golden questions file.

	_, err := EvaluateQuality(QualityOptions{
		RootDir: root,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no golden questions file found")
	assert.Contains(t, err.Error(), "--questions")
}

func TestEvaluateQuality_EmptyRootDir(t *testing.T) {
	t.Parallel()

	_, err := EvaluateQuality(QualityOptions{
		RootDir: "",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "root directory required")
}

func TestEvaluateQuality_CategoryStats(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	// Create some files.
	createRepoFile(t, root, "auth.go", "package main\n")
	createRepoFile(t, root, "config.go", "package main\n")
	// security.go intentionally missing.

	tomlContent := `
[[questions]]
id = "arch-1"
question = "Architecture question 1"
expected_answer = "answer"
category = "architecture"
critical_files = ["auth.go"]

[[questions]]
id = "arch-2"
question = "Architecture question 2"
expected_answer = "answer"
category = "architecture"
critical_files = ["missing-arch.go"]

[[questions]]
id = "config-1"
question = "Config question"
expected_answer = "answer"
category = "configuration"
critical_files = ["config.go"]

[[questions]]
id = "sec-1"
question = "Security question 1"
expected_answer = "answer"
category = "security"
critical_files = ["security.go"]

[[questions]]
id = "sec-2"
question = "Security question 2"
expected_answer = "answer"
category = "security"
critical_files = ["also-missing.go"]
`

	questionsPath := filepath.Join(root, ".harvx", "golden-questions.toml")
	writeGoldenQuestionsTOML(t, questionsPath, tomlContent)

	result, err := EvaluateQuality(QualityOptions{
		RootDir: root,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 5, result.TotalQuestions)
	assert.Equal(t, 2, result.CoveredCount)   // arch-1 and config-1
	assert.Equal(t, 3, result.UncoveredCount) // arch-2, sec-1, sec-2

	// Verify per-category stats.
	require.Contains(t, result.ByCategory, "architecture")
	require.Contains(t, result.ByCategory, "configuration")
	require.Contains(t, result.ByCategory, "security")

	archStats := result.ByCategory["architecture"]
	assert.Equal(t, 2, archStats.Total)
	assert.Equal(t, 1, archStats.Covered)
	assert.InDelta(t, 50.0, archStats.Percent, 0.01)

	configStats := result.ByCategory["configuration"]
	assert.Equal(t, 1, configStats.Total)
	assert.Equal(t, 1, configStats.Covered)
	assert.InDelta(t, 100.0, configStats.Percent, 0.01)

	secStats := result.ByCategory["security"]
	assert.Equal(t, 2, secStats.Total)
	assert.Equal(t, 0, secStats.Covered)
	assert.InDelta(t, 0.0, secStats.Percent, 0.01)
}

func TestEvaluateQuality_CoveragePercent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		covered     int
		total       int
		wantPercent float64
	}{
		{name: "all covered", covered: 5, total: 5, wantPercent: 100.0},
		{name: "none covered", covered: 0, total: 5, wantPercent: 0.0},
		{name: "half covered", covered: 2, total: 4, wantPercent: 50.0},
		{name: "one of three", covered: 1, total: 3, wantPercent: 100.0 / 3.0},
		{name: "two of three", covered: 2, total: 3, wantPercent: 200.0 / 3.0},
		{name: "single question covered", covered: 1, total: 1, wantPercent: 100.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()

			// Create one file that will be the "found" file.
			createRepoFile(t, root, "exists.go", "package main\n")

			var toml string
			for i := 0; i < tt.total; i++ {
				criticalFile := "exists.go"
				if i >= tt.covered {
					criticalFile = "missing.go"
				}
				toml += "\n[[questions]]\n"
				toml += "id = \"q-" + string(rune('a'+i)) + "\"\n"
				toml += "question = \"Question " + string(rune('a'+i)) + "?\"\n"
				toml += "expected_answer = \"answer\"\n"
				toml += "category = \"architecture\"\n"
				toml += "critical_files = [\"" + criticalFile + "\"]\n"
			}

			questionsPath := filepath.Join(root, ".harvx", "golden-questions.toml")
			writeGoldenQuestionsTOML(t, questionsPath, toml)

			result, err := EvaluateQuality(QualityOptions{
				RootDir: root,
			})

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.total, result.TotalQuestions)
			assert.Equal(t, tt.covered, result.CoveredCount)
			assert.InDelta(t, tt.wantPercent, result.CoveragePercent, 0.01)
		})
	}
}

func TestEvaluateQuality_CustomQuestionsPath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	// Create the file referenced by the custom questions file.
	createRepoFile(t, root, "src/main.go", "package main\n")

	// Place the questions file in a non-standard location.
	customDir := t.TempDir()
	tomlContent := `
[[questions]]
id = "custom-q"
question = "Where is the entry point?"
expected_answer = "src/main.go"
category = "architecture"
critical_files = ["src/main.go"]
`

	customPath := filepath.Join(customDir, "my-questions.toml")
	writeGoldenQuestionsTOML(t, customPath, tomlContent)

	result, err := EvaluateQuality(QualityOptions{
		RootDir:       root,
		QuestionsPath: customPath,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 1, result.TotalQuestions)
	assert.Equal(t, 1, result.CoveredCount)
	assert.Equal(t, customPath, result.QuestionsPath)
}

func TestEvaluateQuality_CustomQuestionsPathNotFound(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	_, err := EvaluateQuality(QualityOptions{
		RootDir:       root,
		QuestionsPath: filepath.Join(root, "nonexistent.toml"),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "quality")
}

func TestEvaluateQuality_EmptyQuestionsFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	// An empty TOML file produces an empty questions list, which should
	// fail validation with a meaningful error.
	questionsPath := filepath.Join(root, ".harvx", "golden-questions.toml")
	writeGoldenQuestionsTOML(t, questionsPath, "# Empty golden questions file\n")

	_, err := EvaluateQuality(QualityOptions{
		RootDir: root,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no questions")
}

func TestEvaluateQuality_UncategorizedGrouping(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	createRepoFile(t, root, "main.go", "package main\n")

	tomlContent := `
[[questions]]
id = "no-cat"
question = "A question with no category"
expected_answer = "answer"
critical_files = ["main.go"]
`

	questionsPath := filepath.Join(root, ".harvx", "golden-questions.toml")
	writeGoldenQuestionsTOML(t, questionsPath, tomlContent)

	result, err := EvaluateQuality(QualityOptions{
		RootDir: root,
	})

	require.NoError(t, err)
	require.Contains(t, result.ByCategory, "uncategorized")
	assert.Equal(t, 1, result.ByCategory["uncategorized"].Total)
	assert.Equal(t, 1, result.ByCategory["uncategorized"].Covered)
	assert.InDelta(t, 100.0, result.ByCategory["uncategorized"].Percent, 0.01)
}

func TestEvaluateQuality_QuestionsPathReturned(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	createRepoFile(t, root, "file.go", "package main\n")

	tomlContent := `
[[questions]]
id = "test"
question = "Test question"
expected_answer = "answer"
category = "architecture"
critical_files = ["file.go"]
`

	questionsPath := filepath.Join(root, ".harvx", "golden-questions.toml")
	writeGoldenQuestionsTOML(t, questionsPath, tomlContent)

	result, err := EvaluateQuality(QualityOptions{
		RootDir: root,
	})

	require.NoError(t, err)
	assert.Equal(t, questionsPath, result.QuestionsPath)
}

func TestEvaluateQuality_RootLevelDiscovery(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	createRepoFile(t, root, "app.go", "package main\n")

	// Place the golden questions at root level (fallback location).
	tomlContent := `
[[questions]]
id = "root-q"
question = "Root-level question"
expected_answer = "answer"
category = "conventions"
critical_files = ["app.go"]
`

	questionsPath := filepath.Join(root, "golden-questions.toml")
	writeGoldenQuestionsTOML(t, questionsPath, tomlContent)

	result, err := EvaluateQuality(QualityOptions{
		RootDir: root,
	})

	require.NoError(t, err)
	assert.Equal(t, questionsPath, result.QuestionsPath)
	assert.Equal(t, 1, result.CoveredCount)
}

func TestEvaluateQuality_CriticalFilesSorted(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	createRepoFile(t, root, "b.go", "package main\n")
	createRepoFile(t, root, "a.go", "package main\n")

	tomlContent := `
[[questions]]
id = "sorted"
question = "Are files sorted?"
expected_answer = "yes"
category = "conventions"
critical_files = ["b.go", "a.go"]
`

	questionsPath := filepath.Join(root, ".harvx", "golden-questions.toml")
	writeGoldenQuestionsTOML(t, questionsPath, tomlContent)

	result, err := EvaluateQuality(QualityOptions{
		RootDir: root,
	})

	require.NoError(t, err)
	require.Len(t, result.Questions, 1)

	// CriticalFiles should be sorted alphabetically.
	assert.Equal(t, []string{"a.go", "b.go"}, result.Questions[0].CriticalFiles)
	// FoundFiles should also be sorted.
	assert.Equal(t, []string{"a.go", "b.go"}, result.Questions[0].FoundFiles)
}

func TestIsGlobPattern(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		pattern string
		want    bool
	}{
		{name: "star extension", pattern: "*.go", want: true},
		{name: "literal filename", pattern: "foo.go", want: false},
		{name: "doublestar glob", pattern: "lib/**", want: true},
		{name: "doublestar with extension", pattern: "internal/**/*.go", want: true},
		{name: "question mark", pattern: "file?.go", want: true},
		{name: "bracket character class", pattern: "file[0-9].go", want: true},
		{name: "curly brace alternation", pattern: "*.{go,rs}", want: true},
		{name: "plain directory path", pattern: "internal/auth/token.go", want: false},
		{name: "path with dots", pattern: "config.d/app.toml", want: false},
		{name: "empty string", pattern: "", want: false},
		{name: "just a star", pattern: "*", want: true},
		{name: "nested path no glob", pattern: "a/b/c/d.txt", want: false},
		{name: "star in middle", pattern: "src/*/main.go", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := isGlobPattern(tt.pattern)
			assert.Equal(t, tt.want, got, "isGlobPattern(%q)", tt.pattern)
		})
	}
}

func TestFilePatternExists_LiteralPath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	createRepoFile(t, root, "src/main.go", "package main\n")
	createRepoFile(t, root, "README.md", "# Test\n")

	tests := []struct {
		name    string
		pattern string
		want    bool
	}{
		{name: "existing file", pattern: "src/main.go", want: true},
		{name: "existing file at root", pattern: "README.md", want: true},
		{name: "missing file", pattern: "src/missing.go", want: false},
		{name: "missing directory", pattern: "lib/auth.go", want: false},
		{name: "directory as literal", pattern: "src", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := filePatternExists(root, tt.pattern)
			assert.Equal(t, tt.want, got, "filePatternExists(root, %q)", tt.pattern)
		})
	}
}

func TestFilePatternExists_GlobPattern(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	createRepoFile(t, root, "internal/auth/token.go", "package auth\n")
	createRepoFile(t, root, "internal/auth/session.go", "package auth\n")
	createRepoFile(t, root, "internal/db/config.go", "package db\n")
	createRepoFile(t, root, "lib/utils.go", "package lib\n")
	createRepoFile(t, root, "main.go", "package main\n")

	tests := []struct {
		name    string
		pattern string
		want    bool
	}{
		{name: "doublestar matches nested", pattern: "internal/**/*.go", want: true},
		{name: "star matches single level", pattern: "internal/auth/*.go", want: true},
		{name: "star extension at root", pattern: "*.go", want: true},
		{name: "no match doublestar", pattern: "tests/**/*.go", want: false},
		{name: "no match star extension", pattern: "*.rs", want: false},
		{name: "doublestar all files", pattern: "internal/**", want: true},
		{name: "single level wildcard", pattern: "internal/*/config.go", want: true},
		{name: "single level wildcard no match", pattern: "internal/*/missing.go", want: false},
		{name: "question mark glob", pattern: "lib/utils.g?", want: true},
		{name: "question mark no match", pattern: "lib/utils.g??", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := filePatternExists(root, tt.pattern)
			assert.Equal(t, tt.want, got, "filePatternExists(root, %q)", tt.pattern)
		})
	}
}

func TestComputeCategoryStats(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		questions []QuestionResult
		want      map[string]CategoryStats
	}{
		{
			name:      "empty questions",
			questions: nil,
			want:      map[string]CategoryStats{},
		},
		{
			name: "single category all covered",
			questions: []QuestionResult{
				{ID: "q1", Category: "architecture", Covered: true},
				{ID: "q2", Category: "architecture", Covered: true},
			},
			want: map[string]CategoryStats{
				"architecture": {Total: 2, Covered: 2, Percent: 100.0},
			},
		},
		{
			name: "single category partial coverage",
			questions: []QuestionResult{
				{ID: "q1", Category: "security", Covered: true},
				{ID: "q2", Category: "security", Covered: false},
				{ID: "q3", Category: "security", Covered: false},
			},
			want: map[string]CategoryStats{
				"security": {Total: 3, Covered: 1, Percent: 100.0 / 3.0},
			},
		},
		{
			name: "multiple categories",
			questions: []QuestionResult{
				{ID: "q1", Category: "architecture", Covered: true},
				{ID: "q2", Category: "architecture", Covered: false},
				{ID: "q3", Category: "security", Covered: true},
				{ID: "q4", Category: "configuration", Covered: true},
				{ID: "q5", Category: "configuration", Covered: true},
			},
			want: map[string]CategoryStats{
				"architecture":  {Total: 2, Covered: 1, Percent: 50.0},
				"security":      {Total: 1, Covered: 1, Percent: 100.0},
				"configuration": {Total: 2, Covered: 2, Percent: 100.0},
			},
		},
		{
			name: "empty category becomes uncategorized",
			questions: []QuestionResult{
				{ID: "q1", Category: "", Covered: true},
				{ID: "q2", Category: "", Covered: false},
			},
			want: map[string]CategoryStats{
				"uncategorized": {Total: 2, Covered: 1, Percent: 50.0},
			},
		},
		{
			name: "mixed empty and named categories",
			questions: []QuestionResult{
				{ID: "q1", Category: "architecture", Covered: true},
				{ID: "q2", Category: "", Covered: true},
				{ID: "q3", Category: "", Covered: false},
			},
			want: map[string]CategoryStats{
				"architecture":  {Total: 1, Covered: 1, Percent: 100.0},
				"uncategorized": {Total: 2, Covered: 1, Percent: 50.0},
			},
		},
		{
			name: "none covered",
			questions: []QuestionResult{
				{ID: "q1", Category: "security", Covered: false},
				{ID: "q2", Category: "security", Covered: false},
			},
			want: map[string]CategoryStats{
				"security": {Total: 2, Covered: 0, Percent: 0.0},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := computeCategoryStats(tt.questions)

			assert.Len(t, got, len(tt.want), "number of categories")
			for cat, wantStats := range tt.want {
				gotStats, ok := got[cat]
				require.True(t, ok, "category %q should exist", cat)
				assert.Equal(t, wantStats.Total, gotStats.Total, "category %q total", cat)
				assert.Equal(t, wantStats.Covered, gotStats.Covered, "category %q covered", cat)
				assert.InDelta(t, wantStats.Percent, gotStats.Percent, 0.01, "category %q percent", cat)
			}
		})
	}
}

func TestEvaluateQuality_MultipleCriticalFilesPartialMatch(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	// Create only 2 of 3 critical files.
	createRepoFile(t, root, "middleware/auth.go", "package middleware\n")
	createRepoFile(t, root, "internal/auth/token.go", "package auth\n")
	// internal/auth/session.go is missing.

	tomlContent := `
[[questions]]
id = "auth-pipeline"
question = "How does the auth pipeline work?"
expected_answer = "Chain of middleware"
category = "architecture"
critical_files = ["middleware/auth.go", "internal/auth/token.go", "internal/auth/session.go"]
`

	questionsPath := filepath.Join(root, ".harvx", "golden-questions.toml")
	writeGoldenQuestionsTOML(t, questionsPath, tomlContent)

	result, err := EvaluateQuality(QualityOptions{
		RootDir: root,
	})

	require.NoError(t, err)
	require.Len(t, result.Questions, 1)

	q := result.Questions[0]
	assert.False(t, q.Covered, "question should not be covered when any critical file is missing")
	assert.Len(t, q.FoundFiles, 2)
	assert.Len(t, q.MissingFiles, 1)
	assert.Contains(t, q.MissingFiles, "internal/auth/session.go")
}

func TestEvaluateQuality_ProfileFieldPassedThrough(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	createRepoFile(t, root, "main.go", "package main\n")

	tomlContent := `
[[questions]]
id = "test"
question = "Test"
expected_answer = "answer"
category = "architecture"
critical_files = ["main.go"]
`

	questionsPath := filepath.Join(root, ".harvx", "golden-questions.toml")
	writeGoldenQuestionsTOML(t, questionsPath, tomlContent)

	// Verify the profile option does not cause an error.
	result, err := EvaluateQuality(QualityOptions{
		RootDir: root,
		Profile: "custom-profile",
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 1, result.CoveredCount)
}

func TestEvaluateQuality_QuestionFieldsPreserved(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	createRepoFile(t, root, "auth.go", "package main\n")

	tomlContent := `
[[questions]]
id = "preserve-test"
question = "Is the auth handler in auth.go?"
expected_answer = "Yes, the AuthHandler function"
category = "security"
critical_files = ["auth.go"]
`

	questionsPath := filepath.Join(root, ".harvx", "golden-questions.toml")
	writeGoldenQuestionsTOML(t, questionsPath, tomlContent)

	result, err := EvaluateQuality(QualityOptions{
		RootDir: root,
	})

	require.NoError(t, err)
	require.Len(t, result.Questions, 1)

	q := result.Questions[0]
	assert.Equal(t, "preserve-test", q.ID)
	assert.Equal(t, "Is the auth handler in auth.go?", q.Question)
	assert.Equal(t, "security", q.Category)
	assert.Equal(t, []string{"auth.go"}, q.CriticalFiles)
}

func TestEvaluateQuality_InvalidTOMLSyntax(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	questionsPath := filepath.Join(root, ".harvx", "golden-questions.toml")
	writeGoldenQuestionsTOML(t, questionsPath, "[broken TOML syntax")

	_, err := EvaluateQuality(QualityOptions{
		RootDir: root,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "quality")
}

func TestEvaluateQuality_ValidationErrorDuplicateIDs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	tomlContent := `
[[questions]]
id = "dup"
question = "First question"
expected_answer = "answer"
category = "architecture"

[[questions]]
id = "dup"
question = "Second question with same id"
expected_answer = "answer"
category = "architecture"
`

	questionsPath := filepath.Join(root, ".harvx", "golden-questions.toml")
	writeGoldenQuestionsTOML(t, questionsPath, tomlContent)

	_, err := EvaluateQuality(QualityOptions{
		RootDir: root,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate")
}
