package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// ShowOptions controls the rendering of a resolved profile.
type ShowOptions struct {
	// Profile is the fully merged profile to display.
	Profile *Profile

	// Sources maps flat field names to their origin layer.
	Sources SourceMap

	// ProfileName is the name of the profile being displayed.
	ProfileName string

	// Chain is the inheritance chain in resolution order, e.g. ["finvault", "default"].
	Chain []string
}

// ShowProfile renders a resolved profile as annotated TOML. Each field is
// printed with an inline comment indicating which configuration layer
// provided its value. The output is human-readable and approximately valid
// TOML (inline comments are not part of the TOML spec but are widely
// supported by editors and tooling).
//
// The Chain parameter should come from ProfileResolution.Chain.
func ShowProfile(opts ShowOptions) string {
	var b strings.Builder

	// Header comments.
	fmt.Fprintf(&b, "# Resolved profile: %s\n", opts.ProfileName)
	if len(opts.Chain) > 1 {
		fmt.Fprintf(&b, "# Inheritance chain: %s\n", strings.Join(opts.Chain, " -> "))
	}
	fmt.Fprintf(&b, "\n")

	p := opts.Profile
	src := opts.Sources

	// Scalar fields.
	writeStringField(&b, "output", p.Output, sourceLabel(src, "output"))
	writeStringField(&b, "format", p.Format, sourceLabel(src, "format"))
	writeIntField(&b, "max_tokens", p.MaxTokens, sourceLabel(src, "max_tokens"))
	writeStringField(&b, "tokenizer", p.Tokenizer, sourceLabel(src, "tokenizer"))
	writeBoolField(&b, "compression", p.Compression, sourceLabel(src, "compression"))
	writeBoolField(&b, "redaction", p.Redaction, sourceLabel(src, "redaction"))
	if p.Target != "" {
		writeStringField(&b, "target", p.Target, sourceLabel(src, "target"))
	}

	// Slice fields.
	writeStringSliceField(&b, "ignore", p.Ignore, sourceLabel(src, "ignore"))
	if len(p.PriorityFiles) > 0 {
		writeStringSliceField(&b, "priority_files", p.PriorityFiles, sourceLabel(src, "priority_files"))
	}
	if len(p.Include) > 0 {
		writeStringSliceField(&b, "include", p.Include, sourceLabel(src, "include"))
	}

	// Relevance section.
	b.WriteString("\n")
	writeRelevanceSection(&b, p.Relevance, src)

	// RedactionConfig section.
	if p.RedactionConfig.Enabled || p.RedactionConfig.ConfidenceThreshold != "" || len(p.RedactionConfig.ExcludePaths) > 0 {
		b.WriteString("\n")
		writeRedactionConfigSection(&b, p.RedactionConfig, src)
	}

	return b.String()
}

// ShowProfileJSON serializes the resolved profile to indented JSON. It returns
// the JSON bytes as a string. An error is returned only if marshalling fails,
// which should not happen for well-formed Profile values.
func ShowProfileJSON(p *Profile) (string, error) {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal profile to JSON: %w", err)
	}
	return string(data), nil
}

// sourceLabel returns the Source.String() for a given flat key, defaulting to
// "default" when the key is absent from the SourceMap.
func sourceLabel(src SourceMap, key string) string {
	if s, ok := src[key]; ok {
		return s.String()
	}
	return "default"
}

// writeStringField writes a TOML string assignment with an inline source comment.
func writeStringField(b *strings.Builder, key, value, source string) {
	// TOML string: escape backslashes and double-quotes.
	escaped := strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(value)
	fmt.Fprintf(b, "%-20s = %-30s # %s\n", key, `"`+escaped+`"`, source)
}

// writeIntField writes a TOML integer assignment with an inline source comment.
func writeIntField(b *strings.Builder, key string, value int, source string) {
	fmt.Fprintf(b, "%-20s = %-30d # %s\n", key, value, source)
}

// writeBoolField writes a TOML boolean assignment with an inline source comment.
func writeBoolField(b *strings.Builder, key string, value bool, source string) {
	boolStr := "false"
	if value {
		boolStr = "true"
	}
	fmt.Fprintf(b, "%-20s = %-30s # %s\n", key, boolStr, source)
}

