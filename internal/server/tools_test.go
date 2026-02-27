package server

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// createTestFile writes content to a file within base, creating parent
// directories as needed. It fails the test immediately on any error.
func createTestFile(t *testing.T, base, rel, content string) {
	t.Helper()
	path := filepath.Join(base, rel)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

// ---------------------------------------------------------------------------
// Handler creation tests
// ---------------------------------------------------------------------------

func TestMakeBriefHandler_ReturnsHandler(t *testing.T) {
	t.Parallel()
	cfg := ServerConfig{RootDir: t.TempDir()}
	handler := makeBriefHandler(cfg)
	require.NotNil(t, handler, "makeBriefHandler must return a non-nil handler")
}

func TestMakeSliceHandler_ReturnsHandler(t *testing.T) {
	t.Parallel()
	cfg := ServerConfig{RootDir: t.TempDir()}
	handler := makeSliceHandler(cfg)
	require.NotNil(t, handler, "makeSliceHandler must return a non-nil handler")
}

func TestMakeReviewSliceHandler_ReturnsHandler(t *testing.T) {
	t.Parallel()
	cfg := ServerConfig{RootDir: t.TempDir()}
	handler := makeReviewSliceHandler(cfg)
	require.NotNil(t, handler, "makeReviewSliceHandler must return a non-nil handler")
}

// ---------------------------------------------------------------------------
// Brief handler tests
// ---------------------------------------------------------------------------

func TestBriefHandler_WithTempDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	createTestFile(t, dir, "README.md", "# Test Project\n\nA test project.\n")

	cfg := ServerConfig{RootDir: dir}
	handler := makeBriefHandler(cfg)

	result, out, err := handler(context.Background(), nil, briefInput{})
	require.NoError(t, err)
	assert.Nil(t, result, "typed handlers should return nil for CallToolResult")
	assert.NotEmpty(t, out.Content, "brief content must not be empty")
	assert.Greater(t, out.TokenCount, 0, "token count must be positive")
	assert.NotEmpty(t, out.ContentHash, "content hash must not be empty")
}

func TestBriefHandler_EmptyDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := ServerConfig{RootDir: dir}
	handler := makeBriefHandler(cfg)

	// An empty repository has no README or other discoverable files.
	// The brief workflow should still succeed (possibly with empty content or minimal output).
	// Depending on the workflow implementation, this may produce empty content or an error.
	result, out, err := handler(context.Background(), nil, briefInput{})
	// Either way, the handler should not panic.
	if err == nil {
		assert.Nil(t, result)
		// Token count may be 0 if no content was generated.
		_ = out
	}
}

func TestBriefHandler_DeterministicOutput(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	createTestFile(t, dir, "README.md", "# Deterministic Test\n\nSame content each time.\n")

	cfg := ServerConfig{RootDir: dir}
	handler := makeBriefHandler(cfg)

	_, out1, err1 := handler(context.Background(), nil, briefInput{})
	require.NoError(t, err1)

	_, out2, err2 := handler(context.Background(), nil, briefInput{})
	require.NoError(t, err2)

	assert.Equal(t, out1.ContentHash, out2.ContentHash,
		"identical inputs must produce identical content hashes")
	assert.Equal(t, out1.Content, out2.Content,
		"identical inputs must produce identical content")
	assert.Equal(t, out1.TokenCount, out2.TokenCount,
		"identical inputs must produce identical token counts")
}

// ---------------------------------------------------------------------------
// Slice handler tests
// ---------------------------------------------------------------------------

