package compression

import (
	"context"
	"fmt"
	"strings"
)

// Compile-time interface compliance check.
var _ LanguageCompressor = (*YAMLCompressor)(nil)

// YAMLCompressor implements LanguageCompressor for YAML files using
// line-based parsing. It does not depend on any external YAML library.
// The compressor extracts structural keys and comments up to a configurable
// depth, collapsing deeper content and long lists for token efficiency.
// It is stateless and safe for concurrent use.
type YAMLCompressor struct {
	maxDepth    int // maximum nesting depth to preserve (0-based)
	maxListItems int // maximum consecutive list items before collapsing
	maxStringLen int // maximum length for inline string values
}

// NewYAMLCompressor creates a YAML compressor with sensible defaults:
// maxDepth=2, maxListItems=5, maxStringLen=80.
func NewYAMLCompressor() *YAMLCompressor {
	return &YAMLCompressor{
		maxDepth:     2,
		maxListItems: 5,
		maxStringLen: 80,
	}
}

// Language returns "yaml".
func (c *YAMLCompressor) Language() string {
	return "yaml"
}

// SupportedNodeTypes returns the YAML node types this compressor handles.
func (c *YAMLCompressor) SupportedNodeTypes() []string {
	return []string{"mapping", "sequence", "comment"}
}

// Compress parses YAML source line-by-line and produces a compressed
// structural skeleton. Comments and keys at shallow depths are preserved;
// deep nesting and long lists are collapsed with summary annotations.
// The output contains verbatim source lines; it never summarizes or rewrites
// YAML content beyond truncation and collapsing.
func (c *YAMLCompressor) Compress(ctx context.Context, source []byte) (*CompressedOutput, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if len(source) == 0 {
		return &CompressedOutput{
			Language:     "yaml",
			OriginalSize: 0,
		}, nil
	}

	lines := strings.Split(string(source), "\n")
	p := &yamlParser{
		maxDepth:     c.maxDepth,
		maxListItems: c.maxListItems,
		maxStringLen: c.maxStringLen,
	}

	compressed := p.parse(ctx, lines)

	result := strings.Join(compressed, "\n")
	output := &CompressedOutput{
		Signatures: []Signature{{
			Kind:      KindDocComment,
			Name:      "yaml-structure",
			Source:    result,
			StartLine: 1,
			EndLine:   len(lines),
		}},
		Language:     "yaml",
		OriginalSize: len(source),
		OutputSize:   len(result),
		NodeCount:    1,
	}

	return output, nil
}

// ---------------------------------------------------------------------------
// YAML line-based parser
// ---------------------------------------------------------------------------

// yamlParser holds configuration and mutable state for line-based YAML parsing.
type yamlParser struct {
	maxDepth     int
	maxListItems int
	maxStringLen int
}

