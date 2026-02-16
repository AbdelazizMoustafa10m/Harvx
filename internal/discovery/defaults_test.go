package discovery

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultIgnorePatterns_NotEmpty(t *testing.T) {
	t.Parallel()
	assert.NotEmpty(t, DefaultIgnorePatterns, "DefaultIgnorePatterns must not be empty")
}

func TestDefaultIgnorePatterns_ContainsExpected(t *testing.T) {
	t.Parallel()

	expected := []string{
		".git/",
		"node_modules/",
		"dist/",
		"build/",
		"coverage/",
		"__pycache__/",
		".next/",
		"target/",
		"vendor/",
		".harvx/",
		".env",
		".env.*",
		"*.pem",
		"*.key",
		"*.p12",
		"*.pfx",
		"*secret*",
		"*credential*",
		"*password*",
		"package-lock.json",
		"yarn.lock",
		"pnpm-lock.yaml",
		"Gemfile.lock",
		"Cargo.lock",
		"go.sum",
		"poetry.lock",
		"*.pyc",
		"*.pyo",
		"*.class",
		"*.o",
		"*.obj",
		"*.exe",
		"*.dll",
		"*.so",
		"*.dylib",
		".DS_Store",
		"Thumbs.db",
		".idea/",
		".vscode/",
		"*.swp",
		"*.swo",
	}

	patternSet := make(map[string]bool, len(DefaultIgnorePatterns))
	for _, p := range DefaultIgnorePatterns {
		patternSet[p] = true
	}

	for _, exp := range expected {
		assert.True(t, patternSet[exp], "DefaultIgnorePatterns should contain %q", exp)
	}
}

func TestSensitivePatterns_NotEmpty(t *testing.T) {
	t.Parallel()
	assert.NotEmpty(t, SensitivePatterns, "SensitivePatterns must not be empty")
}

func TestSensitivePatterns_SubsetOfDefaults(t *testing.T) {
	t.Parallel()

	defaultSet := make(map[string]bool, len(DefaultIgnorePatterns))
	for _, p := range DefaultIgnorePatterns {
		defaultSet[p] = true
	}

	for _, sp := range SensitivePatterns {
		assert.True(t, defaultSet[sp],
			"SensitivePatterns entry %q should also be in DefaultIgnorePatterns", sp)
	}
}

func TestNewDefaultIgnoreMatcher(t *testing.T) {
	t.Parallel()

	m := NewDefaultIgnoreMatcher()
	require.NotNil(t, m)
	assert.Equal(t, len(DefaultIgnorePatterns), m.PatternCount())
}

func TestDefaultIgnoreMatcher_Directories(t *testing.T) {
	t.Parallel()

	m := NewDefaultIgnoreMatcher()

	tests := []struct {
		name   string
		path   string
		isDir  bool
		expect bool
	}{
		{name: ".git dir", path: ".git", isDir: true, expect: true},
		{name: "node_modules dir", path: "node_modules", isDir: true, expect: true},
		{name: "dist dir", path: "dist", isDir: true, expect: true},
		{name: "build dir", path: "build", isDir: true, expect: true},
		{name: "coverage dir", path: "coverage", isDir: true, expect: true},
		{name: "__pycache__ dir", path: "__pycache__", isDir: true, expect: true},
		{name: ".next dir", path: ".next", isDir: true, expect: true},
		{name: "target dir", path: "target", isDir: true, expect: true},
		{name: "vendor dir", path: "vendor", isDir: true, expect: true},
		{name: ".harvx dir", path: ".harvx", isDir: true, expect: true},
		{name: ".idea dir", path: ".idea", isDir: true, expect: true},
		{name: ".vscode dir", path: ".vscode", isDir: true, expect: true},
		{name: "nested node_modules", path: "packages/app/node_modules", isDir: true, expect: true},
		{name: "file inside node_modules", path: "node_modules/express/index.js", isDir: false, expect: true},
		{name: "src dir not ignored", path: "src", isDir: true, expect: false},
		{name: "lib dir not ignored", path: "lib", isDir: true, expect: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := m.IsIgnored(tt.path, tt.isDir)
			assert.Equal(t, tt.expect, got, "IsIgnored(%q, %v)", tt.path, tt.isDir)
		})
	}
}