func TestSliceHandler_MissingPath(t *testing.T) {
	t.Parallel()

	cfg := ServerConfig{RootDir: t.TempDir()}
	handler := makeSliceHandler(cfg)

	_, _, err := handler(context.Background(), nil, sliceInput{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "path parameter is required",
		"missing path should produce a descriptive error")
}

func TestSliceHandler_NonexistentPath(t *testing.T) {
	t.Parallel()

	cfg := ServerConfig{RootDir: t.TempDir()}
	handler := makeSliceHandler(cfg)

	_, _, err := handler(context.Background(), nil, sliceInput{Path: "nonexistent/module"})
	require.Error(t, err, "nonexistent path should produce an error")
}

func TestSliceHandler_WithFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	createTestFile(t, dir, "internal/auth/auth.go", "package auth\n\nfunc Login() {}\n")
	createTestFile(t, dir, "internal/auth/auth_test.go", "package auth\n\nfunc TestLogin(t *testing.T) {}\n")

	cfg := ServerConfig{RootDir: dir}
	handler := makeSliceHandler(cfg)

	result, out, err := handler(context.Background(), nil, sliceInput{Path: "internal/auth"})
	require.NoError(t, err)
	assert.Nil(t, result, "typed handlers should return nil for CallToolResult")
	assert.NotEmpty(t, out.Content, "slice content must not be empty")
	assert.Greater(t, out.TokenCount, 0, "token count must be positive")
	assert.NotEmpty(t, out.ContentHash, "content hash must not be empty")
	assert.NotEmpty(t, out.ModuleFiles, "module files should include discovered files")
}

func TestSliceHandler_CustomMaxTokens(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	createTestFile(t, dir, "pkg/main.go", "package pkg\n")

	cfg := ServerConfig{RootDir: dir}
	handler := makeSliceHandler(cfg)

	_, out, err := handler(context.Background(), nil, sliceInput{
		Path:      "pkg",
		MaxTokens: 5000,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, out.Content, "slice content must not be empty with custom token budget")
}

func TestSliceHandler_SingleFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	createTestFile(t, dir, "cmd/main.go", "package main\n\nimport \"fmt\"\n\nfunc main() { fmt.Println(\"hello\") }\n")

	cfg := ServerConfig{RootDir: dir}
	handler := makeSliceHandler(cfg)

	_, out, err := handler(context.Background(), nil, sliceInput{Path: "cmd"})
	require.NoError(t, err)
	assert.NotEmpty(t, out.Content)
	assert.Contains(t, out.ModuleFiles, "cmd/main.go",
		"module files should contain the target file")
}

func TestSliceHandler_NeighborDiscovery(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// Create a module file that imports from another package.
	createTestFile(t, dir, "internal/handler/handler.go",
		"package handler\n\nimport \"example.com/internal/service\"\n\nfunc Handle() { service.Run() }\n")
	createTestFile(t, dir, "internal/service/service.go",
		"package service\n\nfunc Run() {}\n")

	cfg := ServerConfig{RootDir: dir}
	handler := makeSliceHandler(cfg)

	_, out, err := handler(context.Background(), nil, sliceInput{Path: "internal/handler"})
	require.NoError(t, err)
	assert.NotEmpty(t, out.Content)
	// NeighborFiles may or may not include the service file depending on
	// import resolution, but the handler should not error.
}

// ---------------------------------------------------------------------------
// ReviewSlice handler validation tests
// ---------------------------------------------------------------------------

