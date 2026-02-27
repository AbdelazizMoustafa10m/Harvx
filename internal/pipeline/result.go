package pipeline

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// RunOptions encapsulates all parameters for a single pipeline Run invocation.
type RunOptions struct {
	// Dir is the target directory to scan.
	Dir string `json:"dir"`

	// MaxTokens overrides the token budget (0 = use config default).
	MaxTokens int `json:"max_tokens,omitempty"`

	// PathFilter limits processing to specific paths (relative to Dir).
	// Empty means process all discovered files.
	PathFilter []string `json:"path_filter,omitempty"`

	// GitRef is a git reference for diff-based workflows (e.g., "HEAD~1").
	GitRef string `json:"git_ref,omitempty"`

	// GitBaseRef is the base ref for PR-style diff (e.g., "main").
	GitBaseRef string `json:"git_base_ref,omitempty"`

	// GitHeadRef is the head ref for PR-style diff (e.g., "feature-branch").
	GitHeadRef string `json:"git_head_ref,omitempty"`

	// Stages controls which pipeline stages to execute.
	// Nil or empty means run all configured stages.
	Stages *StageSelection `json:"stages,omitempty"`
}

// StageSelection controls which pipeline stages are executed.
// When a field is false, that stage is skipped. By default all stages
// are enabled. Use NewStageSelection to get defaults.
type StageSelection struct {
	Discovery   bool `json:"discovery"`
	Relevance   bool `json:"relevance"`
	Tokenize    bool `json:"tokenize"`
	Budget      bool `json:"budget"`
	Redaction   bool `json:"redaction"`
	Compression bool `json:"compression"`
	Render      bool `json:"render"`
}

// NewStageSelection returns a StageSelection with all stages enabled.
func NewStageSelection() *StageSelection {
	return &StageSelection{
		Discovery:   true,
		Relevance:   true,
		Tokenize:    true,
		Budget:      true,
		Redaction:   true,
		Compression: true,
		Render:      true,
	}
}

// DiscoveryOnly returns a StageSelection with only discovery enabled.
func DiscoveryOnly() *StageSelection {
	return &StageSelection{
		Discovery: true,
	}
}

// DiscoveryAndRelevance returns a StageSelection with discovery and relevance enabled.
func DiscoveryAndRelevance() *StageSelection {
	return &StageSelection{
		Discovery: true,
		Relevance: true,
	}
}

// RunResult is the structured result of a pipeline Run invocation.
// It contains all processed files, aggregate statistics, and timing data.
type RunResult struct {
	// Files is the processed file descriptors after all enabled stages.
	Files []FileDescriptor `json:"files"`

	// Stats holds aggregate statistics about the pipeline run.
	Stats RunStats `json:"stats"`

	// Timings records wall-clock duration for each pipeline stage.
	Timings StageTimings `json:"timings"`

	// ContentHash is the XXH3 hash of all processed content.
	ContentHash uint64 `json:"content_hash"`

	// ExitCode is the pipeline exit code (0=success, 1=error, 2=partial).
	ExitCode ExitCode `json:"exit_code"`
}

// RunStats holds aggregate statistics about a pipeline run.
type RunStats struct {
	// TotalFiles is the number of files included in the result.
	TotalFiles int `json:"total_files"`

	// TotalTokens is the sum of tokens across all included files.
	TotalTokens int `json:"total_tokens"`

	// TokenizerName is the tokenizer encoding used.
	TokenizerName string `json:"tokenizer_name"`

	// TierBreakdown maps tier number to file count.
	TierBreakdown map[int]int `json:"tier_breakdown"`

	// RedactionCount is the total number of secrets redacted.
	RedactionCount int `json:"redaction_count"`

	// CompressedFiles is the number of files that were compressed.
	CompressedFiles int `json:"compressed_files"`

	// DiscoveryTotal is the total files found before filtering.
	DiscoveryTotal int `json:"discovery_total"`

	// DiscoverySkipped is the total files skipped during discovery.
	DiscoverySkipped int `json:"discovery_skipped"`
}

// StageTimings records wall-clock duration for each pipeline stage.
// Zero duration means the stage was not executed.
type StageTimings struct {
	Discovery   time.Duration `json:"discovery"`
	Relevance   time.Duration `json:"relevance"`
	Tokenize    time.Duration `json:"tokenize"`
	Budget      time.Duration `json:"budget"`
	Redaction   time.Duration `json:"redaction"`
	Compression time.Duration `json:"compression"`
	Render      time.Duration `json:"render"`
	Total       time.Duration `json:"total"`
}

// MarshalJSON implements custom JSON serialization for StageTimings,
// encoding durations as human-readable strings (e.g., "1.234s").
func (st StageTimings) MarshalJSON() ([]byte, error) {
	type alias struct {
		Discovery   string `json:"discovery"`
		Relevance   string `json:"relevance"`
		Tokenize    string `json:"tokenize"`
		Budget      string `json:"budget"`
		Redaction   string `json:"redaction"`
		Compression string `json:"compression"`
		Render      string `json:"render"`
		Total       string `json:"total"`
	}
	return json.Marshal(alias{
		Discovery:   st.Discovery.String(),
		Relevance:   st.Relevance.String(),
		Tokenize:    st.Tokenize.String(),
		Budget:      st.Budget.String(),
		Redaction:   st.Redaction.String(),
		Compression: st.Compression.String(),
		Render:      st.Render.String(),
		Total:       st.Total.String(),
	})
}

