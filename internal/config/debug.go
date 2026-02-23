package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"
)

// ConfigFileStatus represents the found/not-found status of a single harvx
// config file, along with a display-friendly path.
type ConfigFileStatus struct {
	Label string `json:"label"` // "Global" or "Repo"
	Path  string `json:"path"`  // display path with ~ or ./
	Found bool   `json:"found"`
}

// EnvVarStatus tracks whether a known HARVX_* environment variable is
// currently set and active.
type EnvVarStatus struct {
	Name    string `json:"name"`
	Value   string `json:"value,omitempty"`
	Applied bool   `json:"applied"`
}

// ConfigEntry is one row in the resolved configuration table, pairing a flat
// field key with its display value and the source layer that provided it.
type ConfigEntry struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Source string `json:"source"`
}

// DebugOutput is the complete structured result produced by BuildDebugOutput.
// It is consumed by FormatDebugOutput for human-readable text and by
// FormatDebugOutputJSON for machine-readable JSON.
type DebugOutput struct {
	ConfigFiles   []ConfigFileStatus `json:"config_files"`
	ActiveProfile string             `json:"active_profile"`
	InheritChain  []string           `json:"inherit_chain,omitempty"`
	EnvVars       []EnvVarStatus     `json:"env_vars"`
	Config        []ConfigEntry      `json:"config"`
}

// DebugOptions configures BuildDebugOutput. All fields are optional and fall
// back to sensible defaults.
type DebugOptions struct {
	// ProfileName selects which named profile to debug. Defaults to "default".
	ProfileName string
	// TargetDir is the directory to search for harvx.toml. Defaults to ".".
	TargetDir string
	// GlobalConfigPath overrides automatic global config discovery. Useful in
	// tests to point at a fixture file instead of the real user config.
	GlobalConfigPath string
	// CLIFlags holds explicit CLI flag overrides (highest precedence layer).
	// Keys are flat Profile field names: "format", "max_tokens", etc.
	CLIFlags map[string]any
}

// BuildDebugOutput collects all configuration debug information and returns a
// structured DebugOutput ready for rendering. It runs the full 5-layer
// resolution pipeline and annotates each field with its origin.
func BuildDebugOutput(opts DebugOptions) (*DebugOutput, error) {
	profileName := opts.ProfileName
	if profileName == "" {
		profileName = "default"
	}
	targetDir := opts.TargetDir
	if targetDir == "" {
		targetDir = "."
	}

	// ── Config file statuses ─────────────────────────────────────────────────
	configFiles, err := buildConfigFileStatuses(targetDir, opts.GlobalConfigPath)
	if err != nil {
		return nil, fmt.Errorf("building config file statuses: %w", err)
	}

	// ── Full 5-layer resolution ──────────────────────────────────────────────
	resolved, err := Resolve(ResolveOptions{
		ProfileName:      profileName,
		TargetDir:        targetDir,
		GlobalConfigPath: opts.GlobalConfigPath,
		CLIFlags:         opts.CLIFlags,
	})
	if err != nil {
		return nil, fmt.Errorf("resolving config: %w", err)
	}

	// ── Inheritance chain ────────────────────────────────────────────────────
	chain, chainErr := resolveChainForDebug(profileName, targetDir, opts.GlobalConfigPath)
	if chainErr != nil {
		// Non-fatal: degrade gracefully to a single-element chain.
		chain = []string{profileName}
	}

	activeProfile := buildActiveProfileLabel(chain)

	// ── Env var statuses ─────────────────────────────────────────────────────
	envVars := buildEnvVarStatuses()

	// ── Ordered config entries ───────────────────────────────────────────────
	configEntries := buildConfigEntries(resolved.Profile, resolved.Sources)

	return &DebugOutput{
		ConfigFiles:   configFiles,
		ActiveProfile: activeProfile,
		InheritChain:  chain,
		EnvVars:       envVars,
		Config:        configEntries,
	}, nil
}

