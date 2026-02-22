package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	koanf "github.com/knadh/koanf/v2"
	"github.com/knadh/koanf/providers/confmap"
)

// ResolveOptions configures the multi-source configuration resolution.
type ResolveOptions struct {
	// ProfileName selects a named profile from loaded configs.
	// If empty, the HARVX_PROFILE env var is checked, then "default" is used.
	ProfileName string

	// ProfileFile is a standalone profile TOML file path (--profile-file flag).
	// When set, the repo config (harvx.toml) is not loaded.
	ProfileFile string

	// TargetDir is the directory to search for harvx.toml.
	// Defaults to "." if empty.
	TargetDir string

	// GlobalConfigPath overrides the default ~/.config/harvx/config.toml.
	// Useful for testing.
	GlobalConfigPath string

	// CLIFlags holds explicit CLI flag overrides (highest precedence).
	// Keys are flat Profile field names: "format", "max_tokens", "output", etc.
	CLIFlags map[string]any
}

// ResolvedConfig is the result of multi-source configuration resolution.
type ResolvedConfig struct {
	// Profile is the final merged profile ready for use by the pipeline.
	Profile *Profile

	// Sources tracks which layer each field value came from.
	Sources SourceMap

	// ProfileName is the name of the resolved profile.
	ProfileName string
}

// Resolve runs the 5-layer configuration resolution pipeline:
//  1. Built-in defaults
//  2. Global config (~/.config/harvx/config.toml)
//  3. Repository config (harvx.toml in TargetDir) OR standalone profile file
//  4. Environment variables (HARVX_* prefix)
//  5. CLI flags (highest precedence)
//
// Missing config files are silently ignored. Invalid files return errors.
// Named profiles not found in any loaded config return an error listing
// available profiles.
func Resolve(opts ResolveOptions) (*ResolvedConfig, error) {
	// Determine profile name: explicit option → HARVX_PROFILE env → "default".
	profileName := opts.ProfileName
	if profileName == "" {
		if v := os.Getenv(EnvProfile); v != "" {
			profileName = v
		} else {
			profileName = "default"
		}
	}

	slog.Debug("resolving config",
		"profile", profileName,
		"targetDir", opts.TargetDir,
		"profileFile", opts.ProfileFile,
	)

	k := koanf.New(".")
	sources := make(SourceMap)

	// ── Layer 1: built-in defaults ─────────────────────────────────────────
	defaultProfile := DefaultProfile()
	if err := loadLayer(k, profileToFlatMap(defaultProfile), sources, SourceDefault); err != nil {
		return nil, fmt.Errorf("loading defaults: %w", err)
	}

	// Track whether the named profile was found in at least one file layer.
	profileFound := false

	// ── Layer 2: global config ─────────────────────────────────────────────
	globalPath := opts.GlobalConfigPath
	if globalPath == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			globalPath = filepath.Join(home, ".config", "harvx", "config.toml")
		}
	}

	if globalPath != "" {
		found, err := loadFileLayer(k, globalPath, profileName, sources, SourceGlobal)
		if err != nil {
			return nil, err
		}
		if found {
			profileFound = true
		}
	}

	// ── Layer 3: repo config OR standalone profile file ────────────────────
	if opts.ProfileFile != "" {
		found, err := loadFileLayer(k, opts.ProfileFile, profileName, sources, SourceRepo)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, fmt.Errorf("profile %q not found in profile file %s", profileName, opts.ProfileFile)
		}
		profileFound = true
	} else {
		targetDir := opts.TargetDir
		if targetDir == "" {
			targetDir = "."
		}
		repoConfigPath := filepath.Join(targetDir, "harvx.toml")
		found, err := loadFileLayer(k, repoConfigPath, profileName, sources, SourceRepo)
		if err != nil {
			return nil, err
		}
		if found {
			profileFound = true
		}
	}

	// If a non-default profile was requested but not found, return a helpful error.
	if profileName != "default" && !profileFound {
		return nil, fmt.Errorf("profile %q not found in any config file", profileName)
	}

	// ── Layer 4: environment variables ────────────────────────────────────
	envMap := buildEnvMap()
	if len(envMap) > 0 {
		if err := loadLayer(k, envMap, sources, SourceEnv); err != nil {
			return nil, fmt.Errorf("loading env vars: %w", err)
		}
	}

	// ── Apply target preset (after env, before CLI flags) ─────────────────
	// CLI flags loaded afterward can still override preset values.
	if target := k.String("target"); target != "" {
		presetProfile := flatMapToProfile(k)
		if err := ApplyTargetPreset(presetProfile, target); err != nil {
			return nil, fmt.Errorf("applying target preset: %w", err)
		}
		// Re-load from preset-applied profile; only changed keys get re-attributed.
		if err := loadLayer(k, profileToFlatMap(presetProfile), sources, SourceEnv); err != nil {
			return nil, fmt.Errorf("loading target preset: %w", err)
		}
	}

	// ── Layer 5: CLI flags ─────────────────────────────────────────────────
	if len(opts.CLIFlags) > 0 {
		if err := loadLayer(k, opts.CLIFlags, sources, SourceFlag); err != nil {
			return nil, fmt.Errorf("loading CLI flags: %w", err)
		}
	}

	finalProfile := flatMapToProfile(k)

	slog.Debug("config resolved",
		"profile", profileName,
		"format", finalProfile.Format,
		"maxTokens", finalProfile.MaxTokens,
		"target", finalProfile.Target,
	)

	return &ResolvedConfig{
		Profile:     finalProfile,
		Sources:     sources,
		ProfileName: profileName,
	}, nil
}