// writeStringSliceField writes a multi-line TOML array with an inline source
// comment on the opening bracket line.
func writeStringSliceField(b *strings.Builder, key string, values []string, source string) {
	if len(values) == 0 {
		fmt.Fprintf(b, "%-20s = []%-27s # %s\n", key, "", source)
		return
	}

	fmt.Fprintf(b, "%-20s = [%-29s # %s\n", key, "", source)
	for _, v := range values {
		// %q produces a Go double-quoted string, which is valid TOML.
		fmt.Fprintf(b, "  %q,\n", v)
	}
	b.WriteString("]\n")
}

// writeRelevanceSection writes the [relevance] TOML table with per-tier source
// annotations.
func writeRelevanceSection(b *strings.Builder, rel RelevanceConfig, src SourceMap) {
	fmt.Fprintf(b, "[relevance]\n")
	writeTierField(b, "tier_0", rel.Tier0, sourceLabel(src, "relevance.tier_0"))
	writeTierField(b, "tier_1", rel.Tier1, sourceLabel(src, "relevance.tier_1"))
	writeTierField(b, "tier_2", rel.Tier2, sourceLabel(src, "relevance.tier_2"))
	writeTierField(b, "tier_3", rel.Tier3, sourceLabel(src, "relevance.tier_3"))
	writeTierField(b, "tier_4", rel.Tier4, sourceLabel(src, "relevance.tier_4"))
	writeTierField(b, "tier_5", rel.Tier5, sourceLabel(src, "relevance.tier_5"))
}

// writeTierField writes a single relevance tier as a TOML array with a source comment.
func writeTierField(b *strings.Builder, key string, values []string, source string) {
	if len(values) == 0 {
		fmt.Fprintf(b, "%-8s = []%-27s # %s\n", key, "", source)
		return
	}

	fmt.Fprintf(b, "%-8s = [%-29s # %s\n", key, "", source)
	for _, v := range values {
		// %q produces a Go double-quoted string, which is valid TOML.
		fmt.Fprintf(b, "  %q,\n", v)
	}
	b.WriteString("]\n")
}

// writeRedactionConfigSection writes the [redaction_config] TOML table.
func writeRedactionConfigSection(b *strings.Builder, rc RedactionConfig, src SourceMap) {
	fmt.Fprintf(b, "[redaction_config]\n")
	writeBoolSectionField(b, "enabled", rc.Enabled, sourceLabel(src, "redaction_config.enabled"))
	writeStringSectionField(b, "confidence_threshold", rc.ConfidenceThreshold, sourceLabel(src, "redaction_config.confidence_threshold"))
	if len(rc.ExcludePaths) > 0 {
		writeArraySectionField(b, "exclude_paths", rc.ExcludePaths, sourceLabel(src, "redaction_config.exclude_paths"))
	}
}

// writeBoolSectionField writes a boolean field inside a TOML section.
func writeBoolSectionField(b *strings.Builder, key string, value bool, source string) {
	boolStr := "false"
	if value {
		boolStr = "true"
	}
	fmt.Fprintf(b, "%-24s = %-26s # %s\n", key, boolStr, source)
}

// writeStringSectionField writes a string field inside a TOML section.
func writeStringSectionField(b *strings.Builder, key, value, source string) {
	escaped := strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(value)
	fmt.Fprintf(b, "%-24s = %-26s # %s\n", key, `"`+escaped+`"`, source)
}

// writeArraySectionField writes an array field inside a TOML section.
func writeArraySectionField(b *strings.Builder, key string, values []string, source string) {
	if len(values) == 0 {
		fmt.Fprintf(b, "%-24s = []%-23s # %s\n", key, "", source)
		return
	}
	var arr bytes.Buffer
	arr.WriteString("[")
	for i, v := range values {
		if i > 0 {
			arr.WriteString(", ")
		}
		fmt.Fprintf(&arr, "%q", v)
	}
	arr.WriteString("]")
	fmt.Fprintf(b, "%-24s = %-26s # %s\n", key, arr.String(), source)
}
