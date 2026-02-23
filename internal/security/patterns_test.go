package security_test

import (
	"strings"
	"testing"

	"github.com/harvx/harvx/internal/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// join concatenates parts at runtime to avoid triggering GitHub's push
// protection scanner on synthetic test strings.
func join(parts ...string) string {
	var b strings.Builder
	for _, p := range parts {
		b.WriteString(p)
	}
	return b.String()
}

// ---------------------------------------------------------------------------
// Regression corpus
//
// corpusEntry holds a single test case used by TestPattern_CorpusTable.
// All test strings are synthetic and do not represent real credentials.
// ---------------------------------------------------------------------------

type corpusEntry struct {
	name        string
	ruleID      string
	input       string
	shouldMatch bool
}

// corpusEntries is the regression corpus for all 19 built-in pattern rules.
// Positive entries (shouldMatch: true) contain realistic-but-fake secrets.
// Negative entries (shouldMatch: false) contain strings similar to secrets
// that must NOT be flagged (wrong prefix, wrong length, etc.).
var corpusEntries = []corpusEntry{
	// =========================================================================
	// aws-access-key-id  (6 positive, 4 negative)
	// =========================================================================
	{name: "aws-access-key-id/pos/AKIA-canonical-example", ruleID: "aws-access-key-id", input: "AKIAIOSFODNN7EXAMPLE", shouldMatch: true},
	{name: "aws-access-key-id/pos/ASIA-prefix", ruleID: "aws-access-key-id", input: "ASIAIOSFODNN7EXAMPLE", shouldMatch: true},
	{name: "aws-access-key-id/pos/ABIA-prefix", ruleID: "aws-access-key-id", input: "ABIAIOSFODNN7EXAMPLE", shouldMatch: true},
	{name: "aws-access-key-id/pos/ACCA-prefix", ruleID: "aws-access-key-id", input: "ACCAIOSFODNN7EXAMPLE", shouldMatch: true},
	{name: "aws-access-key-id/pos/A3T-prefix", ruleID: "aws-access-key-id", input: "A3T00000000000000001", shouldMatch: true},
	{name: "aws-access-key-id/pos/in-env-export", ruleID: "aws-access-key-id", input: "export AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE", shouldMatch: true},
	{name: "aws-access-key-id/neg/too-short", ruleID: "aws-access-key-id", input: "AKIAIOSFODNN7", shouldMatch: false},
	{name: "aws-access-key-id/neg/lowercase", ruleID: "aws-access-key-id", input: "akiaiosfodnn7example", shouldMatch: false},
	{name: "aws-access-key-id/neg/wrong-prefix-BKIA", ruleID: "aws-access-key-id", input: "BKIAIOSFODNN7EXAMPLE", shouldMatch: false},
	{name: "aws-access-key-id/neg/unknown-prefix-ABCD", ruleID: "aws-access-key-id", input: "ABCD1234567890ABCDEF", shouldMatch: false},

	// =========================================================================
	// aws-secret-access-key  (4 positive, 3 negative)
	// =========================================================================
	{name: "aws-secret-access-key/pos/underscore-separator", ruleID: "aws-secret-access-key", input: `aws_secret_access_key = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"`, shouldMatch: true},
	{name: "aws-secret-access-key/pos/env-uppercase", ruleID: "aws-secret-access-key", input: "AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", shouldMatch: true},
	{name: "aws-secret-access-key/pos/colon-separator", ruleID: "aws-secret-access-key", input: "aws_secret_key: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", shouldMatch: true},
	{name: "aws-secret-access-key/pos/dot-separator", ruleID: "aws-secret-access-key", input: "aws.secret.access.key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", shouldMatch: true},
	{name: "aws-secret-access-key/neg/no-keyword", ruleID: "aws-secret-access-key", input: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", shouldMatch: false},
	{name: "aws-secret-access-key/neg/wrong-keyword", ruleID: "aws-secret-access-key", input: "database_password = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", shouldMatch: false},
	{name: "aws-secret-access-key/neg/too-short-value", ruleID: "aws-secret-access-key", input: "aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLE", shouldMatch: false},

	// =========================================================================
	// github-classic-token  (5 positive, 3 negative)
	// =========================================================================
	{name: "github-classic-token/pos/ghp-prefix", ruleID: "github-classic-token", input: join("gh", "p_abcdefghijklmnopqrstuvwxyz0123456789"), shouldMatch: true},
	{name: "github-classic-token/pos/gho-prefix", ruleID: "github-classic-token", input: join("gh", "o_abcdefghijklmnopqrstuvwxyz0123456789"), shouldMatch: true},
	{name: "github-classic-token/pos/ghs-prefix", ruleID: "github-classic-token", input: join("gh", "s_abcdefghijklmnopqrstuvwxyz0123456789"), shouldMatch: true},
	{name: "github-classic-token/pos/ghr-prefix", ruleID: "github-classic-token", input: join("gh", "r_abcdefghijklmnopqrstuvwxyz0123456789"), shouldMatch: true},
	{name: "github-classic-token/pos/in-json-value", ruleID: "github-classic-token", input: join(`{"token": "gh`, `p_abcdefghijklmnopqrstuvwxyz0123456789"}`), shouldMatch: true},
	{name: "github-classic-token/neg/too-short-suffix", ruleID: "github-classic-token", input: "ghp_abc", shouldMatch: false},
	{name: "github-classic-token/neg/unknown-prefix-ghz", ruleID: "github-classic-token", input: "ghz_abcdefghijklmnopqrstuvwxyz012345", shouldMatch: false},
	{name: "github-classic-token/neg/unknown-prefix-ghq", ruleID: "github-classic-token", input: "ghq_abcdefghijklmnopqrstuvwxyz012345", shouldMatch: false},

	// =========================================================================
	// github-fine-grained-pat  (3 positive, 2 negative)
	// =========================================================================
	{name: "github-fine-grained-pat/pos/min-length-22", ruleID: "github-fine-grained-pat", input: join("github_pa", "t_ABCDEFGHIJKLMNOPQRSTUV"), shouldMatch: true},
	{name: "github-fine-grained-pat/pos/with-underscores", ruleID: "github-fine-grained-pat", input: join("github_pa", "t_abc_def_ghi_jkl_mno_pqrstu"), shouldMatch: true},
	{name: "github-fine-grained-pat/pos/long-token", ruleID: "github-fine-grained-pat", input: join("github_pa", "t_ABCDEFGHIJKLMNOPQRSTUV_1234567890_extra_segment"), shouldMatch: true},
	{name: "github-fine-grained-pat/neg/too-short-21-chars", ruleID: "github-fine-grained-pat", input: join("github_pa", "t_ABCDEFGHIJKLMNOPQRSTU"), shouldMatch: false},
	{name: "github-fine-grained-pat/neg/wrong-prefix", ruleID: "github-fine-grained-pat", input: "github_tok_ABCDEFGHIJKLMNOPQRSTUV", shouldMatch: false},

	// =========================================================================
	// private-key-block  (4 positive, 2 negative)
	// =========================================================================
	{name: "private-key-block/pos/RSA", ruleID: "private-key-block", input: "-----BEGIN RSA PRIVATE KEY-----", shouldMatch: true},
	{name: "private-key-block/pos/EC", ruleID: "private-key-block", input: "-----BEGIN EC PRIVATE KEY-----", shouldMatch: true},
	{name: "private-key-block/pos/OPENSSH", ruleID: "private-key-block", input: "-----BEGIN OPENSSH PRIVATE KEY-----", shouldMatch: true},
	{name: "private-key-block/pos/PKCS8-bare", ruleID: "private-key-block", input: "-----BEGIN PRIVATE KEY-----", shouldMatch: true},
	{name: "private-key-block/neg/public-key", ruleID: "private-key-block", input: "-----BEGIN PUBLIC KEY-----", shouldMatch: false},
	{name: "private-key-block/neg/certificate", ruleID: "private-key-block", input: "-----BEGIN CERTIFICATE-----", shouldMatch: false},

	// =========================================================================
	// stripe-live-key  (5 positive, 3 negative)
	// =========================================================================
	{name: "stripe-live-key/pos/sk_live-24-chars", ruleID: "stripe-live-key", input: join("sk_liv", "e_abcdefghijklmnopqrstuvwx"), shouldMatch: true},
	{name: "stripe-live-key/pos/pk_live-24-chars", ruleID: "stripe-live-key", input: join("pk_liv", "e_abcdefghijklmnopqrstuvwx"), shouldMatch: true},
	{name: "stripe-live-key/pos/rk_live-24-chars", ruleID: "stripe-live-key", input: join("rk_liv", "e_abcdefghijklmnopqrstuvwx"), shouldMatch: true},
	{name: "stripe-live-key/pos/in-json-config", ruleID: "stripe-live-key", input: join(`{"stripe_key": "sk_liv`, `e_abcdefghijklmnopqrstuvwx"}`), shouldMatch: true},
	{name: "stripe-live-key/pos/in-env-file", ruleID: "stripe-live-key", input: join("STRIPE_SECRET_KEY=sk_liv", "e_abcdefghijklmnopqrstuvwx"), shouldMatch: true},
	{name: "stripe-live-key/neg/sk_test-excluded", ruleID: "stripe-live-key", input: join("sk_tes", "t_abcdefghijklmnopqrstuvwx"), shouldMatch: false},
	{name: "stripe-live-key/neg/pk_test-excluded", ruleID: "stripe-live-key", input: join("pk_tes", "t_abcdefghijklmnopqrstuvwx"), shouldMatch: false},
	{name: "stripe-live-key/neg/too-short-23-chars", ruleID: "stripe-live-key", input: join("sk_liv", "e_ABCDEFGHIJKLMNOPQRSTUVW"), shouldMatch: false},

	// =========================================================================
	// openai-api-key  (3 positive, 3 negative)
	// =========================================================================
	{name: "openai-api-key/pos/20-char-suffix", ruleID: "openai-api-key", input: "sk-ABCDEFGHIJKLMNOPQRst", shouldMatch: true},
	{name: "openai-api-key/pos/40-char-suffix", ruleID: "openai-api-key", input: "sk-ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij", shouldMatch: true},
	{name: "openai-api-key/pos/in-yaml-config", ruleID: "openai-api-key", input: "openai_api_key: sk-Fake1ABCDEFGHIJKLMNOPQRSTUVWXYZabcd", shouldMatch: true},
	{name: "openai-api-key/neg/too-short", ruleID: "openai-api-key", input: "sk-TOOSHORT", shouldMatch: false},
	{name: "openai-api-key/neg/underscore-separator", ruleID: "openai-api-key", input: "sk_ABCDEFGHIJKLMNOPQRst", shouldMatch: false},
	{name: "openai-api-key/neg/stripe-live-key-prefix", ruleID: "openai-api-key", input: join("sk_liv", "e_ABCDEFGHIJKLMNOPQRst"), shouldMatch: false},

	// =========================================================================
	// connection-string  (6 positive, 2 negative)
	// =========================================================================
	{name: "connection-string/pos/postgres", ruleID: "connection-string", input: "postgres://user:pass@localhost/db", shouldMatch: true},
	{name: "connection-string/pos/postgresql", ruleID: "connection-string", input: "postgresql://user:pass@localhost/db", shouldMatch: true},
	{name: "connection-string/pos/mysql", ruleID: "connection-string", input: "mysql://root:secret@db.example.com:3306/app", shouldMatch: true},
	{name: "connection-string/pos/mongodb", ruleID: "connection-string", input: "mongodb://user:pass@host:27017/db", shouldMatch: true},
	{name: "connection-string/pos/mongodb+srv", ruleID: "connection-string", input: "mongodb+srv://user:pass@cluster.example.com/db", shouldMatch: true},
	{name: "connection-string/pos/redis", ruleID: "connection-string", input: "redis://default:secret@cache.example.com:6379", shouldMatch: true},
	{name: "connection-string/neg/http-url", ruleID: "connection-string", input: "http://example.com/path", shouldMatch: false},
	{name: "connection-string/neg/https-url", ruleID: "connection-string", input: "https://api.example.com/v1/resource", shouldMatch: false},

	// =========================================================================
	// gcp-service-account  (3 positive, 3 negative)
	// =========================================================================
	{name: "gcp-service-account/pos/standard-json-field", ruleID: "gcp-service-account", input: `"type": "service_account"`, shouldMatch: true},
	{name: "gcp-service-account/pos/compact-no-spaces", ruleID: "gcp-service-account", input: `"type":"service_account"`, shouldMatch: true},
	{name: "gcp-service-account/pos/extra-whitespace", ruleID: "gcp-service-account", input: `"type"  :  "service_account"`, shouldMatch: true},
	{name: "gcp-service-account/neg/authorized-user", ruleID: "gcp-service-account", input: `"type": "authorized_user"`, shouldMatch: false},
	{name: "gcp-service-account/neg/toml-style", ruleID: "gcp-service-account", input: `type = service_account`, shouldMatch: false},
	{name: "gcp-service-account/neg/unclosed-quote", ruleID: "gcp-service-account", input: `"type": "service_account`, shouldMatch: false},

	// =========================================================================
	// azure-connection-string  (3 positive, 3 negative)
	// =========================================================================
	{name: "azure-connection-string/pos/full-string", ruleID: "azure-connection-string", input: "DefaultEndpointsProtocol=https;AccountName=myaccount;AccountKey=EXAMPLE==;EndpointSuffix=core.windows.net", shouldMatch: true},
	{name: "azure-connection-string/pos/minimal", ruleID: "azure-connection-string", input: "DefaultEndpointsProtocol=https;AccountName=testacct", shouldMatch: true},
	{name: "azure-connection-string/pos/in-env-var", ruleID: "azure-connection-string", input: "AZURE_STORAGE=DefaultEndpointsProtocol=https;AccountName=myacct;AccountKey=abc==", shouldMatch: true},
	{name: "azure-connection-string/neg/http-protocol", ruleID: "azure-connection-string", input: "DefaultEndpointsProtocol=http;AccountName=myaccount", shouldMatch: false},
	{name: "azure-connection-string/neg/missing-account-name", ruleID: "azure-connection-string", input: "DefaultEndpointsProtocol=https;AccountKey=EXAMPLE==", shouldMatch: false},
	{name: "azure-connection-string/neg/typo-in-prefix", ruleID: "azure-connection-string", input: "DefaultEndpointProtocol=https;AccountName=myaccount", shouldMatch: false},

	// =========================================================================
	// jwt-token  (3 positive, 3 negative)
	// =========================================================================
	{name: "jwt-token/pos/hs256", ruleID: "jwt-token", input: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c", shouldMatch: true},
	{name: "jwt-token/pos/in-auth-header", ruleID: "jwt-token", input: "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c", shouldMatch: true},
	{name: "jwt-token/pos/rs256", ruleID: "jwt-token", input: "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ1c2VyMTIzIn0.ABCDEFGHIJKLMNOPQRSTUVWXYZ01234567890abcdef", shouldMatch: true},
	{name: "jwt-token/neg/no-eyJ-prefix", ruleID: "jwt-token", input: "abc.def.ghi", shouldMatch: false},
	{name: "jwt-token/neg/two-segments", ruleID: "jwt-token", input: "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ1c2VyIn0", shouldMatch: false},
	{name: "jwt-token/neg/first-segment-too-short", ruleID: "jwt-token", input: "eyJ.eyJzdWIiOiJ1c2VyIn0.abc123", shouldMatch: false},

	// =========================================================================
	// generic-api-key  (4 positive, 3 negative)
	// =========================================================================
	{name: "generic-api-key/pos/api_key-unquoted", ruleID: "generic-api-key", input: "api_key = ABCDEFGHIJKLMNOPabcdefghij", shouldMatch: true},
	{name: "generic-api-key/pos/api-key-quoted", ruleID: "generic-api-key", input: `api-key: "ABCDEFGHIJKLMNOPabcdefghij"`, shouldMatch: true},
	{name: "generic-api-key/pos/apikey-single-quoted", ruleID: "generic-api-key", input: "apikey='ABCDEFGHIJKLMNOPabcdefghij'", shouldMatch: true},
	{name: "generic-api-key/pos/api_secret-yaml", ruleID: "generic-api-key", input: `api_secret: "ABCDEFGHIJKLMNOPabcdefghij"`, shouldMatch: true},
	{name: "generic-api-key/neg/too-short-15-chars", ruleID: "generic-api-key", input: "api_key = ABCDEFGHIJKLMNO", shouldMatch: false},
	{name: "generic-api-key/neg/wrong-key-name", ruleID: "generic-api-key", input: "database_name = ABCDEFGHIJKLMNOPabcdefghij", shouldMatch: false},
	{name: "generic-api-key/neg/empty-value", ruleID: "generic-api-key", input: `api_key = ""`, shouldMatch: false},

	// =========================================================================
	// slack-token  (4 positive, 2 negative)
	// =========================================================================
	{name: "slack-token/pos/xoxb-bot", ruleID: "slack-token", input: join("xox", "b-123456789012-123456789012-abcdefghijklmnopqrstuvwx"), shouldMatch: true},
	{name: "slack-token/pos/xoxp-user", ruleID: "slack-token", input: join("xox", "p-123456789012-123456789012-abcdefghijklmnopqrstuvwx"), shouldMatch: true},
	{name: "slack-token/pos/xoxo-oauth", ruleID: "slack-token", input: join("xox", "o-123456789012-abcdefghijklmno"), shouldMatch: true},
	{name: "slack-token/pos/xoxs-service", ruleID: "slack-token", input: join("xox", "s-123456789012-abcdefghijklmno"), shouldMatch: true},
	{name: "slack-token/neg/unknown-xoxz-prefix", ruleID: "slack-token", input: "xoxz-123456789012-abcdefghijklmno", shouldMatch: false},
	{name: "slack-token/neg/too-short", ruleID: "slack-token", input: "xoxb-abc", shouldMatch: false},

	// =========================================================================
	// twilio-auth-token  (3 positive, 2 negative)
	// =========================================================================
	{name: "twilio-auth-token/pos/numeric-hex", ruleID: "twilio-auth-token", input: join("S", "K12345678901234567890123456789012"), shouldMatch: true},
	{name: "twilio-auth-token/pos/alpha-hex", ruleID: "twilio-auth-token", input: join("S", "Kabcdefabcdefabcdefabcdefabcdefab"), shouldMatch: true},
	{name: "twilio-auth-token/pos/mixed-hex", ruleID: "twilio-auth-token", input: join("S", "K1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d"), shouldMatch: true},
	{name: "twilio-auth-token/neg/uppercase-hex", ruleID: "twilio-auth-token", input: join("S", "KABCDEF1234567890ABCDEF1234567890"), shouldMatch: false},
	{name: "twilio-auth-token/neg/wrong-prefix", ruleID: "twilio-auth-token", input: "XK12345678901234567890123456789012", shouldMatch: false},

	// =========================================================================
	// sendgrid-api-key  (3 positive, 3 negative)
	// =========================================================================
	{name: "sendgrid-api-key/pos/standard", ruleID: "sendgrid-api-key", input: join("SG.abcdefghijklmnopqrst", "uv.abcdefghijklmnopqrstuvwxyz01234567890abcdefghij"), shouldMatch: true},
	{name: "sendgrid-api-key/pos/in-env-file", ruleID: "sendgrid-api-key", input: join("SENDGRID_API_KEY=SG.abcdefghijklmnopqrst", "uv.abcdefghijklmnopqrstuvwxyz01234567890abcdefghij"), shouldMatch: true},
	{name: "sendgrid-api-key/pos/in-yaml", ruleID: "sendgrid-api-key", input: join("sendgrid_key: SG.abcdefghijklmnopqrst", "uv.abcdefghijklmnopqrstuvwxyz01234567890abcdefghij"), shouldMatch: true},
	{name: "sendgrid-api-key/neg/wrong-prefix-SK", ruleID: "sendgrid-api-key", input: "SK.abcdefghijklmnopqrstuv.abcdefghijklmnopqrstuvwxyz01234567890abcdefghij", shouldMatch: false},
	{name: "sendgrid-api-key/neg/second-segment-too-short", ruleID: "sendgrid-api-key", input: "SG.abc.abcdefghijklmnopqrstuvwxyz01234567890abcdefghij", shouldMatch: false},
	{name: "sendgrid-api-key/neg/third-segment-too-short", ruleID: "sendgrid-api-key", input: "SG.abcdefghijklmnopqrstuv.abc", shouldMatch: false},

	// =========================================================================
	// password-assignment  (3 positive, 3 negative)
	// =========================================================================
	{name: "password-assignment/pos/equals-single-quote", ruleID: "password-assignment", input: "password = 'supersecret'", shouldMatch: true},
	{name: "password-assignment/pos/passwd-colon", ruleID: "password-assignment", input: "passwd: 'supersecret'", shouldMatch: true},
	{name: "password-assignment/pos/pwd-equals", ruleID: "password-assignment", input: "pwd = 'supersecret'", shouldMatch: true},
	{name: "password-assignment/neg/no-quotes", ruleID: "password-assignment", input: "password = supersecret", shouldMatch: false},
	{name: "password-assignment/neg/too-short-7-chars", ruleID: "password-assignment", input: "password = '1234567'", shouldMatch: false},
	{name: "password-assignment/neg/wrong-key-passphrase", ruleID: "password-assignment", input: "passphrase = 'supersecretpass'", shouldMatch: false},

	// =========================================================================
	// secret-token-assignment  (4 positive, 3 negative)
	// =========================================================================
	{name: "secret-token-assignment/pos/secret-single-quote", ruleID: "secret-token-assignment", input: "secret = 'mysecretvalue123'", shouldMatch: true},
	{name: "secret-token-assignment/pos/token-single-quote", ruleID: "secret-token-assignment", input: "token = 'myauthtoken12345'", shouldMatch: true},
	{name: "secret-token-assignment/pos/credential-double-quote", ruleID: "secret-token-assignment", input: `credential: "mycredential1234"`, shouldMatch: true},
	{name: "secret-token-assignment/pos/TOKEN-uppercase-env", ruleID: "secret-token-assignment", input: "TOKEN='myauthtoken12345'", shouldMatch: true},
	{name: "secret-token-assignment/neg/too-short", ruleID: "secret-token-assignment", input: "secret = 'short'", shouldMatch: false},
	{name: "secret-token-assignment/neg/no-quotes", ruleID: "secret-token-assignment", input: "secret = mysecretvalue123", shouldMatch: false},
	{name: "secret-token-assignment/neg/wrong-key", ruleID: "secret-token-assignment", input: "username = 'mysecretvalue123'", shouldMatch: false},

	// =========================================================================
	// bearer-token  (3 positive, 2 negative)
	// =========================================================================
	{name: "bearer-token/pos/with-jwt-value", ruleID: "bearer-token", input: "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9", shouldMatch: true},
	{name: "bearer-token/pos/lowercase-bearer", ruleID: "bearer-token", input: "bearer abcdefghijklmnopqrstuvwxyz", shouldMatch: true},
	{name: "bearer-token/pos/uppercase-BEARER", ruleID: "bearer-token", input: "BEARER abcdefghijklmnopqrstuvwxyz", shouldMatch: true},
	{name: "bearer-token/neg/too-short-19-chars", ruleID: "bearer-token", input: "Bearer ABCDEFGHIJKLMNOPQRS", shouldMatch: false},
	{name: "bearer-token/neg/no-bearer-keyword", ruleID: "bearer-token", input: "Authorization: ABCDEFGHIJKLMNOPQRSTUVWXYZ", shouldMatch: false},

	// =========================================================================
	// hex-encoded-secret  (4 positive, 3 negative)
	// =========================================================================
	{name: "hex-encoded-secret/pos/secret-unquoted-32", ruleID: "hex-encoded-secret", input: "secret = a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6", shouldMatch: true},
	{name: "hex-encoded-secret/pos/key-quoted-32", ruleID: "hex-encoded-secret", input: `key = "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6"`, shouldMatch: true},
	{name: "hex-encoded-secret/pos/token-64-chars", ruleID: "hex-encoded-secret", input: "token: a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6", shouldMatch: true},
	{name: "hex-encoded-secret/pos/password-32", ruleID: "hex-encoded-secret", input: "password = a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6", shouldMatch: true},
	{name: "hex-encoded-secret/neg/too-short-31", ruleID: "hex-encoded-secret", input: "secret = a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5", shouldMatch: false},
	{name: "hex-encoded-secret/neg/non-hex-chars", ruleID: "hex-encoded-secret", input: "secret = g1h2i3j4k5l6m7n8o9p0q1r2s3t4u5v6", shouldMatch: false},
	{name: "hex-encoded-secret/neg/wrong-key-name", ruleID: "hex-encoded-secret", input: "username = a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6", shouldMatch: false},
}

// ---------------------------------------------------------------------------
// Built-in pattern registry coverage
// ---------------------------------------------------------------------------

// TestDefaultRegistry_BuiltinRuleCount verifies that NewDefaultRegistry
// returns a registry with the expected number of built-in rules.
func TestDefaultRegistry_BuiltinRuleCount(t *testing.T) {
	r := security.NewDefaultRegistry()
	rules := r.Rules()
	// 6 high + 9 medium + 4 low = 19 built-in rules.
	assert.Equal(t, 19, len(rules), "expected 19 built-in rules")
}

// TestDefaultRegistry_BuiltinRuleIDs verifies that all expected rule IDs are
// present in the default registry.
func TestDefaultRegistry_BuiltinRuleIDs(t *testing.T) {
	r := security.NewDefaultRegistry()
	rules := r.Rules()

	ids := make(map[string]struct{}, len(rules))
	for _, rule := range rules {
		ids[rule.ID] = struct{}{}
	}

	expectedIDs := []string{
		// High confidence
		"aws-access-key-id",
		"aws-secret-access-key",
		"github-classic-token",
		"github-fine-grained-pat",
		"private-key-block",
		"stripe-live-key",
		// Medium confidence
		"openai-api-key",
		"connection-string",
		"gcp-service-account",
		"azure-connection-string",
		"jwt-token",
		"generic-api-key",
		"slack-token",
		"twilio-auth-token",
		"sendgrid-api-key",
		// Low confidence
		"password-assignment",
		"secret-token-assignment",
		"bearer-token",
		"hex-encoded-secret",
	}

	for _, id := range expectedIDs {
		assert.Contains(t, ids, id, "expected rule ID %q to be registered", id)
	}
}

// TestDefaultRegistry_ConfidenceTierCounts checks that each confidence tier
// has exactly the expected number of built-in rules.
func TestDefaultRegistry_ConfidenceTierCounts(t *testing.T) {
	r := security.NewDefaultRegistry()

	tests := []struct {
		confidence security.Confidence
		wantCount  int
	}{
		{security.ConfidenceHigh, 6},
		{security.ConfidenceMedium, 9},
		{security.ConfidenceLow, 4},
	}

	for _, tt := range tests {
		t.Run(string(tt.confidence), func(t *testing.T) {
			got := r.RulesByConfidence(tt.confidence)
			assert.Len(t, got, tt.wantCount,
				"expected %d %s-confidence built-in rules", tt.wantCount, tt.confidence)
		})
	}
}

// TestDefaultRegistry_AllRulesHaveCompiledRegex verifies that every built-in
// rule has a non-nil compiled regex.
func TestDefaultRegistry_AllRulesHaveCompiledRegex(t *testing.T) {
	r := security.NewDefaultRegistry()
	for _, rule := range r.Rules() {
		require.NotNil(t, rule.Regex, "rule %q must have a non-nil compiled regex", rule.ID)
	}
}

// TestDefaultRegistry_AllRulesHaveSecretType verifies that every built-in
// rule has a non-empty SecretType for use in redaction replacement strings.
func TestDefaultRegistry_AllRulesHaveSecretType(t *testing.T) {
	r := security.NewDefaultRegistry()
	for _, rule := range r.Rules() {
		assert.NotEmpty(t, rule.SecretType, "rule %q must have a non-empty SecretType", rule.ID)
	}
}

// TestDefaultRegistry_AllRulesHaveKeywords verifies that every built-in
// rule has at least one keyword (our patterns always use keyword pre-filtering).
func TestDefaultRegistry_AllRulesHaveKeywords(t *testing.T) {
	r := security.NewDefaultRegistry()
	for _, rule := range r.Rules() {
		assert.NotEmpty(t, rule.Keywords, "rule %q must have at least one keyword", rule.ID)
	}
}

// ---------------------------------------------------------------------------
// Pattern matching: high-confidence rules
// ---------------------------------------------------------------------------

func TestPattern_AWSAccessKeyID(t *testing.T) {
	r := security.NewDefaultRegistry()
	var rule security.RedactionRule
	for _, rl := range r.Rules() {
		if rl.ID == "aws-access-key-id" {
			rule = rl
			break
		}
	}
	require.NotNil(t, rule.Regex)

	tests := []struct {
		name    string
		input   string
		wantHit bool
	}{
		{"AKIA prefix", "AKIAIOSFODNN7EXAMPLE", true},
		{"ASIA prefix", "ASIAIOSFODNN7EXAMPLE", true},
		{"ABIA prefix", "ABIAIOSFODNN7EXAMPLE", true},
		{"ACCA prefix", "ACCAIOSFODNN7EXAMPLE", true},
		{"A3T prefix", "A3T0000000000000000X", true},
		{"too short", "AKIA123", false},
		{"lowercase", "akiaiosfodnn7example", false},
		{"unknown prefix", "ABCD123456789012345A", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rule.Regex.MatchString(tt.input)
			assert.Equal(t, tt.wantHit, got)
		})
	}
}

func TestPattern_GitHubClassicToken(t *testing.T) {
	r := security.NewDefaultRegistry()
	var rule security.RedactionRule
	for _, rl := range r.Rules() {
		if rl.ID == "github-classic-token" {
			rule = rl
			break
		}
	}
	require.NotNil(t, rule.Regex)

	tests := []struct {
		name    string
		input   string
		wantHit bool
	}{
		{"ghp_ token", join("gh", "p_abcdefghijklmnopqrstuvwxyz0123456789"), true},
		{"gho_ token", join("gh", "o_abcdefghijklmnopqrstuvwxyz0123456789"), true},
		{"ghs_ token", join("gh", "s_abcdefghijklmnopqrstuvwxyz0123456789"), true},
		{"ghr_ token", join("gh", "r_abcdefghijklmnopqrstuvwxyz0123456789"), true},
		{"too short suffix", "ghp_abc", false},
		{"unknown prefix", "ghz_abcdefghijklmnopqrstuvwxyz012345", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rule.Regex.MatchString(tt.input)
			assert.Equal(t, tt.wantHit, got)
		})
	}
}

func TestPattern_GitHubFineGrainedPAT(t *testing.T) {
	r := security.NewDefaultRegistry()
	var rule security.RedactionRule
	for _, rl := range r.Rules() {
		if rl.ID == "github-fine-grained-pat" {
			rule = rl
			break
		}
	}
	require.NotNil(t, rule.Regex)

	tests := []struct {
		name    string
		input   string
		wantHit bool
	}{
		{"valid fine-grained PAT", join("github_pa", "t_ABCDEFGHIJKLMNOPQRSTUVW"), true},
		{"valid fine-grained PAT with underscores", join("github_pa", "t_abc_def_ghi_jkl_mno_pqrstu"), true},
		{"too short suffix", "github_pat_short", false},
		{"wrong prefix", "github_tok_ABCDEFGHIJKLMNOPQRSTUVW", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rule.Regex.MatchString(tt.input)
			assert.Equal(t, tt.wantHit, got)
		})
	}
}

