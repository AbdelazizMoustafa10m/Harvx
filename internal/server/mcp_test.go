package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMCPServer_CreatesServer(t *testing.T) {
	t.Parallel()

	cfg := ServerConfig{RootDir: t.TempDir()}
	s := NewMCPServer(cfg)
	require.NotNil(t, s, "NewMCPServer should return a non-nil server")
}

func TestNewMCPServer_DefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := ServerConfig{}
	s := NewMCPServer(cfg)
	require.NotNil(t, s, "NewMCPServer should accept zero-value ServerConfig")
}

func TestNewMCPServer_WithProfile(t *testing.T) {
	t.Parallel()

	cfg := ServerConfig{
		RootDir: t.TempDir(),
		Profile: "session",
	}
	s := NewMCPServer(cfg)
	require.NotNil(t, s, "NewMCPServer should accept a profile configuration")
}

func TestServerConfig_Fields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     ServerConfig
		wantDir string
		wantPro string
	}{
		{
			name:    "empty config",
			cfg:     ServerConfig{},
			wantDir: "",
			wantPro: "",
		},
		{
			name:    "root dir only",
			cfg:     ServerConfig{RootDir: "/some/path"},
			wantDir: "/some/path",
			wantPro: "",
		},
		{
			name:    "profile only",
			cfg:     ServerConfig{Profile: "session"},
			wantDir: "",
			wantPro: "session",
		},
		{
			name:    "both fields set",
			cfg:     ServerConfig{RootDir: "/some/path", Profile: "session"},
			wantDir: "/some/path",
			wantPro: "session",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.wantDir, tt.cfg.RootDir)
			assert.Equal(t, tt.wantPro, tt.cfg.Profile)
		})
	}
}

func TestNewMCPServer_MultipleInstances(t *testing.T) {
	t.Parallel()

	cfg1 := ServerConfig{RootDir: t.TempDir()}
	cfg2 := ServerConfig{RootDir: t.TempDir()}

	s1 := NewMCPServer(cfg1)
	s2 := NewMCPServer(cfg2)

	require.NotNil(t, s1)
	require.NotNil(t, s2)
	// Each call should produce a distinct server instance.
	assert.NotSame(t, s1, s2, "separate calls should produce distinct server instances")
}
