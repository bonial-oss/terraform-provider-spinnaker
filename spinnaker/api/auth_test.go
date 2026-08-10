package api

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadGateConfig_missingFile(t *testing.T) {
	// The Gate config file is optional, a missing one must not be an error.
	cfg, err := loadGateConfig(filepath.Join(t.TempDir(), "config"))
	require.NoError(t, err)
	require.Nil(t, cfg.Auth)
	require.Empty(t, cfg.Gate.Endpoint)
}

func TestLoadGateConfig(t *testing.T) {
	t.Setenv("TEST_CLIENT_SECRET", "the-secret")

	location := filepath.Join(t.TempDir(), "config")
	contents := `
gate:
  endpoint: https://gate.example.com
auth:
  enabled: true
  oauth2:
    authUrl: https://auth.example.com/authorize
    tokenUrl: https://auth.example.com/token
    clientId: the-client-id
    clientSecret: $TEST_CLIENT_SECRET
    scopes:
      - scope1
      - scope2
`
	require.NoError(t, os.WriteFile(location, []byte(contents), 0600))

	cfg, err := loadGateConfig(location)
	require.NoError(t, err)

	require.Equal(t, "https://gate.example.com", cfg.Gate.Endpoint)
	require.NotNil(t, cfg.Auth)
	require.True(t, cfg.Auth.Enabled)
	require.NotNil(t, cfg.Auth.OAuth2)
	require.Equal(t, "the-client-id", cfg.Auth.OAuth2.ClientId)
	// Environment variables in the config file are expanded.
	require.Equal(t, "the-secret", cfg.Auth.OAuth2.ClientSecret)
	require.Equal(t, []string{"scope1", "scope2"}, cfg.Auth.OAuth2.Scopes)
}

func TestLoadGateConfig_unknownField(t *testing.T) {
	location := filepath.Join(t.TempDir(), "config")
	require.NoError(t, os.WriteFile(location, []byte("gate:\n  nope: true\n"), 0600))

	_, err := loadGateConfig(location)
	require.ErrorContains(t, err, "failed to parse Gate config file")
}