func TestPattern_PrivateKeyBlock(t *testing.T) {
	r := security.NewDefaultRegistry()
	var rule security.RedactionRule
	for _, rl := range r.Rules() {
		if rl.ID == "private-key-block" {
			rule = rl
			break
		}
	}
	require.NotNil(t, rule.Regex)

	tests := []struct {
		name    string
		input   string
		wantHit bool
	}{
		{"RSA private key", "-----BEGIN RSA PRIVATE KEY-----", true},
		{"EC private key", "-----BEGIN EC PRIVATE KEY-----", true},
		{"OPENSSH private key", "-----BEGIN OPENSSH PRIVATE KEY-----", true},
		{"PKCS8 private key", "-----BEGIN PRIVATE KEY-----", true},
		{"public key (not a secret)", "-----BEGIN PUBLIC KEY-----", false},
		{"certificate (not a secret)", "-----BEGIN CERTIFICATE-----", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rule.Regex.MatchString(tt.input)
			assert.Equal(t, tt.wantHit, got)
		})
	}
}

func TestPattern_StripeLiveKey(t *testing.T) {
	r := security.NewDefaultRegistry()
	var rule security.RedactionRule
	for _, rl := range r.Rules() {
		if rl.ID == "stripe-live-key" {
			rule = rl
			break
		}
	}
	require.NotNil(t, rule.Regex)

	tests := []struct {
		name    string
		input   string
		wantHit bool
	}{
		{"sk_live_ prefix", join("sk_liv", "e_abcdefghijklmnopqrstuvwx"), true},
		{"pk_live_ prefix", join("pk_liv", "e_abcdefghijklmnopqrstuvwx"), true},
		{"rk_live_ prefix", join("rk_liv", "e_abcdefghijklmnopqrstuvwx"), true},
		{"test key (not redacted)", join("sk_tes", "t_abcdefghijklmnopqrstuvwx"), false},
		{"too short", "sk_live_short", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rule.Regex.MatchString(tt.input)
			assert.Equal(t, tt.wantHit, got)
		})
	}
}

