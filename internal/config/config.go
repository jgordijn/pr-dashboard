// Package config handles configuration loading and parsing for pr-dashboard.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// ErrConfigNotFound is returned when the configuration file does not exist.
// Use errors.Is(err, ErrConfigNotFound) to check for this condition.
var ErrConfigNotFound = errors.New("configuration file not found")

// Default values for optional configuration fields.
const (
	DefaultRefreshInterval  = 30
	DefaultShowDrafts       = true
	DefaultInitialMode      = "full"
	DefaultGrouping         = "organization"
	DefaultHighlightChanges = true
)

// Config is the top-level configuration structure.
type Config struct {
	General       GeneralConfig        `toml:"general"`
	Organizations []OrganizationConfig `toml:"organizations"`
	Display       DisplayConfig        `toml:"display"`
	Notifications NotificationsConfig  `toml:"notifications"`
}

// GeneralConfig contains general application settings.
type GeneralConfig struct {
	Username        string `toml:"username"`         // Required: GitHub username (login)
	RefreshInterval int    `toml:"refresh_interval"` // Optional: seconds, default 30, min 10, max 300
}

// OrganizationConfig represents a GitHub organization to query.
type OrganizationConfig struct {
	Login string `toml:"login"` // Required: GitHub org login
}

// DisplayConfig contains display-related settings.
type DisplayConfig struct {
	ShowDrafts  bool   `toml:"show_drafts"`  // Optional: default true
	InitialMode string `toml:"initial_mode"` // Optional: "full", "compact", "minimal", default "full"
	ASCII       bool   `toml:"ascii"`        // Optional: use pure-ASCII status symbols, default false
	Grouping    string `toml:"grouping"`     // Optional: "organization" or "repository", default "organization"
}

// NotificationsConfig contains notification-related settings.
type NotificationsConfig struct {
	HighlightChanges bool `toml:"highlight_changes"` // Optional: default true
}

// DefaultConfigPath returns the default path to the configuration file.
// This is ~/.config/pr-dashboard/config.toml on Unix-like systems.
func DefaultConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("unable to determine home directory: %w", err)
	}
	return filepath.Join(homeDir, ".config", "pr-dashboard", "config.toml"), nil
}

// Load reads and parses the configuration file from the default path.
// It applies default values for any missing optional fields.
// Returns ErrConfigNotFound if the config file does not exist.
func Load() (*Config, error) {
	configPath, err := DefaultConfigPath()
	if err != nil {
		return nil, err
	}
	return LoadFromPath(configPath)
}

// LoadFromPath reads and parses the configuration file from the specified path.
// It applies default values for any missing optional fields.
// Returns ErrConfigNotFound if the config file does not exist.
func LoadFromPath(path string) (*Config, error) {
	cfg := &Config{}

	// Apply defaults before parsing so TOML values override them
	applyDefaults(cfg)

	// Read the config file
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("%w: %s", ErrConfigNotFound, path)
		}
		return nil, fmt.Errorf("unable to read config file %s: %w", path, err)
	}

	// Parse TOML and get metadata to detect unknown keys
	meta, err := toml.Decode(string(data), cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to parse config file %s: %w", path, err)
	}

	// Check for unknown keys (typos, unsupported fields)
	if undecoded := meta.Undecoded(); len(undecoded) > 0 {
		keys := make([]string, len(undecoded))
		for i, key := range undecoded {
			keys[i] = key.String()
		}
		return nil, fmt.Errorf("unknown configuration keys in %s: %s", path, strings.Join(keys, ", "))
	}

	return cfg, nil
}

// applyDefaults sets all optional fields to their default values.
// This is called before TOML parsing so that explicitly set values in the
// config file will override these defaults.
func applyDefaults(cfg *Config) {
	cfg.General.RefreshInterval = DefaultRefreshInterval
	cfg.Display.ShowDrafts = DefaultShowDrafts
	cfg.Display.InitialMode = DefaultInitialMode
	cfg.Display.Grouping = DefaultGrouping
	cfg.Notifications.HighlightChanges = DefaultHighlightChanges
}
