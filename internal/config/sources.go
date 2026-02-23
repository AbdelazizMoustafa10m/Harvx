package config

// Source identifies which configuration layer provided a value.
// Higher values indicate higher precedence.
type Source int

const (
	// SourceDefault is the built-in fallback (lowest precedence).
	SourceDefault Source = iota
	// SourceGlobal is the user's global config (~/.config/harvx/config.toml).
	SourceGlobal
	// SourceRepo is the project-local harvx.toml in the target directory.
	SourceRepo
	// SourceEnv is an HARVX_* environment variable override.
	SourceEnv
	// SourceFlag is an explicit CLI flag (highest precedence).
	SourceFlag
)

// String returns the human-readable name of the source.
func (s Source) String() string {
	switch s {
	case SourceDefault:
		return "default"
	case SourceGlobal:
		return "global"
	case SourceRepo:
		return "repo"
	case SourceEnv:
		return "env"
	case SourceFlag:
		return "flag"
	default:
		return "unknown"
	}
}

// SourceMap tracks where each Profile field value originated.
// Keys are flat field names like "format", "max_tokens", "relevance.tier_0".
type SourceMap map[string]Source
