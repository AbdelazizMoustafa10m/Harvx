package workflows

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ParsedFile represents a single file extracted from a harvx output document.
type ParsedFile struct {
	// Path is the relative file path as recorded in the output document.
	Path string

	// Content is the raw file content extracted from the code block or CDATA section.
	Content string

	// IsCompressed indicates whether the file was marked as compressed in the output.
	IsCompressed bool

	// Redactions is the number of redaction placeholders detected in the content.
	Redactions int
}

// ParsedOutput represents the complete parsed output document.
type ParsedOutput struct {
	// Format is the detected output format: "markdown" or "xml".
	Format string

	// Files is the ordered list of files extracted from the output document.
	Files []ParsedFile
}

// ParseOutput detects the format of a harvx output document and parses it,
// extracting individual file blocks with their paths and content.
//
// Format detection uses the document prefix: content starting with "<?xml" is
// treated as XML; everything else is treated as Markdown.
func ParseOutput(content string) (*ParsedOutput, error) {
	trimmed := strings.TrimSpace(content)
	if strings.HasPrefix(trimmed, "<?xml") {
		return ParseXMLOutput(content)
	}
	return ParseMarkdownOutput(content)
}

// mdFileHeaderRe matches Markdown file headers of the form: ### `path/to/file`
var mdFileHeaderRe = regexp.MustCompile("(?m)^### `([^`]+)`")

// mdMetaCompressedRe matches the "Compressed: yes" portion of the metadata line.
var mdMetaCompressedRe = regexp.MustCompile(`\*\*Compressed:\*\*\s*(yes|no)`)

// mdFencedCodeRe matches the opening of a fenced code block (``` optionally followed by a language).
var mdFencedCodeRe = regexp.MustCompile("(?m)^```[a-zA-Z0-9]*\\s*$")

// mdFenceCloseRe matches the closing of a fenced code block (``` on its own line).
var mdFenceCloseRe = regexp.MustCompile("(?m)^```\\s*$")

// mdFilesSectionRe matches the "## Files" heading that delimits the start of file blocks.
var mdFilesSectionRe = regexp.MustCompile(`(?m)^## Files\s*$`)

// redactionPlaceholderRe counts redaction placeholders in content.
// Matches the standard redaction format: [REDACTED:type]
var redactionPlaceholderRe = regexp.MustCompile(`\[REDACTED:[^\]]+\]`)

// ParseMarkdownOutput extracts file blocks from a Markdown-formatted harvx output
// document. Each file block is identified by a level-3 heading with the path in
// backticks, followed by a metadata line and a fenced code block containing the
// file content.
func ParseMarkdownOutput(content string) (*ParsedOutput, error) {
	// Find the "## Files" section. Everything before it is header/summary.
	filesIdx := findFilesSectionIndex(content)
	if filesIdx < 0 {
		return &ParsedOutput{Format: "markdown"}, nil
	}

	filesSection := content[filesIdx:]

	// Find all file header positions within the files section.
	headerMatches := mdFileHeaderRe.FindAllStringSubmatchIndex(filesSection, -1)
	if len(headerMatches) == 0 {
		return &ParsedOutput{Format: "markdown"}, nil
	}

	files := make([]ParsedFile, 0, len(headerMatches))

	for i, match := range headerMatches {
		path := filesSection[match[2]:match[3]]

		// Determine the region for this file block: from after the header
		// to the start of the next file header (or end of content).
		blockStart := match[1]
		blockEnd := len(filesSection)
		if i+1 < len(headerMatches) {
			blockEnd = headerMatches[i+1][0]
		}

		block := filesSection[blockStart:blockEnd]

		// Parse metadata line for compressed status.
		isCompressed := false
		if m := mdMetaCompressedRe.FindStringSubmatch(block); len(m) > 1 {
			isCompressed = m[1] == "yes"
		}

		// Extract content from fenced code block.
		fileContent := extractFencedContent(block)

		// Unescape triple backticks that were escaped by the template.
		// The escapeTripleBackticks function replaces ``` with `` ` (two backticks, space, one backtick).
		fileContent = strings.ReplaceAll(fileContent, "`` `", "```")

		// Count redaction placeholders.
		redactions := len(redactionPlaceholderRe.FindAllString(fileContent, -1))

		files = append(files, ParsedFile{
			Path:         path,
			Content:      fileContent,
			IsCompressed: isCompressed,
			Redactions:   redactions,
		})
	}

	return &ParsedOutput{
		Format: "markdown",
		Files:  files,
	}, nil
}

// findFilesSectionIndex locates the "## Files" heading in a Markdown document
// and returns the byte offset of its start, or -1 if not found.
func findFilesSectionIndex(content string) int {
	loc := mdFilesSectionRe.FindStringIndex(content)
	if loc == nil {
		return -1
	}
	return loc[0]
}

// extractFencedContent finds the first fenced code block in the given text and
// returns its inner content. Returns an empty string if no fenced block is found.
func extractFencedContent(block string) string {
	lines := strings.Split(block, "\n")

	inFence := false
	var contentLines []string

	for _, line := range lines {
		if !inFence {
			// Look for opening fence: ``` optionally followed by a language identifier.
			if mdFencedCodeRe.MatchString(line) {
				inFence = true
				continue
			}
			continue
		}

		// Inside a fence: look for closing ```.
		if mdFenceCloseRe.MatchString(line) {
			break
		}
		contentLines = append(contentLines, line)
	}

	if len(contentLines) == 0 {
		return ""
	}

	return strings.Join(contentLines, "\n")
}