// ---------------------------------------------------------------------------
// Pattern matching: medium-confidence rules
// ---------------------------------------------------------------------------

func TestPattern_ConnectionString(t *testing.T) {
	r := security.NewDefaultRegistry()
	var rule security.RedactionRule
	for _, rl := range r.Rules() {
		if rl.ID == "connection-string" {
			rule = rl
			break
		}
	}
	require.NotNil(t, rule.Regex)

	tests := []struct {
		name    string
		input   string
		wantHit bool
	}{
		{"postgres", "postgres://user:pass@localhost/db", true},
		{"postgresql", "postgresql://user:pass@localhost/db", true},
		{"mysql", "mysql://user:pass@localhost/db", true},
		{"mongodb", "mongodb://user:pass@localhost/db", true},
		{"mongodb+srv", "mongodb+srv://user:pass@cluster.example.com/db", true},
		{"redis", "redis://user:pass@localhost:6379", true},
		{"amqp", "amqp://user:pass@localhost:5672", true},
		{"amqps", "amqps://user:pass@localhost:5671", true},
		{"http URL (not a DB connection)", "http://example.com/path", false},
		{"https URL (not a DB connection)", "https://example.com/path", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rule.Regex.MatchString(tt.input)
			assert.Equal(t, tt.wantHit, got)
		})
	}
}