// UnmarshalJSON implements custom JSON deserialization for StageTimings,
// parsing human-readable duration strings (e.g., "1.234s") back to time.Duration.
func (st *StageTimings) UnmarshalJSON(data []byte) error {
	type alias struct {
		Discovery   string `json:"discovery"`
		Relevance   string `json:"relevance"`
		Tokenize    string `json:"tokenize"`
		Budget      string `json:"budget"`
		Redaction   string `json:"redaction"`
		Compression string `json:"compression"`
		Render      string `json:"render"`
		Total       string `json:"total"`
	}
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	var err error
	if st.Discovery, err = time.ParseDuration(a.Discovery); err != nil {
		return fmt.Errorf("parsing discovery duration: %w", err)
	}
	if st.Relevance, err = time.ParseDuration(a.Relevance); err != nil {
		return fmt.Errorf("parsing relevance duration: %w", err)
	}
	if st.Tokenize, err = time.ParseDuration(a.Tokenize); err != nil {
		return fmt.Errorf("parsing tokenize duration: %w", err)
	}
	if st.Budget, err = time.ParseDuration(a.Budget); err != nil {
		return fmt.Errorf("parsing budget duration: %w", err)
	}
	if st.Redaction, err = time.ParseDuration(a.Redaction); err != nil {
		return fmt.Errorf("parsing redaction duration: %w", err)
	}
	if st.Compression, err = time.ParseDuration(a.Compression); err != nil {
		return fmt.Errorf("parsing compression duration: %w", err)
	}
	if st.Render, err = time.ParseDuration(a.Render); err != nil {
		return fmt.Errorf("parsing render duration: %w", err)
	}
	if st.Total, err = time.ParseDuration(a.Total); err != nil {
		return fmt.Errorf("parsing total duration: %w", err)
	}
	return nil
}

// PreviewResult is the JSON output for `harvx preview --json`.
// It provides machine-readable pipeline metadata for orchestration scripts
// and CI pipelines that need to make programmatic decisions based on Harvx's
// analysis without parsing human-readable text.
type PreviewResult struct {
	// TotalFiles is the number of files included after filtering.
	TotalFiles int `json:"total_files"`

	// TotalTokens is the sum of tokens across all included files.
	TotalTokens int `json:"total_tokens"`

	// Tokenizer is the tokenizer encoding name (e.g., "cl100k_base", "o200k_base").
	Tokenizer string `json:"tokenizer"`

	// Tiers maps tier number (as string key) to the count of files in that tier.
	// String keys are used because JSON object keys must be strings.
	Tiers map[string]int `json:"tiers"`

	// Redactions is the total number of secrets redacted across all files.
	Redactions int `json:"redactions"`

	// EstimatedTimeMs is the actual pipeline execution time in milliseconds.
	EstimatedTimeMs int64 `json:"estimated_time_ms"`

	// ContentHash is the XXH3 hash of all processed content, formatted as
	// lowercase hexadecimal.
	ContentHash string `json:"content_hash"`

	// Profile is the config profile name used for the run.
	Profile string `json:"profile"`

	// BudgetUtilizationPercent is the percentage of the token budget used.
	// Nil when no budget (MaxTokens == 0) is set, serialized as JSON null.
	BudgetUtilizationPercent *float64 `json:"budget_utilization_percent"`

	// FilesTruncated is the number of files that were truncated to fit the budget.
	FilesTruncated int `json:"files_truncated"`

	// FilesOmitted is the number of files skipped during discovery (ignored,
	// binary, oversized, etc.).
	FilesOmitted int `json:"files_omitted"`
}

// BuildPreviewResult converts a RunResult into a PreviewResult for JSON output.
// The profile parameter is the config profile name. The maxTokens parameter is
// the token budget (0 means no budget). When maxTokens is 0,
// BudgetUtilizationPercent is set to nil (JSON null).
func BuildPreviewResult(result *RunResult, profile string, maxTokens int) *PreviewResult {
	// Convert TierBreakdown map[int]int to map[string]int for JSON string keys.
	tiers := make(map[string]int, len(result.Stats.TierBreakdown))
	for tier, count := range result.Stats.TierBreakdown {
		tiers[strconv.Itoa(tier)] = count
	}

	// Budget utilization: nil when no budget is set.
	var budgetPct *float64
	if maxTokens > 0 {
		pct := (float64(result.Stats.TotalTokens) / float64(maxTokens)) * 100
		budgetPct = &pct
	}

	// Content hash as lowercase hex.
	contentHash := fmt.Sprintf("%x", result.ContentHash)

	// Estimated time from actual pipeline duration.
	estimatedTimeMs := result.Timings.Total.Milliseconds()

	return &PreviewResult{
		TotalFiles:               result.Stats.TotalFiles,
		TotalTokens:              result.Stats.TotalTokens,
		Tokenizer:                result.Stats.TokenizerName,
		Tiers:                    tiers,
		Redactions:               result.Stats.RedactionCount,
		EstimatedTimeMs:          estimatedTimeMs,
		ContentHash:              contentHash,
		Profile:                  profile,
		BudgetUtilizationPercent: budgetPct,
		FilesTruncated:           0, // Populated from BudgetResult when budget stage runs.
		FilesOmitted:             result.Stats.DiscoverySkipped,
	}
}

// PreviewStages returns a StageSelection configured for preview mode:
// discovery, relevance, and tokenization are enabled. Budget, redaction,
// compression, and rendering are disabled for speed.
func PreviewStages() *StageSelection {
	return &StageSelection{
		Discovery: true,
		Relevance: true,
		Tokenize:  true,
	}
}
