package output

import (
	"context"
	"fmt"
	"io"
)

// Compile-time interface compliance check.
var _ Renderer = (*MarkdownRenderer)(nil)

// MarkdownRenderer produces Markdown-formatted context documents using Go templates.
// It implements the Renderer interface.
type MarkdownRenderer struct{}

// NewMarkdownRenderer creates a new MarkdownRenderer.
func NewMarkdownRenderer() *MarkdownRenderer {
	return &MarkdownRenderer{}
}

// Render writes the complete Markdown context document to w. The template streams
// directly to the writer, avoiding full in-memory buffering.
func (r *MarkdownRenderer) Render(ctx context.Context, w io.Writer, data *RenderData) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if data == nil {
		return fmt.Errorf("render data is nil")
	}

	return markdownTemplate.ExecuteTemplate(w, "markdown-root", data)
}