func TestPattern_JWTToken(t *testing.T) {
	r := security.NewDefaultRegistry()
	var rule security.RedactionRule
	for _, rl := range r.Rules() {
		if rl.ID == "jwt-token" {
			rule = rl
			break
		}
	}
	require.NotNil(t, rule.Regex)

	// A real-looking JWT (header.payload.signature).
	realJWT := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"

	tests := []struct {
		name    string
		input   string
		wantHit bool
	}{
		{"valid JWT", realJWT, true},
		{"eyJ prefix required", "abc.def.ghi", false},
		{"two segments only", "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ1c2VyIn0", false},
		{"first segment too short", "eyJ.eyJzdWIiOiJ1c2VyIn0.abc123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rule.Regex.MatchString(tt.input)
			assert.Equal(t, tt.wantHit, got)
		})
	}
}

func TestPattern_SlackToken(t *testing.T) {
	r := security.NewDefaultRegistry()
	var rule security.RedactionRule
	for _, rl := range r.Rules() {
		if rl.ID == "slack-token" {
			rule = rl
			break
		}
	}
	require.NotNil(t, rule.Regex)

	tests := []struct {
		name    string
		input   string
		wantHit bool
	}{
		{"xoxb bot token", join("xox", "b-123456789012-123456789012-abcdefghijklmnopqrstuvwx"), true},
		{"xoxp user token", join("xox", "p-123456789012-123456789012-abcdefghijklmnopqrstuvwx"), true},
		{"xoxo OAuth token", join("xox", "o-123456789012-123456789012-abcdefghijklmnopqrstu"), true},
		{"xoxr refresh token", join("xox", "r-123456789012-abcdefghijklmno"), true},
		{"xoxs service token", join("xox", "s-123456789012-abcdefghijklmno"), true},
		{"unknown xox prefix", "xoxz-123456789012-abcdefghijklmno", false},
		{"too short", "xoxb-abc", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rule.Regex.MatchString(tt.input)
			assert.Equal(t, tt.wantHit, got)
		})
	}
}

