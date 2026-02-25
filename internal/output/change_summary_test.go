package output

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/harvx/harvx/internal/diff"
)

// ---------------------------------------------------------------------------
// TestNewDiffSummaryData
// ---------------------------------------------------------------------------

func TestNewDiffSummaryData(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		result    *diff.DiffResult
		wantNil   bool
		wantAdded int
		wantMod   int
		wantDel   int
		wantUnch  int
	}{
		{
			name:    "nil result returns nil",
			result:  nil,
			wantNil: true,
		},
		{
			name: "copies all fields",
			result: &diff.DiffResult{
				Added:     []string{"a.go", "b.go"},
				Modified:  []string{"c.go"},
				Deleted:   []string{"d.go", "e.go", "f.go"},
				Unchanged: 42,
			},
			wantAdded: 2,
			wantMod:   1,
			wantDel:   3,
			wantUnch:  42,
		},
		{
			name: "empty slices",
			result: &diff.DiffResult{
				Added:     nil,
				Modified:  nil,
				Deleted:   nil,
				Unchanged: 0,
			},
			wantAdded: 0,
			wantMod:   0,
			wantDel:   0,
			wantUnch:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := NewDiffSummaryData(tt.result)

			if tt.wantNil {
				assert.Nil(t, got)
				return
			}

			require.NotNil(t, got)
			assert.Len(t, got.AddedFiles, tt.wantAdded)
			assert.Len(t, got.ModifiedFiles, tt.wantMod)
			assert.Len(t, got.DeletedFiles, tt.wantDel)
			assert.Equal(t, tt.wantUnch, got.Unchanged)
		})
	}
}

func TestNewDiffSummaryData_DefensiveCopy(t *testing.T) {
	t.Parallel()

	original := &diff.DiffResult{
		Added:     []string{"a.go", "b.go"},
		Modified:  []string{"c.go"},
		Deleted:   []string{"d.go"},
		Unchanged: 10,
	}

	got := NewDiffSummaryData(original)
	require.NotNil(t, got)

	// Mutate the original slices; the copy should be unaffected.
	original.Added[0] = "MUTATED"
	original.Modified[0] = "MUTATED"
	original.Deleted[0] = "MUTATED"

	assert.Equal(t, "a.go", got.AddedFiles[0], "added files should be a defensive copy")
	assert.Equal(t, "c.go", got.ModifiedFiles[0], "modified files should be a defensive copy")
	assert.Equal(t, "d.go", got.DeletedFiles[0], "deleted files should be a defensive copy")
}

// ---------------------------------------------------------------------------
// TestRenderChangeSummary_Nil
// ---------------------------------------------------------------------------

func TestRenderChangeSummary_Nil(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "", RenderChangeSummary(nil, FormatMarkdown))
	assert.Equal(t, "", RenderChangeSummary(nil, FormatXML))
}

// ---------------------------------------------------------------------------
// TestRenderChangeSummary_Markdown
// ---------------------------------------------------------------------------

