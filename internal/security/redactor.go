package security

import (
	"context"
	"path/filepath"
	"strings"
	"sync"

	"github.com/bmatcuk/doublestar/v4"
)

// Redactor scans file content for secrets and returns a sanitised copy with
// all detected values replaced by canonical placeholders.
//
// Implementations must be safe for concurrent use from multiple goroutines;
// the pipeline processes files in parallel and shares a single Redactor
// instance.
type Redactor interface {
	// Redact scans content for secrets defined by the implementation's rule
	// set and returns:
	//   - redacted: a copy of content with every matched secret replaced by
	//     FormatReplacement(rule.SecretType).
	//   - matches: an ordered slice of RedactionMatch values describing each
	//     replacement, sorted by LineNumber then StartCol.
	//   - err: a non-nil error if scanning could not complete (e.g. context
	//     cancellation). Partial results in redacted and matches must not be
	//     trusted when err is non-nil.
	//
	// filePath is used only for populating RedactionMatch.FilePath; it is not
	// opened or read by Redact.
	//
	// When the implementation's RedactionConfig.Enabled is false, Redact
	// returns content unchanged with a nil matches slice and nil error.
	Redact(ctx context.Context, content string, filePath string) (redacted string, matches []RedactionMatch, err error)
}

// Compile-time interface compliance check.
var _ Redactor = (*StreamRedactor)(nil)

// StreamRedactor is the concrete implementation of the Redactor interface.
// It processes file content through the detection patterns and entropy
// analyzer, replacing detected secrets with [REDACTED:type] markers.
//
// StreamRedactor is safe for concurrent use. It holds no mutable state after
// construction; each Redact call operates entirely on its own stack.
// The only shared mutable state is the aggregated summary, which is protected
// by a mutex.
type StreamRedactor struct {
	registry *PatternRegistry
	analyzer *EntropyAnalyzer
	config   RedactionConfig
	rules    []RedactionRule  // cached from registry at construction time; read-only after init
	mu       sync.Mutex       // protects summary -- only field mutated after construction
	summary  RedactionSummary // aggregated across all Redact calls
}

// NewStreamRedactor constructs a StreamRedactor with the given registry,
// analyzer, and config. Custom patterns from config.CustomPatterns are
// compiled and appended to the registry's built-in rules.
//
// Passing a nil registry uses NewDefaultRegistry(). Passing a nil analyzer
// uses NewEntropyAnalyzer().
func NewStreamRedactor(registry *PatternRegistry, analyzer *EntropyAnalyzer, cfg RedactionConfig) *StreamRedactor {
	if registry == nil {
		registry = NewDefaultRegistry()
	}
	if analyzer == nil {
		analyzer = NewEntropyAnalyzer()
	}

	// Start from the registry's built-in rules.
	rules := registry.Rules()

	// Append custom patterns from config.
	for _, cp := range cfg.CustomPatterns {
		rule, err := NewRedactionRule(
			cp.ID,
			cp.Description,
			cp.Pattern,
			nil,
			cp.SecretType,
			cp.Confidence,
			0,
		)
		if err != nil {
			// Skip invalid custom patterns; caller validation should catch these.
			continue
		}
		rules = append(rules, rule)
	}

	return &StreamRedactor{
		registry: registry,
		analyzer: analyzer,
		config:   cfg,
		rules:    rules,
		summary: RedactionSummary{
			ByType:       make(map[string]int),
			ByConfidence: make(map[Confidence]int),
		},
	}
}

