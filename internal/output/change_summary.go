package output

import (
	"fmt"
	"html"
	"strings"

	"github.com/harvx/harvx/internal/diff"
)

// NewDiffSummaryData converts a diff.DiffResult into a DiffSummaryData suitable
// for use by the output renderers. It copies all fields from the DiffResult,
// including the unchanged count. Returns nil if result is nil.
func NewDiffSummaryData(result *diff.DiffResult) *DiffSummaryData {
	if result == nil {
		return nil
	}

	// DC-1: Copy slices at package boundaries.
	added := make([]string, len(result.Added))
	copy(added, result.Added)

	modified := make([]string, len(result.Modified))
	copy(modified, result.Modified)

	deleted := make([]string, len(result.Deleted))
	copy(deleted, result.Deleted)

	return &DiffSummaryData{
		AddedFiles:    added,
		ModifiedFiles: modified,
		DeletedFiles:  deleted,
		Unchanged:     result.Unchanged,
	}
}

// RenderChangeSummary renders a formatted change summary section from diff data.
// The format parameter selects between FormatMarkdown and FormatXML output.
// Returns an empty string when diff is nil.
func RenderChangeSummary(diff *DiffSummaryData, format string) string {
	if diff == nil {
		return ""
	}

	switch strings.ToLower(format) {
	case FormatXML:
		return renderChangeSummaryXML(diff)
	default:
		return renderChangeSummaryMarkdown(diff)
	}
}

// renderChangeSummaryMarkdown renders the change summary in Markdown format.
func renderChangeSummaryMarkdown(d *DiffSummaryData) string {
	hasChanges := len(d.AddedFiles) > 0 || len(d.ModifiedFiles) > 0 || len(d.DeletedFiles) > 0

	var sb strings.Builder
	sb.WriteString("## Changes Since Last Run\n\n")

	if !hasChanges {
		sb.WriteString("No changes detected since last run.\n")
		return sb.String()
	}

	// Header with counts.
	sb.WriteString(fmt.Sprintf("**%d added** | **%d modified** | **%d deleted** (%d unchanged)\n",
		len(d.AddedFiles), len(d.ModifiedFiles), len(d.DeletedFiles), d.Unchanged))

	// Added files subsection.
	if len(d.AddedFiles) > 0 {
		sb.WriteString("\n### Added Files\n")
		for _, f := range d.AddedFiles {
			sb.WriteString(fmt.Sprintf("- `%s`\n", f))
		}
	}

	// Modified files subsection.
	if len(d.ModifiedFiles) > 0 {
		sb.WriteString("\n### Modified Files\n")
		for _, f := range d.ModifiedFiles {
			sb.WriteString(fmt.Sprintf("- `%s`\n", f))
		}
	}

	// Deleted files subsection.
	if len(d.DeletedFiles) > 0 {
		sb.WriteString("\n### Deleted Files\n")
		for _, f := range d.DeletedFiles {
			sb.WriteString(fmt.Sprintf("- `%s`\n", f))
		}
	}

	return sb.String()
}

// renderChangeSummaryXML renders the change summary in XML format.
func renderChangeSummaryXML(d *DiffSummaryData) string {
	hasChanges := len(d.AddedFiles) > 0 || len(d.ModifiedFiles) > 0 || len(d.DeletedFiles) > 0

	var sb strings.Builder
	sb.WriteString("<change_summary>\n")

	sb.WriteString(fmt.Sprintf("  <counts added=\"%d\" modified=\"%d\" deleted=\"%d\" unchanged=\"%d\"/>\n",
		len(d.AddedFiles), len(d.ModifiedFiles), len(d.DeletedFiles), d.Unchanged))

	if hasChanges {
		if len(d.AddedFiles) > 0 {
			sb.WriteString("  <added_files>\n")
			for _, f := range d.AddedFiles {
				sb.WriteString(fmt.Sprintf("    <file path=\"%s\"/>\n", html.EscapeString(f)))
			}
			sb.WriteString("  </added_files>\n")
		}

		if len(d.ModifiedFiles) > 0 {
			sb.WriteString("  <modified_files>\n")
			for _, f := range d.ModifiedFiles {
				sb.WriteString(fmt.Sprintf("    <file path=\"%s\"/>\n", html.EscapeString(f)))
			}
			sb.WriteString("  </modified_files>\n")
		}

		if len(d.DeletedFiles) > 0 {
			sb.WriteString("  <deleted_files>\n")
			for _, f := range d.DeletedFiles {
				sb.WriteString(fmt.Sprintf("    <file path=\"%s\"/>\n", html.EscapeString(f)))
			}
			sb.WriteString("  </deleted_files>\n")
		}
	}

	sb.WriteString("</change_summary>\n")
	return sb.String()
}