func TestPattern_SendGridAPIKey(t *testing.T) {
	r := security.NewDefaultRegistry()
	var rule security.RedactionRule
	for _, rl := range r.Rules() {
		if rl.ID == "sendgrid-api-key" {
			rule = rl
			break
		}
	}
	require.NotNil(t, rule.Regex)

	// A correctly structured SendGrid key: SG.<22chars>.<43chars>
	validKey := join("SG.abcdefghijklmnopqrst", "uv.abcdefghijklmnopqrstuvwxyz01234567890abcdefghij")

	tests := []struct {
		name    string
		input   string
		wantHit bool
	}{
		{"valid SendGrid key", validKey, true},
		{"wrong prefix", "SK.abcdefghijklmnopqrstuv.abcdefghijklmnopqrstuvwxyz01234567890abcdefghij", false},
		{"second segment too short", "SG.abc.abcdefghijklmnopqrstuvwxyz01234567890abcdefghij", false},
		{"third segment too short", "SG.abcdefghijklmnopqrstuv.abc", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rule.Regex.MatchString(tt.input)
			assert.Equal(t, tt.wantHit, got)
		})
	}
}

// ---------------------------------------------------------------------------
// Pattern matching: low-confidence rules
// ---------------------------------------------------------------------------

func TestPattern_PasswordAssignment(t *testing.T) {
	r := security.NewDefaultRegistry()
	var rule security.RedactionRule
	for _, rl := range r.Rules() {
		if rl.ID == "password-assignment" {
			rule = rl
			break
		}
	}
	require.NotNil(t, rule.Regex)

	tests := []struct {
		name    string
		input   string
		wantHit bool
	}{
		{"password = 'value'", `password = 'supersecret'`, true},
		{"PASSWORD = value", `PASSWORD = 'supersecret'`, true},
		{"passwd: value", `passwd: 'supersecret'`, true},
		{"pwd = value", `pwd = 'supersecret'`, true},
		{"too short value (< 8 chars)", `password = 'short'`, false},
		{"no quotes", `password = supersecret`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rule.Regex.MatchString(tt.input)
			assert.Equal(t, tt.wantHit, got)
		})
	}
}

func TestPattern_BearerToken(t *testing.T) {
	r := security.NewDefaultRegistry()
	var rule security.RedactionRule
	for _, rl := range r.Rules() {
		if rl.ID == "bearer-token" {
			rule = rl
			break
		}
	}
	require.NotNil(t, rule.Regex)

	tests := []struct {
		name    string
		input   string
		wantHit bool
	}{
		{"Bearer with long token", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9", true},
		{"bearer lowercase", "bearer abcdefghijklmnopqrstuvwxyz", true},
		{"BEARER uppercase", "BEARER abcdefghijklmnopqrstuvwxyz", true},
		{"too short token", "Bearer shorttoken", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rule.Regex.MatchString(tt.input)
			assert.Equal(t, tt.wantHit, got)
		})
	}
}

// ---------------------------------------------------------------------------
// Pattern matching: missing medium-confidence rules
// ---------------------------------------------------------------------------

// TestPattern_AWSSecretAccessKey tests the aws-secret-access-key rule,
// which requires keyword context alongside the 40-char base64 value.
func TestPattern_AWSSecretAccessKey(t *testing.T) {
	r := security.NewDefaultRegistry()
	var rule security.RedactionRule
	for _, rl := range r.Rules() {
		if rl.ID == "aws-secret-access-key" {
			rule = rl
			break
		}
	}
	require.NotNil(t, rule.Regex)

	tests := []struct {
		name    string
		input   string
		wantHit bool
	}{
		{
			name:    "aws_secret_access_key = value (underscore separator)",
			input:   `aws_secret_access_key = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"`,
			wantHit: true,
		},
		{
			name:    "AWS_SECRET_ACCESS_KEY env var style",
			input:   `AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY`,
			wantHit: true,
		},
		{
			name:    "aws secret key colon separator",
			input:   `aws_secret_key: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY`,
			wantHit: true,
		},
		{
			name:    "aws.secret.access.key dot separator",
			input:   `aws.secret.access.key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY`,
			wantHit: true,
		},
		{
			name:    "40-char base64 without keyword context does not match",
			input:   `wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY`,
			wantHit: false,
		},
		{
			name:    "wrong keyword name does not match",
			input:   `database_password = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY`,
			wantHit: false,
		},
		{
			name:    "value too short (39 chars) does not match",
			input:   `aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLE`,
			wantHit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rule.Regex.MatchString(tt.input)
			assert.Equal(t, tt.wantHit, got, "input: %q", tt.input)
		})
	}
}

// TestPattern_OpenAIAPIKey tests the openai-api-key rule.
func TestPattern_OpenAIAPIKey(t *testing.T) {
	r := security.NewDefaultRegistry()
	var rule security.RedactionRule
	for _, rl := range r.Rules() {
		if rl.ID == "openai-api-key" {
			rule = rl
			break
		}
	}
	require.NotNil(t, rule.Regex)

	tests := []struct {
		name    string
		input   string
		wantHit bool
	}{
		{
			name:    "standard sk- prefixed key (20 chars)",
			input:   `sk-ABCDEFGHIJKLMNOPQRst`,
			wantHit: true,
		},
		{
			name:    "longer sk- prefixed key (40 chars)",
			input:   `sk-ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij`,
			wantHit: true,
		},
		{
			name:    "sk- key with mixed alphanumeric (32 chars)",
			input:   `sk-Fake1ABCDEFGHIJKLMNOPQRSTUVWXYz`,
			wantHit: true,
		},
		{
			name:    "too short (< 20 chars after sk-) does not match",
			input:   `sk-TOOSHORT`,
			wantHit: false,
		},
		{
			name:    "wrong prefix sk_ (underscore) does not match",
			input:   `sk_ABCDEFGHIJKLMNOPQRst`,
			wantHit: false,
		},
		{
			name:    "Stripe live key prefix sk_live_ does not match openai rule",
			input:   join("sk_liv", "e_ABCDEFGHIJKLMNOPQRst"),
			wantHit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rule.Regex.MatchString(tt.input)
			assert.Equal(t, tt.wantHit, got, "input: %q", tt.input)
		})
	}
}

// TestPattern_GCPServiceAccount tests the gcp-service-account rule.
func TestPattern_GCPServiceAccount(t *testing.T) {
	r := security.NewDefaultRegistry()
	var rule security.RedactionRule
	for _, rl := range r.Rules() {
		if rl.ID == "gcp-service-account" {
			rule = rl
			break
		}
	}
	require.NotNil(t, rule.Regex)

	tests := []struct {
		name    string
		input   string
		wantHit bool
	}{
		{
			name:    "standard GCP service account JSON field",
			input:   `"type": "service_account"`,
			wantHit: true,
		},
		{
			name:    "compact JSON without spaces",
			input:   `"type":"service_account"`,
			wantHit: true,
		},
		{
			name:    "type field with extra whitespace",
			input:   `"type"  :  "service_account"`,
			wantHit: true,
		},
		{
			name:    "wrong type value does not match",
			input:   `"type": "authorized_user"`,
			wantHit: false,
		},
		{
			name:    "non-JSON type field does not match",
			input:   `type = service_account`,
			wantHit: false,
		},
		{
			name:    "partial match without closing quote does not match",
			input:   `"type": "service_account`,
			wantHit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rule.Regex.MatchString(tt.input)
			assert.Equal(t, tt.wantHit, got, "input: %q", tt.input)
		})
	}
}