// Redact scans content for secrets and returns:
//   - redacted: content with secrets replaced by [REDACTED:type] markers
//   - matches: all RedactionMatch values, sorted by LineNumber then StartCol
//   - err: non-nil only if ctx is cancelled mid-scan
//
// When cfg.Enabled is false, Redact returns content unchanged with nil matches.
// When filePath matches any cfg.ExcludePaths pattern, Redact returns unchanged.
func (r *StreamRedactor) Redact(ctx context.Context, content string, filePath string) (string, []RedactionMatch, error) {
	// Fast path: redaction disabled.
	if !r.config.Enabled {
		return content, nil, nil
	}

	// Check path exclusions.
	normalized := filepath.ToSlash(filePath)
	for _, pattern := range r.config.ExcludePaths {
		if ok, _ := doublestar.Match(pattern, normalized); ok {
			return content, nil, nil
		}
	}

	// Determine if heightened scanning mode applies.
	heightened := IsSensitiveFile(filePath)

	// Compute effective confidence threshold (lower by one level in heightened mode).
	effectiveThreshold := r.config.ConfidenceThreshold
	if heightened {
		effectiveThreshold = lowerConfidence(effectiveThreshold)
	}

	// Build the set of active rules at or above the effective threshold.
	activeRules := make([]RedactionRule, 0, len(r.rules))
	for _, rule := range r.rules {
		if rule.Regex == nil {
			continue
		}
		if confidenceLevel(rule.Confidence) >= confidenceLevel(effectiveThreshold) {
			activeRules = append(activeRules, rule)
		}
	}

	// Split into lines, process each, then rejoin.
	lines := strings.Split(content, "\n")
	var matches []RedactionMatch

	inPEMBlock := false
	pemBlockStartLine := 0

	for i, line := range lines {
		lineNum := i + 1

		// Check for context cancellation every 100 lines.
		if i%100 == 0 {
			select {
			case <-ctx.Done():
				return "", nil, ctx.Err()
			default:
			}
		}

		// --- Multi-line PEM block detection ---
		if !inPEMBlock {
			if strings.Contains(line, "-----BEGIN") && containsPrivateKey(line) {
				inPEMBlock = true
				pemBlockStartLine = lineNum
				// Replace this opening line with the redaction marker.
				lines[i] = "[REDACTED:private_key_block]"
				matches = append(matches, RedactionMatch{
					RuleID:      "private-key-block",
					SecretType:  "private_key_block",
					Confidence:  ConfidenceHigh,
					FilePath:    filePath,
					LineNumber:  pemBlockStartLine,
					StartCol:    0,
					EndCol:      len(line),
					Replacement: "[REDACTED:private_key_block]",
				})
				continue
			}
		} else {
			// Inside a PEM block: blank out lines until END.
			if strings.Contains(line, "-----END") && containsPrivateKey(line) {
				inPEMBlock = false
			}
			lines[i] = ""
			continue
		}

		// --- Per-line regex processing ---
		processedLine, lineMatches := r.processLine(line, lineNum, filePath, activeRules)
		lines[i] = processedLine
		matches = append(matches, lineMatches...)

		// --- Entropy analysis pass ---
		// Run in heightened mode, or when effective threshold is low.
		if heightened || confidenceLevel(effectiveThreshold) <= confidenceLevel(ConfidenceLow) {
			entropyLine, entropyMatches := r.processEntropy(processedLine, lineNum, filePath, effectiveThreshold)
			if len(entropyMatches) > 0 {
				lines[i] = entropyLine
				matches = append(matches, entropyMatches...)
			}
		}
	}

	redacted := strings.Join(lines, "\n")

	// Update aggregated summary under the mutex.
	r.mu.Lock()
	r.summary.TotalCount += len(matches)
	if len(matches) > 0 {
		r.summary.FileCount++
		for _, m := range matches {
			r.summary.ByType[m.SecretType]++
			r.summary.ByConfidence[m.Confidence]++
		}
	}
	r.mu.Unlock()

	return redacted, matches, nil
}

// Summary returns a copy of the aggregated RedactionSummary across all
// Redact calls made on this StreamRedactor instance.
func (r *StreamRedactor) Summary() RedactionSummary {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Deep-copy the maps so callers cannot mutate internal state.
	byType := make(map[string]int, len(r.summary.ByType))
	for k, v := range r.summary.ByType {
		byType[k] = v
	}
	byConf := make(map[Confidence]int, len(r.summary.ByConfidence))
	for k, v := range r.summary.ByConfidence {
		byConf[k] = v
	}

	return RedactionSummary{
		TotalCount:   r.summary.TotalCount,
		ByType:       byType,
		ByConfidence: byConf,
		FileCount:    r.summary.FileCount,
	}
}

// lineReplacement represents a single in-line text replacement to be applied
// to a processed line.
type lineReplacement struct {
	start       int
	end         int
	replacement string
	match       RedactionMatch
}

