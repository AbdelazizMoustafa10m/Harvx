package security

import "regexp"

// builtinRules holds all pre-compiled RedactionRules registered by
// registerBuiltinPatterns. Compiled once at package init time; treated as
// read-only thereafter.
//
// Rule ordering within a confidence tier is stable: rules registered first
// take precedence when the redactor deduplicates overlapping matches.
var builtinRules = []RedactionRule{
	// -----------------------------------------------------------------------
	// High confidence -- structural patterns with near-zero false-positive rates.
	// -----------------------------------------------------------------------

	// AWS Access Key IDs are exactly 20 characters.  4-character prefixes
	// (AKIA, ASIA, ABIA, ACCA) are followed by 16 uppercase alphanumeric
	// characters; the 3-character prefix A3T is followed by 17 characters.
	// The alternation captures both forms in a single rule.
	{
		ID:          "aws-access-key-id",
		Description: "AWS Access Key ID (AKIA/ASIA/ABIA/ACCA/A3T prefix, 20 chars total)",
		Regex:       regexp.MustCompile(`((?:AKIA|ASIA|ABIA|ACCA)[A-Z0-9]{16}|A3T[A-Z0-9]{17})`),
		Keywords:    []string{"AKIA", "ASIA", "ABIA", "ACCA", "A3T"},
		SecretType:  "aws_access_key_id",
		Confidence:  ConfidenceHigh,
	},

	// AWS Secret Access Keys are 40-character base64 strings.  The pattern
	// is intentionally broad; the keyword list narrows it to lines that
	// mention AWS credential context, keeping false positives low.
	{
		ID:               "aws-secret-access-key",
		Description:      "AWS Secret Access Key (40-char base64 near AWS context keyword)",
		Regex:            regexp.MustCompile(`(?i)(?:aws[_\-. ]?secret[_\-. ]?(?:access[_\-. ]?)?key)\s*[=:]\s*['"]?([A-Za-z0-9+/]{40})['"]?`),
		Keywords:         []string{"aws_secret", "secret_access_key", "AWS_SECRET"},
		SecretType:       "aws_secret_access_key",
		Confidence:       ConfidenceHigh,
		EntropyThreshold: 4.5,
	},

	// GitHub classic personal access tokens use a well-known 4-char prefix
	// followed by exactly 36 base36 characters.
	{
		ID:          "github-classic-token",
		Description: "GitHub classic personal access token (ghp_/gho_/ghs_/ghr_ prefix)",
		Regex:       regexp.MustCompile(`(gh[pors]_[A-Za-z0-9]{36})`),
		Keywords:    []string{"ghp_", "gho_", "ghs_", "ghr_"},
		SecretType:  "github_token",
		Confidence:  ConfidenceHigh,
	},

	// GitHub fine-grained PATs introduced in 2022 use the github_pat_ prefix.
	{
		ID:          "github-fine-grained-pat",
		Description: "GitHub fine-grained personal access token (github_pat_ prefix)",
		Regex:       regexp.MustCompile(`(github_pat_[A-Za-z0-9_]{22,})`),
		Keywords:    []string{"github_pat_"},
		SecretType:  "github_token",
		Confidence:  ConfidenceHigh,
	},

	// PEM private key blocks are unambiguous -- the header line is unique
	// to actual private key material.
	{
		ID:          "private-key-block",
		Description: "PEM private key block (RSA, EC, OPENSSH, PKCS8, etc.)",
		Regex:       regexp.MustCompile(`(-----BEGIN [A-Z ]*PRIVATE KEY-----)`),
		Keywords:    []string{"BEGIN", "PRIVATE KEY"},
		SecretType:  "private_key",
		Confidence:  ConfidenceHigh,
	},

	// Stripe live-mode keys have an unmistakable prefix; test-mode keys
	// (sk_test_) are intentionally excluded since they carry no risk.
	{
		ID:          "stripe-live-key",
		Description: "Stripe live-mode API key (sk_live_/pk_live_/rk_live_ prefix)",
		Regex:       regexp.MustCompile(`((?:sk|pk|rk)_live_[A-Za-z0-9]{24,})`),
		Keywords:    []string{"sk_live_", "pk_live_", "rk_live_"},
		SecretType:  "stripe_api_key",
		Confidence:  ConfidenceHigh,
	},

	// -----------------------------------------------------------------------
	// Medium confidence -- patterns that need keyword or structural context
	// to separate real secrets from coincidental matches.
	// -----------------------------------------------------------------------

	// OpenAI API keys begin with sk- followed by 20+ alphanumeric characters.
	// The keyword list ensures we only flag lines with clear OpenAI context.
	{
		ID:               "openai-api-key",
		Description:      "OpenAI API key (sk- prefix with OpenAI context keyword)",
		Regex:            regexp.MustCompile(`(sk-[A-Za-z0-9]{20,})`),
		Keywords:         []string{"openai", "OPENAI", "sk-"},
		SecretType:       "openai_api_key",
		Confidence:       ConfidenceMedium,
		EntropyThreshold: 3.5,
	},

	// Connection strings embed credentials directly in the URI and cover all
	// major database and messaging systems supported by Harvx targets.
	// The character class [^\s'"] stops at whitespace, single-quotes, and
	// double-quotes to avoid greedily consuming surrounding text.
	{
		ID:          "connection-string",
		Description: "Database / broker connection string with embedded credentials",
		Regex:       regexp.MustCompile(`((?:postgres|postgresql|mysql|mongodb(?:\+srv)?|redis|amqps?)://[^\s'"]+)`),
		Keywords:    []string{"postgres", "mysql", "mongodb", "redis", "amqp"},
		SecretType:  "connection_string",
		Confidence:  ConfidenceMedium,
	},

	// GCP service account JSON files are identified by the "service_account"
	// type field.  The regex is intentionally narrow to avoid matching
	// unrelated JSON keys named "type".
	{
		ID:          "gcp-service-account",
		Description: `GCP service account JSON ("type": "service_account")`,
		Regex:       regexp.MustCompile(`("type"\s*:\s*"service_account")`),
		Keywords:    []string{"service_account", "private_key"},
		SecretType:  "gcp_service_account",
		Confidence:  ConfidenceMedium,
	},

	// Azure Storage connection strings have a fixed prefix that is unique to
	// Azure SDK credential blocks.
	{
		ID:          "azure-connection-string",
		Description: "Azure Storage connection string (DefaultEndpointsProtocol=https)",
		Regex:       regexp.MustCompile(`(DefaultEndpointsProtocol=https;AccountName=[^\s]+)`),
		Keywords:    []string{"DefaultEndpointsProtocol", "AccountName"},
		SecretType:  "azure_connection_string",
		Confidence:  ConfidenceMedium,
	},

	// JWT tokens have three base64url-encoded segments separated by dots.
	// The validateJWT post-match validator (validate.go) confirms the
	// segment structure before the redactor treats it as a real secret.
	{
		ID:          "jwt-token",
		Description: "JSON Web Token (three base64url segments separated by dots)",
		Regex:       regexp.MustCompile(`(eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,})`),
		Keywords:    []string{"eyJ"},
		SecretType:  "jwt_token",
		Confidence:  ConfidenceMedium,
	},

	// Generic API key assignment pattern matches common configuration formats
	// across languages (YAML, TOML, .env, JSON, Python, Ruby, etc.).
	// The keyword portion uses a non-capturing group; capture group 1 is
	// the secret value so T-036 redacts only the value, not the key name.
	{
		ID:               "generic-api-key",
		Description:      "Generic API key or secret assignment",
		Regex:            regexp.MustCompile(`(?i)(?:api[_-]?key|apikey|api[_-]?secret)\s*[=:]\s*['"]?([A-Za-z0-9_\-]{16,})['"]?`),
		Keywords:         []string{"api_key", "apikey", "api-key", "api_secret"},
		SecretType:       "generic_api_key",
		Confidence:       ConfidenceMedium,
		EntropyThreshold: 3.5,
	},

	// Slack bot, user, and workspace tokens use distinctive xox prefixes.
	{
		ID:          "slack-token",
		Description: "Slack API token (xoxb/xoxp/xoxo/xoxr/xoxs prefix)",
		Regex:       regexp.MustCompile(`(xox[bpors]-[A-Za-z0-9-]{10,})`),
		Keywords:    []string{"xox"},
		SecretType:  "slack_token",
		Confidence:  ConfidenceMedium,
	},

	// Twilio auth tokens are 32-character lowercase hex strings.  They are
	// only flagged when the line contains Twilio-related keywords to avoid
	// false positives on other 32-char hex identifiers.
	{
		ID:          "twilio-auth-token",
		Description: "Twilio authentication token (32-char hex with Twilio context)",
		Regex:       regexp.MustCompile(`(SK[a-f0-9]{32})`),
		Keywords:    []string{"SK", "twilio", "TWILIO"},
		SecretType:  "twilio_auth_token",
		Confidence:  ConfidenceMedium,
	},

	// SendGrid API keys have a fixed two-segment structure with known lengths.
	{
		ID:          "sendgrid-api-key",
		Description: "SendGrid API key (SG.<22>.<43> format)",
		Regex:       regexp.MustCompile(`(SG\.[A-Za-z0-9_-]{22}\.[A-Za-z0-9_-]{43})`),
		Keywords:    []string{"SG.", "sendgrid"},
		SecretType:  "sendgrid_api_key",
		Confidence:  ConfidenceMedium,
	},

	// -----------------------------------------------------------------------
	// Low confidence -- generic patterns with elevated false-positive rates.
	// Use only as a last-resort sweep or when combined with entropy checks.
	// -----------------------------------------------------------------------

	// Password assignments in config files and source code.
	// The keyword portion uses a non-capturing group; capture group 1 is
	// the secret value so T-036 redacts only the value, not the key name.
	{
		ID:          "password-assignment",
		Description: "Password assignment in config or source (password/passwd/pwd = '...')",
		Regex:       regexp.MustCompile(`(?i)(?:password|passwd|pwd)\s*[=:]\s*['"]([^\s'"]{8,})['"]`),
		Keywords:    []string{"password", "passwd", "pwd"},
		SecretType:  "password",
		Confidence:  ConfidenceLow,
	},

	// Secret/token/credential assignments broader than the API-key pattern.
	// The keyword portion uses a non-capturing group; capture group 1 is
	// the secret value so T-036 redacts only the value, not the key name.
	{
		ID:          "secret-token-assignment",
		Description: "Secret, token, or credential assignment",
		Regex:       regexp.MustCompile(`(?i)(?:secret|token|credential)\s*[=:]\s*['"]([^\s'"]{8,})['"]`),
		Keywords:    []string{"secret", "token", "credential"},
		SecretType:  "generic_secret",
		Confidence:  ConfidenceLow,
	},

	// Bearer token header values as seen in HTTP configuration files and tests.
	// Capture group 1 is just the token value, not the "Bearer" keyword, so
	// T-036 replaces only the credential and leaves the scheme word intact.
	{
		ID:          "bearer-token",
		Description: "HTTP Bearer token value",
		Regex:       regexp.MustCompile(`(?i)bearer\s+([A-Za-z0-9_\-.]{20,})`),
		Keywords:    []string{"bearer", "Bearer"},
		SecretType:  "bearer_token",
		Confidence:  ConfidenceLow,
	},

	// Hex-encoded 128-bit or larger secrets frequently appear in .env files
	// and generated configuration values.
	{
		ID:               "hex-encoded-secret",
		Description:      "Hex-encoded secret value (32+ hex chars with secret/key/token/password context)",
		Regex:            regexp.MustCompile(`(?i)(?:secret|key|token|password)\s*[=:]\s*['"]?([0-9a-f]{32,})['"]?`),
		Keywords:         []string{"secret", "key", "token", "password"},
		SecretType:       "hex_secret",
		Confidence:       ConfidenceLow,
		EntropyThreshold: 3.0,
	},
}

// registerBuiltinPatterns registers all built-in detection rules into r.
// It is called once by NewDefaultRegistry and is the single source of truth
// for the Gitleaks-inspired ruleset shipped with Harvx.
//
// Rules are registered in declaration order: high-confidence first, then
// medium-confidence, then low-confidence. Within each tier, rule order
// follows the definition order in builtinRules.
func registerBuiltinPatterns(r *PatternRegistry) {
	for _, rule := range builtinRules {
		r.Register(rule)
	}
}
