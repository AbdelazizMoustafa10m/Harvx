package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoadGoldenQuestions_ValidFile loads a golden questions TOML file from a
// temp directory and verifies all fields decode correctly.
func TestLoadGoldenQuestions_ValidFile(t *testing.T) {
	t.Parallel()

	const data = `
[[questions]]
id = "auth-jwt"
question = "Where is JWT token validation performed?"
expected_answer = "middleware.ts, verifyToken function"
category = "architecture"
critical_files = ["middleware.ts", "lib/auth/jwt.ts"]

[[questions]]
id = "retry-default"
question = "What is the default retry count for API calls?"
expected_answer = "3 retries with exponential backoff"
category = "configuration"
critical_files = ["lib/config/defaults.ts"]
`

	dir := t.TempDir()
	path := filepath.Join(dir, "golden-questions.toml")
	require.NoError(t, os.WriteFile(path, []byte(data), 0o644))

	cfg, err := LoadGoldenQuestions(path)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Len(t, cfg.Questions, 2)

	q0 := cfg.Questions[0]
	assert.Equal(t, "auth-jwt", q0.ID)
	assert.Equal(t, "Where is JWT token validation performed?", q0.Question)
	assert.Equal(t, "middleware.ts, verifyToken function", q0.ExpectedAnswer)
	assert.Equal(t, "architecture", q0.Category)
	assert.Equal(t, []string{"middleware.ts", "lib/auth/jwt.ts"}, q0.CriticalFiles)

	q1 := cfg.Questions[1]
	assert.Equal(t, "retry-default", q1.ID)
	assert.Equal(t, "configuration", q1.Category)
	assert.Equal(t, []string{"lib/config/defaults.ts"}, q1.CriticalFiles)
}

// TestLoadGoldenQuestions_InvalidSyntax verifies that malformed TOML returns
// an error that includes the file path.
func TestLoadGoldenQuestions_InvalidSyntax(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "bad.toml")
	require.NoError(t, os.WriteFile(path, []byte("[broken"), 0o644))

	_, err := LoadGoldenQuestions(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad.toml")
}

// TestLoadGoldenQuestions_NonExistentFile verifies that a missing file returns
// an error.
func TestLoadGoldenQuestions_NonExistentFile(t *testing.T) {
	t.Parallel()

	_, err := LoadGoldenQuestions("/nonexistent/golden-questions.toml")
	require.Error(t, err)
}

// TestLoadGoldenQuestions_EmptyFile verifies that an empty TOML file returns
// a config with no questions.
func TestLoadGoldenQuestions_EmptyFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "empty.toml")
	require.NoError(t, os.WriteFile(path, []byte{}, 0o644))

	cfg, err := LoadGoldenQuestions(path)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Empty(t, cfg.Questions)
}

// TestLoadGoldenQuestions_NoCriticalFiles verifies that questions without
// critical_files decode with a nil slice.
func TestLoadGoldenQuestions_NoCriticalFiles(t *testing.T) {
	t.Parallel()

	const data = `
[[questions]]
id = "bare-question"
question = "What framework is used?"
expected_answer = "Next.js"
category = "architecture"
`

	dir := t.TempDir()
	path := filepath.Join(dir, "no-critical.toml")
	require.NoError(t, os.WriteFile(path, []byte(data), 0o644))

	cfg, err := LoadGoldenQuestions(path)
	require.NoError(t, err)
	require.Len(t, cfg.Questions, 1)
	assert.Nil(t, cfg.Questions[0].CriticalFiles)
}

// TestLoadGoldenQuestions_UnknownKeys verifies that unknown TOML keys do not
// cause an error (forward compatibility).
func TestLoadGoldenQuestions_UnknownKeys(t *testing.T) {
	t.Parallel()

	const data = `
[[questions]]
id = "test"
question = "Does this work?"
expected_answer = "Yes"
category = "architecture"
future_field = "should be ignored"
`

	dir := t.TempDir()
	path := filepath.Join(dir, "unknown-keys.toml")
	require.NoError(t, os.WriteFile(path, []byte(data), 0o644))

	cfg, err := LoadGoldenQuestions(path)
	require.NoError(t, err, "unknown keys must not cause an error")
	require.Len(t, cfg.Questions, 1)
	assert.Equal(t, "test", cfg.Questions[0].ID)
}