// loadFileLayer loads a named profile from a TOML config file, merges its
// explicitly-set fields into k, and records source attribution. Missing files
// and missing profiles are silently skipped (returns false, nil). Parse errors
// and I/O errors are returned.
func loadFileLayer(k *koanf.Koanf, path, profileName string, sources SourceMap, src Source) (bool, error) {
	flat, err := extractProfileFlat(path, profileName)
	if err != nil {
		return false, fmt.Errorf("loading config %s: %w", path, err)
	}
	if flat == nil {
		return false, nil
	}

	slog.Debug("loading profile from config",
		"profile", profileName,
		"path", path,
		"source", src.String(),
	)

	if err := loadLayer(k, flat, sources, src); err != nil {
		return false, err
	}
	return true, nil
}

// extractProfileFlat parses a TOML config file into a raw Go map and returns a
// flat koanf-compatible map containing only the fields that are explicitly
// present in the TOML for the given profile. Returns nil if the file does not
// exist or the profile is not found in the file.
func extractProfileFlat(path, profileName string) (map[string]any, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			slog.Debug("config file not found, skipping", "path", path)
			return nil, nil
		}
		return nil, fmt.Errorf("stat %s: %w", path, err)
	}

	// Parse into a raw map so we only see keys present in the TOML file.
	var raw map[string]interface{}
	if _, err := toml.DecodeFile(path, &raw); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	profilesRaw, ok := raw["profile"].(map[string]interface{})
	if !ok {
		available := listConfigProfileNames(path)
		slog.Debug("no [profile] section in config",
			"path", path,
			"available", strings.Join(available, ", "),
		)
		return nil, nil
	}

	profileRaw, ok := profilesRaw[profileName].(map[string]interface{})
	if !ok {
		available := make([]string, 0, len(profilesRaw))
		for name := range profilesRaw {
			available = append(available, name)
		}
		sort.Strings(available)
		slog.Debug("profile not found in config",
			"profile", profileName,
			"path", path,
			"available", strings.Join(available, ", "),
		)
		return nil, nil
	}

	return flattenProfileRaw(profileRaw), nil
}