// FormatDebugOutput renders a DebugOutput as a human-readable text report.
// The resolved configuration table is aligned using text/tabwriter.
//
// Example output:
//
//	Harvx Configuration Debug
//	==========================
//
//	Config Files:
//	  Global:  ~/.config/harvx/config.toml (not found)
//	  Repo:    ./harvx.toml (loaded)
//
//	Active Profile: finvault (extends: base -> default)
//
//	Environment Variables:
//	  HARVX_MAX_TOKENS = 150000 (applied)
//	  HARVX_COMPRESS   = (not set)
//
//	Resolved Configuration:
//	  KEY          VALUE           SOURCE
//	  output       harvx-out.md    repo
func FormatDebugOutput(out *DebugOutput, w io.Writer) error {
	// Header.
	fmt.Fprintln(w, "Harvx Configuration Debug")
	fmt.Fprintln(w, "==========================")
	fmt.Fprintln(w)

	// Config Files section.
	fmt.Fprintln(w, "Config Files:")
	for _, cf := range out.ConfigFiles {
		status := "not found"
		if cf.Found {
			status = "loaded"
		}
		// Pad label to 9 characters so paths align:
		//   "Global:  " (7 chars + 2 spaces = 9)
		//   "Repo:    " (5 chars + 4 spaces = 9)
		fmt.Fprintf(w, "  %-9s%s (%s)\n", cf.Label+":", cf.Path, status)
	}
	fmt.Fprintln(w)

	// Active Profile section.
	fmt.Fprintf(w, "Active Profile: %s\n", out.ActiveProfile)
	fmt.Fprintln(w)

	// Environment Variables section.
	fmt.Fprintln(w, "Environment Variables:")
	if len(out.EnvVars) > 0 {
		// Compute max name length once for right-aligned padding.
		maxLen := 0
		for _, ev := range out.EnvVars {
			if len(ev.Name) > maxLen {
				maxLen = len(ev.Name)
			}
		}
		for _, ev := range out.EnvVars {
			padded := ev.Name + strings.Repeat(" ", maxLen-len(ev.Name))
			if ev.Applied {
				fmt.Fprintf(w, "  %s = %s (applied)\n", padded, ev.Value)
			} else {
				fmt.Fprintf(w, "  %s = (not set)\n", padded)
			}
		}
	}
	fmt.Fprintln(w)

	// Resolved Configuration section.
	fmt.Fprintln(w, "Resolved Configuration:")
	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	fmt.Fprintln(tw, "  KEY\tVALUE\tSOURCE")
	for _, ce := range out.Config {
		fmt.Fprintf(tw, "  %s\t%s\t%s\n", ce.Key, ce.Value, ce.Source)
	}
	if err := tw.Flush(); err != nil {
		return fmt.Errorf("flushing config table: %w", err)
	}

	return nil
}

// FormatDebugOutputJSON marshals a DebugOutput to indented JSON and writes it
// to w. The output includes a trailing newline.
func FormatDebugOutputJSON(out *DebugOutput, w io.Writer) error {
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal debug output to JSON: %w", err)
	}
	_, err = fmt.Fprintln(w, string(data))
	return err
}

// ── Internal builders ────────────────────────────────────────────────────────

// buildConfigFileStatuses computes the Found/not-found status and display path
// for the global and repo config files.
func buildConfigFileStatuses(targetDir, globalConfigPathOverride string) ([]ConfigFileStatus, error) {
	statuses := make([]ConfigFileStatus, 0, 2)

	// Global config: compute canonical expected path via globalConfigDir.
	configDir, err := globalConfigDir()
	if err != nil {
		return nil, fmt.Errorf("global config dir: %w", err)
	}
	globalExpected := filepath.Join(configDir, "harvx", "config.toml")
	globalDisplay := displayTildePath(globalExpected)

	var globalFound bool
	if globalConfigPathOverride != "" {
		// A test-supplied override is treated as "found" by convention.
		globalFound = true
	} else {
		discovered, discErr := DiscoverGlobalConfig()
		if discErr == nil && discovered != "" {
			globalFound = true
		}
	}

	statuses = append(statuses, ConfigFileStatus{
		Label: "Global",
		Path:  globalDisplay,
		Found: globalFound,
	})

	// Repo config: expected path is harvx.toml directly inside targetDir.
	repoExpected := filepath.Join(targetDir, "harvx.toml")
	repoDisplay := displayDotPath(repoExpected, targetDir)

	var repoFound bool
	discovered, discErr := DiscoverRepoConfig(targetDir)
	if discErr == nil && discovered != "" {
		repoFound = true
	}

	statuses = append(statuses, ConfigFileStatus{
		Label: "Repo",
		Path:  repoDisplay,
		Found: repoFound,
	})

	return statuses, nil
}

// buildActiveProfileLabel formats the active profile display string. When the
// chain has more than one element, the ancestry is shown as "extends" list.
//
// Examples:
//   - ["default"]                    → "default"
//   - ["finvault", "base", "default"] → "finvault (extends: base -> default)"
func buildActiveProfileLabel(chain []string) string {
	if len(chain) == 0 {
		return "default"
	}
	if len(chain) == 1 {
		return chain[0]
	}
	return chain[0] + " (extends: " + strings.Join(chain[1:], " -> ") + ")"
}