// TestValidateGoldenQuestions covers the validation logic with table-driven
// tests for both success and error cases.
func TestValidateGoldenQuestions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     *GoldenQuestionsConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil config",
			cfg:     nil,
			wantErr: true,
			errMsg:  "nil",
		},
		{
			name:    "empty questions slice",
			cfg:     &GoldenQuestionsConfig{Questions: []GoldenQuestion{}},
			wantErr: true,
			errMsg:  "no questions",
		},
		{
			name: "valid single question",
			cfg: &GoldenQuestionsConfig{
				Questions: []GoldenQuestion{
					{
						ID:             "auth-jwt",
						Question:       "Where is JWT validated?",
						ExpectedAnswer: "middleware.ts",
						Category:       "architecture",
						CriticalFiles:  []string{"middleware.ts"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing id",
			cfg: &GoldenQuestionsConfig{
				Questions: []GoldenQuestion{
					{
						Question:       "Where is JWT validated?",
						ExpectedAnswer: "middleware.ts",
					},
				},
			},
			wantErr: true,
			errMsg:  "id is required",
		},
		{
			name: "missing question text",
			cfg: &GoldenQuestionsConfig{
				Questions: []GoldenQuestion{
					{
						ID:             "auth-jwt",
						ExpectedAnswer: "middleware.ts",
					},
				},
			},
			wantErr: true,
			errMsg:  "question text is required",
		},
		{
			name: "duplicate ids",
			cfg: &GoldenQuestionsConfig{
				Questions: []GoldenQuestion{
					{ID: "auth-jwt", Question: "Question 1"},
					{ID: "auth-jwt", Question: "Question 2"},
				},
			},
			wantErr: true,
			errMsg:  "duplicate id",
		},
		{
			name: "multiple errors accumulated",
			cfg: &GoldenQuestionsConfig{
				Questions: []GoldenQuestion{
					{ID: "", Question: ""},
					{ID: "good", Question: "Valid question"},
					{ID: "good", Question: "Duplicate id"},
				},
			},
			wantErr: true,
			errMsg:  "id is required",
		},
		{
			name: "valid with no critical files",
			cfg: &GoldenQuestionsConfig{
				Questions: []GoldenQuestion{
					{
						ID:       "bare",
						Question: "What framework?",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid with empty category",
			cfg: &GoldenQuestionsConfig{
				Questions: []GoldenQuestion{
					{
						ID:       "no-cat",
						Question: "What is this?",
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateGoldenQuestions(tt.cfg)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}
			require.NoError(t, err)
		})
	}
}

// TestDiscoverGoldenQuestionsPath_HarvxDirFirst verifies that .harvx/ is
// checked before the root directory.
func TestDiscoverGoldenQuestionsPath_HarvxDirFirst(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	harvxDir := filepath.Join(dir, ".harvx")
	require.NoError(t, os.MkdirAll(harvxDir, 0o755))

	// Create both files.
	require.NoError(t, os.WriteFile(
		filepath.Join(harvxDir, "golden-questions.toml"),
		[]byte("# harvx dir"),
		0o644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "golden-questions.toml"),
		[]byte("# root dir"),
		0o644,
	))

	got := DiscoverGoldenQuestionsPath(dir)
	assert.Equal(t, filepath.Join(harvxDir, "golden-questions.toml"), got)
}

// TestDiscoverGoldenQuestionsPath_FallbackToRoot verifies that the root-level
// file is returned when .harvx/ does not contain the file.
func TestDiscoverGoldenQuestionsPath_FallbackToRoot(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "golden-questions.toml"),
		[]byte("# root dir"),
		0o644,
	))

	got := DiscoverGoldenQuestionsPath(dir)
	assert.Equal(t, filepath.Join(dir, "golden-questions.toml"), got)
}

// TestDiscoverGoldenQuestionsPath_NotFound verifies that an empty string is
// returned when no golden questions file exists.
func TestDiscoverGoldenQuestionsPath_NotFound(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	got := DiscoverGoldenQuestionsPath(dir)
	assert.Equal(t, "", got)
}

// TestGenerateGoldenQuestionsInit_ValidTOML verifies that the generated
// starter TOML is valid and contains 3 example questions.
func TestGenerateGoldenQuestionsInit_ValidTOML(t *testing.T) {
	t.Parallel()

	content := GenerateGoldenQuestionsInit()
	assert.NotEmpty(t, content)

	// Write to a temp file and load to verify it is valid TOML.
	dir := t.TempDir()
	path := filepath.Join(dir, "init.toml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	cfg, err := LoadGoldenQuestions(path)
	require.NoError(t, err)
	require.Len(t, cfg.Questions, 3, "init template must contain 3 example questions")

	// Verify each question has the required fields.
	for i, q := range cfg.Questions {
		assert.NotEmpty(t, q.ID, "question[%d] must have a non-empty ID", i)
		assert.NotEmpty(t, q.Question, "question[%d] must have a non-empty question", i)
		assert.NotEmpty(t, q.ExpectedAnswer, "question[%d] must have an expected_answer", i)
		assert.NotEmpty(t, q.Category, "question[%d] must have a category", i)
		assert.NotEmpty(t, q.CriticalFiles, "question[%d] must have critical_files", i)
	}

	// Verify the generated template passes validation.
	err = ValidateGoldenQuestions(cfg)
	require.NoError(t, err, "init template must pass validation")
}

// TestGenerateGoldenQuestionsInit_Categories verifies that all categories in
// the starter template are valid.
func TestGenerateGoldenQuestionsInit_Categories(t *testing.T) {
	t.Parallel()

	content := GenerateGoldenQuestionsInit()
	dir := t.TempDir()
	path := filepath.Join(dir, "init.toml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	cfg, err := LoadGoldenQuestions(path)
	require.NoError(t, err)

	for _, q := range cfg.Questions {
		assert.True(t, ValidCategories[q.Category],
			"question %q has unknown category %q", q.ID, q.Category)
	}
}

// TestGenerateGoldenQuestionsInit_UniqueIDs verifies that the starter template
// has no duplicate IDs.
func TestGenerateGoldenQuestionsInit_UniqueIDs(t *testing.T) {
	t.Parallel()

	content := GenerateGoldenQuestionsInit()
	dir := t.TempDir()
	path := filepath.Join(dir, "init.toml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	cfg, err := LoadGoldenQuestions(path)
	require.NoError(t, err)

	seen := make(map[string]bool, len(cfg.Questions))
	for _, q := range cfg.Questions {
		assert.False(t, seen[q.ID], "duplicate ID %q in init template", q.ID)
		seen[q.ID] = true
	}
}

// TestValidCategories_KnownSet verifies the exported ValidCategories map
// contains exactly the 5 known categories from the spec.
func TestValidCategories_KnownSet(t *testing.T) {
	t.Parallel()

	expected := []string{
		"architecture",
		"configuration",
		"security",
		"conventions",
		"integration",
	}

	assert.Len(t, ValidCategories, len(expected))
	for _, cat := range expected {
		assert.True(t, ValidCategories[cat], "category %q must be in ValidCategories", cat)
	}
}

// TestLoadGoldenQuestions_AllCategories verifies that questions with every
// valid category decode correctly.
func TestLoadGoldenQuestions_AllCategories(t *testing.T) {
	t.Parallel()

	const data = `
[[questions]]
id = "q1"
question = "Q1?"
category = "architecture"

[[questions]]
id = "q2"
question = "Q2?"
category = "configuration"

[[questions]]
id = "q3"
question = "Q3?"
category = "security"

[[questions]]
id = "q4"
question = "Q4?"
category = "conventions"

[[questions]]
id = "q5"
question = "Q5?"
category = "integration"
`

	dir := t.TempDir()
	path := filepath.Join(dir, "all-cats.toml")
	require.NoError(t, os.WriteFile(path, []byte(data), 0o644))

	cfg, err := LoadGoldenQuestions(path)
	require.NoError(t, err)
	require.Len(t, cfg.Questions, 5)

	for i, q := range cfg.Questions {
		assert.True(t, ValidCategories[q.Category],
			"question[%d] category %q must be valid", i, q.Category)
	}
}

// TestLoadGoldenQuestions_MultipleCriticalFiles verifies that questions with
// many critical files decode the slice correctly and in order.
func TestLoadGoldenQuestions_MultipleCriticalFiles(t *testing.T) {
	t.Parallel()

	const data = `
[[questions]]
id = "multi-files"
question = "How does the auth pipeline work?"
expected_answer = "Through a chain of middleware"
category = "architecture"
critical_files = [
  "middleware/auth.go",
  "middleware/cors.go",
  "internal/auth/token.go",
  "internal/auth/session.go",
  "cmd/server/main.go",
]
`

	dir := t.TempDir()
	path := filepath.Join(dir, "multi.toml")
	require.NoError(t, os.WriteFile(path, []byte(data), 0o644))

	cfg, err := LoadGoldenQuestions(path)
	require.NoError(t, err)
	require.Len(t, cfg.Questions, 1)

	assert.Equal(t, []string{
		"middleware/auth.go",
		"middleware/cors.go",
		"internal/auth/token.go",
		"internal/auth/session.go",
		"cmd/server/main.go",
	}, cfg.Questions[0].CriticalFiles)
}