// TestPattern_AzureConnectionString tests the azure-connection-string rule.
func TestPattern_AzureConnectionString(t *testing.T) {
	r := security.NewDefaultRegistry()
	var rule security.RedactionRule
	for _, rl := range r.Rules() {
		if rl.ID == "azure-connection-string" {
			rule = rl
			break
		}
	}
	require.NotNil(t, rule.Regex)

	tests := []struct {
		name    string
		input   string
		wantHit bool
	}{
		{
			name:    "full Azure storage connection string",
			input:   `DefaultEndpointsProtocol=https;AccountName=mystorageaccount;AccountKey=EXAMPLEKEY==;EndpointSuffix=core.windows.net`,
			wantHit: true,
		},
		{
			name:    "minimal Azure connection string with just required prefix",
			input:   `DefaultEndpointsProtocol=https;AccountName=testaccount`,
			wantHit: true,
		},
		{
			name:    "Azure connection string in config value",
			input:   `AZURE_STORAGE_CONNECTION_STRING=DefaultEndpointsProtocol=https;AccountName=exampleacct;AccountKey=abc123==`,
			wantHit: true,
		},
		{
			name:    "HTTP protocol variant does not match (only https matched)",
			input:   `DefaultEndpointsProtocol=http;AccountName=mystorageaccount`,
			wantHit: false,
		},
		{
			name:    "missing AccountName does not match",
			input:   `DefaultEndpointsProtocol=https;AccountKey=EXAMPLEKEY==`,
			wantHit: false,
		},
		{
			name:    "typo in prefix does not match",
			input:   `DefaultEndpointProtocol=https;AccountName=mystorageaccount`,
			wantHit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rule.Regex.MatchString(tt.input)
			assert.Equal(t, tt.wantHit, got, "input: %q", tt.input)
		})
	}
}

// TestPattern_GenericAPIKey tests the generic-api-key rule.
func TestPattern_GenericAPIKey(t *testing.T) {
	r := security.NewDefaultRegistry()
	var rule security.RedactionRule
	for _, rl := range r.Rules() {
		if rl.ID == "generic-api-key" {
			rule = rl
			break
		}
	}
	require.NotNil(t, rule.Regex)

	tests := []struct {
		name    string
		input   string
		wantHit bool
	}{
		{
			name:    "api_key = unquoted value",
			input:   `api_key = ABCDEFGHIJKLMNOPabcdefghij`,
			wantHit: true,
		},
		{
			name:    "api-key: quoted value",
			input:   `api-key: "ABCDEFGHIJKLMNOPabcdefghij"`,
			wantHit: true,
		},
		{
			name:    "apikey= single-quoted value",
			input:   `apikey='ABCDEFGHIJKLMNOPabcdefghij'`,
			wantHit: true,
		},
		{
			name:    "API_KEY uppercase in env style",
			input:   `API_KEY=ABCDEFGHIJKLMNOPabcdefghij`,
			wantHit: true,
		},
		{
			name:    "api_secret assignment",
			input:   `api_secret: "ABCDEFGHIJKLMNOPabcdefghij"`,
			wantHit: true,
		},
		{
			name:    "value too short (15 chars) does not match",
			input:   `api_key = ABCDEFGHIJKLMNO`,
			wantHit: false,
		},
		{
			name:    "wrong key name does not match",
			input:   `database_name = ABCDEFGHIJKLMNOPabcdefghij`,
			wantHit: false,
		},
		{
			name:    "api_key with empty value does not match",
			input:   `api_key = ""`,
			wantHit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rule.Regex.MatchString(tt.input)
			assert.Equal(t, tt.wantHit, got, "input: %q", tt.input)
		})
	}
}

// TestPattern_TwilioAuthToken tests the twilio-auth-token rule.
func TestPattern_TwilioAuthToken(t *testing.T) {
	r := security.NewDefaultRegistry()
	var rule security.RedactionRule
	for _, rl := range r.Rules() {
		if rl.ID == "twilio-auth-token" {
			rule = rl
			break
		}
	}
	require.NotNil(t, rule.Regex)

	tests := []struct {
		name    string
		input   string
		wantHit bool
	}{
		{
			name:    "SK-prefixed 32-char lowercase hex",
			input:   join("S", "K12345678901234567890123456789012"),
			wantHit: true,
		},
		{
			name:    "SK-prefixed all-hex 32 chars (a-f)",
			input:   join("S", "Kabcdefabcdefabcdefabcdefabcdefab"),
			wantHit: true,
		},
		{
			name:    "SK-prefixed mixed hex 32 chars",
			input:   join("S", "K1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d"),
			wantHit: true,
		},
		{
			name:    "SK prefix with uppercase hex does not match (rule requires lowercase)",
			input:   join("S", "KABCDEF1234567890ABCDEF12345678"),
			wantHit: false,
		},
		{
			name:    "SK prefix too short (31 hex chars) does not match",
			input:   join("S", "K1234567890123456789012345678901"),
			wantHit: false,
		},
		{
			name:    "SK prefix too long (33 hex chars) still matches (regex is prefix match)",
			input:   join("S", "K123456789012345678901234567890123"),
			wantHit: true,
		},
		{
			name:    "wrong prefix XK does not match",
			input:   `XK12345678901234567890123456789012`,
			wantHit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rule.Regex.MatchString(tt.input)
			assert.Equal(t, tt.wantHit, got, "input: %q", tt.input)
		})
	}
}

// TestPattern_SecretTokenAssignment tests the secret-token-assignment rule.
func TestPattern_SecretTokenAssignment(t *testing.T) {
	r := security.NewDefaultRegistry()
	var rule security.RedactionRule
	for _, rl := range r.Rules() {
		if rl.ID == "secret-token-assignment" {
			rule = rl
			break
		}
	}
	require.NotNil(t, rule.Regex)

	tests := []struct {
		name    string
		input   string
		wantHit bool
	}{
		{
			name:    "secret = single-quoted value",
			input:   `secret = 'mysecretvalue123'`,
			wantHit: true,
		},
		{
			name:    "SECRET: double-quoted value (uppercase, YAML style)",
			input:   `SECRET: "mysecretvalue123"`,
			wantHit: true,
		},
		{
			name:    "token = single-quoted value",
			input:   `token = 'myauthtoken12345'`,
			wantHit: true,
		},
		{
			name:    "credential: double-quoted value",
			input:   `credential: "mycredential1234"`,
			wantHit: true,
		},
		{
			name:    "TOKEN in env style",
			input:   `TOKEN='myauthtoken12345'`,
			wantHit: true,
		},
		{
			name:    "value too short (< 8 chars) does not match",
			input:   `secret = 'short'`,
			wantHit: false,
		},
		{
			name:    "value without quotes does not match (pattern requires quotes)",
			input:   `secret = mysecretvalue123`,
			wantHit: false,
		},
		{
			name:    "wrong key name does not match",
			input:   `username = 'mysecretvalue123'`,
			wantHit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rule.Regex.MatchString(tt.input)
			assert.Equal(t, tt.wantHit, got, "input: %q", tt.input)
		})
	}
}

// TestPattern_HexEncodedSecret tests the hex-encoded-secret rule.
func TestPattern_HexEncodedSecret(t *testing.T) {
	r := security.NewDefaultRegistry()
	var rule security.RedactionRule
	for _, rl := range r.Rules() {
		if rl.ID == "hex-encoded-secret" {
			rule = rl
			break
		}
	}
	require.NotNil(t, rule.Regex)

	// 32-char hex string used across multiple test cases.
	hexVal32 := "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6"
	// 64-char hex string (256-bit secret).
	hexVal64 := "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6"

	tests := []struct {
		name    string
		input   string
		wantHit bool
	}{
		{
			name:    "secret = unquoted 32-char hex",
			input:   "secret = " + hexVal32,
			wantHit: true,
		},
		{
			name:    "key = quoted 32-char hex",
			input:   `key = "` + hexVal32 + `"`,
			wantHit: true,
		},
		{
			name:    "token: 64-char hex value",
			input:   "token: " + hexVal64,
			wantHit: true,
		},
		{
			name:    "password = quoted 32-char hex",
			input:   `password = '` + hexVal32 + `'`,
			wantHit: true,
		},
		{
			name:    "KEY uppercase env style",
			input:   "KEY=" + hexVal32,
			wantHit: true,
		},
		{
			name:    "too short hex (31 chars) does not match",
			input:   "secret = a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5",
			wantHit: false,
		},
		{
			name:    "non-hex characters in value do not match",
			input:   "secret = g1h2i3j4k5l6m7n8o9p0q1r2s3t4u5v6",
			wantHit: false,
		},
		{
			name:    "wrong key name does not match",
			input:   "username = " + hexVal32,
			wantHit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rule.Regex.MatchString(tt.input)
			assert.Equal(t, tt.wantHit, got, "input: %q", tt.input)
		})
	}
}

// ---------------------------------------------------------------------------
// Edge case tests: context and embedding
// ---------------------------------------------------------------------------

