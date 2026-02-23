package config

// DefaultProfile returns a new Profile populated with the built-in defaults
// described in PRD Section 5.2 and Section 5.3. This profile is used as the
// base when no harvx.toml is present or when a named profile omits fields.
//
// Callers receive a fresh copy each time; mutating the returned value does not
// affect subsequent calls.
func DefaultProfile() *Profile {
	return &Profile{
		Output:      "harvx-output.md",
		Format:      "markdown",
		MaxTokens:   128000,
		Tokenizer:   "cl100k_base",
		Compression: false,
		Redaction:   true,
		Target:      "",
		Ignore: []string{
			"node_modules",
			"dist",
			".git",
			"coverage",
			"__pycache__",
			".next",
			"target",
			"vendor",
		},
		Relevance: defaultRelevanceTiers(),
		RedactionConfig: RedactionConfig{
			Enabled:             true,
			ConfidenceThreshold: "high",
		},
	}
}

// defaultRelevanceTiers returns the built-in tier glob patterns per PRD Section 5.3.
//
//   - Tier 0: Configuration files (package.json, Cargo.toml, go.mod, Makefile, etc.)
//   - Tier 1: Primary source directories (src/, lib/, app/, cmd/, internal/, pkg/)
//   - Tier 2: Secondary source files, components, utilities
//   - Tier 3: Test files (*_test.go, *.test.ts, *.spec.js, __tests__/)
//   - Tier 4: Documentation (*.md, docs/, README*)
//   - Tier 5: CI/CD configs, lock files
func defaultRelevanceTiers() RelevanceConfig {
	return RelevanceConfig{
		Tier0: []string{
			"package.json",
			"tsconfig.json",
			"tsconfig.*.json",
			"Cargo.toml",
			"go.mod",
			"go.sum",
			"Makefile",
			"Dockerfile",
			"docker-compose.yml",
			"docker-compose.yaml",
			"*.config.*",
			"pyproject.toml",
			"setup.py",
			"setup.cfg",
			"pom.xml",
			"build.gradle",
			"build.gradle.kts",
		},
		Tier1: []string{
			"src/**",
			"lib/**",
			"app/**",
			"cmd/**",
			"internal/**",
			"pkg/**",
		},
		Tier2: []string{
			"components/**",
			"hooks/**",
			"utils/**",
			"helpers/**",
			"middleware/**",
			"services/**",
			"models/**",
			"types/**",
		},
		Tier3: []string{
			"**/*_test.go",
			"**/*.test.ts",
			"**/*.test.tsx",
			"**/*.test.js",
			"**/*.spec.ts",
			"**/*.spec.tsx",
			"**/*.spec.js",
			"**/__tests__/**",
			"**/*_test.py",
			"**/tests/**",
		},
		Tier4: []string{
			"**/*.md",
			"docs/**",
			"README*",
			"CHANGELOG*",
			"CONTRIBUTING*",
			"LICENSE*",
		},
		Tier5: []string{
			".github/**",
			".gitlab-ci.yml",
			".gitlab/**",
			"**/*.lock",
			"package-lock.json",
			"yarn.lock",
			"pnpm-lock.yaml",
			"Cargo.lock",
		},
	}
}