// buildEnvVarStatuses returns the Applied/not-applied status of every known
// HARVX_* environment variable in a consistent display order.
func buildEnvVarStatuses() []EnvVarStatus {
	known := []string{
		EnvProfile,
		EnvMaxTokens,
		EnvFormat,
		EnvTokenizer,
		EnvOutput,
		EnvTarget,
		EnvCompress,
		EnvRedact,
		EnvLogFormat,
	}

	statuses := make([]EnvVarStatus, 0, len(known))
	for _, name := range known {
		value := os.Getenv(name)
		statuses = append(statuses, EnvVarStatus{
			Name:    name,
			Value:   value,
			Applied: value != "",
		})
	}
	return statuses
}

// buildConfigEntries constructs the ordered list of configuration rows from a
// resolved Profile and its source attribution map. The display order follows
// the canonical field sequence documented in the harvx PRD.
func buildConfigEntries(p *Profile, sources SourceMap) []ConfigEntry {
	entries := make([]ConfigEntry, 0, 19)

	// Scalar fields.
	entries = append(entries, stringEntry("output", p.Output, sources))
	entries = append(entries, stringEntry("format", p.Format, sources))
	entries = append(entries, intEntry("max_tokens", p.MaxTokens, sources))
	entries = append(entries, stringEntry("tokenizer", p.Tokenizer, sources))
	entries = append(entries, boolEntry("compression", p.Compression, sources))
	entries = append(entries, boolEntry("redaction", p.Redaction, sources))
	entries = append(entries, stringEntry("target", p.Target, sources))

	// Top-level slice fields.
	entries = append(entries, sliceEntry("ignore", p.Ignore, sources))
	entries = append(entries, sliceEntry("priority_files", p.PriorityFiles, sources))
	entries = append(entries, sliceEntry("include", p.Include, sources))

	// RedactionConfig nested fields.
	entries = append(entries, boolEntry("redaction_config.enabled", p.RedactionConfig.Enabled, sources))
	entries = append(entries, stringEntry("redaction_config.confidence_threshold", p.RedactionConfig.ConfidenceThreshold, sources))
	entries = append(entries, sliceEntry("redaction_config.exclude_paths", p.RedactionConfig.ExcludePaths, sources))

	// Relevance tier slice fields.
	entries = append(entries, sliceEntry("relevance.tier_0", p.Relevance.Tier0, sources))
	entries = append(entries, sliceEntry("relevance.tier_1", p.Relevance.Tier1, sources))
	entries = append(entries, sliceEntry("relevance.tier_2", p.Relevance.Tier2, sources))
	entries = append(entries, sliceEntry("relevance.tier_3", p.Relevance.Tier3, sources))
	entries = append(entries, sliceEntry("relevance.tier_4", p.Relevance.Tier4, sources))
	entries = append(entries, sliceEntry("relevance.tier_5", p.Relevance.Tier5, sources))

	return entries
}

// stringEntry builds a ConfigEntry for a string-valued profile field.
// An empty string is rendered as "(not set)" with source "-".
func stringEntry(key, value string, sources SourceMap) ConfigEntry {
	if value == "" {
		return ConfigEntry{Key: key, Value: "(not set)", Source: "-"}
	}
	return ConfigEntry{
		Key:    key,
		Value:  value,
		Source: sourceDetailLabel(key, sources[key]),
	}
}

// boolEntry builds a ConfigEntry for a boolean-valued profile field.
// Booleans are always shown with their source attribution (false is meaningful).
func boolEntry(key string, value bool, sources SourceMap) ConfigEntry {
	return ConfigEntry{
		Key:    key,
		Value:  strconv.FormatBool(value),
		Source: sourceDetailLabel(key, sources[key]),
	}
}

// intEntry builds a ConfigEntry for an integer-valued profile field.
// Integers are always shown with their source attribution.
func intEntry(key string, value int, sources SourceMap) ConfigEntry {
	return ConfigEntry{
		Key:    key,
		Value:  strconv.Itoa(value),
		Source: sourceDetailLabel(key, sources[key]),
	}
}

// sliceEntry builds a ConfigEntry for a slice-valued profile field.
// An empty slice is rendered as "(not set)" with source "-".
func sliceEntry(key string, values []string, sources SourceMap) ConfigEntry {
	abbreviated := abbreviateSlice(values)
	if abbreviated == "" {
		return ConfigEntry{Key: key, Value: "(not set)", Source: "-"}
	}
	return ConfigEntry{
		Key:    key,
		Value:  abbreviated,
		Source: sourceDetailLabel(key, sources[key]),
	}
}

