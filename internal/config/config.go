// Package config loads this tool's configuration from a TOML file at an
// XDG-standard path (or an explicit override), with the environment
// variables it used to read exclusively kept as an optional override on
// top — the natural way to configure the Docker image, where mounting a
// file usually isn't worth it.
package config

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Config is the fully resolved, validated set of settings this tool needs
// to run.
type Config struct {
	LinkwardenURL   string `mapstructure:"linkwarden_url"`
	LinkwardenToken string `mapstructure:"linkwarden_token"`
	VaultPath       string `mapstructure:"vault_path"`
	VaultSubdir     string `mapstructure:"vault_subdir"`
	StateDir        string `mapstructure:"state_dir"`
	SkipGit         bool   `mapstructure:"skip_git"`
}

// ErrMissingRequired is wrapped with the actual comma-joined field names
// rather than building a one-off dynamic error, so callers can match on it
// with errors.Is if they ever need to. errConfigExists is likewise wrapped
// with the actual path in WriteTemplate.
var (
	ErrMissingRequired = errors.New("config: missing required settings")
	errConfigExists    = errors.New("config file already exists (pass --force to overwrite)")
)

// envBindings maps each config key to the environment variable that has
// historically configured it, preserved as an override so the setting
// still works the old way — most importantly for the Docker image, which
// has no config file mounted by default.
var envBindings = map[string]string{
	"linkwarden_url":   "LINKWARDEN_URL",
	"linkwarden_token": "LINKWARDEN_TOKEN",
	"vault_path":       "VAULT_PATH",
	"vault_subdir":     "VAULT_SUBDIR",
	"state_dir":        "LINKWARDEN_SYNC_STATE_DIR",
	"skip_git":         "LINKWARDEN_SYNC_SKIP_GIT",
}

// Load reads configuration from the TOML file at path, or, if path is
// empty, from $XDG_CONFIG_HOME/linkwarden-obsidian-sync/config.toml (or its
// XDG-spec fallback, ~/.config/linkwarden-obsidian-sync/config.toml). A
// missing file at the default location is not an error by itself — the
// environment variables in envBindings, or the defaults below, may still
// satisfy every required field — but a missing file at an explicitly given
// path is.
func Load(path string) (Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Config{}, fmt.Errorf("resolving home directory: %w", err)
	}

	v := viper.New()
	v.SetConfigType("toml")
	v.SetDefault("vault_path", filepath.Join(home, "Documents", "obsidian"))
	v.SetDefault("vault_subdir", "Linkwarden")
	v.SetDefault("state_dir", filepath.Join(xdgDir("XDG_STATE_HOME", home, ".local", "state"), "linkwarden-obsidian-sync"))

	explicit := path != ""
	if explicit {
		v.SetConfigFile(path)
	} else {
		v.AddConfigPath(filepath.Join(xdgDir("XDG_CONFIG_HOME", home, ".config"), "linkwarden-obsidian-sync"))
		v.SetConfigName("config")
	}

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if explicit || !errors.As(err, &notFound) {
			return Config{}, fmt.Errorf("reading config: %w", err)
		}
	}

	for key, env := range envBindings {
		if err := v.BindEnv(key, env); err != nil {
			return Config{}, fmt.Errorf("binding %s: %w", env, err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("parsing config: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) validate() error {
	var missing []string
	if c.LinkwardenURL == "" {
		missing = append(missing, "linkwarden_url (LINKWARDEN_URL)")
	}
	if c.LinkwardenToken == "" {
		missing = append(missing, "linkwarden_token (LINKWARDEN_TOKEN)")
	}
	if len(missing) > 0 {
		return fmt.Errorf("%w: %s", ErrMissingRequired, strings.Join(missing, ", "))
	}

	return nil
}

// ResolvePath returns the config file path this tool reads from or writes
// to: path itself if non-empty, otherwise the XDG default
// ($XDG_CONFIG_HOME/linkwarden-obsidian-sync/config.toml, falling back to
// ~/.config/linkwarden-obsidian-sync/config.toml).
func ResolvePath(path string) (string, error) {
	if path != "" {
		return path, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}

	return filepath.Join(xdgDir("XDG_CONFIG_HOME", home, ".config"), "linkwarden-obsidian-sync", "config.toml"), nil
}

// xdgDir resolves an XDG base-directory variable per spec: an unset or
// relative value is ignored (the spec requires an absolute path), falling
// back to home joined with fallbackParts.
func xdgDir(envVar, home string, fallbackParts ...string) string {
	if v := os.Getenv(envVar); v != "" && filepath.IsAbs(v) {
		return v
	}

	return filepath.Join(append([]string{home}, fallbackParts...)...)
}

//go:embed config.toml.tmpl
var configTemplate string

// WriteTemplate writes a commented template config file to path, or, if
// path is empty, to the default XDG location. It refuses to overwrite an
// existing file unless force is true, and returns the path it wrote to.
func WriteTemplate(path string, force bool) (string, error) {
	path, err := ResolvePath(path)
	if err != nil {
		return "", err
	}

	if !force {
		if _, err := os.Stat(path); err == nil {
			return "", fmt.Errorf("%s: %w", path, errConfigExists)
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("checking %s: %w", path, err)
		}
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return "", fmt.Errorf("creating %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(configTemplate), 0o600); err != nil {
		return "", fmt.Errorf("writing %s: %w", path, err)
	}

	return path, nil
}
