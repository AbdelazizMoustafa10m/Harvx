package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// DefaultOutput is the default output file path when --output is not specified.
const DefaultOutput = "harvx-output.md"

// DefaultSkipLargeFiles is the default file size threshold (1MB) above which
// files are skipped during discovery.
const DefaultSkipLargeFiles int64 = 1 * 1024 * 1024

// FlagValues collects all parsed global flag values from the CLI. This struct
// is populated by BindFlags and passed to downstream pipeline stages.
type FlagValues struct {
	Dir             string
	Output          string
	Filters         []string // file extensions (without leading dots)
	Includes        []string // include glob patterns
	Excludes        []string // exclude glob patterns
	Format          string
	Target          string
	GitTrackedOnly  bool
	SkipLargeFiles  int64 // bytes
	Stdout          bool
	LineNumbers     bool
	NoRedact        bool
	FailOnRedaction bool
	RedactionReport string // path for --redaction-report; empty means flag not set
	Verbose         bool
	Quiet           bool
	Yes             bool
	ClearCache      bool

	// Token counting flags (T-033)
	Tokenizer          string // Tokenizer encoding: cl100k_base, o200k_base, none
	MaxTokens          int    // Token budget (0 = unlimited)
	TruncationStrategy string // Budget overflow: truncate or skip
	TokenCountOnly     bool   // Report token counts only, no output file generated
	TopFiles           int    // Show N largest files by token count (0 = disabled)
	Heatmap            bool   // Show token density heatmap (preview only)

	// Compression flags (T-049, T-050)
	Compress        bool   // Enable tree-sitter compression
	CompressTimeout int    // Per-file timeout in milliseconds (default: 5000)
	CompressEngine  string // Compression engine: ast, regex, auto (default: auto)

	// Split flag (T-056)
	Split int // Max tokens per part for multi-part output (0 = disabled)

	// Metadata sidecar flag (T-057)
	OutputMetadata bool // Generate .meta.json sidecar alongside output

	// Diff flags (T-064)
	DiffOnly bool   // Output only changed files since last run
	Profile  string // Profile name for config and state caching
}

// BindFlags registers all global persistent flags on the given Cobra command
// and returns a FlagValues pointer that will be populated when the command is
// executed. Callers should access the returned struct after flag parsing.
func BindFlags(cmd *cobra.Command) *FlagValues {
	fv := &FlagValues{}

	pf := cmd.PersistentFlags()
	pf.StringVarP(&fv.Dir, "dir", "d", ".", "target directory to scan")
	pf.StringVarP(&fv.Output, "output", "o", DefaultOutput, "output file path")
	pf.StringArrayVarP(&fv.Filters, "filter", "f", nil, "filter by file extension (repeatable, e.g. -f ts -f go)")
	pf.StringArrayVar(&fv.Includes, "include", nil, "include glob pattern (repeatable)")
	pf.StringArrayVar(&fv.Excludes, "exclude", nil, "exclude glob pattern (repeatable)")
	pf.StringVar(&fv.Format, "format", "markdown", "output format: markdown, xml")
	pf.StringVar(&fv.Target, "target", "generic", "LLM target: claude, chatgpt, generic")
	pf.BoolVar(&fv.GitTrackedOnly, "git-tracked-only", false, "only include files in git index")
	pf.StringVar(&skipLargeFilesRaw, "skip-large-files", "1MB", "skip files larger than threshold (e.g. 500KB, 2MB)")
	pf.BoolVar(&fv.Stdout, "stdout", false, "output to stdout instead of file")
	pf.BoolVar(&fv.LineNumbers, "line-numbers", false, "add line numbers to code blocks")
	pf.BoolVar(&fv.NoRedact, "no-redact", false, "disable secret redaction")
	pf.BoolVar(&fv.FailOnRedaction, "fail-on-redaction", false, "exit 1 if secrets are detected")
	pf.StringVar(&fv.RedactionReport, "redaction-report", "", "write redaction report to path (default: harvx-redaction-report.json when flag is set)")
	pf.BoolVarP(&fv.Verbose, "verbose", "v", false, "enable debug logging")
	pf.BoolVarP(&fv.Quiet, "quiet", "q", false, "suppress all output except errors")
	pf.BoolVar(&fv.Yes, "yes", false, "skip confirmation prompts")
	pf.BoolVar(&fv.ClearCache, "clear-cache", false, "clear cached state before running")

	// Token reporting flags (T-033)
	pf.StringVar(&fv.Tokenizer, "tokenizer", "cl100k_base", "Tokenizer encoding: cl100k_base, o200k_base, none")
	pf.IntVar(&fv.MaxTokens, "max-tokens", 0, "Token budget (0 = unlimited)")
	pf.StringVar(&fv.TruncationStrategy, "truncation-strategy", "skip", "Budget overflow: truncate or skip")
	pf.BoolVar(&fv.TokenCountOnly, "token-count", false, "Report token counts only, no output file generated")
	pf.IntVar(&fv.TopFiles, "top-files", 0, "Show N largest files by token count (0 = disabled)")

	// Compression flags (T-049, T-050)
	pf.BoolVar(&fv.Compress, "compress", false, "Enable tree-sitter code compression")
	pf.IntVar(&fv.CompressTimeout, "compress-timeout", 5000, "Per-file compression timeout in milliseconds")
	pf.StringVar(&fv.CompressEngine, "compress-engine", "auto", "Compression engine: ast, regex, auto, wasm (default: auto)")

	// Split flag (T-056)
	pf.IntVar(&fv.Split, "split", 0, "Split output into parts with max tokens per part (0 = disabled)")

	// Metadata sidecar flag (T-057)
	pf.BoolVar(&fv.OutputMetadata, "output-metadata", false, "Generate .meta.json sidecar file alongside output")

	// Diff flags (T-064)
	pf.BoolVar(&fv.DiffOnly, "diff-only", false, "Output only changed files since last run")
	pf.StringVar(&fv.Profile, "profile", "default", "Profile name for config and state caching")

	return fv
}