func TestDefaultIgnoreMatcher_EnvironmentFiles(t *testing.T) {
	t.Parallel()

	m := NewDefaultIgnoreMatcher()

	tests := []struct {
		name   string
		path   string
		expect bool
	}{
		{name: ".env", path: ".env", expect: true},
		{name: ".env.local", path: ".env.local", expect: true},
		{name: ".env.production", path: ".env.production", expect: true},
		{name: ".env.development", path: ".env.development", expect: true},
		{name: "nested .env", path: "config/.env", expect: true},
		{name: "nested .env.local", path: "config/.env.local", expect: true},
		{name: "env.go not ignored", path: "env.go", expect: false},
		{name: "environment.yml not ignored", path: "environment.yml", expect: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := m.IsIgnored(tt.path, false)
			assert.Equal(t, tt.expect, got, "IsIgnored(%q, false)", tt.path)
		})
	}
}

func TestDefaultIgnoreMatcher_CertificateFiles(t *testing.T) {
	t.Parallel()

	m := NewDefaultIgnoreMatcher()

	tests := []struct {
		name   string
		path   string
		expect bool
	}{
		{name: "pem file", path: "server.pem", expect: true},
		{name: "key file", path: "server.key", expect: true},
		{name: "p12 file", path: "keystore.p12", expect: true},
		{name: "pfx file", path: "cert.pfx", expect: true},
		{name: "nested pem", path: "certs/server.pem", expect: true},
		{name: "nested key", path: "ssl/private.key", expect: true},
		{name: "key.go not ignored", path: "key.go", expect: false},
		{name: "pem_utils.py not ignored", path: "pem_utils.py", expect: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := m.IsIgnored(tt.path, false)
			assert.Equal(t, tt.expect, got, "IsIgnored(%q, false)", tt.path)
		})
	}
}

func TestDefaultIgnoreMatcher_SensitiveNamingPatterns(t *testing.T) {
	t.Parallel()

	m := NewDefaultIgnoreMatcher()

	tests := []struct {
		name   string
		path   string
		expect bool
	}{
		{name: "secret in name", path: "my-secret-config.yml", expect: true},
		{name: "credential in name", path: "credential-store.json", expect: true},
		{name: "password in name", path: "password-hash.txt", expect: true},
		{name: "secrets dir file", path: "secrets.yaml", expect: true},
		{name: "nested secret", path: "config/db-secret.env", expect: true},
		{name: "app-credentials", path: "app-credentials.json", expect: true},
		{name: "normal config not ignored", path: "config.yml", expect: false},
		{name: "main.go not ignored", path: "main.go", expect: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := m.IsIgnored(tt.path, false)
			assert.Equal(t, tt.expect, got, "IsIgnored(%q, false)", tt.path)
		})
	}
}

func TestDefaultIgnoreMatcher_LockFiles(t *testing.T) {
	t.Parallel()

	m := NewDefaultIgnoreMatcher()

	tests := []struct {
		name   string
		path   string
		expect bool
	}{
		{name: "package-lock.json", path: "package-lock.json", expect: true},
		{name: "yarn.lock", path: "yarn.lock", expect: true},
		{name: "pnpm-lock.yaml", path: "pnpm-lock.yaml", expect: true},
		{name: "Gemfile.lock", path: "Gemfile.lock", expect: true},
		{name: "Cargo.lock", path: "Cargo.lock", expect: true},
		{name: "go.sum", path: "go.sum", expect: true},
		{name: "poetry.lock", path: "poetry.lock", expect: true},
		{name: "nested package-lock", path: "packages/ui/package-lock.json", expect: true},
		{name: "go.mod not ignored", path: "go.mod", expect: false},
		{name: "Gemfile not ignored", path: "Gemfile", expect: false},
		{name: "Cargo.toml not ignored", path: "Cargo.toml", expect: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := m.IsIgnored(tt.path, false)
			assert.Equal(t, tt.expect, got, "IsIgnored(%q, false)", tt.path)
		})
	}
}

func TestDefaultIgnoreMatcher_CompiledArtifacts(t *testing.T) {
	t.Parallel()

	m := NewDefaultIgnoreMatcher()

	tests := []struct {
		name   string
		path   string
		expect bool
	}{
		{name: "pyc file", path: "module.pyc", expect: true},
		{name: "pyo file", path: "module.pyo", expect: true},
		{name: "class file", path: "Main.class", expect: true},
		{name: "object file .o", path: "main.o", expect: true},
		{name: "object file .obj", path: "main.obj", expect: true},
		{name: "exe file", path: "app.exe", expect: true},
		{name: "dll file", path: "library.dll", expect: true},
		{name: "so file", path: "libcrypto.so", expect: true},
		{name: "dylib file", path: "libssl.dylib", expect: true},
		{name: "nested pyc", path: "src/__pycache__/module.pyc", expect: true},
		{name: "go file not ignored", path: "main.go", expect: false},
		{name: "py file not ignored", path: "main.py", expect: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := m.IsIgnored(tt.path, false)
			assert.Equal(t, tt.expect, got, "IsIgnored(%q, false)", tt.path)
		})
	}
}