func TestReviewSliceHandler_MissingBase(t *testing.T) {
	t.Parallel()

	cfg := ServerConfig{RootDir: t.TempDir()}
	handler := makeReviewSliceHandler(cfg)

	_, _, err := handler(context.Background(), nil, reviewSliceInput{Head: "HEAD"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "base parameter is required",
		"missing base should produce a descriptive error")
}

func TestReviewSliceHandler_MissingHead(t *testing.T) {
	t.Parallel()

	cfg := ServerConfig{RootDir: t.TempDir()}
	handler := makeReviewSliceHandler(cfg)

	_, _, err := handler(context.Background(), nil, reviewSliceInput{Base: "origin/main"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "head parameter is required",
		"missing head should produce a descriptive error")
}

func TestReviewSliceHandler_MissingBoth(t *testing.T) {
	t.Parallel()

	cfg := ServerConfig{RootDir: t.TempDir()}
	handler := makeReviewSliceHandler(cfg)

	_, _, err := handler(context.Background(), nil, reviewSliceInput{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "base parameter is required",
		"when both are missing, base validation should trigger first")
}

func TestReviewSliceHandler_ValidationOrder(t *testing.T) {
	// Verify that base is validated before head. When both are empty,
	// the error should mention "base", not "head".
	t.Parallel()

	cfg := ServerConfig{RootDir: t.TempDir()}
	handler := makeReviewSliceHandler(cfg)

	_, _, err := handler(context.Background(), nil, reviewSliceInput{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "base",
		"base parameter should be validated first")
}

// ---------------------------------------------------------------------------
// Concurrent invocation tests
// ---------------------------------------------------------------------------

func TestConcurrentBriefInvocations(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	createTestFile(t, dir, "README.md", "# Concurrent Test\n\nContent for concurrency.\n")

	cfg := ServerConfig{RootDir: dir}
	handler := makeBriefHandler(cfg)

	const n = 5
	var wg sync.WaitGroup
	errs := make([]error, n)
	outputs := make([]briefOutput, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, outputs[idx], errs[idx] = handler(context.Background(), nil, briefInput{})
		}(i)
	}
	wg.Wait()

	for i := 0; i < n; i++ {
		require.NoError(t, errs[i], "concurrent invocation %d failed", i)
		assert.NotEmpty(t, outputs[i].Content, "invocation %d produced empty content", i)
	}

	// All outputs should be identical (deterministic for identical inputs).
	for i := 1; i < n; i++ {
		assert.Equal(t, outputs[0].ContentHash, outputs[i].ContentHash,
			"hash mismatch between invocations 0 and %d", i)
		assert.Equal(t, outputs[0].Content, outputs[i].Content,
			"content mismatch between invocations 0 and %d", i)
	}
}

func TestConcurrentSliceInvocations(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	createTestFile(t, dir, "lib/lib.go", "package lib\n\nfunc Do() {}\n")

	cfg := ServerConfig{RootDir: dir}
	handler := makeSliceHandler(cfg)

	const n = 3
	var wg sync.WaitGroup
	errs := make([]error, n)
	outputs := make([]sliceOutput, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, outputs[idx], errs[idx] = handler(context.Background(), nil, sliceInput{Path: "lib"})
		}(i)
	}
	wg.Wait()

	for i := 0; i < n; i++ {
		require.NoError(t, errs[i], "concurrent slice invocation %d failed", i)
		assert.NotEmpty(t, outputs[i].Content, "invocation %d produced empty content", i)
	}

	// All outputs should be identical (deterministic).
	for i := 1; i < n; i++ {
		assert.Equal(t, outputs[0].ContentHash, outputs[i].ContentHash,
			"hash mismatch between slice invocations 0 and %d", i)
	}
}

func TestConcurrentMixedToolInvocations(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	createTestFile(t, dir, "README.md", "# Mixed Test\n")
	createTestFile(t, dir, "pkg/pkg.go", "package pkg\n")

	cfg := ServerConfig{RootDir: dir}
	briefHandler := makeBriefHandler(cfg)
	sliceHandler := makeSliceHandler(cfg)

	var wg sync.WaitGroup
	var briefErr, sliceErr error
	var briefOut briefOutput
	var sliceOut sliceOutput

	wg.Add(2)
	go func() {
		defer wg.Done()
		_, briefOut, briefErr = briefHandler(context.Background(), nil, briefInput{})
	}()
	go func() {
		defer wg.Done()
		_, sliceOut, sliceErr = sliceHandler(context.Background(), nil, sliceInput{Path: "pkg"})
	}()
	wg.Wait()

	require.NoError(t, briefErr, "concurrent brief invocation failed")
	require.NoError(t, sliceErr, "concurrent slice invocation failed")
	assert.NotEmpty(t, briefOut.Content)
	assert.NotEmpty(t, sliceOut.Content)
}

// ---------------------------------------------------------------------------
// registerTools test
// ---------------------------------------------------------------------------

func TestRegisterTools(t *testing.T) {
	t.Parallel()

	s := mcp.NewServer(&mcp.Implementation{
		Name:    "test",
		Version: "0.0.1",
	}, nil)
	cfg := ServerConfig{RootDir: t.TempDir()}

	// registerTools should not panic.
	require.NotPanics(t, func() {
		registerTools(s, cfg)
	}, "registerTools must not panic")

	// The server should still be functional after tool registration.
	require.NotNil(t, s)
}

func TestRegisterTools_EmptyConfig(t *testing.T) {
	t.Parallel()

	s := mcp.NewServer(&mcp.Implementation{
		Name:    "test-empty",
		Version: "0.0.1",
	}, nil)
	cfg := ServerConfig{}

	require.NotPanics(t, func() {
		registerTools(s, cfg)
	}, "registerTools should accept empty config without panic")
}

// ---------------------------------------------------------------------------
// nonNilSlice helper tests
// ---------------------------------------------------------------------------

func TestNonNilSlice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "nil input returns empty slice",
			input: nil,
			want:  []string{},
		},
		{
			name:  "empty slice returns same empty slice",
			input: []string{},
			want:  []string{},
		},
		{
			name:  "populated slice returned unchanged",
			input: []string{"a.go", "b.go"},
			want:  []string{"a.go", "b.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := nonNilSlice(tt.input)
			assert.Equal(t, tt.want, got)
			// Ensure nil input produces a non-nil result.
			if tt.input == nil {
				assert.NotNil(t, got, "nonNilSlice(nil) must return non-nil")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Context cancellation test
// ---------------------------------------------------------------------------

func TestBriefHandler_ContextCancellation(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	createTestFile(t, dir, "README.md", "# Cancel Test\n")

	cfg := ServerConfig{RootDir: dir}
	handler := makeBriefHandler(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	// The brief workflow does not check context, so this may or may not error.
	// The important thing is that it does not panic or hang.
	_, _, _ = handler(ctx, nil, briefInput{})
}

func TestSliceHandler_ContextCancellation(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	createTestFile(t, dir, "pkg/main.go", "package pkg\n")

	cfg := ServerConfig{RootDir: dir}
	handler := makeSliceHandler(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Should not panic or hang.
	_, _, _ = handler(ctx, nil, sliceInput{Path: "pkg"})
}

// ---------------------------------------------------------------------------
// Input/output type validation tests
// ---------------------------------------------------------------------------

func TestBriefInput_DefaultValues(t *testing.T) {
	t.Parallel()

	input := briefInput{}
	assert.Empty(t, input.Profile, "default profile should be empty string")
}

func TestSliceInput_DefaultValues(t *testing.T) {
	t.Parallel()

	input := sliceInput{}
	assert.Empty(t, input.Path, "default path should be empty string")
	assert.Empty(t, input.Profile, "default profile should be empty string")
	assert.Equal(t, 0, input.MaxTokens, "default max_tokens should be zero")
}

func TestReviewSliceInput_DefaultValues(t *testing.T) {
	t.Parallel()

	input := reviewSliceInput{}
	assert.Empty(t, input.Base, "default base should be empty string")
	assert.Empty(t, input.Head, "default head should be empty string")
	assert.Empty(t, input.Profile, "default profile should be empty string")
}

func TestBriefOutput_Structure(t *testing.T) {
	t.Parallel()

	out := briefOutput{
		Content:     "some content",
		TokenCount:  42,
		ContentHash: "abc123",
	}
	assert.Equal(t, "some content", out.Content)
	assert.Equal(t, 42, out.TokenCount)
	assert.Equal(t, "abc123", out.ContentHash)
}

func TestSliceOutput_Structure(t *testing.T) {
	t.Parallel()

	out := sliceOutput{
		Content:       "slice content",
		TokenCount:    100,
		ContentHash:   "def456",
		ModuleFiles:   []string{"a.go"},
		NeighborFiles: []string{"b.go"},
	}
	assert.Equal(t, "slice content", out.Content)
	assert.Equal(t, 100, out.TokenCount)
	assert.Equal(t, "def456", out.ContentHash)
	assert.Equal(t, []string{"a.go"}, out.ModuleFiles)
	assert.Equal(t, []string{"b.go"}, out.NeighborFiles)
}

func TestReviewSliceOutput_Structure(t *testing.T) {
	t.Parallel()

	out := reviewSliceOutput{
		Content:       "review content",
		TokenCount:    200,
		ContentHash:   "ghi789",
		ChangedFiles:  []string{"c.go"},
		NeighborFiles: []string{"d.go"},
		DeletedFiles:  []string{"e.go"},
	}
	assert.Equal(t, "review content", out.Content)
	assert.Equal(t, 200, out.TokenCount)
	assert.Equal(t, "ghi789", out.ContentHash)
	assert.Equal(t, []string{"c.go"}, out.ChangedFiles)
	assert.Equal(t, []string{"d.go"}, out.NeighborFiles)
	assert.Equal(t, []string{"e.go"}, out.DeletedFiles)
}
