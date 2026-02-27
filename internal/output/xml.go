package output

import (
	"context"
	"fmt"
	"io"
	"strings"
)

// Compile-time interface compliance check.
var _ Renderer = (*XMLRenderer)(nil)

// XMLRenderer produces XML-formatted context documents optimized for Claude.
// It uses semantic XML tags following Anthropic's best practices, with CDATA
// sections for file content to avoid escaping issues. It implements the
// Renderer interface.
type XMLRenderer struct{}

// NewXMLRenderer creates a new XMLRenderer.
func NewXMLRenderer() *XMLRenderer {
	return &XMLRenderer{}
}

// Render writes the complete XML context document to w. The template streams
// directly to the writer, avoiding full in-memory buffering.
func (r *XMLRenderer) Render(ctx context.Context, w io.Writer, data *RenderData) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if data == nil {
		return fmt.Errorf("render data is nil")
	}

	return xmlTemplate.ExecuteTemplate(w, "xml-root", data)
}

// wrapCDATA wraps content in a CDATA section, properly handling content that
// contains the CDATA end sequence "]]>". When "]]>" appears in the content, it
// is split across two CDATA sections: "...]]" + "]]><![CDATA[>" + "...".
func wrapCDATA(content string) string {
	if content == "" {
		return "<![CDATA[]]>"
	}
	// Split on "]]>" and rejoin with the CDATA split technique.
	escaped := strings.ReplaceAll(content, "]]>", "]]]]><![CDATA[>")
	return "<![CDATA[" + escaped + "]]>"
}

// xmlEscapeAttr escapes special XML characters in attribute values and element
// text content. The five predefined XML entities are replaced.
func xmlEscapeAttr(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}