// skipLargeFilesRaw holds the raw string value for --skip-large-files before
// parsing. This is a package-level variable because Cobra needs a string target
// for binding, and we parse it into FlagValues.SkipLargeFiles during validation.
var skipLargeFilesRaw string

// ValidateFlags checks the parsed flag values for correctness and mutual
// exclusion. It also applies environment variable fallbacks and normalizes
// values. Call this from PersistentPreRunE after Cobra has parsed the flags.
func ValidateFlags(fv *FlagValues, cmd *cobra.Command) error {
	// Apply environment variable fallbacks for flags not explicitly set.
	applyEnvOverrides(fv, cmd)

	// Mutual exclusion: --verbose and --quiet
	if fv.Verbose && fv.Quiet {
		return fmt.Errorf("--verbose and --quiet are mutually exclusive")
	}

	// Mutual exclusion warning: --no-redact and --fail-on-redaction
	// --no-redact takes precedence; warn the user.
	if fv.NoRedact && fv.FailOnRedaction {
		slog.Warn("--no-redact and --fail-on-redaction are both set; --no-redact takes precedence, redaction is disabled")
	}

	// Validate --dir exists and is a directory
	info, err := os.Stat(fv.Dir)
	if err != nil {
		return fmt.Errorf("--dir: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("--dir: %s is not a directory", fv.Dir)
	}

	// Validate --format
	switch fv.Format {
	case "markdown", "xml":
		// valid
	default:
		return fmt.Errorf("--format: invalid value %q (allowed: markdown, xml)", fv.Format)
	}

	// Validate --target
	switch fv.Target {
	case "claude", "chatgpt", "generic":
		// valid
	default:
		return fmt.Errorf("--target: invalid value %q (allowed: claude, chatgpt, generic)", fv.Target)
	}

	// Parse --skip-large-files
	size, err := ParseSize(skipLargeFilesRaw)
	if err != nil {
		return fmt.Errorf("--skip-large-files: %w", err)
	}
	fv.SkipLargeFiles = size

	// Validate --tokenizer
	switch fv.Tokenizer {
	case "cl100k_base", "o200k_base", "none":
		// valid
	default:
		return fmt.Errorf("--tokenizer: invalid value %q (allowed: cl100k_base, o200k_base, none)", fv.Tokenizer)
	}

	// Validate --truncation-strategy
	switch fv.TruncationStrategy {
	case "truncate", "skip":
		// valid
	default:
		return fmt.Errorf("--truncation-strategy: invalid value %q (allowed: truncate, skip)", fv.TruncationStrategy)
	}

	// Validate --compress-timeout
	if fv.CompressTimeout <= 0 {
		return fmt.Errorf("--compress-timeout: must be a positive integer, got %d", fv.CompressTimeout)
	}

	// Validate --compress-engine
	switch fv.CompressEngine {
	case "ast", "regex", "auto", "wasm":
		// valid
	default:
		return fmt.Errorf("--compress-engine: invalid value %q (allowed: ast, regex, auto, wasm)", fv.CompressEngine)
	}

	// Validate --split
	if fv.Split < 0 {
		return fmt.Errorf("--split: must be a non-negative integer, got %d", fv.Split)
	}

	// Normalize --filter: strip leading dots
	for i, f := range fv.Filters {
		fv.Filters[i] = strings.TrimLeft(f, ".")
	}

	return nil
}

// applyEnvOverrides applies environment variable fallbacks for flags that were
// not explicitly set on the command line. The prefix is HARVX_.
func applyEnvOverrides(fv *FlagValues, cmd *cobra.Command) {
	envMap := map[string]func(string){
		"HARVX_DIR": func(v string) { fv.Dir = v },
		"HARVX_OUTPUT": func(v string) { fv.Output = v },
		"HARVX_FORMAT": func(v string) { fv.Format = v },
		"HARVX_TARGET": func(v string) { fv.Target = v },
	}

	for env, setter := range envMap {
		v := os.Getenv(env)
		if v == "" {
			continue
		}
		// Only apply if the corresponding flag was not explicitly set.
		flagName := strings.ToLower(strings.TrimPrefix(env, "HARVX_"))
		if !cmd.Flags().Changed(flagName) {
			setter(v)
		}
	}

	// Boolean env vars
	if os.Getenv("HARVX_VERBOSE") == "1" && !cmd.Flags().Changed("verbose") {
		fv.Verbose = true
	}
	if os.Getenv("HARVX_QUIET") == "1" && !cmd.Flags().Changed("quiet") {
		fv.Quiet = true
	}
	// HARVX_NO_REDACT=1 disables redaction (equivalent to --no-redact)
	if os.Getenv("HARVX_NO_REDACT") == "1" && !cmd.Flags().Changed("no-redact") {
		fv.NoRedact = true
	}
	// HARVX_FAIL_ON_REDACTION=1 enables CI enforcement mode
	if os.Getenv("HARVX_FAIL_ON_REDACTION") == "1" && !cmd.Flags().Changed("fail-on-redaction") {
		fv.FailOnRedaction = true
	}
}

// ParseSize parses a human-readable size string into bytes. It supports KB, MB,
// and GB suffixes (case-insensitive). Plain numbers without a suffix are treated
// as bytes. KB = 1024, MB = 1048576, GB = 1073741824.
func ParseSize(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty size string")
	}

	upper := strings.ToUpper(s)

	var suffix string
	var multiplier int64

	switch {
	case strings.HasSuffix(upper, "GB"):
		suffix = "GB"
		multiplier = 1024 * 1024 * 1024
	case strings.HasSuffix(upper, "MB"):
		suffix = "MB"
		multiplier = 1024 * 1024
	case strings.HasSuffix(upper, "KB"):
		suffix = "KB"
		multiplier = 1024
	default:
		// Plain number, treat as bytes
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid size: %q", s)
		}
		if n < 0 {
			return 0, fmt.Errorf("size must be non-negative: %q", s)
		}
		return n, nil
	}

	numStr := strings.TrimSpace(s[:len(s)-len(suffix)])
	n, err := strconv.ParseInt(numStr, 10, 64)
	if err != nil {
		// Try float for things like "1.5MB"
		f, ferr := strconv.ParseFloat(numStr, 64)
		if ferr != nil {
			return 0, fmt.Errorf("invalid size: %q", s)
		}
		if f < 0 {
			return 0, fmt.Errorf("size must be non-negative: %q", s)
		}
		return int64(f * float64(multiplier)), nil
	}
	if n < 0 {
		return 0, fmt.Errorf("size must be non-negative: %q", s)
	}
	return n * multiplier, nil
}
