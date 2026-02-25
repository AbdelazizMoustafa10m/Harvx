// Package workflows implements high-level workflow commands for harvx.
// This file implements the golden questions coverage analysis workflow.
// It checks whether critical files referenced by golden questions are
// present in the repository, providing a coverage metric for LLM context
// quality evaluation.
package workflows

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/harvx/harvx/internal/config"
)

// QualityOptions configures the quality evaluation workflow.
type QualityOptions struct {
	// RootDir is the repository root directory to scan.
	RootDir string

	// QuestionsPath is the path to a golden questions TOML file.
	// When empty, auto-discovery is used via config.DiscoverGoldenQuestionsPath.
	QuestionsPath string

	// Profile is the active profile name, used for logging context.
	Profile string
}

// QuestionResult holds the coverage result for a single golden question.
type QuestionResult struct {
	// ID is the unique question identifier.
	ID string `json:"id"`

	// Question is the natural-language question text.
	Question string `json:"question"`

	// Category classifies the question (e.g., "architecture", "security").
	Category string `json:"category"`

	// CriticalFiles lists the file patterns declared in the question.
	CriticalFiles []string `json:"critical_files"`

	// FoundFiles lists critical file patterns that matched at least one
	// file in the repository.
	FoundFiles []string `json:"found_files"`

	// MissingFiles lists critical file patterns that matched zero files
	// in the repository.
	MissingFiles []string `json:"missing_files"`

	// Covered is true when every critical file pattern matched at least
	// one file in the repository.
	Covered bool `json:"covered"`
}

// QualityResult holds the aggregate quality evaluation results.
type QualityResult struct {
	// TotalQuestions is the total number of golden questions evaluated.
	TotalQuestions int `json:"total_questions"`

	// CoveredCount is the number of questions with all critical files present.
	CoveredCount int `json:"covered_count"`

	// UncoveredCount is the number of questions with at least one missing
	// critical file.
	UncoveredCount int `json:"uncovered_count"`

	// CoveragePercent is the percentage of covered questions (0-100).
	CoveragePercent float64 `json:"coverage_percent"`

	// Questions holds per-question coverage results in the order they
	// appear in the golden questions file.
	Questions []QuestionResult `json:"questions"`

	// ByCategory holds per-category aggregate coverage statistics.
	ByCategory map[string]CategoryStats `json:"by_category"`

	// QuestionsPath is the absolute path to the golden questions file
	// that was evaluated.
	QuestionsPath string `json:"questions_path"`
}

// CategoryStats holds per-category aggregate coverage statistics.
type CategoryStats struct {
	// Total is the number of questions in this category.
	Total int `json:"total"`

	// Covered is the number of covered questions in this category.
	Covered int `json:"covered"`

	// Percent is the coverage percentage for this category (0-100).
	Percent float64 `json:"percent"`
}

// EvaluateQuality runs the golden questions coverage evaluation. For each
// question, it checks whether the critical_files exist in the repository.
// Glob patterns in critical_files are expanded using doublestar. A question
// is "covered" when ALL critical file patterns match at least one file.
// Questions with empty critical_files are always considered covered (with
// a debug-level warning logged).
func EvaluateQuality(opts QualityOptions) (*QualityResult, error) {
	if opts.RootDir == "" {
		return nil, fmt.Errorf("quality: root directory required")
	}

	// Step 1: Discover or use provided questions path.
	questionsPath := opts.QuestionsPath
	if questionsPath == "" {
		questionsPath = config.DiscoverGoldenQuestionsPath(opts.RootDir)
		if questionsPath == "" {
			return nil, fmt.Errorf("quality: no golden questions file found in %s (expected .harvx/golden-questions.toml or golden-questions.toml); use --questions to specify a path", opts.RootDir)
		}
	}

	slog.Debug("quality: loading golden questions",
		"path", questionsPath,
		"root", opts.RootDir,
		"profile", opts.Profile,
	)

	// Step 2: Load and validate golden questions.
	cfg, err := config.LoadGoldenQuestions(questionsPath)
	if err != nil {
		return nil, fmt.Errorf("quality: %w", err)
	}

	if err := config.ValidateGoldenQuestions(cfg); err != nil {
		return nil, fmt.Errorf("quality: %w", err)
	}

	slog.Debug("quality: loaded golden questions",
		"count", len(cfg.Questions),
		"path", questionsPath,
	)

	// Step 3: Evaluate each question.
	questions := make([]QuestionResult, 0, len(cfg.Questions))
	coveredCount := 0

	for _, q := range cfg.Questions {
		qr := evaluateQuestion(opts.RootDir, q)
		questions = append(questions, qr)
		if qr.Covered {
			coveredCount++
		}
	}

	// Step 4: Compute aggregate statistics.
	totalQuestions := len(questions)
	uncoveredCount := totalQuestions - coveredCount
	coveragePercent := 0.0
	if totalQuestions > 0 {
		coveragePercent = float64(coveredCount) / float64(totalQuestions) * 100
	}

	// Step 5: Compute per-category statistics.
	byCategory := computeCategoryStats(questions)

	result := &QualityResult{
		TotalQuestions:  totalQuestions,
		CoveredCount:    coveredCount,
		UncoveredCount:  uncoveredCount,
		CoveragePercent: coveragePercent,
		Questions:       questions,
		ByCategory:      byCategory,
		QuestionsPath:   questionsPath,
	}

	slog.Info("quality evaluation complete",
		"total_questions", totalQuestions,
		"covered", coveredCount,
		"uncovered", uncoveredCount,
		"coverage_percent", coveragePercent,
		"questions_path", questionsPath,
	)

	return result, nil
}

