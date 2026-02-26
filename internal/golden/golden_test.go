package golden

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/harvx/harvx/internal/discovery"
	"github.com/harvx/harvx/internal/output"
	"github.com/harvx/harvx/internal/pipeline"
	"github.com/harvx/harvx/internal/relevance"
	"github.com/harvx/harvx/internal/security"
	"github.com/harvx/harvx/internal/tokenizer"
)

// fixedTimestamp is used for all golden tests to ensure deterministic output.
var fixedTimestamp = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

// goldenTests defines the test scenarios for golden file comparison. Each
// scenario walks a testdata directory, runs it through the pipeline stages,
// renders the output, and compares against a golden reference file.
var goldenTests = []struct {
	name      string
	dir       string // relative to the project root testdata/ directory
	format    string // "markdown" or "xml"
	redact    bool
	maxTokens int    // 0 = unlimited
}{
	{
		name:   "default-profile-markdown.md",
		dir:    "sample-repo",
		format: "markdown",
	},
	{
		name:   "default-profile-xml.xml",
		dir:    "sample-repo",
		format: "xml",
	},
	{
		name:   "redacted-markdown.md",
		dir:    "secrets",
		format: "markdown",
		redact: true,
	},
	{
		name:      "token-budget-10k.md",
		dir:       "sample-repo",
		format:    "markdown",
		maxTokens: 10000,
	},
	{
		name:   "monorepo-markdown.md",
		dir:    "monorepo",
		format: "markdown",
	},
}

// TestGolden runs the golden test suite. For each scenario it:
//  1. Discovers files via discovery.Walker
//  2. Classifies relevance tiers via relevance.TierMatcher
//  3. Optionally applies secret redaction
//  4. Counts tokens using the "none" estimator tokenizer
//  5. Applies budget enforcement if maxTokens > 0
//  6. Renders output to a buffer
//  7. Compares the output against the golden reference file
//
// Run with -update to regenerate golden files:
//
//	go test ./internal/golden/ -update -count=1
func TestGolden(t *testing.T) {
	// Resolve the testdata directory. The test working directory is
	// internal/golden/, so testdata is at ../../testdata/.
	testdataRoot, err := filepath.Abs(filepath.Join("..", "..", "testdata"))
	require.NoError(t, err)

	for _, tt := range goldenTests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			targetDir := filepath.Join(testdataRoot, tt.dir)

			// Verify the target directory exists.
			info, err := os.Stat(targetDir)
			require.NoError(t, err, "target directory %s must exist", targetDir)
			require.True(t, info.IsDir(), "target %s must be a directory", targetDir)

			// Step 1: Discover files.
			files := discoverFiles(t, ctx, targetDir)

			// Step 2: Classify relevance tiers.
			files = classifyTiers(files)

			// Step 3: Optionally redact secrets (before tokenization).
			if tt.redact {
				files = redactFiles(t, ctx, files)
			}

			// Step 4: Count tokens using "none" estimator.
			// Tokenization runs after redaction in the real pipeline.
			tok := countTokens(t, files)

			// Step 5: Budget enforcement.
			if tt.maxTokens > 0 {
				files = enforceBudget(files, tt.maxTokens)
			}

			// Step 6: Render output to buffer.
			buf := renderOutput(t, ctx, files, tt.format, tok.Name(), targetDir, tt.maxTokens)

			// Step 7: Compare against golden file.
			GoldenFile(t, tt.name, buf.Bytes(), targetDir)
		})
	}
}