func TestRenderChangeSummary_Markdown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		diff         *DiffSummaryData
		wantContains []string
		wantMissing  []string
	}{
		{
			name: "all change types",
			diff: &DiffSummaryData{
				AddedFiles:    []string{"src/new-feature.go", "src/helper.go", "tests/new_test.go"},
				ModifiedFiles: []string{"README.md", "go.mod", "internal/handler.go", "src/config.go", "src/main.go"},
				DeletedFiles:  []string{"src/deprecated.go"},
				Unchanged:     42,
			},
			wantContains: []string{
				"## Changes Since Last Run",
				"**3 added** | **5 modified** | **1 deleted** (42 unchanged)",
				"### Added Files",
				"- `src/new-feature.go`",
				"- `src/helper.go`",
				"- `tests/new_test.go`",
				"### Modified Files",
				"- `README.md`",
				"- `go.mod`",
				"- `internal/handler.go`",
				"- `src/config.go`",
				"- `src/main.go`",
				"### Deleted Files",
				"- `src/deprecated.go`",
			},
		},
		{
			name: "only additions",
			diff: &DiffSummaryData{
				AddedFiles:    []string{"new.go"},
				ModifiedFiles: nil,
				DeletedFiles:  nil,
				Unchanged:     5,
			},
			wantContains: []string{
				"## Changes Since Last Run",
				"**1 added** | **0 modified** | **0 deleted** (5 unchanged)",
				"### Added Files",
				"- `new.go`",
			},
			wantMissing: []string{
				"### Modified Files",
				"### Deleted Files",
			},
		},
		{
			name: "only modifications",
			diff: &DiffSummaryData{
				AddedFiles:    nil,
				ModifiedFiles: []string{"changed.go"},
				DeletedFiles:  nil,
				Unchanged:     10,
			},
			wantContains: []string{
				"### Modified Files",
				"- `changed.go`",
			},
			wantMissing: []string{
				"### Added Files",
				"### Deleted Files",
			},
		},
		{
			name: "only deletions",
			diff: &DiffSummaryData{
				AddedFiles:    nil,
				ModifiedFiles: nil,
				DeletedFiles:  []string{"removed.go"},
				Unchanged:     3,
			},
			wantContains: []string{
				"### Deleted Files",
				"- `removed.go`",
			},
			wantMissing: []string{
				"### Added Files",
				"### Modified Files",
			},
		},
		{
			name: "no changes",
			diff: &DiffSummaryData{
				AddedFiles:    nil,
				ModifiedFiles: nil,
				DeletedFiles:  nil,
				Unchanged:     0,
			},
			wantContains: []string{
				"## Changes Since Last Run",
				"No changes detected since last run.",
			},
			wantMissing: []string{
				"### Added Files",
				"### Modified Files",
				"### Deleted Files",
			},
		},
		{
			name: "no changes with empty slices",
			diff: &DiffSummaryData{
				AddedFiles:    []string{},
				ModifiedFiles: []string{},
				DeletedFiles:  []string{},
				Unchanged:     100,
			},
			wantContains: []string{
				"No changes detected since last run.",
			},
			wantMissing: []string{
				"### Added Files",
				"### Modified Files",
				"### Deleted Files",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := RenderChangeSummary(tt.diff, FormatMarkdown)

			for _, want := range tt.wantContains {
				assert.Contains(t, got, want, "output should contain: %s", want)
			}
			for _, missing := range tt.wantMissing {
				assert.NotContains(t, got, missing, "output should not contain: %s", missing)
			}
		})
	}
}

func TestRenderChangeSummary_Markdown_CountsMatchLists(t *testing.T) {
	t.Parallel()

	d := &DiffSummaryData{
		AddedFiles:    []string{"a.go", "b.go"},
		ModifiedFiles: []string{"c.go", "d.go", "e.go"},
		DeletedFiles:  []string{"f.go"},
		Unchanged:     7,
	}

	got := RenderChangeSummary(d, FormatMarkdown)

	assert.Contains(t, got, "**2 added**")
	assert.Contains(t, got, "**3 modified**")
	assert.Contains(t, got, "**1 deleted**")
	assert.Contains(t, got, "(7 unchanged)")

	// Verify actual file list lengths match the header counts.
	lines := strings.Split(got, "\n")
	fileEntries := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "- `") {
			fileEntries++
		}
	}
	assert.Equal(t, 6, fileEntries, "total file entries should match sum of added+modified+deleted")
}

// ---------------------------------------------------------------------------
// TestRenderChangeSummary_XML
// ---------------------------------------------------------------------------

