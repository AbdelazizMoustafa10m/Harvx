// Package workflows implements high-level workflow commands for harvx.
// This file implements the golden questions coverage analysis workflow.
// It checks whether critical files referenced by golden questions are
// present in the repository, providing a coverage metric for LLM context
// quality evaluation.
package workflows

import (
	"fmt"
	"log/slog"
	"sort"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/harvx/harvx/internal/config"
)

// QualityOptions configures the quality evaluation workflow.
type QualityOptions struct {
	// RootDir is the repository root directory to scan.
	RootDir string

	// QuestionsPath is the path to a golden questions TOML file.
	// When empty, auto-discovery is used via config.DiscoverGoldenQuestions.
	QuestionsPath string

	// ProfileName is the active profile name, used for logging context.
	ProfileName string
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

	// IncludedFiles lists critical file patterns that matched at least one
	// file in the repository (i.e., critical files that ARE in the output).
	IncludedFiles []string `json:"included_files"`

	// MissingFiles lists critical file patterns that matched zero files
	// in the repository (i.e., critical files NOT in the output).
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

	// QuestionsPath is the absolute path to the golden questions file
	// that was evaluated.
	QuestionsPath string `json:"questions_path"`
}

// EvaluateQuality runs the golden questions coverage evaluation. For each
// question, it walks the RootDir to collect file paths and checks whether
// the critical_files exist using doublestar.Match for glob patterns. A
// question is "covered" when ALL critical file patterns match at least one
// file. Questions with empty critical_files are always considered covered
// (with a debug-level note logged).
//
// If the golden questions file contains zero questions, the function returns
// a meaningful result with 0 total questions and 100% coverage.
func EvaluateQuality(opts QualityOptions) (*QualityResult, error) {
	if opts.RootDir == "" {
		return nil, fmt.Errorf("quality: root directory required")
	}

	// Step 1: Discover or use provided questions path.
	questionsPath := opts.QuestionsPath
	if questionsPath == "" {
		discovered, discoverErr := config.DiscoverGoldenQuestions(opts.RootDir)
		if discoverErr != nil {
			return nil, fmt.Errorf("quality: discovering golden questions: %w", discoverErr)
		}
		if discovered == "" {
			return nil, fmt.Errorf("quality: no golden questions file found in %s (expected .harvx/golden-questions.toml); use --questions to specify a path", opts.RootDir)
		}
		questionsPath = discovered
	}

	slog.Debug("quality: loading golden questions",
		"path", questionsPath,
		"root", opts.RootDir,
		"profile", opts.ProfileName,
	)

	// Step 2: Load golden questions.
	cfg, err := config.LoadGoldenQuestions(questionsPath)
	if err != nil {
		return nil, fmt.Errorf("quality: %w", err)
	}

	// Validate unless the file is empty (empty is a valid edge case:
	// 0 questions, 100% coverage).
	if len(cfg.Questions) > 0 {
		if err := config.ValidateGoldenQuestions(cfg); err != nil {
			return nil, fmt.Errorf("quality: %w", err)
		}
	}

	slog.Debug("quality: loaded golden questions",
		"count", len(cfg.Questions),
		"path", questionsPath,
	)

	// Step 3: Walk the repository to collect file paths (relative, skip hidden dirs).
	repoFiles, err := collectRepoFiles(opts.RootDir)
	if err != nil {
		return nil, fmt.Errorf("quality: collecting repo files: %w", err)
	}

	slog.Debug("quality: collected repo files",
		"count", len(repoFiles),
		"root", opts.RootDir,
	)

	// Step 4: Evaluate each question against the collected files.
	questions := make([]QuestionResult, 0, len(cfg.Questions))
	coveredCount := 0

	for _, q := range cfg.Questions {
		qr := evaluateQuestion(repoFiles, q)
		questions = append(questions, qr)
		if qr.Covered {
			coveredCount++
		}
	}

	// Step 5: Compute aggregate statistics.
	totalQuestions := len(questions)
	uncoveredCount := totalQuestions - coveredCount
	coveragePercent := 100.0
	if totalQuestions > 0 {
		coveragePercent = float64(coveredCount) / float64(totalQuestions) * 100
	}

	result := &QualityResult{
		TotalQuestions:  totalQuestions,
		CoveredCount:    coveredCount,
		UncoveredCount:  uncoveredCount,
		CoveragePercent: coveragePercent,
		Questions:       questions,
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
// the collected repository files. Each critical file pattern is matched using
// doublestar.Match against all repo file paths.
func evaluateQuestion(repoFiles []string, q config.GoldenQuestion) QuestionResult {
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

	var included []string
	var missing []string

	for _, pattern := range q.CriticalFiles {
		if patternMatchesAny(pattern, repoFiles) {
			included = append(included, pattern)
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
	sort.Strings(included)
	sort.Strings(missing)

	qr.IncludedFiles = included
	qr.MissingFiles = missing
	qr.Covered = len(missing) == 0

	return qr
}

// patternMatchesAny checks whether the given pattern matches at least one
// file in the list using doublestar.Match. Both literal paths and glob
// patterns are handled uniformly by doublestar.Match.
func patternMatchesAny(pattern string, files []string) bool {
	for _, f := range files {
		ok, err := doublestar.Match(pattern, f)
		if err != nil {
			slog.Debug("quality: match error",
				"pattern", pattern,
				"file", f,
				"error", err,
			)
			continue
		}
		if ok {
			return true
		}
	}
	return false
}