func TestDefaultIgnoreMatcher_OSEditorFiles(t *testing.T) {
	t.Parallel()

	m := NewDefaultIgnoreMatcher()

	tests := []struct {
		name   string
		path   string
		isDir  bool
		expect bool
	}{
		{name: ".DS_Store", path: ".DS_Store", isDir: false, expect: true},
		{name: "Thumbs.db", path: "Thumbs.db", isDir: false, expect: true},
		{name: "swp file", path: ".main.go.swp", isDir: false, expect: true},
		{name: "swo file", path: ".main.go.swo", isDir: false, expect: true},
		{name: "nested .DS_Store", path: "src/.DS_Store", isDir: false, expect: true},
		{name: ".idea dir", path: ".idea", isDir: true, expect: true},
		{name: ".vscode dir", path: ".vscode", isDir: true, expect: true},
		{name: "file in .idea", path: ".idea/workspace.xml", isDir: false, expect: true},
		{name: "file in .vscode", path: ".vscode/settings.json", isDir: false, expect: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := m.IsIgnored(tt.path, tt.isDir)
			assert.Equal(t, tt.expect, got, "IsIgnored(%q, %v)", tt.path, tt.isDir)
		})
	}
}

func TestDefaultIgnoreMatcher_EmptyAndDotPaths(t *testing.T) {
	t.Parallel()

	m := NewDefaultIgnoreMatcher()

	assert.False(t, m.IsIgnored("", false), "empty path should not be ignored")
	assert.False(t, m.IsIgnored(".", false), "dot path should not be ignored")
	assert.False(t, m.IsIgnored("./", true), "dot-slash path should not be ignored")
}

func TestDefaultIgnoreMatcher_NormalFilesNotIgnored(t *testing.T) {
	t.Parallel()

	m := NewDefaultIgnoreMatcher()

	normalFiles := []string{
		"main.go",
		"README.md",
		"Makefile",
		"src/app.ts",
		"internal/config/config.go",
		"docs/architecture.md",
		"Dockerfile",
		"docker-compose.yml",
		".golangci.yml",
		".editorconfig",
	}

	for _, path := range normalFiles {
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			assert.False(t, m.IsIgnored(path, false),
				"normal file %q should not be ignored", path)
		})
	}
}

func TestIsSensitivePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		path   string
		expect bool
	}{
		{name: ".env is sensitive", path: ".env", expect: true},
		{name: ".env.local is sensitive", path: ".env.local", expect: true},
		{name: "server.pem is sensitive", path: "server.pem", expect: true},
		{name: "private.key is sensitive", path: "private.key", expect: true},
		{name: "cert.p12 is sensitive", path: "cert.p12", expect: true},
		{name: "cert.pfx is sensitive", path: "cert.pfx", expect: true},
		{name: "my-secret.yml is sensitive", path: "my-secret.yml", expect: true},
		{name: "credential-store is sensitive", path: "credential-store.json", expect: true},
		{name: "password-file is sensitive", path: "password-file.txt", expect: true},
		{name: "main.go is not sensitive", path: "main.go", expect: false},
		{name: "README.md is not sensitive", path: "README.md", expect: false},
		{name: "config.yml is not sensitive", path: "config.yml", expect: false},
		{name: "empty path", path: "", expect: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := IsSensitivePath(tt.path)
			assert.Equal(t, tt.expect, got, "IsSensitivePath(%q)", tt.path)
		})
	}
}

func BenchmarkDefaultIgnoreMatcher_IsIgnored(b *testing.B) {
	m := NewDefaultIgnoreMatcher()

	paths := []string{
		"main.go",
		"node_modules/express/index.js",
		"src/app.ts",
		".env",
		"server.pem",
		"package-lock.json",
		"README.md",
		"internal/config/config.go",
		".DS_Store",
		"build/output.js",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, p := range paths {
			m.IsIgnored(p, false)
		}
	}
}