func TestRenderChangeSummary_XML(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		diff         *DiffSummaryData
		wantContains []string
		wantMissing  []string
	}{
		{
			name: "all change types",
			diff: &DiffSummaryData{
				AddedFiles:    []string{"src/new-feature.go", "src/helper.go"},
				ModifiedFiles: []string{"src/main.go"},
				DeletedFiles:  []string{"src/deprecated.go"},
				Unchanged:     42,
			},
			wantContains: []string{
				"<change_summary>",
				`<counts added="2" modified="1" deleted="1" unchanged="42"/>`,
				"<added_files>",
				`<file path="src/new-feature.go"/>`,
				`<file path="src/helper.go"/>`,
				"</added_files>",
				"<modified_files>",
				`<file path="src/main.go"/>`,
				"</modified_files>",
				"<deleted_files>",
				`<file path="src/deprecated.go"/>`,
				"</deleted_files>",
				"</change_summary>",
			},
		},
		{
			name: "no changes",
			diff: &DiffSummaryData{
				AddedFiles:    nil,
				ModifiedFiles: nil,
				DeletedFiles:  nil,
				Unchanged:     0,
			},
			wantContains: []string{
				"<change_summary>",
				`<counts added="0" modified="0" deleted="0" unchanged="0"/>`,
				"</change_summary>",
			},
			wantMissing: []string{
				"<added_files>",
				"<modified_files>",
				"<deleted_files>",
			},
		},
		{
			name: "only additions omits other sections",
			diff: &DiffSummaryData{
				AddedFiles:    []string{"new.go"},
				ModifiedFiles: nil,
				DeletedFiles:  nil,
				Unchanged:     5,
			},
			wantContains: []string{
				"<added_files>",
				`<file path="new.go"/>`,
				"</added_files>",
			},
			wantMissing: []string{
				"<modified_files>",
				"<deleted_files>",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := RenderChangeSummary(tt.diff, FormatXML)

			for _, want := range tt.wantContains {
				assert.Contains(t, got, want, "output should contain: %s", want)
			}
			for _, missing := range tt.wantMissing {
				assert.NotContains(t, got, missing, "output should not contain: %s", missing)
			}
		})
	}
}

func TestRenderChangeSummary_XML_EscapesSpecialChars(t *testing.T) {
	t.Parallel()

	d := &DiffSummaryData{
		AddedFiles:    []string{`path/with"quotes.go`, "path/with<angle>.go"},
		ModifiedFiles: []string{"path/with&ampersand.go"},
		DeletedFiles:  nil,
		Unchanged:     0,
	}

	got := RenderChangeSummary(d, FormatXML)

	assert.Contains(t, got, `path/with&#34;quotes.go`, "double quotes should be escaped")
	assert.Contains(t, got, `path/with&lt;angle&gt;.go`, "angle brackets should be escaped")
	assert.Contains(t, got, `path/with&amp;ampersand.go`, "ampersands should be escaped")
}

func TestRenderChangeSummary_XML_EmptySlices(t *testing.T) {
	t.Parallel()

	d := &DiffSummaryData{
		AddedFiles:    []string{},
		ModifiedFiles: []string{},
		DeletedFiles:  []string{},
		Unchanged:     50,
	}

	got := RenderChangeSummary(d, FormatXML)

	assert.Contains(t, got, `<counts added="0" modified="0" deleted="0" unchanged="50"/>`)
	assert.NotContains(t, got, "<added_files>")
	assert.NotContains(t, got, "<modified_files>")
	assert.NotContains(t, got, "<deleted_files>")
}

// ---------------------------------------------------------------------------
// TestRenderChangeSummary_UnknownFormat
// ---------------------------------------------------------------------------

func TestRenderChangeSummary_UnknownFormat(t *testing.T) {
	t.Parallel()

	d := &DiffSummaryData{
		AddedFiles: []string{"file.go"},
		Unchanged:  1,
	}

	// Unknown format defaults to markdown.
	got := RenderChangeSummary(d, "unknown")
	assert.Contains(t, got, "## Changes Since Last Run",
		"unknown format should fall back to markdown")
}

func TestRenderChangeSummary_CaseInsensitiveFormat(t *testing.T) {
	t.Parallel()

	d := &DiffSummaryData{
		AddedFiles: []string{"file.go"},
		Unchanged:  1,
	}

	xmlOutput := RenderChangeSummary(d, "XML")
	assert.Contains(t, xmlOutput, "<change_summary>",
		"format should be case-insensitive")

	mdOutput := RenderChangeSummary(d, "MARKDOWN")
	assert.Contains(t, mdOutput, "## Changes Since Last Run",
		"format should be case-insensitive")
}
