package config

import (
	"os"
	"strconv"
)

// Environment variable name constants for HARVX_ prefixed overrides.
const (
	// EnvProfile selects the named profile to activate.
	EnvProfile = "HARVX_PROFILE"
	// EnvMaxTokens overrides the token budget cap.
	EnvMaxTokens = "HARVX_MAX_TOKENS"
	// EnvFormat overrides the output format.
	EnvFormat = "HARVX_FORMAT"
	// EnvTokenizer overrides the token counting model.
	EnvTokenizer = "HARVX_TOKENIZER"
	// EnvOutput overrides the output file path.
	EnvOutput = "HARVX_OUTPUT"
	// EnvTarget overrides the LLM target preset.
	EnvTarget = "HARVX_TARGET"
	// EnvLogFormat overrides the log output format (not a profile field).
	EnvLogFormat = "HARVX_LOG_FORMAT"
	// EnvCompress overrides the compression flag.
	EnvCompress = "HARVX_COMPRESS"
	// EnvRedact overrides the redaction flag.
	EnvRedact = "HARVX_REDACT"
)

// buildEnvMap reads HARVX_* environment variables and returns a flat map
// suitable for use with a koanf confmap provider. Only non-empty env vars that
// parse successfully are included. Invalid numeric/boolean values are silently
// skipped so that a bad env var does not block the entire resolution pipeline.
func buildEnvMap() map[string]any {
	m := make(map[string]any)

	if v := os.Getenv(EnvFormat); v != "" {
		m["format"] = v
	}
	if v := os.Getenv(EnvMaxTokens); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			m["max_tokens"] = n
		}
	}
	if v := os.Getenv(EnvTokenizer); v != "" {
		m["tokenizer"] = v
	}
	if v := os.Getenv(EnvOutput); v != "" {
		m["output"] = v
	}
	if v := os.Getenv(EnvTarget); v != "" {
		m["target"] = v
	}
	if v := os.Getenv(EnvCompress); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			m["compression"] = b
		}
	}
	if v := os.Getenv(EnvRedact); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			m["redaction"] = b
		}
	}

	return m
}
