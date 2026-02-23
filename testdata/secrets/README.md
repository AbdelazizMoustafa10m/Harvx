# Secret Detection Test Corpus

This directory contains synthetic test fixtures for the Harvx secret detection and
redaction subsystem (`internal/security`).

## Purpose

These fixtures are used by:

- `internal/security/golden_test.go` — golden regression tests
- `internal/security/fuzz_test.go` — fuzz testing invariants
- `internal/security/corpus_bench_test.go` — performance benchmarks

**All credentials in this directory are SYNTHETIC and non-functional.**
They are designed to match the structural format of real secrets without being
valid credentials for any service.

## File Inventory

| File | Description | Rules Exercised |
|------|-------------|-----------------|
| `aws_keys.txt` | AWS INI-style credentials file | `aws-access-key-id`, `aws-secret-access-key` |
| `github_tokens.txt` | GitHub classic and fine-grained PATs | `github-classic-token`, `github-fine-grained-pat` |
| `stripe_keys.txt` | Stripe live and test keys | `stripe-live-key` |
| `openai_keys.txt` | OpenAI API key formats | `openai-api-key` |
| `private_keys.txt` | PEM private key blocks | `private-key-block` |
| `connection_strings.txt` | Database/broker URIs | `connection-string` |
| `jwt_tokens.txt` | JSON Web Tokens | `jwt-token` |
| `cloud_credentials.txt` | GCP and Azure credentials | `gcp-service-account`, `azure-connection-string` |
| `generic_assignments.txt` | Generic config assignments | `generic-api-key`, `slack-token`, `twilio-auth-token`, `sendgrid-api-key`, `password-assignment`, `secret-token-assignment`, `hex-encoded-secret` |
| `mixed_file.go` | Go source with embedded secrets | Multiple rules |
| `mixed_file.ts` | TypeScript config with secrets | Multiple rules |
| `mixed_file.py` | Python config with secrets | Multiple rules |
| `config.env` | .env file with all credential types | Multiple rules |
| `docker-compose.yml` | Docker Compose with credentials | Multiple rules |
| `false_positives.txt` | Strings that look like secrets but are not | None (all should pass) |

## Expected Results Format

Each fixture file has a corresponding `.expected` JSON file that lists the
expected redaction matches:

```json
{
  "expected_redactions": [
    {
      "line": 4,
      "rule_id": "aws-access-key-id",
      "secret_type": "aws_access_key_id",
      "confidence": "high"
    }
  ]
}
```

Fields:
- `line`: 1-based line number in the fixture file
- `rule_id`: the ID of the rule that should fire
- `secret_type`: the secret type category
- `confidence`: `"high"`, `"medium"`, or `"low"`

## Adding New Fixtures

1. Create the fixture file with synthetic (non-functional) credentials
2. Create the `.expected` JSON file with exact line numbers
3. Run `go test ./internal/security/... -run TestGoldenCorpus` to verify

## Performance Expectations

The full corpus should process in under 500ms on a modern machine.
Run benchmarks with: `go test ./internal/security/... -bench BenchmarkFullCorpus`.