// evaluateQuestion checks a single golden question's critical files against
// the repository. Each critical file pattern is checked: literal paths via
// os.Stat and glob patterns via doublestar.Glob.
func evaluateQuestion(rootDir string, q config.GoldenQuestion) QuestionResult {
	qr := QuestionResult{
		ID:            q.ID,
		Question:      q.Question,
		Category:      q.Category,
		CriticalFiles: make([]string, len(q.CriticalFiles)),
	}
	copy(qr.CriticalFiles, q.CriticalFiles)
	sort.Strings(qr.CriticalFiles)

	// Questions with no critical files are always considered covered.
	if len(q.CriticalFiles) == 0 {
		slog.Debug("quality: question has no critical files, marking as covered",
			"id", q.ID,
		)
		qr.Covered = true
		return qr
	}

	var found []string
	var missing []string

	for _, pattern := range q.CriticalFiles {
		if filePatternExists(rootDir, pattern) {
			found = append(found, pattern)
			slog.Debug("quality: critical file found",
				"id", q.ID,
				"pattern", pattern,
			)
		} else {
			missing = append(missing, pattern)
			slog.Debug("quality: critical file missing",
				"id", q.ID,
				"pattern", pattern,
			)
		}
	}

	// Sort for deterministic output.
	sort.Strings(found)
	sort.Strings(missing)

	qr.FoundFiles = found
	qr.MissingFiles = missing
	qr.Covered = len(missing) == 0

	return qr
}

// isGlobPattern reports whether the pattern contains glob metacharacters.
func isGlobPattern(pattern string) bool {
	return strings.ContainsAny(pattern, "*?[{")
}

// filePatternExists checks whether a critical file pattern matches at least
// one file in the repository. Glob patterns are expanded using doublestar;
// literal paths are checked with os.Stat.
func filePatternExists(rootDir, pattern string) bool {
	if isGlobPattern(pattern) {
		matches, err := doublestar.Glob(os.DirFS(rootDir), pattern)
		if err != nil {
			slog.Debug("quality: glob expansion error",
				"pattern", pattern,
				"error", err,
			)
			return false
		}
		return len(matches) > 0
	}

	// Literal path check.
	absPath := filepath.Join(rootDir, filepath.FromSlash(pattern))
	_, err := os.Stat(absPath)
	return err == nil
}

// computeCategoryStats aggregates per-category coverage statistics from
// the evaluated question results. Categories are sorted alphabetically in
// the returned map. Questions with an empty category are grouped under
// the key "uncategorized".
func computeCategoryStats(questions []QuestionResult) map[string]CategoryStats {
	stats := make(map[string]CategoryStats)

	for _, q := range questions {
		cat := q.Category
		if cat == "" {
			cat = "uncategorized"
		}

		s := stats[cat]
		s.Total++
		if q.Covered {
			s.Covered++
		}
		stats[cat] = s
	}

	// Compute percentages.
	for cat, s := range stats {
		if s.Total > 0 {
			s.Percent = float64(s.Covered) / float64(s.Total) * 100
		}
		stats[cat] = s
	}

	return stats
}
