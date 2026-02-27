package output

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRenderer_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		format  string
		wantMD  bool
		wantXML bool
	}{
		{
			name:   "markdown format returns MarkdownRenderer",
			format: FormatMarkdown,
			wantMD: true,
		},
		{
			name:    "xml format returns XMLRenderer",
			format:  FormatXML,
			wantXML: true,
		},
		{
			name:   "uppercase MARKDOWN is accepted",
			format: "MARKDOWN",
			wantMD: true,
		},
		{
			name:    "mixed case Xml is accepted",
			format:  "Xml",
			wantXML: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r, err := NewRenderer(tt.format)
			require.NoError(t, err)
			require.NotNil(t, r)
			assert.Implements(t, (*Renderer)(nil), r)

			if tt.wantMD {
				_, ok := r.(*MarkdownRenderer)
				assert.True(t, ok, "expected *MarkdownRenderer, got %T", r)
			}
			if tt.wantXML {
				_, ok := r.(*XMLRenderer)
				assert.True(t, ok, "expected *XMLRenderer, got %T", r)
			}
		})
	}
}

func TestNewRenderer_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		format string
	}{
		{name: "unknown format", format: "html"},
		{name: "empty format", format: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r, err := NewRenderer(tt.format)
			require.Error(t, err)
			assert.Nil(t, r)
			assert.Contains(t, err.Error(), "unknown output format")
		})
	}
}

func TestExtensionForFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		format string
		want   string
	}{
		{name: "markdown returns .md", format: FormatMarkdown, want: ExtensionMarkdown},
		{name: "xml returns .xml", format: FormatXML, want: ExtensionXML},
		{name: "unknown defaults to .md", format: "html", want: ExtensionMarkdown},
		{name: "empty defaults to .md", format: "", want: ExtensionMarkdown},
		{name: "uppercase XML returns .xml", format: "XML", want: ExtensionXML},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ExtensionForFormat(tt.format)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDefaultOutputPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		format string
		want   string
	}{
		{name: "markdown default path", format: FormatMarkdown, want: "harvx-output.md"},
		{name: "xml default path", format: FormatXML, want: "harvx-output.xml"},
		{name: "unknown format defaults to .md", format: "other", want: "harvx-output.md"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := DefaultOutputPath(tt.format)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestResolveOutputPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		outputFlag    string
		profileOutput string
		format        string
		want          string
	}{
		{
			name:          "CLI flag takes highest precedence",
			outputFlag:    "custom-output.md",
			profileOutput: "profile-output.md",
			format:        FormatMarkdown,
			want:          "custom-output.md",
		},
		{
			name:          "profile output used when no CLI flag",
			outputFlag:    "",
			profileOutput: "profile-output.xml",
			format:        FormatXML,
			want:          "profile-output.xml",
		},
		{
			name:          "default path when both empty",
			outputFlag:    "",
			profileOutput: "",
			format:        FormatMarkdown,
			want:          "harvx-output.md",
		},
		{
			name:          "default XML path when both empty",
			outputFlag:    "",
			profileOutput: "",
			format:        FormatXML,
			want:          "harvx-output.xml",
		},
		{
			name:          "extension appended to CLI flag without extension",
			outputFlag:    "output",
			profileOutput: "",
			format:        FormatMarkdown,
			want:          "output.md",
		},
		{
			name:          "xml extension appended to CLI flag without extension",
			outputFlag:    "output",
			profileOutput: "",
			format:        FormatXML,
			want:          "output.xml",
		},
		{
			name:          "extension appended to profile path without extension",
			outputFlag:    "",
			profileOutput: "my-output",
			format:        FormatXML,
			want:          "my-output.xml",
		},
		{
			name:          "existing extension preserved on CLI flag",
			outputFlag:    "report.txt",
			profileOutput: "",
			format:        FormatMarkdown,
			want:          "report.txt",
		},
		{
			name:          "existing extension preserved on profile output",
			outputFlag:    "",
			profileOutput: "report.txt",
			format:        FormatXML,
			want:          "report.txt",
		},
		{
			name:          "path with directory and no extension",
			outputFlag:    "build/output",
			profileOutput: "",
			format:        FormatMarkdown,
			want:          "build/output.md",
		},
		{
			name:          "CLI flag overrides profile even when profile has extension",
			outputFlag:    "mine",
			profileOutput: "profile.xml",
			format:        FormatXML,
			want:          "mine.xml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ResolveOutputPath(tt.outputFlag, tt.profileOutput, tt.format)
			assert.Equal(t, tt.want, got)
		})
	}
}
