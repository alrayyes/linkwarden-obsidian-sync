package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alrayyes/linkwarden-obsidian-sync/internal/config"
	"github.com/stretchr/testify/require"
)

func writeConfigFile(t *testing.T, dir, contents string) string {
	t.Helper()
	path := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))

	return path
}

func TestLoadFromExplicitPath(t *testing.T) {
	dir := t.TempDir()
	path := writeConfigFile(t, dir, `
linkwarden_url = "https://linkwarden.example.com"
linkwarden_token = "file-token"
vault_path = "/vault"
vault_subdir = "Links"
state_dir = "/state"
skip_git = true
`)

	cfg, err := config.Load(path)
	require.NoError(t, err)
	require.Equal(t, "https://linkwarden.example.com", cfg.LinkwardenURL)
	require.Equal(t, "file-token", cfg.LinkwardenToken)
	require.Equal(t, "/vault", cfg.VaultPath)
	require.Equal(t, "Links", cfg.VaultSubdir)
	require.Equal(t, "/state", cfg.StateDir)
	require.True(t, cfg.SkipGit)
}

func TestLoadExplicitPathMissingFileIsAnError(t *testing.T) {
	_, err := config.Load(filepath.Join(t.TempDir(), "does-not-exist.toml"))
	require.Error(t, err)
}

func TestLoadDefaults(t *testing.T) {
	dir := t.TempDir()
	path := writeConfigFile(t, dir, `
linkwarden_url = "https://linkwarden.example.com"
linkwarden_token = "file-token"
`)
	t.Setenv("HOME", dir)
	t.Setenv("XDG_STATE_HOME", "")

	cfg, err := config.Load(path)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(dir, "Documents", "obsidian"), cfg.VaultPath)
	require.Equal(t, "Linkwarden", cfg.VaultSubdir)
	require.Equal(t, filepath.Join(dir, ".local", "state", "linkwarden-obsidian-sync"), cfg.StateDir)
	require.False(t, cfg.SkipGit)
}

func TestLoadStateDirRespectsXDGStateHome(t *testing.T) {
	dir := t.TempDir()
	path := writeConfigFile(t, dir, `
linkwarden_url = "https://linkwarden.example.com"
linkwarden_token = "file-token"
`)
	t.Setenv("XDG_STATE_HOME", "/custom-state")

	cfg, err := config.Load(path)
	require.NoError(t, err)
	require.Equal(t, "/custom-state/linkwarden-obsidian-sync", cfg.StateDir)
}

func TestLoadMissingRequiredFields(t *testing.T) {
	dir := t.TempDir()
	path := writeConfigFile(t, dir, `vault_subdir = "Links"`)

	_, err := config.Load(path)
	require.Error(t, err)
	require.ErrorContains(t, err, "linkwarden_url")
	require.ErrorContains(t, err, "linkwarden_token")
}

func TestLoadEnvVarsOverrideFile(t *testing.T) {
	dir := t.TempDir()
	path := writeConfigFile(t, dir, `
linkwarden_url = "https://from-file.example.com"
linkwarden_token = "file-token"
vault_path = "/from-file"
`)
	t.Setenv("LINKWARDEN_URL", "https://from-env.example.com")
	t.Setenv("VAULT_PATH", "/from-env")

	cfg, err := config.Load(path)
	require.NoError(t, err)
	require.Equal(t, "https://from-env.example.com", cfg.LinkwardenURL)
	require.Equal(t, "file-token", cfg.LinkwardenToken)
	require.Equal(t, "/from-env", cfg.VaultPath)
}

func TestLoadEnvVarsAloneAreEnough(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("LINKWARDEN_URL", "https://from-env.example.com")
	t.Setenv("LINKWARDEN_TOKEN", "env-token")

	// No config file at the default XDG location — the Docker image's
	// situation, where env vars have to carry the whole config on their own.
	cfg, err := config.Load("")
	require.NoError(t, err)
	require.Equal(t, "https://from-env.example.com", cfg.LinkwardenURL)
	require.Equal(t, "env-token", cfg.LinkwardenToken)
}

func TestLoadSkipGitEnvVar(t *testing.T) {
	dir := t.TempDir()
	path := writeConfigFile(t, dir, `
linkwarden_url = "https://linkwarden.example.com"
linkwarden_token = "file-token"
`)
	t.Setenv("LINKWARDEN_SYNC_SKIP_GIT", "true")

	cfg, err := config.Load(path)
	require.NoError(t, err)
	require.True(t, cfg.SkipGit)
}

func TestLoadDefaultXDGPath(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "linkwarden-obsidian-sync")
	require.NoError(t, os.MkdirAll(configDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(`
linkwarden_url = "https://xdg-default.example.com"
linkwarden_token = "xdg-token"
`), 0o600))
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")

	cfg, err := config.Load("")
	require.NoError(t, err)
	require.Equal(t, "https://xdg-default.example.com", cfg.LinkwardenURL)
}

func TestLoadXDGConfigHomeOverride(t *testing.T) {
	home := t.TempDir()
	xdgConfigHome := t.TempDir()
	configDir := filepath.Join(xdgConfigHome, "linkwarden-obsidian-sync")
	require.NoError(t, os.MkdirAll(configDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(`
linkwarden_url = "https://custom-xdg.example.com"
linkwarden_token = "xdg-token"
`), 0o600))
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", xdgConfigHome)

	cfg, err := config.Load("")
	require.NoError(t, err)
	require.Equal(t, "https://custom-xdg.example.com", cfg.LinkwardenURL)
}

func TestWriteTemplateToExplicitPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.toml")

	written, err := config.WriteTemplate(path, false)
	require.NoError(t, err)
	require.Equal(t, path, written)

	data, err := os.ReadFile(path) //nolint:gosec // path is this test's own t.TempDir() fixture, not user input
	require.NoError(t, err)
	require.Contains(t, string(data), "linkwarden_url")
	require.Contains(t, string(data), "LINKWARDEN_TOKEN")
}

func TestWriteTemplateRefusesToOverwriteWithoutForce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	_, err := config.WriteTemplate(path, false)
	require.NoError(t, err)

	_, err = config.WriteTemplate(path, false)
	require.Error(t, err)
}

func TestWriteTemplateForceOverwrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	_, err := config.WriteTemplate(path, false)
	require.NoError(t, err)

	_, err = config.WriteTemplate(path, true)
	require.NoError(t, err)
}

func TestWriteTemplateDefaultXDGPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")

	written, err := config.WriteTemplate("", false)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(home, ".config", "linkwarden-obsidian-sync", "config.toml"), written)

	_, err = os.Stat(written)
	require.NoError(t, err)
}

func TestLoadNoConfigFileAnywhereAndNoEnvIsAnError(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("LINKWARDEN_URL", "")
	t.Setenv("LINKWARDEN_TOKEN", "")

	_, err := config.Load("")
	require.Error(t, err)
}