// parse processes YAML lines and returns compressed output lines.
func (p *yamlParser) parse(ctx context.Context, lines []string) []string {
	var out []string
	prevBlank := false

	// Multi-line string tracking.
	inMultiLine := false
	multiLineIndent := 0
	multiLineCount := 0
	multiLineMaxLines := 3

	// List collapsing tracking.
	listDepth := -1       // indent level of current list being tracked
	listCount := 0        // number of consecutive list items at listDepth
	listEmitted := 0      // number of list items actually emitted
	listSkipping := false // true when we've exceeded maxListItems

	for i, line := range lines {
		// Check context cancellation every 1000 lines.
		if i%1000 == 0 {
			select {
			case <-ctx.Done():
				return out
			default:
			}
		}

		trimmed := strings.TrimSpace(line)

		// Handle multi-line string continuation.
		if inMultiLine {
			lineIndent := yamlIndentLevel(line)
			// Multi-line scalar continues while indent is deeper than the key
			// or the line is blank (blank lines are valid in block scalars).
			if trimmed == "" || lineIndent > multiLineIndent {
				multiLineCount++
				if multiLineCount <= multiLineMaxLines {
					out = append(out, line)
				} else if multiLineCount == multiLineMaxLines+1 {
					// Emit truncation marker at the same indentation.
					indent := strings.Repeat(" ", multiLineIndent+2)
					out = append(out, indent+"# ... (truncated)")
				}
				// Skip additional lines silently.
				continue
			}
			// No longer in multi-line mode; fall through to process this line normally.
			inMultiLine = false
		}

		// Blank line handling: collapse consecutive blanks to single.
		if trimmed == "" {
			if !prevBlank {
				out = append(out, "")
				prevBlank = true
			}
			// Reset list tracking on blank lines.
			if listSkipping {
				p.emitListSummary(&out, listDepth, listCount, listEmitted)
				listSkipping = false
			}
			listDepth = -1
			listCount = 0
			listEmitted = 0
			continue
		}
		prevBlank = false

		indent := yamlIndentLevel(line)
		depth := indent / 2

		// Comment lines: always preserve at depth <= maxDepth.
		if yamlIsComment(trimmed) {
			if depth <= p.maxDepth {
				// If we were skipping list items, flush the summary before the comment.
				if listSkipping {
					p.emitListSummary(&out, listDepth, listCount, listEmitted)
					listSkipping = false
					listDepth = -1
					listCount = 0
					listEmitted = 0
				}
				out = append(out, line)
			}
			continue
		}

		// Document separator lines (--- or ...).
		if trimmed == "---" || trimmed == "..." {
			// Flush any pending list summary.
			if listSkipping {
				p.emitListSummary(&out, listDepth, listCount, listEmitted)
				listSkipping = false
				listDepth = -1
				listCount = 0
				listEmitted = 0
			}
			out = append(out, line)
			continue
		}

		// Check if this is a list item.
		isList := yamlIsListItem(trimmed)

		// If we were tracking a list and this line is not a list item at the
		// same or deeper depth, or is a list item at a different depth, flush.
		if listDepth >= 0 && (!isList || indent != listDepth) {
			if listSkipping {
				p.emitListSummary(&out, listDepth, listCount, listEmitted)
				listSkipping = false
			}
			listDepth = -1
			listCount = 0
			listEmitted = 0
		}

		// Lines at depth > maxDepth: skip.
		if depth > p.maxDepth {
			continue
		}

		// Handle list items.
		if isList {
			if listDepth < 0 {
				// Start tracking a new list.
				listDepth = indent
				listCount = 0
				listEmitted = 0
				listSkipping = false
			}

			listCount++

			if listCount <= p.maxListItems {
				out = append(out, p.maybeTruncateLine(line, trimmed, indent))
				listEmitted++
			} else if !listSkipping {
				listSkipping = true
				// Don't emit summary yet -- wait for end of list.
			}
			continue
		}

		// Regular key line at allowed depth.
		emittedLine := p.processKeyLine(line, trimmed, indent)
		out = append(out, emittedLine)

		// Check for multi-line string indicator.
		if yamlHasBlockScalarIndicator(trimmed) {
			inMultiLine = true
			multiLineIndent = indent
			multiLineCount = 0
		}
	}

	// Flush any pending list summary at EOF.
	if listSkipping {
		p.emitListSummary(&out, listDepth, listCount, listEmitted)
	}

	// Remove trailing blank lines.
	for len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == "" {
		out = out[:len(out)-1]
	}

	return out
}

// emitListSummary appends a "# ... (N more items)" comment at the appropriate indent.
func (p *yamlParser) emitListSummary(out *[]string, indent, totalCount, emittedCount int) {
	remaining := totalCount - emittedCount
	if remaining <= 0 {
		return
	}
	prefix := strings.Repeat(" ", indent)
	*out = append(*out, fmt.Sprintf("%s# ... (%d more items)", prefix, remaining))
}

// processKeyLine handles a key: value line, potentially truncating long string values.
func (p *yamlParser) processKeyLine(line, trimmed string, indent int) string {
	return p.maybeTruncateLine(line, trimmed, indent)
}