// discoverFiles runs the discovery walker against the target directory.
func discoverFiles(t *testing.T, ctx context.Context, dir string) []pipeline.FileDescriptor {
	t.Helper()

	walker := discovery.NewWalker()

	// Build ignore matchers for the target directory.
	gitignoreMatcher, err := discovery.NewGitignoreMatcher(dir)
	require.NoError(t, err)

	harvxignoreMatcher, err := discovery.NewHarvxignoreMatcher(dir)
	require.NoError(t, err)

	defaultIgnorer := discovery.NewDefaultIgnoreMatcher()

	cfg := discovery.WalkerConfig{
		Root:                      dir,
		GitignoreMatcher:          gitignoreMatcher,
		HarvxignoreMatcher:        harvxignoreMatcher,
		DefaultIgnorer:            defaultIgnorer,
		Concurrency:               1, // Sequential for deterministic ordering.
		SuppressSensitiveWarnings: true,
	}

	result, err := walker.Walk(ctx, cfg)
	require.NoError(t, err)

	return result.Files
}

// classifyTiers assigns relevance tiers to files using default tier definitions.
func classifyTiers(files []pipeline.FileDescriptor) []pipeline.FileDescriptor {
	defs := relevance.DefaultTierDefinitions()
	ptrs := make([]*pipeline.FileDescriptor, len(files))
	for i := range files {
		ptrs[i] = &files[i]
	}

	ptrs = relevance.ClassifyAndSort(ptrs, defs)

	result := make([]pipeline.FileDescriptor, len(ptrs))
	for i, p := range ptrs {
		result[i] = *p
	}
	return result
}

// countTokens applies the "none" estimator tokenizer to all files.
func countTokens(t *testing.T, files []pipeline.FileDescriptor) tokenizer.Tokenizer {
	t.Helper()

	tok, err := tokenizer.NewTokenizer("none")
	require.NoError(t, err)

	for i := range files {
		if files[i].Content != "" {
			files[i].TokenCount = tok.Count(files[i].Content)
		}
	}

	return tok
}

// redactFiles applies secret redaction to all files with content.
func redactFiles(t *testing.T, ctx context.Context, files []pipeline.FileDescriptor) []pipeline.FileDescriptor {
	t.Helper()

	cfg := security.RedactionConfig{
		Enabled:             true,
		ConfidenceThreshold: security.ConfidenceMedium,
	}
	redactor := security.NewStreamRedactor(nil, nil, cfg)

	for i := range files {
		if files[i].Content == "" {
			continue
		}
		redacted, matches, err := redactor.Redact(ctx, files[i].Content, files[i].Path)
		require.NoError(t, err)
		files[i].Content = redacted
		files[i].Redactions = len(matches)
	}

	return files
}

// enforceBudget drops files that exceed the token budget. Files are included
// in tier order (lowest tier first), then alphabetically. When the budget is
// exhausted, remaining files are dropped.
func enforceBudget(files []pipeline.FileDescriptor, maxTokens int) []pipeline.FileDescriptor {
	// Files should already be sorted by tier then path from classifyTiers.
	// Sort again to be safe.
	sort.SliceStable(files, func(i, j int) bool {
		if files[i].Tier != files[j].Tier {
			return files[i].Tier < files[j].Tier
		}
		return files[i].Path < files[j].Path
	})

	var included []pipeline.FileDescriptor
	total := 0
	for _, fd := range files {
		if total+fd.TokenCount > maxTokens {
			continue
		}
		total += fd.TokenCount
		included = append(included, fd)
	}
	return included
}

// renderOutput renders the file descriptors into the specified format and
// returns the rendered bytes.
func renderOutput(
	t *testing.T,
	ctx context.Context,
	files []pipeline.FileDescriptor,
	format string,
	tokenizerName string,
	dir string,
	maxTokens int,
) *bytes.Buffer {
	t.Helper()

	// Use a buffer-based OutputWriter so nothing goes to real stdout.
	var buf bytes.Buffer
	writer := output.NewOutputWriterWithStreams(&buf, &buf)

	cfg := output.OutputConfig{
		Format:        format,
		Target:        "generic",
		UseStdout:     true,
		ProjectName:   filepath.Base(dir),
		ProfileName:   "default",
		TokenizerName: tokenizerName,
		Timestamp:     fixedTimestamp,
		MaxTokens:     maxTokens,
		Writer:        writer,
	}

	_, err := output.RenderOutput(ctx, cfg, files)
	require.NoError(t, err)

	return &buf
}