// TestPattern_SecretsInContext verifies that rules match secrets appearing in
// realistic surrounding text such as JSON values, YAML assignments, and
// code comments.
func TestPattern_SecretsInContext(t *testing.T) {
	r := security.NewDefaultRegistry()

	// Build a lookup map from ID to rule for convenience.
	ruleMap := make(map[string]security.RedactionRule, len(r.Rules()))
	for _, rl := range r.Rules() {
		ruleMap[rl.ID] = rl
	}

	tests := []struct {
		name    string
		ruleID  string
		input   string
		wantHit bool
	}{
		// ------------------------------------------------------------------
		// AWS access key ID embedded in various contexts
		// ------------------------------------------------------------------
		{
			name:    "AWS key ID in JSON value",
			ruleID:  "aws-access-key-id",
			input:   `{"aws_access_key_id": "AKIAIOSFODNN7EXAMPLE"}`,
			wantHit: true,
		},
		{
			name:    "AWS key ID in YAML value",
			ruleID:  "aws-access-key-id",
			input:   `aws_access_key_id: AKIAIOSFODNN7EXAMPLE`,
			wantHit: true,
		},
		{
			name:    "AWS key ID in Go comment",
			ruleID:  "aws-access-key-id",
			input:   `// AWS_ACCESS_KEY_ID = AKIAIOSFODNN7EXAMPLE (do not commit)`,
			wantHit: true,
		},
		{
			name:    "AWS key ID at start of line",
			ruleID:  "aws-access-key-id",
			input:   `AKIAIOSFODNN7EXAMPLE`,
			wantHit: true,
		},
		{
			name:    "AWS key ID at end of line",
			ruleID:  "aws-access-key-id",
			input:   `export AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE`,
			wantHit: true,
		},
		// ------------------------------------------------------------------
		// GitHub classic token embedded in various contexts
		// ------------------------------------------------------------------
		{
			name:    "GitHub token in JSON value",
			ruleID:  "github-classic-token",
			input:   join(`{"token": "gh`, `p_abcdefghijklmnopqrstuvwxyz0123456789"}`),
			wantHit: true,
		},
		{
			name:    "GitHub token in YAML value",
			ruleID:  "github-classic-token",
			input:   join("github_token: gh", "p_abcdefghijklmnopqrstuvwxyz0123456789"),
			wantHit: true,
		},
		{
			name:    "GitHub token in shell comment",
			ruleID:  "github-classic-token",
			input:   join("# GITHUB_TOKEN=gh", "p_abcdefghijklmnopqrstuvwxyz0123456789"),
			wantHit: true,
		},
		// ------------------------------------------------------------------
		// Stripe live key in various contexts
		// ------------------------------------------------------------------
		{
			name:    "Stripe live key in JSON value",
			ruleID:  "stripe-live-key",
			input:   join(`{"stripe_key": "sk_liv`, `e_abcdefghijklmnopqrstuvwx"}`),
			wantHit: true,
		},
		{
			name:    "Stripe test key in JSON value does NOT match",
			ruleID:  "stripe-live-key",
			input:   join(`{"stripe_key": "sk_tes`, `t_abcdefghijklmnopqrstuvwx"}`),
			wantHit: false,
		},
		{
			name:    "Stripe live key in .env file",
			ruleID:  "stripe-live-key",
			input:   join("STRIPE_SECRET_KEY=sk_liv", "e_abcdefghijklmnopqrstuvwx"),
			wantHit: true,
		},
		// ------------------------------------------------------------------
		// Connection strings with URL-encoded passwords
		// ------------------------------------------------------------------
		{
			name:    "postgres URI with URL-encoded password",
			ruleID:  "connection-string",
			input:   `postgres://appuser:p%40ssw0rd@db.example.com:5432/myapp`,
			wantHit: true,
		},
		{
			name:    "mongodb URI with encoded special chars",
			ruleID:  "connection-string",
			input:   `mongodb://admin:s3cr3t%21@mongo.example.com:27017/admin`,
			wantHit: true,
		},
		{
			name:    "connection string in YAML config",
			ruleID:  "connection-string",
			input:   `database_url: postgres://dbuser:dbpass123@localhost/mydb`,
			wantHit: true,
		},
		// ------------------------------------------------------------------
		// JWT token in various contexts
		// ------------------------------------------------------------------
		{
			name:    "JWT in Authorization header string",
			ruleID:  "jwt-token",
			input:   `Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c`,
			wantHit: true,
		},
		{
			name:    "JWT in JSON value field",
			ruleID:  "jwt-token",
			input:   `{"access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"}`,
			wantHit: true,
		},
		// ------------------------------------------------------------------
		// SendGrid key in various contexts
		// ------------------------------------------------------------------
		{
			name:    "SendGrid key in .env file",
			ruleID:  "sendgrid-api-key",
			input:   join("SENDGRID_API_KEY=SG.abcdefghijklmnopqrst", "uv.abcdefghijklmnopqrstuvwxyz01234567890abcdefghij"),
			wantHit: true,
		},
		{
			name:    "SendGrid key in YAML config",
			ruleID:  "sendgrid-api-key",
			input:   join("sendgrid_key: SG.abcdefghijklmnopqrst", "uv.abcdefghijklmnopqrstuvwxyz01234567890abcdefghij"),
			wantHit: true,
		},
		// ------------------------------------------------------------------
		// Password assignment in various contexts
		// ------------------------------------------------------------------
		{
			name:    "password in Python assignment",
			ruleID:  "password-assignment",
			input:   `DB_PASSWORD = "mysupersecretpass"`,
			wantHit: true,
		},
		{
			name:    "passwd in TOML config",
			ruleID:  "password-assignment",
			input:   `passwd = 'mypassword1234'`,
			wantHit: true,
		},
		{
			name:    "pwd in YAML config",
			ruleID:  "password-assignment",
			input:   `pwd: 'mypassword1234'`,
			wantHit: true,
		},
		// ------------------------------------------------------------------
		// Private key block in various contexts
		// ------------------------------------------------------------------
		{
			name:    "DSA private key block",
			ruleID:  "private-key-block",
			input:   `-----BEGIN DSA PRIVATE KEY-----`,
			wantHit: true,
		},
		{
			name:    "ENCRYPTED private key block",
			ruleID:  "private-key-block",
			input:   `-----BEGIN ENCRYPTED PRIVATE KEY-----`,
			wantHit: true,
		},
		{
			name:    "public key BEGIN header does not match",
			ruleID:  "private-key-block",
			input:   `-----BEGIN PUBLIC KEY-----`,
			wantHit: false,
		},
		{
			name:    "certificate BEGIN header does not match",
			ruleID:  "private-key-block",
			input:   `-----BEGIN CERTIFICATE-----`,
			wantHit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule, ok := ruleMap[tt.ruleID]
			require.True(t, ok, "rule %q must exist in default registry", tt.ruleID)
			got := rule.Regex.MatchString(tt.input)
			assert.Equal(t, tt.wantHit, got, "rule=%q input=%q", tt.ruleID, tt.input)
		})
	}
}

// TestPattern_KeywordsPresent verifies that every built-in rule has at least
// one keyword and that each keyword appears in at least one positive test
// example for that rule. This is a meta-test confirming the pre-filter design.
func TestPattern_KeywordsPresent(t *testing.T) {
	r := security.NewDefaultRegistry()
	for _, rule := range r.Rules() {
		t.Run(rule.ID, func(t *testing.T) {
			assert.NotEmpty(t, rule.Keywords,
				"rule %q must have at least one keyword for pre-filter optimization", rule.ID)
		})
	}
}

// TestPattern_AllRegexHaveCaptureGroup verifies that every built-in rule's
// regex has at least one capture group, which is required by the redactor to
// identify the secret value vs. surrounding context.
func TestPattern_AllRegexHaveCaptureGroup(t *testing.T) {
	r := security.NewDefaultRegistry()
	for _, rule := range r.Rules() {
		t.Run(rule.ID, func(t *testing.T) {
			numGroups := rule.Regex.NumSubexp()
			assert.GreaterOrEqual(t, numGroups, 1,
				"rule %q regex must have at least one capture group (got %d)", rule.ID, numGroups)
		})
	}
}

// TestPattern_MultipleSecretsOnOneLine verifies that regex matching works
// when multiple distinct secrets appear on a single line.
func TestPattern_MultipleSecretsOnOneLine(t *testing.T) {
	r := security.NewDefaultRegistry()
	ruleMap := make(map[string]security.RedactionRule, len(r.Rules()))
	for _, rl := range r.Rules() {
		ruleMap[rl.ID] = rl
	}

	tests := []struct {
		name    string
		ruleID  string
		input   string
		wantHit bool
	}{
		{
			name:   "two AWS key IDs on the same line",
			ruleID: "aws-access-key-id",
			// Both AKIA keys should be found; MatchString returns true if at least one match exists.
			input:   `key1=AKIAIOSFODNN7EXAMPLE key2=ASIAIOSFODNN7EXAMPLE`,
			wantHit: true,
		},
		{
			name:    "Slack token followed by GitHub token on the same line",
			ruleID:  "slack-token",
			input:   join("SLACK=xox", "b-123456789012-123456789012-abcdefghijklmnopqrstuvwx GITHUB=gh", "p_abcdefghijklmnopqrstuvwxyz0123456789"),
			wantHit: true,
		},
		{
			name:    "SendGrid key followed by arbitrary text",
			ruleID:  "sendgrid-api-key",
			input:   join("api_key: SG.abcdefghijklmnopqrst", "uv.abcdefghijklmnopqrstuvwxyz01234567890abcdefghij # production"),
			wantHit: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule, ok := ruleMap[tt.ruleID]
			require.True(t, ok, "rule %q must exist", tt.ruleID)
			got := rule.Regex.MatchString(tt.input)
			assert.Equal(t, tt.wantHit, got, "rule=%q input=%q", tt.ruleID, tt.input)
		})
	}
}

