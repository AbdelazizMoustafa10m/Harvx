# T-039: Redaction Report and Output Summary

**Priority:** Must Have
**Effort:** Medium (6-10hrs)
**Dependencies:** T-034, T-037
**Phase:** 3 - Security

---

## Description

Implement the redaction reporting system that provides two output mechanisms: (1) an inline redaction summary embedded in the main Harvx output document showing aggregate counts by type, and (2) a detailed `--redaction-report` flag that generates a standalone report listing every redaction with file path, line number, rule ID, and secret type -- without revealing the actual secret value. The report enables developers to audit what was redacted and take corrective action (rotate keys, move secrets to vaults).

## User Story

As a developer, I want to see a summary of what secrets were redacted in my output and optionally generate a detailed report, so that I can verify the redaction is working correctly and know which credentials to rotate.

## Acceptance Criteria

- [ ] **Inline summary in output**: The output document's summary/metadata section includes a "Redaction Summary" block:
  ```
  Redactions:  7 (3 API keys, 2 connection strings, 1 private key block, 1 JWT token)
  ```
- [ ] Summary is generated from the aggregated `RedactionSummary` struct produced by the redactor
- [ ] **`--redaction-report` flag**: When set, writes a detailed report to a file (default: `harvx-redaction-report.json`)
- [ ] The report file path can be customized: `--redaction-report=path/to/report.json`
- [ ] **Report structure** (JSON format):
  ```json
  {
    "generated_at": "2026-02-16T10:30:00Z",
    "profile": "finvault",
    "confidence_threshold": "high",
    "summary": {
      "total_redactions": 7,
      "files_affected": 4,
      "by_type": {
        "aws_access_key": 2,
        "connection_string": 2,
        "private_key_block": 1,
        "github_token": 1,
        "jwt_token": 1
      },
      "by_confidence": {
        "high": 5,
        "medium": 2,
        "low": 0
      }
    },
    "redactions": [
      {
        "file": "config/database.go",
        "line": 42,
        "rule_id": "connection-string",
        "secret_type": "connection_string",
        "confidence": "high",
        "replacement": "[REDACTED:connection_string]"
      }
    ]
  }
  ```
- [ ] The report NEVER contains the actual secret value -- only metadata about what was found and where
- [ ] **Human-readable report option**: `--redaction-report=report.txt` (detected by extension) produces a text-formatted report:
  ```
  Harvx Redaction Report
  ======================
  Generated: 2026-02-16 10:30:00 UTC
  Profile: finvault
  Threshold: high

  Summary: 7 redactions in 4 files
    3 API keys (high confidence)
    2 connection strings (high confidence)
    1 private key block (high confidence)
    1 JWT token (medium confidence)

  Details:
    config/database.go:42    [connection_string]  high
    config/database.go:43    [connection_string]  high
    lib/auth.go:15           [aws_access_key]     high
    ...
  ```
- [ ] When `--redaction-report` is used without a value, default to `harvx-redaction-report.json`
- [ ] Report generation does not affect the main output pipeline timing (write report after main output is complete)
- [ ] When no redactions are found, report states "No secrets detected" with zero counts
- [ ] Unit tests for report serialization and formatting

## Technical Notes

- **Report generator location**: `internal/security/report.go`
- **Integration with output rendering**: The inline summary is consumed by the output renderer (Markdown/XML). The renderer calls a formatting function that takes `RedactionSummary` and returns a human-readable string for embedding in the output document. This integrates with the output rendering system from Phase 1.
- **JSON serialization**: Use `encoding/json` with `json.MarshalIndent` for readable output. All struct fields have proper `json:"..."` tags (defined in T-034).
- **Text format detection**: If the report path ends in `.txt` or `.text`, use text format. Otherwise default to JSON.
- **File writing**: Use `os.WriteFile` with `0644` permissions. Create parent directories if needed with `os.MkdirAll`.
- **CLI integration**: The `--redaction-report` flag is defined in the CLI layer (`internal/cli/root.go` or `generate.go`). The flag value is passed through config to the pipeline orchestrator, which passes the collected matches to the report generator after output rendering completes.
- **Sorting**: Redactions in the detailed list are sorted by file path, then by line number for consistent output.
- **No-redact mode**: When `--no-redact` is active, the report flag is ignored (no report generated since no scanning was performed).

## Files to Create/Modify

- `internal/security/report.go` - `ReportGenerator` struct, `GenerateJSON`, `GenerateText`, `FormatInlineSummary` methods
- `internal/security/report_test.go` - Tests for all report formats and edge cases
- `internal/output/markdown.go` - Integration point: embed inline summary in output metadata section (modify existing)

## Testing Requirements

- **JSON report tests**:
  - Generate report with multiple redactions -> verify JSON structure and field values
  - Generate report with zero redactions -> verify "No secrets detected" output
  - Verify report never contains actual secret content (scan output for common secret patterns)
  - Verify sorting: redactions ordered by file path then line number
- **Text report tests**:
  - Generate text report -> verify human-readable formatting
  - Verify alignment of columns in detail section
  - Verify summary counts match detail counts
- **Inline summary tests**:
  - Single redaction type -> `"1 API key"`
  - Multiple types -> `"3 API keys, 2 connection strings, 1 private key block"`
  - Zero redactions -> `"No secrets detected"`
  - Pluralization: `"1 API key"` vs `"3 API keys"`
- **File format detection tests**:
  - `.json` extension -> JSON format
  - `.txt` extension -> text format
  - No extension -> JSON format (default)
- **Edge cases**:
  - Very large number of redactions (1000+) -> report generates without issues
  - File paths with special characters in report
  - Unicode content in surrounding context (non-secret parts of lines)