// sourceDetailLabel returns a human-readable source label for a profile field.
// For SourceEnv and SourceFlag it embeds the specific env var or flag name.
//
// Examples:
//   - SourceEnv  + "max_tokens" → "env (HARVX_MAX_TOKENS)"
//   - SourceFlag + "output"     → "flag (--output)"
//   - SourceRepo               → "repo"
func sourceDetailLabel(key string, src Source) string {
	switch src {
	case SourceDefault:
		return "default"
	case SourceGlobal:
		return "global"
	case SourceRepo:
		return "repo"
	case SourceEnv:
		if envVar := keyToEnvVar(key); envVar != "" {
			return "env (" + envVar + ")"
		}
		return "env"
	case SourceFlag:
		if flag := keyToFlag(key); flag != "" {
			return "flag (" + flag + ")"
		}
		return "flag"
	default:
		return src.String()
	}
}

// keyToEnvVar returns the HARVX_* env var name for a flat profile key, or an
// empty string when no env var is defined for that key.
func keyToEnvVar(key string) string {
	m := map[string]string{
		"format":      EnvFormat,
		"max_tokens":  EnvMaxTokens,
		"tokenizer":   EnvTokenizer,
		"output":      EnvOutput,
		"target":      EnvTarget,
		"compression": EnvCompress,
		"redaction":   EnvRedact,
	}
	return m[key]
}

// keyToFlag returns the CLI flag name for a flat profile key, or an empty
// string when no flag is defined for that key.
func keyToFlag(key string) string {
	m := map[string]string{
		"output":     "--output",
		"format":     "--format",
		"target":     "--target",
		"max_tokens": "--max-tokens",
	}
	return m[key]
}

// abbreviateSlice formats a string slice for compact single-line display.
//
//   - 0 items → "" (caller shows "(not set)")
//   - 1–3 items → "[item1, item2, item3]"
//   - >3 items → "[item1, item2, item3 ...N more]" where N = len-3
func abbreviateSlice(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1, 2, 3:
		return "[" + strings.Join(items, ", ") + "]"
	default:
		head := strings.Join(items[:3], ", ")
		more := strconv.Itoa(len(items) - 3)
		return "[" + head + " ..." + more + " more]"
	}
}

// displayTildePath replaces the user's home directory prefix in path with "~".
// Falls back to the original path when the home directory cannot be determined
// or when path does not reside under the home directory.
func displayTildePath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}

// displayDotPath converts path to a "./" prefixed path relative to baseDir.
// Falls back to the absolute form of path when the relative path would escape
// baseDir (i.e. start with "..") or when any path computation fails.
func displayDotPath(path, baseDir string) string {
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return path
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	rel, err := filepath.Rel(absBase, absPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return absPath
	}
	return "./" + rel
}

// resolveChainForDebug loads profiles from the repo and global config files
// and calls ResolveProfile to compute the full inheritance chain. The
// globalConfigPath parameter overrides automatic discovery when non-empty,
// which is useful in tests.
//
// Returns the chain in resolution order, e.g. ["finvault", "base", "default"].
// On error (e.g. profile not found), the error is returned so the caller can
// degrade gracefully.
func resolveChainForDebug(profileName, targetDir, globalConfigPath string) ([]string, error) {
	profiles := make(map[string]*Profile)

	// Load repo config first (repo profiles take precedence over global).
	repoPath, err := DiscoverRepoConfig(targetDir)
	if err == nil && repoPath != "" {
		cfg, loadErr := LoadFromFile(repoPath)
		if loadErr == nil {
			for name, p := range cfg.Profile {
				profiles[name] = p
			}
		}
	}

	// Resolve global config path: use override when provided, otherwise discover.
	gPath := globalConfigPath
	if gPath == "" {
		discovered, discErr := DiscoverGlobalConfig()
		if discErr == nil {
			gPath = discovered
		}
	}
	if gPath != "" {
		cfg, loadErr := LoadFromFile(gPath)
		if loadErr == nil {
			for name, p := range cfg.Profile {
				// Do not overwrite profiles already loaded from repo.
				if _, exists := profiles[name]; !exists {
					profiles[name] = p
				}
			}
		}
	}

	res, err := ResolveProfile(profileName, profiles)
	if err != nil {
		return nil, err
	}
	return res.Chain, nil
}
