package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// ValidCategories is the set of recognized categories for golden questions.
// Questions with categories outside this set still parse but produce a
// validation warning.
var ValidCategories = map[string]bool{
	"architecture":  true,
	"configuration": true,
	"security":      true,
	"conventions":   true,
	"integration":   true,
}

// GoldenQuestionsConfig is the top-level type for golden questions TOML files.
// It holds an ordered list of questions used to evaluate LLM context quality.
type GoldenQuestionsConfig struct {
	// Questions is the ordered list of golden questions loaded from TOML.
	Questions []GoldenQuestion `toml:"questions"`
}

// GoldenQuestion represents a single golden question for quality evaluation.
// Each question pairs a natural-language query with an expected answer and
// the set of files that must be present in the context for an LLM to answer
// correctly.
type GoldenQuestion struct {
	// ID is a unique, short identifier for this question (e.g. "auth-jwt").
	ID string `toml:"id"`

	// Question is the natural-language question to ask the LLM.
	Question string `toml:"question"`

	// ExpectedAnswer is the expected correct answer, used for manual or
	// automated comparison against the LLM's response.
	ExpectedAnswer string `toml:"expected_answer"`

	// Category classifies the question. Known categories are listed in
	// ValidCategories; custom categories are permitted but produce warnings.
	Category string `toml:"category"`

	// CriticalFiles lists the file paths (relative to the repo root) that
	// must be included in the Harvx output for the LLM to answer this
	// question correctly. Glob patterns (doublestar syntax) are supported.
	CriticalFiles []string `toml:"critical_files"`
}

// LoadGoldenQuestions reads and parses a golden questions TOML file at path.
// Unknown TOML keys produce slog warnings (not errors) to maintain forward
// compatibility. Invalid TOML syntax causes an error that includes the file
// path and line information from the decoder.
func LoadGoldenQuestions(path string) (*GoldenQuestionsConfig, error) {
	var cfg GoldenQuestionsConfig
	meta, err := toml.DecodeFile(path, &cfg)
	if err != nil {
		return nil, fmt.Errorf("parse golden questions %s: %w", path, err)
	}

	undecoded := meta.Undecoded()
	if len(undecoded) > 0 {
		keys := make([]string, 0, len(undecoded))
		for _, k := range undecoded {
			keys = append(keys, k.String())
		}
		slog.Warn("unknown golden questions keys will be ignored",
			"source", path,
			"keys", strings.Join(keys, ", "),
		)
	}

	return &cfg, nil
}

// ValidateGoldenQuestions checks the loaded golden questions configuration
// for structural problems. It returns an error if:
//   - the config contains no questions
//   - any question has an empty ID
//   - any question has an empty Question field
//   - duplicate question IDs exist
//
// Unknown categories produce slog warnings but are not errors.
func ValidateGoldenQuestions(cfg *GoldenQuestionsConfig) error {
	if cfg == nil {
		return fmt.Errorf("golden questions config is nil")
	}
	if len(cfg.Questions) == 0 {
		return fmt.Errorf("golden questions file contains no questions")
	}

	var errs []string
	seen := make(map[string]bool, len(cfg.Questions))

	for i, q := range cfg.Questions {
		if q.ID == "" {
			errs = append(errs, fmt.Sprintf("questions[%d]: id is required", i))
		}
		if q.Question == "" {
			errs = append(errs, fmt.Sprintf("questions[%d]: question text is required", i))
		}
		if q.ID != "" {
			if seen[q.ID] {
				errs = append(errs, fmt.Sprintf("questions[%d]: duplicate id %q", i, q.ID))
			}
			seen[q.ID] = true
		}
		if q.Category != "" && !ValidCategories[q.Category] {
			slog.Warn("unknown golden question category",
				"id", q.ID,
				"category", q.Category,
				"valid", "architecture, configuration, security, conventions, integration",
			)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("golden questions validation failed:\n  %s", strings.Join(errs, "\n  "))
	}

	return nil
}

// DiscoverGoldenQuestionsPath checks for a golden questions TOML file in
// standard locations relative to rootDir. It returns the absolute path of
// the first file found, or an empty string if no file exists.
//
// Search order:
//  1. <rootDir>/.harvx/golden-questions.toml
//  2. <rootDir>/golden-questions.toml
func DiscoverGoldenQuestionsPath(rootDir string) string {
	candidates := []string{
		filepath.Join(rootDir, ".harvx", "golden-questions.toml"),
		filepath.Join(rootDir, "golden-questions.toml"),
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			slog.Debug("discovered golden questions file",
				"path", path,
			)
			return path
		}
	}

	slog.Debug("no golden questions file found",
		"root", rootDir,
	)
	return ""
}

// GenerateGoldenQuestionsInit returns a starter TOML string with three
// example golden questions. This is used by the `harvx quality init`
// command to bootstrap a new golden questions file.
func GenerateGoldenQuestionsInit() string {
	return `# Golden Questions Harness
# Each question tests whether the LLM can answer correctly given Harvx context.
# See: https://github.com/harvx/harvx/docs/guides/golden-questions.md
#
# Categories: architecture, configuration, security, conventions, integration

[[questions]]
id = "auth-flow"
question = "Where is user authentication performed and what middleware enforces it?"
expected_answer = "Authentication is handled in middleware/auth.go via the AuthMiddleware function"
category = "architecture"
critical_files = ["middleware/auth.go", "internal/auth/token.go"]

[[questions]]
id = "db-retry"
question = "What is the default retry policy for database connections?"
expected_answer = "3 retries with exponential backoff starting at 100ms"
category = "configuration"
critical_files = ["internal/db/config.go", "internal/db/connection.go"]

[[questions]]
id = "api-key-storage"
question = "How are API keys stored and where are they validated?"
expected_answer = "API keys are stored as bcrypt hashes in the keys table and validated in the API gateway middleware"
category = "security"
critical_files = ["internal/api/keys.go", "internal/gateway/auth.go"]
`
}