// xmlFileRe matches <file> elements with their path attribute in XML output.
var xmlFileRe = regexp.MustCompile(`<file\s+([^>]*)>`)

// xmlPathAttrRe extracts the path attribute value from a <file> element.
var xmlPathAttrRe = regexp.MustCompile(`path="([^"]*)"`)

// xmlCompressedAttrRe extracts the compressed attribute value from a <file> element.
var xmlCompressedAttrRe = regexp.MustCompile(`compressed="([^"]*)"`)

// ParseXMLOutput extracts file blocks from an XML-formatted harvx output document.
// Each file is identified by a <file path="..."> element containing a <content>
// element with CDATA-wrapped file content.
func ParseXMLOutput(content string) (*ParsedOutput, error) {
	// Find the <files> section.
	filesStart := strings.Index(content, "<files>")
	if filesStart < 0 {
		return &ParsedOutput{Format: "xml"}, nil
	}
	filesEnd := strings.Index(content, "</files>")
	if filesEnd < 0 {
		return nil, fmt.Errorf("parsing xml output: missing </files> closing tag")
	}

	filesSection := content[filesStart : filesEnd+len("</files>")]

	// Split the files section into individual <file>...</file> blocks.
	fileBlocks := splitXMLFileBlocks(filesSection)

	files := make([]ParsedFile, 0, len(fileBlocks))

	for _, block := range fileBlocks {
		// Extract path attribute.
		fileTagMatch := xmlFileRe.FindStringSubmatch(block)
		if len(fileTagMatch) < 2 {
			continue
		}
		attrs := fileTagMatch[1]

		pathMatch := xmlPathAttrRe.FindStringSubmatch(attrs)
		if len(pathMatch) < 2 {
			continue
		}
		path := unescapeXMLAttr(pathMatch[1])

		// Extract compressed attribute.
		isCompressed := false
		compMatch := xmlCompressedAttrRe.FindStringSubmatch(attrs)
		if len(compMatch) > 1 {
			isCompressed = compMatch[1] == "true"
		}

		// Extract content from <content>CDATA</content>.
		fileContent := extractXMLContent(block)

		// Count redaction placeholders.
		redactions := len(redactionPlaceholderRe.FindAllString(fileContent, -1))

		files = append(files, ParsedFile{
			Path:         path,
			Content:      fileContent,
			IsCompressed: isCompressed,
			Redactions:   redactions,
		})
	}

	return &ParsedOutput{
		Format: "xml",
		Files:  files,
	}, nil
}

// splitXMLFileBlocks splits the <files>...</files> section into individual
// <file>...</file> blocks. This avoids using a full XML parser while handling
// nested content correctly.
func splitXMLFileBlocks(filesSection string) []string {
	var blocks []string

	remaining := filesSection
	for {
		start := strings.Index(remaining, "<file ")
		if start < 0 {
			break
		}

		end := strings.Index(remaining[start:], "</file>")
		if end < 0 {
			break
		}

		block := remaining[start : start+end+len("</file>")]
		blocks = append(blocks, block)
		remaining = remaining[start+end+len("</file>"):]
	}

	return blocks
}

// extractXMLContent extracts file content from a <content>CDATA</content> element.
// It handles CDATA sections and the split technique used to escape "]]>" sequences
// within content (split as "]]]]><![CDATA[>").
func extractXMLContent(block string) string {
	contentStart := strings.Index(block, "<content>")
	if contentStart < 0 {
		return ""
	}
	contentEnd := strings.Index(block, "</content>")
	if contentEnd < 0 {
		return ""
	}

	inner := block[contentStart+len("<content>") : contentEnd]

	// Unwrap CDATA sections. The content may be wrapped in one or more
	// CDATA sections due to the split technique for escaping "]]>".
	return unwrapCDATA(inner)
}

// unwrapCDATA removes CDATA wrappers from XML content. It handles the split
// CDATA technique where "]]>" in content is encoded as "]]]]><![CDATA[>".
func unwrapCDATA(s string) string {
	// Strip all CDATA markers.
	s = strings.ReplaceAll(s, "<![CDATA[", "")
	s = strings.ReplaceAll(s, "]]>", "")

	return s
}

// unescapeXMLAttr reverses the five predefined XML entity escapes in attribute values.
func unescapeXMLAttr(s string) string {
	// Order matters: &amp; must be last to avoid double-unescaping.
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&apos;", "'")
	s = strings.ReplaceAll(s, "&amp;", "&")
	return s
}

// countRedactions counts the number of [REDACTED:type] placeholders in content.
// This is exported for use by other packages that need redaction counting.
func countRedactions(content string) int {
	return len(redactionPlaceholderRe.FindAllString(content, -1))
}

// parseIntAttr parses an integer from an XML attribute match, returning 0
// if the attribute is not found or cannot be parsed.
func parseIntAttr(attrs, attrName string) int {
	re := regexp.MustCompile(attrName + `="([^"]*)"`)
	m := re.FindStringSubmatch(attrs)
	if len(m) < 2 {
		return 0
	}
	v, err := strconv.Atoi(m[1])
	if err != nil {
		return 0
	}
	return v
}
