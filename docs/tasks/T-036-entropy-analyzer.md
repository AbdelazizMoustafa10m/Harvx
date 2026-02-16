# T-036: Shannon Entropy Analyzer for High-Entropy String Detection

**Priority:** Must Have
**Effort:** Medium (6-10hrs)
**Dependencies:** T-034
**Phase:** 3 - Security

---

## Description

Implement a Shannon entropy analyzer that calculates the information entropy of strings to detect high-entropy values that may be secrets (API keys, tokens, random passwords) even when no specific regex pattern matches. The analyzer operates as a secondary signal alongside regex-based detection: it can promote low-confidence regex matches to medium confidence, or independently flag unrecognized high-entropy strings in sensitive contexts (e.g., assignments to variables named `key`, `secret`, `token`). Configurable thresholds allow tuning the sensitivity.

## User Story

As a developer, I want Harvx to detect secrets that do not match any known regex pattern but have the statistical characteristics of randomly generated credentials, so that novel or custom secret formats are still caught before I share my code with an LLM.

## Acceptance Criteria

- [ ] `EntropyAnalyzer` struct with configurable thresholds for different character sets
- [ ] `Calculate(s string) float64` returns Shannon entropy of a string (bits per character)
- [ ] `IsHighEntropy(s string, charset CharacterSet) bool` checks against threshold for the given character set
- [ ] Character set detection: `DetectCharset(s string) CharacterSet` identifies if string is hex, base64, alphanumeric, or mixed
- [ ] Default thresholds calibrated to minimize false positives:
  - Hex strings (0-9a-f): threshold >= 3.0 (max ~4.0)
  - Base64 strings (A-Za-z0-9+/=): threshold >= 4.5 (max ~6.0)
  - Generic alphanumeric: threshold >= 4.0 (max ~5.7)
- [ ] Minimum string length filter: strings shorter than 16 characters are skipped (too short for reliable entropy measurement)
- [ ] Maximum string length filter: strings longer than 256 characters use only the first 256 chars for entropy calculation (performance bound)
- [ ] `AnalyzeToken(token string, context TokenContext) EntropyResult` returns entropy value, charset, whether it exceeds threshold, and suggested confidence level
- [ ] `TokenContext` struct includes: variable name, file path, line content -- used to boost confidence when entropy is borderline but context is suspicious (e.g., variable named `api_key`)
- [ ] Integration point: `EntropyAnalyzer` is usable both standalone and as a field in `RedactionRule` (optional entropy threshold per rule)
- [ ] Unit tests achieve >= 90% coverage
- [ ] Benchmarks for entropy calculation on various string lengths (target: < 1 microsecond for 100-char string)

## Technical Notes

- **Shannon entropy formula**: `H = -sum(p_i * log2(p_i))` where `p_i` is the frequency of character `i` divided by string length. This is a simple O(n) computation.
- **Do NOT use external entropy packages** (lazybeaver/entropy, chrisjchandler/entropy). The calculation is trivial (~15 lines of Go) and avoids an unnecessary dependency. Implement it directly in `internal/security/entropy.go`.
- **Calibration data from research**: Non-secret source code text typically has entropy around 3.5-4.0 bits/char. Truly random secrets (API keys, tokens) typically have entropy 4.5-5.5 bits/char for base64, and 3.5-4.0 for hex. The overlap zone (4.0-4.5) is where false positives occur. Handle this by requiring both high entropy AND contextual signals (suspicious variable name, sensitive file path).
- **Character set detection**: Classify strings by their character composition:
  - Hex: only `[0-9a-fA-F]`
  - Base64: only `[A-Za-z0-9+/=_-]` (includes URL-safe base64)
  - Alphanumeric: only `[A-Za-z0-9]`
  - Mixed: contains special characters beyond the above
- **Context boosting**: When the token appears in an assignment like `API_KEY = "..."` or `password: "..."`, lower the entropy threshold by 0.5 to catch borderline cases.
- **Performance**: Entropy calculation should be done AFTER regex matching fails, not on every line. It is a supplementary signal, not a primary detector.
- Reference: https://blog.miloslavhomer.cz/p/secret-detection-shannon-entropy for calibration insights.

## Files to Create/Modify

- `internal/security/entropy.go` - `EntropyAnalyzer`, `Calculate`, `IsHighEntropy`, `DetectCharset`, `AnalyzeToken`
- `internal/security/entropy_test.go` - Comprehensive tests including calibration validation
- `internal/security/entropy_bench_test.go` - Benchmarks for performance validation

## Testing Requirements

- **Known-entropy tests**: Strings with pre-calculated expected entropy values
  - `"aaaa"` -> entropy 0.0
  - `"abcd"` -> entropy 2.0
  - `"aabb"` -> entropy 1.0
  - Random 32-char hex string -> entropy ~3.8-4.0
  - Random 40-char base64 string -> entropy ~5.0-5.5
- **Real-world calibration tests**:
  - Common English words should NOT trigger high-entropy (e.g., `"authentication"`, `"configuration"`)
  - UUIDs (e.g., `"550e8400-e29b-41d4-a716-446655440000"`) should be borderline -- only flagged with context
  - Base64-encoded content should trigger (e.g., AWS secret keys are 40-char base64)
  - Hex-encoded SHA256 hashes should trigger
- **Character set detection tests**: Verify correct classification of hex, base64, alphanumeric, mixed strings
- **Context boosting tests**: Verify that `API_KEY = "<borderline-entropy>"` triggers but the same string without context does not
- **Edge cases**:
  - Empty string returns entropy 0.0
  - Single character returns entropy 0.0
  - Very long string (10K chars) completes in bounded time
  - Unicode strings are handled without panics (skip non-ASCII for entropy calc or handle gracefully)
- **Benchmarks**:
  - `BenchmarkCalculate/short_16` (16-char string)
  - `BenchmarkCalculate/medium_64` (64-char string)
  - `BenchmarkCalculate/long_256` (256-char string)
  - Target: all under 1 microsecond
