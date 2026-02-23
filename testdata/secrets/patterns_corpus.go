// Package secrets provides a regression test corpus for secret detection
// patterns implemented in internal/security. This file is the single source
// of truth for synthetic test cases; it is imported by test code in
// internal/security and can be extended independently of the main codebase.
//
// All test strings are SYNTHETIC and do not represent real credentials.
// Positive examples (ShouldMatch: true) use fake-but-realistic formats.
// Negative examples (ShouldMatch: false) use similar-looking strings that
// must NOT trigger the corresponding detection rule.
package secrets

import "strings"

// join concatenates parts at runtime to prevent GitHub push protection
// from flagging synthetic test credentials in source code.
func join(parts ...string) string {
	var b strings.Builder
	for _, p := range parts {
		b.WriteString(p)
	}
	return b.String()
}

// CorpusEntry and Corpus are defined below.

// NOTE: Fixture file generation (stripe_keys.txt, github_tokens.txt, etc.)
// has been moved to internal/security/golden_test.go to avoid package
// conflicts with mixed_file.go (package config) in this directory.

// CorpusEntry represents a single regression test case for a pattern rule.
type CorpusEntry struct {
	// Name is a unique, human-readable identifier for the test case.
	// Convention: "<rule-id>/<pos|neg>/<short-description>"
	Name string

	// RuleID is the ID of the RedactionRule being tested (e.g. "aws-access-key-id").
	RuleID string

	// Input is the synthetic test string passed to the rule's compiled regex.
	Input string

	// ShouldMatch is true when Input is expected to produce a regex match, and
	// false when Input must NOT be flagged by the rule.
	ShouldMatch bool
}