// processLine applies all activeRules to a single line and returns the
// (possibly redacted) line along with any RedactionMatch records.
func (r *StreamRedactor) processLine(line string, lineNum int, filePath string, activeRules []RedactionRule) (string, []RedactionMatch) {
	var replacements []lineReplacement
	lowerLine := strings.ToLower(line)

	for _, rule := range activeRules {
		// Keyword pre-filter: skip if none of the keywords appear in the line.
		if len(rule.Keywords) > 0 {
			found := false
			for _, kw := range rule.Keywords {
				if strings.Contains(lowerLine, strings.ToLower(kw)) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Apply regex to the line.
		allMatches := rule.Regex.FindAllStringSubmatchIndex(line, -1)
		for _, loc := range allMatches {
			if len(loc) < 2 {
				continue
			}

			// Determine capture group positions.
			// loc[0]:loc[1] is the full match.
			// loc[2]:loc[3] is capture group 1 (the secret value).
			var secretStart, secretEnd int
			if len(loc) >= 4 && loc[2] >= 0 {
				// Use capture group 1.
				secretStart = loc[2]
				secretEnd = loc[3]
			} else {
				// Fall back to full match.
				secretStart = loc[0]
				secretEnd = loc[1]
			}

			secretValue := line[secretStart:secretEnd]

			// Entropy threshold check.
			if rule.EntropyThreshold > 0 {
				entropy := r.analyzer.Calculate(secretValue)
				if entropy < rule.EntropyThreshold {
					continue
				}
			}

			repl := FormatReplacement(rule.SecretType)
			replacements = append(replacements, lineReplacement{
				start:       secretStart,
				end:         secretEnd,
				replacement: repl,
				match: RedactionMatch{
					RuleID:      rule.ID,
					SecretType:  rule.SecretType,
					Confidence:  rule.Confidence,
					FilePath:    filePath,
					LineNumber:  lineNum,
					StartCol:    secretStart,
					EndCol:      secretEnd,
					Replacement: repl,
				},
			})
		}
	}

	if len(replacements) == 0 {
		return line, nil
	}

	// Deduplicate overlapping replacements: keep leftmost, then rightmost
	// non-overlapping. Sort by start position, then process right-to-left.
	replacements = deduplicateReplacements(replacements)

	// Apply replacements right-to-left to preserve correct offsets.
	var matches []RedactionMatch
	result := line
	for i := len(replacements) - 1; i >= 0; i-- {
		rep := replacements[i]
		// Guard against out-of-bounds after earlier replacements.
		if rep.start < 0 || rep.end > len(result) || rep.start > rep.end {
			continue
		}
		result = result[:rep.start] + rep.replacement + result[rep.end:]
		matches = append(matches, rep.match)
	}

	// Matches were built right-to-left; reverse to get left-to-right order.
	for left, right := 0, len(matches)-1; left < right; left, right = left+1, right-1 {
		matches[left], matches[right] = matches[right], matches[left]
	}

	return result, matches
}

// processEntropy tokenizes the line and runs entropy analysis on tokens that
// are >= 16 chars and have not already been redacted.
func (r *StreamRedactor) processEntropy(line string, lineNum int, filePath string, threshold Confidence) (string, []RedactionMatch) {
	// Tokenize: split on whitespace and common delimiters.
	tokens := tokenizeLine(line)
	if len(tokens) == 0 {
		return line, nil
	}

	varName := extractVarName(line)
	tctx := TokenContext{
		VariableName: varName,
		FilePath:     filePath,
		LineContent:  line,
	}

	var matches []RedactionMatch
	result := line

	// tokenMatch holds a candidate token along with the entropy analysis result.
	type tokenMatch struct {
		start      int
		end        int
		confidence Confidence
	}
	var toRedact []tokenMatch

	// Find byte ranges already occupied by [REDACTED:...] markers so we
	// don't re-process fragments of those markers as secrets.
	redactedSpans := findRedactedSpans(line)

	for _, tok := range tokens {
		// Skip tokens whose byte range overlaps an existing [REDACTED:...] span.
		if overlapsRedacted(tok.start, tok.end, redactedSpans) {
			continue
		}
		// Minimum 16 characters for entropy analysis.
		if len(tok.value) < 16 {
			continue
		}

		entropyResult := r.analyzer.AnalyzeToken(tok.value, tctx)
		if !entropyResult.IsHigh {
			continue
		}
		if confidenceLevel(entropyResult.Confidence) < confidenceLevel(threshold) {
			continue
		}

		toRedact = append(toRedact, tokenMatch{
			start:      tok.start,
			end:        tok.end,
			confidence: entropyResult.Confidence,
		})
	}

	if len(toRedact) == 0 {
		return line, nil
	}

	// Apply replacements right-to-left to preserve correct offsets.
	for i := len(toRedact) - 1; i >= 0; i-- {
		tm := toRedact[i]
		if tm.start < 0 || tm.end > len(result) || tm.start > tm.end {
			continue
		}
		repl := "[REDACTED:high_entropy_secret]"
		result = result[:tm.start] + repl + result[tm.end:]
		matches = append(matches, RedactionMatch{
			RuleID:      "entropy-analyzer",
			SecretType:  "high_entropy_secret",
			Confidence:  tm.confidence,
			FilePath:    filePath,
			LineNumber:  lineNum,
			StartCol:    tm.start,
			EndCol:      tm.end,
			Replacement: repl,
		})
	}

	// Reverse matches to get left-to-right order.
	for left, right := 0, len(matches)-1; left < right; left, right = left+1, right-1 {
		matches[left], matches[right] = matches[right], matches[left]
	}

	return result, matches
}

// lineToken represents a token extracted from a line with its byte positions.
type lineToken struct {
	value string
	start int
	end   int
}

// tokenizeLine splits a line into tokens by whitespace and common delimiters
// (=, :, ", '). Returns tokens with their original byte positions in the line.
func tokenizeLine(line string) []lineToken {
	var tokens []lineToken
	start := -1
	for i := 0; i < len(line); i++ {
		c := line[i]
		isSep := c == ' ' || c == '\t' || c == '=' || c == ':' || c == '"' || c == '\''
		if isSep {
			if start >= 0 {
				tokens = append(tokens, lineToken{
					value: line[start:i],
					start: start,
					end:   i,
				})
				start = -1
			}
		} else {
			if start < 0 {
				start = i
			}
		}
	}
	if start >= 0 {
		tokens = append(tokens, lineToken{
			value: line[start:],
			start: start,
			end:   len(line),
		})
	}
	return tokens
}

// extractVarName parses the line to find the variable name before the first
// '=' or ':' separator. Returns the last identifier segment of the left-hand side.
func extractVarName(line string) string {
	// Find first = or :
	idx := strings.IndexAny(line, "=:")
	if idx <= 0 {
		return ""
	}
	lhs := strings.TrimSpace(line[:idx])
	if lhs == "" {
		return ""
	}
	// Return the last segment after '.', '[', or whitespace.
	for i := len(lhs) - 1; i >= 0; i-- {
		c := lhs[i]
		if c == '.' || c == '[' || c == ' ' || c == '\t' {
			return lhs[i+1:]
		}
	}
	return lhs
}

// containsPrivateKey reports whether the line contains "PRIVATE KEY-----"
// (with optional trailing \r). This is used to detect PEM block boundaries.
func containsPrivateKey(line string) bool {
	trimmed := strings.TrimRight(line, "\r")
	return strings.Contains(trimmed, "PRIVATE KEY-----")
}

// confidenceLevel converts a Confidence value to a comparable integer.
// Higher numbers mean higher confidence.
func confidenceLevel(c Confidence) int {
	switch c {
	case ConfidenceHigh:
		return 2
	case ConfidenceMedium:
		return 1
	default: // ConfidenceLow or empty
		return 0
	}
}

// lowerConfidence reduces a confidence level by one step:
//
//	high   -> medium
//	medium -> low
//	low    -> low (floor)
//	empty  -> low
func lowerConfidence(c Confidence) Confidence {
	switch c {
	case ConfidenceHigh:
		return ConfidenceMedium
	case ConfidenceMedium:
		return ConfidenceLow
	default:
		return ConfidenceLow
	}
}

// deduplicateReplacements removes overlapping replacement ranges, keeping the
// first (leftmost start) match when two ranges overlap. The returned slice is
// sorted by start position ascending.
func deduplicateReplacements(reps []lineReplacement) []lineReplacement {
	if len(reps) <= 1 {
		return reps
	}

	// Sort by start position (insertion sort; typically very few replacements per line).
	for i := 1; i < len(reps); i++ {
		for j := i; j > 0 && reps[j].start < reps[j-1].start; j-- {
			reps[j], reps[j-1] = reps[j-1], reps[j]
		}
	}

	// Remove overlaps: only keep a replacement if its start >= the end of
	// the last accepted replacement.
	result := reps[:0:len(reps)]
	result = append(result, reps[0])
	for i := 1; i < len(reps); i++ {
		last := result[len(result)-1]
		if reps[i].start >= last.end {
			result = append(result, reps[i])
		}
	}
	return result
}

// findRedactedSpans returns the [start, end) byte positions of all
// [REDACTED:...] markers already present in line.
func findRedactedSpans(line string) [][2]int {
	const prefix = "[REDACTED:"
	var spans [][2]int
	search := line
	offset := 0
	for {
		idx := strings.Index(search, prefix)
		if idx < 0 {
			break
		}
		absStart := offset + idx
		rest := search[idx:]
		endIdx := strings.Index(rest, "]")
		if endIdx < 0 {
			break
		}
		absEnd := absStart + endIdx + 1
		spans = append(spans, [2]int{absStart, absEnd})
		advance := idx + endIdx + 1
		if advance >= len(search) {
			break
		}
		offset += advance
		search = search[advance:]
	}
	return spans
}

// overlapsRedacted reports whether the byte range [tokStart, tokEnd) overlaps
// any of the given spans.
func overlapsRedacted(tokStart, tokEnd int, spans [][2]int) bool {
	for _, s := range spans {
		if tokStart < s[1] && tokEnd > s[0] {
			return true
		}
	}
	return false
}