// listConfigProfileNames returns profile names from a TOML file, for debug
// logging. Returns nil on any error.
func listConfigProfileNames(path string) []string {
	var raw map[string]interface{}
	if _, err := toml.DecodeFile(path, &raw); err != nil {
		return nil
	}
	profiles, ok := raw["profile"].(map[string]interface{})
	if !ok {
		return nil
	}
	names := make([]string, 0, len(profiles))
	for name := range profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// flattenProfileRaw converts a raw TOML profile map (as decoded by
// BurntSushi/toml into map[string]interface{}) into a flat koanf-compatible
// map. Only fields explicitly present in the raw map are included.
func flattenProfileRaw(raw map[string]interface{}) map[string]any {
	flat := make(map[string]any)

	// Scalar string fields.
	for _, key := range []string{"output", "format", "tokenizer", "target"} {
		if v, ok := raw[key]; ok {
			flat[key] = v
		}
	}

	// Integer fields: BurntSushi/toml decodes TOML integers as int64 in raw maps.
	if v, ok := raw["max_tokens"]; ok {
		switch n := v.(type) {
		case int64:
			flat["max_tokens"] = int(n)
		case int:
			flat["max_tokens"] = n
		default:
			flat["max_tokens"] = v
		}
	}

	// Boolean fields.
	for _, key := range []string{"compression", "redaction"} {
		if v, ok := raw[key]; ok {
			flat[key] = v
		}
	}

	// Slice fields.
	for _, key := range []string{"ignore", "priority_files", "include"} {
		if v, ok := raw[key]; ok {
			flat[key] = rawToStringSlice(v)
		}
	}

	// Nested: relevance tiers.
	if relRaw, ok := raw["relevance"].(map[string]interface{}); ok {
		for _, tier := range []string{"tier_0", "tier_1", "tier_2", "tier_3", "tier_4", "tier_5"} {
			if v, ok := relRaw[tier]; ok {
				flat["relevance."+tier] = rawToStringSlice(v)
			}
		}
	}

	// Nested: redaction_config.
	if rcRaw, ok := raw["redaction_config"].(map[string]interface{}); ok {
		if v, ok := rcRaw["enabled"]; ok {
			flat["redaction_config.enabled"] = v
		}
		if v, ok := rcRaw["exclude_paths"]; ok {
			flat["redaction_config.exclude_paths"] = rawToStringSlice(v)
		}
		if v, ok := rcRaw["confidence_threshold"]; ok {
			flat["redaction_config.confidence_threshold"] = v
		}
	}

	return flat
}

// rawToStringSlice converts a raw TOML array value ([]interface{}) into
// []string. Returns nil for unrecognised types.
func rawToStringSlice(v interface{}) []string {
	switch s := v.(type) {
	case []string:
		return s
	case []interface{}:
		result := make([]string, 0, len(s))
		for _, item := range s {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	default:
		return nil
	}
}

// loadLayer merges a flat map into k and marks every key in the map as
// originating from src. This approach correctly attributes source even when
// a later layer provides the same value as a prior layer (e.g. CLI flag
// setting the same value as an env var).
func loadLayer(k *koanf.Koanf, m map[string]any, sources SourceMap, src Source) error {
	if err := k.Load(confmap.Provider(m, "."), nil); err != nil {
		return fmt.Errorf("merge layer %s: %w", src.String(), err)
	}
	for key := range m {
		sources[key] = src
	}
	return nil
}

// profileToFlatMap converts a Profile to a flat map for koanf's confmap
// provider. All fields are included (used for the defaults layer where every
// field has an authoritative default value).
func profileToFlatMap(p *Profile) map[string]any {
	return map[string]any{
		"output":      p.Output,
		"format":      p.Format,
		"max_tokens":  p.MaxTokens,
		"tokenizer":   p.Tokenizer,
		"compression": p.Compression,
		"redaction":   p.Redaction,
		"target":      p.Target,

		"ignore":         p.Ignore,
		"priority_files": p.PriorityFiles,
		"include":        p.Include,

		"relevance.tier_0": p.Relevance.Tier0,
		"relevance.tier_1": p.Relevance.Tier1,
		"relevance.tier_2": p.Relevance.Tier2,
		"relevance.tier_3": p.Relevance.Tier3,
		"relevance.tier_4": p.Relevance.Tier4,
		"relevance.tier_5": p.Relevance.Tier5,

		"redaction_config.enabled":              p.RedactionConfig.Enabled,
		"redaction_config.exclude_paths":        p.RedactionConfig.ExcludePaths,
		"redaction_config.confidence_threshold": p.RedactionConfig.ConfidenceThreshold,
	}
}

// flatMapToProfile converts the current koanf state into a Profile struct.
func flatMapToProfile(k *koanf.Koanf) *Profile {
	return &Profile{
		Output:      k.String("output"),
		Format:      k.String("format"),
		MaxTokens:   k.Int("max_tokens"),
		Tokenizer:   k.String("tokenizer"),
		Compression: k.Bool("compression"),
		Redaction:   k.Bool("redaction"),
		Target:      k.String("target"),

		Ignore:        k.Strings("ignore"),
		PriorityFiles: k.Strings("priority_files"),
		Include:       k.Strings("include"),

		Relevance: RelevanceConfig{
			Tier0: k.Strings("relevance.tier_0"),
			Tier1: k.Strings("relevance.tier_1"),
			Tier2: k.Strings("relevance.tier_2"),
			Tier3: k.Strings("relevance.tier_3"),
			Tier4: k.Strings("relevance.tier_4"),
			Tier5: k.Strings("relevance.tier_5"),
		},

		RedactionConfig: RedactionConfig{
			Enabled:             k.Bool("redaction_config.enabled"),
			ExcludePaths:        k.Strings("redaction_config.exclude_paths"),
			ConfidenceThreshold: k.String("redaction_config.confidence_threshold"),
		},
	}
}