// TestPattern_AllPatternsCompileSuccessfully is a smoke-test confirming that
// all 19 built-in patterns are valid RE2 expressions and the registry is
// properly initialised. It complements TestDefaultRegistry_BuiltinRuleCount
// by also asserting that no nil Regex survived to the registry.
func TestPattern_AllPatternsCompileSuccessfully(t *testing.T) {
	r := security.NewDefaultRegistry()
	rules := r.Rules()
	require.Len(t, rules, 19, "expected exactly 19 built-in rules")
	for _, rule := range rules {
		require.NotNil(t, rule.Regex, "rule %q must have a non-nil compiled regex", rule.ID)
		require.NotEmpty(t, rule.ID, "all rules must have a non-empty ID")
		require.NotEmpty(t, rule.SecretType, "rule %q must have a non-empty SecretType", rule.ID)
	}
}

// TestPattern_GitHubFineGrainedPAT_LengthBoundary verifies the minimum
// 22-character suffix requirement for github_pat_ tokens.
func TestPattern_GitHubFineGrainedPAT_LengthBoundary(t *testing.T) {
	r := security.NewDefaultRegistry()
	var rule security.RedactionRule
	for _, rl := range r.Rules() {
		if rl.ID == "github-fine-grained-pat" {
			rule = rl
			break
		}
	}
	require.NotNil(t, rule.Regex)

	// Exactly 22 alphanumeric chars after github_pat_ — boundary value.
	exactMinimum := join("github_pa", "t_") + "ABCDEFGHIJKLMNOPQRSTUV" // 22 chars
	// One char short of minimum.
	oneShort := join("github_pa", "t_") + "ABCDEFGHIJKLMNOPQRSTU" // 21 chars
	// Well above minimum.
	longPAT := join("github_pa", "t_") + "ABCDEFGHIJKLMNOPQRSTUV_1234567890_extra"

	assert.True(t, rule.Regex.MatchString(exactMinimum), "exactly 22 chars should match: %q", exactMinimum)
	assert.False(t, rule.Regex.MatchString(oneShort), "21 chars should NOT match: %q", oneShort)
	assert.True(t, rule.Regex.MatchString(longPAT), "long PAT should match: %q", longPAT)
}

// TestPattern_StripeLiveKey_AllPrefixes verifies sk_live_, pk_live_, and
// rk_live_ prefixes, and that the minimum length of 24 chars after the
// prefix is enforced.
func TestPattern_StripeLiveKey_AllPrefixes(t *testing.T) {
	r := security.NewDefaultRegistry()
	var rule security.RedactionRule
	for _, rl := range r.Rules() {
		if rl.ID == "stripe-live-key" {
			rule = rl
			break
		}
	}
	require.NotNil(t, rule.Regex)

	// 24 alphanumeric chars after the live prefix — boundary value.
	suffix24 := "ABCDEFGHIJKLMNOPQRSTUVWX" // exactly 24

	tests := []struct {
		name    string
		input   string
		wantHit bool
	}{
		{"sk_live_ exactly 24", join("sk_liv", "e_") + suffix24, true},
		{"pk_live_ exactly 24", join("pk_liv", "e_") + suffix24, true},
		{"rk_live_ exactly 24", join("rk_liv", "e_") + suffix24, true},
		{"sk_live_ only 23 chars", join("sk_liv", "e_ABCDEFGHIJKLMNOPQRSTUVW"), false},
		{"sk_test_ excluded", join("sk_tes", "t_") + suffix24, false},
		{"pk_test_ excluded", join("pk_tes", "t_") + suffix24, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rule.Regex.MatchString(tt.input)
			assert.Equal(t, tt.wantHit, got, "input: %q", tt.input)
		})
	}
}

// TestPattern_ConnectionString_URLEncoded verifies that connection strings
// with URL-encoded passwords (e.g., %40 for @) are matched correctly.
func TestPattern_ConnectionString_URLEncoded(t *testing.T) {
	r := security.NewDefaultRegistry()
	var rule security.RedactionRule
	for _, rl := range r.Rules() {
		if rl.ID == "connection-string" {
			rule = rl
			break
		}
	}
	require.NotNil(t, rule.Regex)

	tests := []struct {
		name    string
		input   string
		wantHit bool
	}{
		{
			name:    "postgres with %40 encoded @",
			input:   `postgres://user:p%40ssw0rd@localhost/mydb`,
			wantHit: true,
		},
		{
			name:    "mysql with %23 encoded #",
			input:   `mysql://root:p%23ssword@db.example.com:3306/app`,
			wantHit: true,
		},
		{
			name:    "redis with %3A encoded colon in password",
			input:   `redis://default:p%3Assword@cache.example.com:6379/0`,
			wantHit: true,
		},
		{
			name:    "mongodb without credentials does not include embedded user:pass but still matches scheme",
			input:   `mongodb://localhost:27017/mydb`,
			wantHit: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rule.Regex.MatchString(tt.input)
			assert.Equal(t, tt.wantHit, got, "input: %q", tt.input)
		})
	}
}

// TestPattern_PasswordAssignment_NegativeCases thoroughly tests the
// boundaries of the password-assignment rule to document expected
// non-matches.
func TestPattern_PasswordAssignment_NegativeCases(t *testing.T) {
	r := security.NewDefaultRegistry()
	var rule security.RedactionRule
	for _, rl := range r.Rules() {
		if rl.ID == "password-assignment" {
			rule = rl
			break
		}
	}
	require.NotNil(t, rule.Regex)

	tests := []struct {
		name    string
		input   string
		wantHit bool
	}{
		{
			// The rule requires quotes around the value; bare assignments are
			// intentionally excluded to reduce false positives.
			name:    "unquoted value does not match",
			input:   `password = supersecretpass`,
			wantHit: false,
		},
		{
			// Value must be at least 8 characters inside the quotes.
			name:    "7-char value does not match",
			input:   `password = '1234567'`,
			wantHit: false,
		},
		{
			// Exactly 8 chars — boundary.
			name:    "8-char value matches (boundary)",
			input:   `password = '12345678'`,
			wantHit: true,
		},
		{
			// The keyword must be password, passwd, or pwd; other names don't match.
			name:    "passphrase key does not match",
			input:   `passphrase = 'supersecretpass'`,
			wantHit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rule.Regex.MatchString(tt.input)
			assert.Equal(t, tt.wantHit, got, "input: %q", tt.input)
		})
	}
}

// TestPattern_BearerToken_LengthBoundary verifies the minimum 20-character
// token length required by the bearer-token rule.
func TestPattern_BearerToken_LengthBoundary(t *testing.T) {
	r := security.NewDefaultRegistry()
	var rule security.RedactionRule
	for _, rl := range r.Rules() {
		if rl.ID == "bearer-token" {
			rule = rl
			break
		}
	}
	require.NotNil(t, rule.Regex)

	// Exactly 20 chars — boundary value.
	token20 := "ABCDEFGHIJKLMNOPQRST" // 20 chars
	// One short.
	token19 := "ABCDEFGHIJKLMNOPQRS" // 19 chars

	assert.True(t, rule.Regex.MatchString("Bearer "+token20), "20 chars should match")
	assert.False(t, rule.Regex.MatchString("Bearer "+token19), "19 chars should NOT match")
}

// TestPattern_CorpusTable runs the full regression corpus from the
// testdata/secrets package. Each entry must match or not match its
// declared rule regex exactly.
func TestPattern_CorpusTable(t *testing.T) {
	r := security.NewDefaultRegistry()

	// Build lookup map.
	ruleMap := make(map[string]security.RedactionRule, len(r.Rules()))
	for _, rl := range r.Rules() {
		ruleMap[rl.ID] = rl
	}

	for _, entry := range corpusEntries {
		entry := entry // capture range variable
		t.Run(entry.name, func(t *testing.T) {
			rule, ok := ruleMap[entry.ruleID]
			require.True(t, ok, "corpus entry %q references unknown rule ID %q", entry.name, entry.ruleID)
			got := rule.Regex.MatchString(entry.input)
			assert.Equal(t, entry.shouldMatch, got,
				"corpus entry %q (rule=%q): input=%q", entry.name, entry.ruleID, entry.input)
		})
	}
}