// maybeTruncateLine truncates long inline string values on key: value lines.
func (p *yamlParser) maybeTruncateLine(line, trimmed string, indent int) string {
	// Find the colon that separates key from value.
	colonIdx := strings.Index(trimmed, ":")
	if colonIdx < 0 {
		return line
	}

	// For list items, strip the "- " prefix before looking for key: value.
	effective := trimmed
	if yamlIsListItem(trimmed) {
		effective = strings.TrimPrefix(trimmed, "- ")
		effective = strings.TrimSpace(effective)
		colonIdx = strings.Index(effective, ":")
		if colonIdx < 0 {
			// Simple list item without key (e.g., "- value"), truncate the value.
			listVal := strings.TrimPrefix(trimmed, "- ")
			listVal = strings.TrimSpace(listVal)
			if len(listVal) > p.maxStringLen {
				prefix := strings.Repeat(" ", indent)
				return fmt.Sprintf("%s- %s...", prefix, listVal[:p.maxStringLen])
			}
			return line
		}
	}

	afterColon := ""
	if colonIdx+1 < len(effective) {
		afterColon = strings.TrimSpace(effective[colonIdx+1:])
	}

	// Empty value or block scalar indicator -- keep as is.
	if afterColon == "" || afterColon == "|" || afterColon == ">" ||
		afterColon == "|-" || afterColon == ">-" ||
		afterColon == "|+" || afterColon == ">+" {
		return line
	}

	// Value that is a nested mapping or anchor -- keep as is.
	if strings.HasPrefix(afterColon, "{") || strings.HasPrefix(afterColon, "[") ||
		strings.HasPrefix(afterColon, "&") || strings.HasPrefix(afterColon, "*") {
		return line
	}

	// Truncate long string values.
	if len(afterColon) > p.maxStringLen {
		// Reconstruct the line with truncated value.
		// Find the position of the value in the original line.
		origColonIdx := strings.Index(line, ":")
		if origColonIdx >= 0 {
			keyPart := line[:origColonIdx+1]
			// Preserve space after colon.
			rest := line[origColonIdx+1:]
			spacePrefix := ""
			for _, ch := range rest {
				if ch == ' ' {
					spacePrefix += " "
				} else {
					break
				}
			}
			return keyPart + spacePrefix + afterColon[:p.maxStringLen] + "..."
		}
	}

	return line
}

// ---------------------------------------------------------------------------
// YAML helper functions
// ---------------------------------------------------------------------------

// yamlIndentLevel returns the number of leading spaces in a line.
// Tabs are treated as a single space for depth calculation.
func yamlIndentLevel(line string) int {
	count := 0
	for _, ch := range line {
		if ch == ' ' {
			count++
		} else if ch == '\t' {
			count++
		} else {
			break
		}
	}
	return count
}

// yamlIsComment returns true if the trimmed line is a YAML comment.
func yamlIsComment(trimmed string) bool {
	return strings.HasPrefix(trimmed, "#")
}

// yamlIsListItem returns true if the trimmed line starts with "- ".
func yamlIsListItem(trimmed string) bool {
	return strings.HasPrefix(trimmed, "- ") || trimmed == "-"
}

// yamlHasBlockScalarIndicator returns true if the line's value ends with
// a block scalar indicator (| or > with optional chomping modifier).
func yamlHasBlockScalarIndicator(trimmed string) bool {
	colonIdx := strings.Index(trimmed, ":")
	if colonIdx < 0 {
		return false
	}
	afterColon := strings.TrimSpace(trimmed[colonIdx+1:])

	// Strip inline comment.
	if commentIdx := strings.Index(afterColon, " #"); commentIdx >= 0 {
		afterColon = strings.TrimSpace(afterColon[:commentIdx])
	}

	return afterColon == "|" || afterColon == ">" ||
		afterColon == "|-" || afterColon == ">-" ||
		afterColon == "|+" || afterColon == ">+"
}