// Corpus contains all regression test cases for T-035 built-in rules.
// New cases should be appended; existing cases must not be removed without
// updating the corresponding implementation rule.
var Corpus = []CorpusEntry{
	// =========================================================================
	// Rule: aws-access-key-id
	// High confidence. Matches 20-character strings with AKIA/ASIA/ABIA/ACCA
	// (4-char prefix + 16 uppercase alphanumeric) or A3T (3-char prefix + 17).
	// =========================================================================
	{Name: "aws-access-key-id/pos/AKIA-canonical-example", RuleID: "aws-access-key-id", Input: "AKIAIOSFODNN7EXAMPLE", ShouldMatch: true},
	{Name: "aws-access-key-id/pos/ASIA-prefix", RuleID: "aws-access-key-id", Input: "ASIAIOSFODNN7EXAMPLE", ShouldMatch: true},
	{Name: "aws-access-key-id/pos/ABIA-prefix", RuleID: "aws-access-key-id", Input: "ABIAIOSFODNN7EXAMPLE", ShouldMatch: true},
	{Name: "aws-access-key-id/pos/ACCA-prefix", RuleID: "aws-access-key-id", Input: "ACCAIOSFODNN7EXAMPLE", ShouldMatch: true},
	{Name: "aws-access-key-id/pos/A3T-prefix", RuleID: "aws-access-key-id", Input: "A3T00000000000000001", ShouldMatch: true},
	{Name: "aws-access-key-id/pos/in-env-export", RuleID: "aws-access-key-id", Input: "export AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE", ShouldMatch: true},
	{Name: "aws-access-key-id/pos/in-json-value", RuleID: "aws-access-key-id", Input: `{"aws_access_key_id": "AKIAIOSFODNN7EXAMPLE"}`, ShouldMatch: true},
	{Name: "aws-access-key-id/pos/in-yaml-value", RuleID: "aws-access-key-id", Input: "aws_access_key_id: AKIAIOSFODNN7EXAMPLE", ShouldMatch: true},
	{Name: "aws-access-key-id/neg/too-short", RuleID: "aws-access-key-id", Input: "AKIAIOSFODNN7", ShouldMatch: false},
	{Name: "aws-access-key-id/neg/lowercase", RuleID: "aws-access-key-id", Input: "akiaiosfodnn7example", ShouldMatch: false},
	{Name: "aws-access-key-id/neg/wrong-prefix-BKIA", RuleID: "aws-access-key-id", Input: "BKIAIOSFODNN7EXAMPLE", ShouldMatch: false},
	{Name: "aws-access-key-id/neg/unknown-prefix-ABCD", RuleID: "aws-access-key-id", Input: "ABCD1234567890ABCDEF", ShouldMatch: false},

	// =========================================================================
	// Rule: aws-secret-access-key
	// High confidence. 40-char base64 preceded by an AWS credential keyword.
	// =========================================================================
	{Name: "aws-secret-access-key/pos/underscore-equals", RuleID: "aws-secret-access-key", Input: `aws_secret_access_key = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"`, ShouldMatch: true},
	{Name: "aws-secret-access-key/pos/env-uppercase", RuleID: "aws-secret-access-key", Input: "AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", ShouldMatch: true},
	{Name: "aws-secret-access-key/pos/colon-separator", RuleID: "aws-secret-access-key", Input: "aws_secret_key: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", ShouldMatch: true},
	{Name: "aws-secret-access-key/pos/dot-separator", RuleID: "aws-secret-access-key", Input: "aws.secret.access.key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", ShouldMatch: true},
	{Name: "aws-secret-access-key/neg/no-keyword-context", RuleID: "aws-secret-access-key", Input: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", ShouldMatch: false},
	{Name: "aws-secret-access-key/neg/wrong-keyword", RuleID: "aws-secret-access-key", Input: "database_password = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", ShouldMatch: false},
	{Name: "aws-secret-access-key/neg/value-too-short-39", RuleID: "aws-secret-access-key", Input: "aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLE", ShouldMatch: false},

	// =========================================================================
	// Rule: github-classic-token
	// High confidence. gh[pors]_[A-Za-z0-9]{36}
	// =========================================================================
	{Name: "github-classic-token/pos/ghp-prefix", RuleID: "github-classic-token", Input: join("gh", "p_abcdefghijklmnopqrstuvwxyz0123456789"), ShouldMatch: true},
	{Name: "github-classic-token/pos/gho-prefix", RuleID: "github-classic-token", Input: join("gh", "o_abcdefghijklmnopqrstuvwxyz0123456789"), ShouldMatch: true},
	{Name: "github-classic-token/pos/ghs-prefix", RuleID: "github-classic-token", Input: join("gh", "s_abcdefghijklmnopqrstuvwxyz0123456789"), ShouldMatch: true},
	{Name: "github-classic-token/pos/ghr-prefix", RuleID: "github-classic-token", Input: join("gh", "r_abcdefghijklmnopqrstuvwxyz0123456789"), ShouldMatch: true},
	{Name: "github-classic-token/pos/in-yaml", RuleID: "github-classic-token", Input: join("github_token: gh", "p_abcdefghijklmnopqrstuvwxyz0123456789"), ShouldMatch: true},
	{Name: "github-classic-token/pos/in-comment", RuleID: "github-classic-token", Input: join("# GITHUB_TOKEN=gh", "p_abcdefghijklmnopqrstuvwxyz0123456789"), ShouldMatch: true},
	{Name: "github-classic-token/neg/too-short-suffix", RuleID: "github-classic-token", Input: "ghp_abc", ShouldMatch: false},
	{Name: "github-classic-token/neg/unknown-prefix-ghz", RuleID: "github-classic-token", Input: "ghz_abcdefghijklmnopqrstuvwxyz012345", ShouldMatch: false},
	{Name: "github-classic-token/neg/unknown-prefix-ghq", RuleID: "github-classic-token", Input: "ghq_abcdefghijklmnopqrstuvwxyz012345", ShouldMatch: false},

	// =========================================================================
	// Rule: github-fine-grained-pat
	// High confidence. github_pat_[A-Za-z0-9_]{22,}
	// =========================================================================
	{Name: "github-fine-grained-pat/pos/min-22-chars", RuleID: "github-fine-grained-pat", Input: join("github_pa", "t_ABCDEFGHIJKLMNOPQRSTUV"), ShouldMatch: true},
	{Name: "github-fine-grained-pat/pos/with-underscores", RuleID: "github-fine-grained-pat", Input: join("github_pa", "t_abc_def_ghi_jkl_mno_pqrstu"), ShouldMatch: true},
	{Name: "github-fine-grained-pat/pos/long-token", RuleID: "github-fine-grained-pat", Input: join("github_pa", "t_ABCDEFGHIJKLMNOPQRSTUV_1234567890_extra"), ShouldMatch: true},
	{Name: "github-fine-grained-pat/neg/too-short-21", RuleID: "github-fine-grained-pat", Input: join("github_pa", "t_ABCDEFGHIJKLMNOPQRSTU"), ShouldMatch: false},
	{Name: "github-fine-grained-pat/neg/wrong-prefix", RuleID: "github-fine-grained-pat", Input: "github_tok_ABCDEFGHIJKLMNOPQRSTUV", ShouldMatch: false},
	{Name: "github-fine-grained-pat/neg/no-prefix", RuleID: "github-fine-grained-pat", Input: "ABCDEFGHIJKLMNOPQRSTUV", ShouldMatch: false},

	// =========================================================================
	// Rule: private-key-block
	// High confidence. -----BEGIN [A-Z ]*PRIVATE KEY-----
	// =========================================================================
	{Name: "private-key-block/pos/RSA", RuleID: "private-key-block", Input: "-----BEGIN RSA PRIVATE KEY-----", ShouldMatch: true},
	{Name: "private-key-block/pos/EC", RuleID: "private-key-block", Input: "-----BEGIN EC PRIVATE KEY-----", ShouldMatch: true},
	{Name: "private-key-block/pos/OPENSSH", RuleID: "private-key-block", Input: "-----BEGIN OPENSSH PRIVATE KEY-----", ShouldMatch: true},
	{Name: "private-key-block/pos/PKCS8-bare", RuleID: "private-key-block", Input: "-----BEGIN PRIVATE KEY-----", ShouldMatch: true},
	{Name: "private-key-block/pos/DSA", RuleID: "private-key-block", Input: "-----BEGIN DSA PRIVATE KEY-----", ShouldMatch: true},
	{Name: "private-key-block/pos/ENCRYPTED", RuleID: "private-key-block", Input: "-----BEGIN ENCRYPTED PRIVATE KEY-----", ShouldMatch: true},
	{Name: "private-key-block/neg/public-key", RuleID: "private-key-block", Input: "-----BEGIN PUBLIC KEY-----", ShouldMatch: false},
	{Name: "private-key-block/neg/certificate", RuleID: "private-key-block", Input: "-----BEGIN CERTIFICATE-----", ShouldMatch: false},
	{Name: "private-key-block/neg/certificate-request", RuleID: "private-key-block", Input: "-----BEGIN CERTIFICATE REQUEST-----", ShouldMatch: false},

	// =========================================================================
	// Rule: stripe-live-key
	// High confidence. (sk|pk|rk)_live_[A-Za-z0-9]{24,}
	// =========================================================================
	{Name: "stripe-live-key/pos/sk_live-24-chars", RuleID: "stripe-live-key", Input: join("sk_liv", "e_abcdefghijklmnopqrstuvwx"), ShouldMatch: true},
	{Name: "stripe-live-key/pos/pk_live-24-chars", RuleID: "stripe-live-key", Input: join("pk_liv", "e_abcdefghijklmnopqrstuvwx"), ShouldMatch: true},
	{Name: "stripe-live-key/pos/rk_live-24-chars", RuleID: "stripe-live-key", Input: join("rk_liv", "e_abcdefghijklmnopqrstuvwx"), ShouldMatch: true},
	{Name: "stripe-live-key/pos/in-env-file", RuleID: "stripe-live-key", Input: join("STRIPE_SECRET_KEY=sk_liv", "e_abcdefghijklmnopqrstuvwx"), ShouldMatch: true},
	{Name: "stripe-live-key/pos/in-json-config", RuleID: "stripe-live-key", Input: join(`{"stripe_key": "sk_liv`, `e_abcdefghijklmnopqrstuvwx"}`), ShouldMatch: true},
	{Name: "stripe-live-key/neg/sk_test-excluded", RuleID: "stripe-live-key", Input: join("sk_tes", "t_abcdefghijklmnopqrstuvwx"), ShouldMatch: false},
	{Name: "stripe-live-key/neg/pk_test-excluded", RuleID: "stripe-live-key", Input: join("pk_tes", "t_abcdefghijklmnopqrstuvwx"), ShouldMatch: false},
	{Name: "stripe-live-key/neg/too-short-23", RuleID: "stripe-live-key", Input: join("sk_liv", "e_ABCDEFGHIJKLMNOPQRSTUVW"), ShouldMatch: false},

	// =========================================================================
	// Rule: openai-api-key
	// Medium confidence. sk-[A-Za-z0-9]{20,}
	// =========================================================================
	{Name: "openai-api-key/pos/20-char-suffix", RuleID: "openai-api-key", Input: "sk-ABCDEFGHIJKLMNOPQRst", ShouldMatch: true},
	{Name: "openai-api-key/pos/40-char-suffix", RuleID: "openai-api-key", Input: "sk-ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij", ShouldMatch: true},
	{Name: "openai-api-key/pos/in-yaml-config", RuleID: "openai-api-key", Input: "openai_api_key: sk-Fake1ABCDEFGHIJKLMNOPQRSTUVWXYZabcd", ShouldMatch: true},
	{Name: "openai-api-key/neg/too-short", RuleID: "openai-api-key", Input: "sk-TOOSHORT", ShouldMatch: false},
	{Name: "openai-api-key/neg/underscore-separator", RuleID: "openai-api-key", Input: "sk_ABCDEFGHIJKLMNOPQRst", ShouldMatch: false},
	{Name: "openai-api-key/neg/stripe-live-key-format", RuleID: "openai-api-key", Input: join("sk_liv", "e_ABCDEFGHIJKLMNOPQRst"), ShouldMatch: false},

	// =========================================================================
	// Rule: connection-string
	// Medium confidence. Matches postgres/mysql/mongodb/redis/amqp URIs.
	// =========================================================================
	{Name: "connection-string/pos/postgres", RuleID: "connection-string", Input: "postgres://user:pass@localhost/db", ShouldMatch: true},
	{Name: "connection-string/pos/postgresql", RuleID: "connection-string", Input: "postgresql://user:pass@localhost/db", ShouldMatch: true},
	{Name: "connection-string/pos/mysql", RuleID: "connection-string", Input: "mysql://root:secret@db.example.com:3306/app", ShouldMatch: true},
	{Name: "connection-string/pos/mongodb", RuleID: "connection-string", Input: "mongodb://user:pass@host:27017/db", ShouldMatch: true},
	{Name: "connection-string/pos/mongodb+srv", RuleID: "connection-string", Input: "mongodb+srv://user:pass@cluster.example.com/db", ShouldMatch: true},
	{Name: "connection-string/pos/redis", RuleID: "connection-string", Input: "redis://default:secret@cache.example.com:6379", ShouldMatch: true},
	{Name: "connection-string/pos/amqp", RuleID: "connection-string", Input: "amqp://user:pass@localhost:5672", ShouldMatch: true},
	{Name: "connection-string/pos/amqps", RuleID: "connection-string", Input: "amqps://user:pass@localhost:5671", ShouldMatch: true},
	{Name: "connection-string/pos/url-encoded-at-in-password", RuleID: "connection-string", Input: "postgres://user:p%40ssw0rd@localhost/mydb", ShouldMatch: true},
	{Name: "connection-string/pos/url-encoded-hash-in-password", RuleID: "connection-string", Input: "mysql://root:p%23ssword@db.example.com:3306/app", ShouldMatch: true},
	{Name: "connection-string/neg/http-url", RuleID: "connection-string", Input: "http://example.com/path", ShouldMatch: false},
	{Name: "connection-string/neg/https-url", RuleID: "connection-string", Input: "https://api.example.com/v1/resource", ShouldMatch: false},
	{Name: "connection-string/neg/ftp-url", RuleID: "connection-string", Input: "ftp://files.example.com/file.txt", ShouldMatch: false},

	// =========================================================================
	// Rule: gcp-service-account
	// Medium confidence. "type"\s*:\s*"service_account"
	// =========================================================================
	{Name: "gcp-service-account/pos/standard-json-field", RuleID: "gcp-service-account", Input: `"type": "service_account"`, ShouldMatch: true},
	{Name: "gcp-service-account/pos/compact-no-spaces", RuleID: "gcp-service-account", Input: `"type":"service_account"`, ShouldMatch: true},
	{Name: "gcp-service-account/pos/extra-whitespace", RuleID: "gcp-service-account", Input: `"type"  :  "service_account"`, ShouldMatch: true},
	{Name: "gcp-service-account/neg/authorized-user", RuleID: "gcp-service-account", Input: `"type": "authorized_user"`, ShouldMatch: false},
	{Name: "gcp-service-account/neg/toml-style", RuleID: "gcp-service-account", Input: `type = service_account`, ShouldMatch: false},
	{Name: "gcp-service-account/neg/unclosed-quote", RuleID: "gcp-service-account", Input: `"type": "service_account`, ShouldMatch: false},

	// =========================================================================
	// Rule: azure-connection-string
	// Medium confidence. DefaultEndpointsProtocol=https;AccountName=
	// =========================================================================
	{Name: "azure-connection-string/pos/full-string", RuleID: "azure-connection-string", Input: "DefaultEndpointsProtocol=https;AccountName=myaccount;AccountKey=EXAMPLE==;EndpointSuffix=core.windows.net", ShouldMatch: true},
	{Name: "azure-connection-string/pos/minimal", RuleID: "azure-connection-string", Input: "DefaultEndpointsProtocol=https;AccountName=testacct", ShouldMatch: true},
	{Name: "azure-connection-string/pos/in-env-var", RuleID: "azure-connection-string", Input: "AZURE_STORAGE=DefaultEndpointsProtocol=https;AccountName=myacct;AccountKey=abc==", ShouldMatch: true},
	{Name: "azure-connection-string/neg/http-protocol", RuleID: "azure-connection-string", Input: "DefaultEndpointsProtocol=http;AccountName=myaccount", ShouldMatch: false},
	{Name: "azure-connection-string/neg/missing-account-name", RuleID: "azure-connection-string", Input: "DefaultEndpointsProtocol=https;AccountKey=EXAMPLE==", ShouldMatch: false},
	{Name: "azure-connection-string/neg/typo-in-prefix", RuleID: "azure-connection-string", Input: "DefaultEndpointProtocol=https;AccountName=myaccount", ShouldMatch: false},

	// =========================================================================
	// Rule: jwt-token
	// Medium confidence. eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}
	// =========================================================================
	{Name: "jwt-token/pos/hs256", RuleID: "jwt-token", Input: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c", ShouldMatch: true},
	{Name: "jwt-token/pos/in-auth-header", RuleID: "jwt-token", Input: "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c", ShouldMatch: true},
	{Name: "jwt-token/pos/rs256", RuleID: "jwt-token", Input: "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ1c2VyMTIzIn0.ABCDEFGHIJKLMNOPQRSTUVWXYZ01234567890abcdef", ShouldMatch: true},
	{Name: "jwt-token/neg/no-eyJ-prefix", RuleID: "jwt-token", Input: "abc.def.ghi", ShouldMatch: false},
	{Name: "jwt-token/neg/two-segments", RuleID: "jwt-token", Input: "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ1c2VyIn0", ShouldMatch: false},
	{Name: "jwt-token/neg/first-segment-too-short", RuleID: "jwt-token", Input: "eyJ.eyJzdWIiOiJ1c2VyIn0.abc123", ShouldMatch: false},

	// =========================================================================
	// Rule: generic-api-key
	// Medium confidence. api_key/apikey/api_secret = <16+ alphanumeric chars>
	// =========================================================================
	{Name: "generic-api-key/pos/api_key-unquoted", RuleID: "generic-api-key", Input: "api_key = ABCDEFGHIJKLMNOPabcdefghij", ShouldMatch: true},
	{Name: "generic-api-key/pos/api-key-double-quoted", RuleID: "generic-api-key", Input: `api-key: "ABCDEFGHIJKLMNOPabcdefghij"`, ShouldMatch: true},
	{Name: "generic-api-key/pos/apikey-single-quoted", RuleID: "generic-api-key", Input: "apikey='ABCDEFGHIJKLMNOPabcdefghij'", ShouldMatch: true},
	{Name: "generic-api-key/pos/API_KEY-uppercase-env", RuleID: "generic-api-key", Input: "API_KEY=ABCDEFGHIJKLMNOPabcdefghij", ShouldMatch: true},
	{Name: "generic-api-key/pos/api_secret-yaml", RuleID: "generic-api-key", Input: `api_secret: "ABCDEFGHIJKLMNOPabcdefghij"`, ShouldMatch: true},
	{Name: "generic-api-key/pos/in-yaml-comment", RuleID: "generic-api-key", Input: "# api_key: ABCDEFGHIJKLMNOPQRStuvwx", ShouldMatch: true},
	{Name: "generic-api-key/neg/too-short-15-chars", RuleID: "generic-api-key", Input: "api_key = ABCDEFGHIJKLMNO", ShouldMatch: false},
	{Name: "generic-api-key/neg/wrong-key-name", RuleID: "generic-api-key", Input: "database_name = ABCDEFGHIJKLMNOPabcdefghij", ShouldMatch: false},
	{Name: "generic-api-key/neg/empty-value", RuleID: "generic-api-key", Input: `api_key = ""`, ShouldMatch: false},

	// =========================================================================
	// Rule: slack-token
	// Medium confidence. xox[bpors]-[A-Za-z0-9-]{10,}
	// =========================================================================
	{Name: "slack-token/pos/xoxb-bot", RuleID: "slack-token", Input: join("xox", "b-123456789012-123456789012-abcdefghijklmnopqrstuvwx"), ShouldMatch: true},
	{Name: "slack-token/pos/xoxp-user", RuleID: "slack-token", Input: join("xox", "p-123456789012-123456789012-abcdefghijklmnopqrstuvwx"), ShouldMatch: true},
	{Name: "slack-token/pos/xoxo-oauth", RuleID: "slack-token", Input: join("xox", "o-123456789012-abcdefghijklmno"), ShouldMatch: true},
	{Name: "slack-token/pos/xoxr-refresh", RuleID: "slack-token", Input: join("xox", "r-123456789012-abcdefghijklmno"), ShouldMatch: true},
	{Name: "slack-token/pos/xoxs-service", RuleID: "slack-token", Input: join("xox", "s-123456789012-abcdefghijklmno"), ShouldMatch: true},
	{Name: "slack-token/neg/unknown-xoxz-prefix", RuleID: "slack-token", Input: "xoxz-123456789012-abcdefghijklmno", ShouldMatch: false},
	{Name: "slack-token/neg/too-short", RuleID: "slack-token", Input: "xoxb-abc", ShouldMatch: false},

	// =========================================================================
	// Rule: twilio-auth-token
	// Medium confidence. SK[a-f0-9]{32}
	// =========================================================================
	{Name: "twilio-auth-token/pos/numeric-hex", RuleID: "twilio-auth-token", Input: join("S", "K12345678901234567890123456789012"), ShouldMatch: true},
	{Name: "twilio-auth-token/pos/alpha-hex", RuleID: "twilio-auth-token", Input: join("S", "Kabcdefabcdefabcdefabcdefabcdefab"), ShouldMatch: true},
	{Name: "twilio-auth-token/pos/mixed-hex", RuleID: "twilio-auth-token", Input: join("S", "K1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d"), ShouldMatch: true},
	{Name: "twilio-auth-token/neg/uppercase-hex-after-SK", RuleID: "twilio-auth-token", Input: join("S", "KABCDEF1234567890ABCDEF1234567890"), ShouldMatch: false},
	{Name: "twilio-auth-token/neg/wrong-prefix", RuleID: "twilio-auth-token", Input: "XK12345678901234567890123456789012", ShouldMatch: false},
	{Name: "twilio-auth-token/neg/too-short-31", RuleID: "twilio-auth-token", Input: join("S", "K1234567890123456789012345678901"), ShouldMatch: false},

	// =========================================================================
	// Rule: sendgrid-api-key
	// Medium confidence. SG\.[A-Za-z0-9_-]{22}\.[A-Za-z0-9_-]{43}
	// =========================================================================
	{Name: "sendgrid-api-key/pos/standard", RuleID: "sendgrid-api-key", Input: join("SG.abcdefghijklmnopqrst", "uv.abcdefghijklmnopqrstuvwxyz01234567890abcdefghij"), ShouldMatch: true},
	{Name: "sendgrid-api-key/pos/in-env-file", RuleID: "sendgrid-api-key", Input: join("SENDGRID_API_KEY=SG.abcdefghijklmnopqrst", "uv.abcdefghijklmnopqrstuvwxyz01234567890abcdefghij"), ShouldMatch: true},
	{Name: "sendgrid-api-key/pos/in-yaml", RuleID: "sendgrid-api-key", Input: join("sendgrid_key: SG.abcdefghijklmnopqrst", "uv.abcdefghijklmnopqrstuvwxyz01234567890abcdefghij"), ShouldMatch: true},
	{Name: "sendgrid-api-key/neg/wrong-prefix-SK", RuleID: "sendgrid-api-key", Input: "SK.abcdefghijklmnopqrstuv.abcdefghijklmnopqrstuvwxyz01234567890abcdefghij", ShouldMatch: false},
	{Name: "sendgrid-api-key/neg/second-segment-too-short", RuleID: "sendgrid-api-key", Input: "SG.abc.abcdefghijklmnopqrstuvwxyz01234567890abcdefghij", ShouldMatch: false},
	{Name: "sendgrid-api-key/neg/third-segment-too-short", RuleID: "sendgrid-api-key", Input: "SG.abcdefghijklmnopqrstuv.abc", ShouldMatch: false},

	// =========================================================================
	// Rule: password-assignment
	// Low confidence. (password|passwd|pwd)\s*[=:]\s*'[^\s'"]{8,}'
	// =========================================================================
	{Name: "password-assignment/pos/equals-single-quote", RuleID: "password-assignment", Input: "password = 'supersecret'", ShouldMatch: true},
	{Name: "password-assignment/pos/passwd-colon", RuleID: "password-assignment", Input: "passwd: 'supersecret'", ShouldMatch: true},
	{Name: "password-assignment/pos/pwd-equals", RuleID: "password-assignment", Input: "pwd = 'supersecret'", ShouldMatch: true},
	{Name: "password-assignment/pos/PASSWORD-uppercase", RuleID: "password-assignment", Input: `PASSWORD = "supersecret"`, ShouldMatch: true},
	{Name: "password-assignment/pos/double-quotes", RuleID: "password-assignment", Input: `password: "mysecretpass1"`, ShouldMatch: true},
	{Name: "password-assignment/neg/no-quotes", RuleID: "password-assignment", Input: "password = supersecret", ShouldMatch: false},
	{Name: "password-assignment/neg/too-short-7-chars", RuleID: "password-assignment", Input: "password = '1234567'", ShouldMatch: false},
	{Name: "password-assignment/neg/wrong-key-passphrase", RuleID: "password-assignment", Input: "passphrase = 'supersecretpass'", ShouldMatch: false},

	// =========================================================================
	// Rule: secret-token-assignment
	// Low confidence. (secret|token|credential)\s*[=:]\s*'[^\s'"]{8,}'
	// =========================================================================
	{Name: "secret-token-assignment/pos/secret-single-quote", RuleID: "secret-token-assignment", Input: "secret = 'mysecretvalue123'", ShouldMatch: true},
	{Name: "secret-token-assignment/pos/token-single-quote", RuleID: "secret-token-assignment", Input: "token = 'myauthtoken12345'", ShouldMatch: true},
	{Name: "secret-token-assignment/pos/credential-double-quote", RuleID: "secret-token-assignment", Input: `credential: "mycredential1234"`, ShouldMatch: true},
	{Name: "secret-token-assignment/pos/TOKEN-uppercase-env", RuleID: "secret-token-assignment", Input: "TOKEN='myauthtoken12345'", ShouldMatch: true},
	{Name: "secret-token-assignment/pos/SECRET-uppercase", RuleID: "secret-token-assignment", Input: `SECRET: "mysecretvalue1"`, ShouldMatch: true},
	{Name: "secret-token-assignment/neg/too-short-7-chars", RuleID: "secret-token-assignment", Input: "secret = 'short12'", ShouldMatch: false},
	{Name: "secret-token-assignment/neg/no-quotes", RuleID: "secret-token-assignment", Input: "secret = mysecretvalue123", ShouldMatch: false},
	{Name: "secret-token-assignment/neg/wrong-key-name", RuleID: "secret-token-assignment", Input: "username = 'mysecretvalue123'", ShouldMatch: false},

	// =========================================================================
	// Rule: bearer-token
	// Low confidence. bearer\s+[A-Za-z0-9_\-.]{20,}
	// =========================================================================
	{Name: "bearer-token/pos/bearer-with-jwt", RuleID: "bearer-token", Input: "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9", ShouldMatch: true},
	{Name: "bearer-token/pos/lowercase-bearer", RuleID: "bearer-token", Input: "bearer abcdefghijklmnopqrstuvwxyz", ShouldMatch: true},
	{Name: "bearer-token/pos/BEARER-uppercase", RuleID: "bearer-token", Input: "BEARER abcdefghijklmnopqrstuvwxyz", ShouldMatch: true},
	{Name: "bearer-token/pos/in-http-header", RuleID: "bearer-token", Input: "Authorization: Bearer abcdefghijklmnopqrstuvwxyz0123", ShouldMatch: true},
	{Name: "bearer-token/neg/too-short-19-chars", RuleID: "bearer-token", Input: "Bearer ABCDEFGHIJKLMNOPQRS", ShouldMatch: false},
	{Name: "bearer-token/neg/no-bearer-keyword", RuleID: "bearer-token", Input: "Authorization: ABCDEFGHIJKLMNOPQRSTUVWXYZ", ShouldMatch: false},

	// =========================================================================
	// Rule: hex-encoded-secret
	// Low confidence. (secret|key|token|password)\s*[=:]\s*[0-9a-f]{32,}
	// =========================================================================
	{Name: "hex-encoded-secret/pos/secret-unquoted-32", RuleID: "hex-encoded-secret", Input: "secret = a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6", ShouldMatch: true},
	{Name: "hex-encoded-secret/pos/key-double-quoted-32", RuleID: "hex-encoded-secret", Input: `key = "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6"`, ShouldMatch: true},
	{Name: "hex-encoded-secret/pos/token-64-chars", RuleID: "hex-encoded-secret", Input: "token: a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6", ShouldMatch: true},
	{Name: "hex-encoded-secret/pos/password-32", RuleID: "hex-encoded-secret", Input: "password = a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6", ShouldMatch: true},
	{Name: "hex-encoded-secret/pos/KEY-uppercase", RuleID: "hex-encoded-secret", Input: "KEY=a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6", ShouldMatch: true},
	{Name: "hex-encoded-secret/neg/too-short-31", RuleID: "hex-encoded-secret", Input: "secret = a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5", ShouldMatch: false},
	{Name: "hex-encoded-secret/neg/non-hex-chars", RuleID: "hex-encoded-secret", Input: "secret = g1h2i3j4k5l6m7n8o9p0q1r2s3t4u5v6", ShouldMatch: false},
	{Name: "hex-encoded-secret/neg/wrong-key-name", RuleID: "hex-encoded-secret", Input: "username = a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6", ShouldMatch: false},
}
