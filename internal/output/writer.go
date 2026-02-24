package output

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// OutputOpts configures where and how the rendered output is written.
// It is populated from CLI flags and profile configuration.
type OutputOpts struct {
	// OutputPath is the explicit output file path from --output/-o CLI flag.
	// When empty, the path is resolved from ProfileOutput or the default.
	OutputPath string

	// ProfileOutput is the output path from the active profile configuration.
	// Lower priority than OutputPath.
	ProfileOutput string

	// Format is the output format: "markdown" or "xml".
	Format string

	// UseStdout writes to stdout instead of a file when true.
	UseStdout bool
}

// OutputResult holds the result of a successful write operation.
type OutputResult struct {
	// Path is the final output file path. Empty when writing to stdout.
	Path string

	// Hash is the XXH3 64-bit content hash of the rendered output.
	Hash uint64

	// HashHex is the hex-formatted content hash string.
	HashHex string

	// TotalTokens is the total token count from the render data.
	TotalTokens int

	// BytesWritten is the total number of bytes written to the output.
	BytesWritten int64
}

// OutputWriter orchestrates rendering output to a file or stdout. It coordinates
// the renderer, content hasher, and output destination to produce the final
// context document.
type OutputWriter struct {
	stdout io.Writer
	stderr io.Writer
}

// NewOutputWriter creates a new OutputWriter that writes to os.Stdout and
// os.Stderr.
func NewOutputWriter() *OutputWriter {
	return &OutputWriter{
		stdout: os.Stdout,
		stderr: os.Stderr,
	}
}

// NewOutputWriterWithStreams creates an OutputWriter with injectable writers,
// primarily for testing.
func NewOutputWriterWithStreams(stdout, stderr io.Writer) *OutputWriter {
	return &OutputWriter{
		stdout: stdout,
		stderr: stderr,
	}
}

// Write renders the context document and writes it to the configured destination.
// In stdout mode, it streams directly to stdout while computing the content hash.
// In file mode, it performs an atomic write using a temporary file and rename.
func (ow *OutputWriter) Write(ctx context.Context, data *RenderData, opts OutputOpts) (*OutputResult, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if data == nil {
		return nil, fmt.Errorf("writing output: render data is nil")
	}

	if opts.Format != "markdown" && opts.Format != "xml" {
		return nil, fmt.Errorf("writing output: unsupported format %q", opts.Format)
	}

	renderer, err := NewRenderer(opts.Format)
	if err != nil {
		return nil, fmt.Errorf("writing output: creating renderer: %w", err)
	}

	if opts.UseStdout {
		return ow.writeStdout(ctx, data, renderer)
	}

	return ow.writeFile(ctx, data, renderer, opts)
}

// writeStdout streams the rendered output to stdout while computing the content
// hash incrementally.
func (ow *OutputWriter) writeStdout(ctx context.Context, data *RenderData, renderer Renderer) (*OutputResult, error) {
	hasher := NewIncrementalHasher()
	cw := &countingWriter{w: ow.stdout}
	mw := io.MultiWriter(cw, hasher)

	if err := renderer.Render(ctx, mw, data); err != nil {
		return nil, fmt.Errorf("writing to stdout: %w", err)
	}

	hash := hasher.Sum64()

	return &OutputResult{
		Path:         "",
		Hash:         hash,
		HashHex:      FormatHash(hash),
		TotalTokens:  data.TotalTokens,
		BytesWritten: cw.written,
	}, nil
}

// writeFile performs an atomic file write: render to a temporary file in the
// same directory, sync, close, then rename to the final path. On any error the
// temporary file is removed.
func (ow *OutputWriter) writeFile(ctx context.Context, data *RenderData, renderer Renderer, opts OutputOpts) (_ *OutputResult, retErr error) {
	finalPath := ResolveOutputPath(opts.OutputPath, opts.ProfileOutput, opts.Format)

	dir := filepath.Dir(finalPath)
	if _, err := os.Stat(dir); err != nil {
		return nil, fmt.Errorf("writing output: output directory %q: %w", dir, err)
	}

	tmpFile, err := os.CreateTemp(dir, ".harvx-*.tmp")
	if err != nil {
		return nil, fmt.Errorf("writing output: creating temp file in %q: %w", dir, err)
	}
	tmpPath := tmpFile.Name()

	// Clean up the temp file on any error.
	defer func() {
		if retErr != nil {
			tmpFile.Close()
			os.Remove(tmpPath)
		}
	}()

	hasher := NewIncrementalHasher()
	cw := &countingWriter{w: tmpFile}
	mw := io.MultiWriter(cw, hasher)

	if err := renderer.Render(ctx, mw, data); err != nil {
		return nil, fmt.Errorf("writing output to temp file: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		return nil, fmt.Errorf("writing output: syncing temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return nil, fmt.Errorf("writing output: closing temp file: %w", err)
	}

	if err := os.Rename(tmpPath, finalPath); err != nil {
		return nil, fmt.Errorf("writing output: renaming %q to %q: %w", tmpPath, finalPath, err)
	}

	hash := hasher.Sum64()

	return &OutputResult{
		Path:         finalPath,
		Hash:         hash,
		HashHex:      FormatHash(hash),
		TotalTokens:  data.TotalTokens,
		BytesWritten: cw.written,
	}, nil
}

// countingWriter wraps an io.Writer and tracks the total number of bytes written.
type countingWriter struct {
	w       io.Writer
	written int64
}

// Write writes p to the underlying writer and adds the number of bytes written
// to the running total.
func (cw *countingWriter) Write(p []byte) (int, error) {
	n, err := cw.w.Write(p)
	cw.written += int64(n)
	return n, err
}
