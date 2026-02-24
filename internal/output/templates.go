package output

import (
	"sort"
	"text/template"
)

// markdownFuncMap provides helper functions available within the Markdown template.
var markdownFuncMap = template.FuncMap{
	"formatBytes":           formatBytes,
	"formatNumber":          formatNumber,
	"languageFromExt":       languageFromExt,
	"addLineNumbers":        addLineNumbers,
	"repeatString":          repeatString,
	"tierLabel":             tierLabel,
	"escapeTripleBackticks": escapeTripleBackticks,
	"sortedKeys": func(m map[string]int) []string {
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		return keys
	},
	"tierNumbers": func() []int {
		return []int{0, 1, 2, 3, 4, 5}
	},
	"tierCount": func(counts map[int]int, tier int) int {
		return counts[tier]
	},
	"fileLang": func(f FileRenderEntry) string {
		if f.Language != "" {
			return f.Language
		}
		return languageFromExt(f.Path)
	},
}

// markdownTemplate is the compiled Markdown output template, parsed once at
// package level. It is goroutine-safe for concurrent Execute calls.
var markdownTemplate = template.Must(
	template.New("markdown").Funcs(markdownFuncMap).Parse(markdownTmpl),
)

// markdownTmpl is the complete Markdown template string composed of named
// sub-templates for header, summary, tree, files, and change summary sections.
//
// Whitespace strategy: each sub-template produces exactly its own section
// without leading or trailing blank lines. The root template controls spacing
// between sections using explicit newlines.
const markdownTmpl = headerTmpl + summaryTmpl + treeTmpl + filesTmpl + changeSummaryTmpl + rootTmpl

// rootTmpl is the top-level composition template that invokes sub-templates
// with controlled spacing between sections.
const rootTmpl = `{{- define "markdown-root" -}}
{{- template "header" . }}

{{ template "summary" . }}

{{ template "tree" . }}

{{ template "files" . }}
{{- template "changeSummary" . -}}
{{- end -}}`

// headerTmpl renders the project title and metadata table.
const headerTmpl = `{{- define "header" -}}
# Harvx Context: {{.ProjectName}}

| Field | Value |
|-------|-------|
| Generated | {{.Timestamp.Format "2006-01-02T15:04:05Z07:00"}} |
| Content Hash | {{.ContentHash}} |
| Profile | {{.ProfileName}} |
| Tokenizer | {{.TokenizerName}} |
| Total Tokens | {{formatNumber .TotalTokens}} |
| Total Files | {{.TotalFiles}} |
{{- end -}}`

// summaryTmpl renders file counts, tier breakdown, top files, and redaction summary.
const summaryTmpl = `{{- define "summary" -}}
## File Summary

**Total Files:** {{.TotalFiles}} | **Total Tokens:** {{formatNumber .TotalTokens}}

### Files by Tier

| Tier | Label | Count |
|------|-------|-------|
{{- range $tier := tierNumbers}}
{{- $count := tierCount $.TierCounts $tier}}
{{- if gt $count 0}}
| {{$tier}} | {{tierLabel $tier}} | {{$count}} |
{{- end}}
{{- end}}
{{- if gt (len .TopFilesByTokens) 0}}

### Top Files by Token Count

| File | Tokens | Size |
|------|--------|------|
{{- range .TopFilesByTokens}}
| {{.Path}} | {{formatNumber .TokenCount}} | {{formatBytes .Size}} |
{{- end}}
{{- end}}
{{- if gt .TotalRedactions 0}}

### Redaction Summary

| Type | Count |
|------|-------|
{{- range $key := sortedKeys .RedactionSummary}}
| {{$key}} | {{index $.RedactionSummary $key}} |
{{- end}}
{{- end}}
{{- end -}}`

// treeTmpl renders the directory tree section.
const treeTmpl = `{{- define "tree" -}}
## Directory Tree

` + "```" + `
{{.TreeString}}
` + "```" + `
{{- end -}}`

// filesTmpl renders each file with metadata and content in fenced code blocks.
const filesTmpl = `{{- define "files" -}}
## Files
{{- range .Files}}

### ` + "`" + `{{.Path}}` + "`" + `

> **Size:** {{formatBytes .Size}} | **Tokens:** {{formatNumber .TokenCount}} | **Tier:** {{if .TierLabel}}{{.TierLabel}}{{else}}{{tierLabel .Tier}}{{end}} | **Compressed:** {{if .IsCompressed}}yes{{else}}no{{end}}
{{- if .Error}}

**Error:** {{.Error}}
{{- else}}

` + "```" + `{{fileLang .}}
{{- if $.ShowLineNumbers}}
{{addLineNumbers (escapeTripleBackticks .Content)}}
{{- else}}
{{escapeTripleBackticks .Content}}
{{- end}}
` + "```" + `
{{- end}}
{{- end}}
{{- end -}}`

// changeSummaryTmpl renders the diff mode change summary section, only when
// DiffSummary is non-nil.
const changeSummaryTmpl = `{{- define "changeSummary" -}}
{{- if .DiffSummary}}

## Change Summary

| Change Type | Count |
|-------------|-------|
| Added | {{len .DiffSummary.AddedFiles}} |
| Modified | {{len .DiffSummary.ModifiedFiles}} |
| Deleted | {{len .DiffSummary.DeletedFiles}} |
{{- if gt (len .DiffSummary.AddedFiles) 0}}

### Added Files
{{range .DiffSummary.AddedFiles}}
- {{.}}
{{- end}}
{{- end}}
{{- if gt (len .DiffSummary.ModifiedFiles) 0}}

### Modified Files
{{range .DiffSummary.ModifiedFiles}}
- {{.}}
{{- end}}
{{- end}}
{{- if gt (len .DiffSummary.DeletedFiles) 0}}

### Deleted Files
{{range .DiffSummary.DeletedFiles}}
- {{.}}
{{- end}}
{{- end}}
{{- end}}
{{- end -}}`
