package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	configDirName = "macsetgo"
	configFile    = "config.json"
)

// Config holds application-wide settings.
type Config struct {
	// DaemonProfile is the profile name to auto-apply when the daemon detects new interfaces.
	DaemonProfile string `json:"daemon_profile,omitempty"`
	// DaemonPollInterval is seconds between interface polls in daemon mode.
	DaemonPollInterval int `json:"daemon_poll_interval,omitempty"`
	// MatchByPortName controls whether the daemon matches by hardware port name (true)
	// or interface name (false).
	MatchByPortName bool `json:"match_by_port_name"`
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		DaemonPollInterval: 5,
		MatchByPortName:    true,
	}
}

// ConfigDir returns the path to ~/.config/macsetgo, creating it if needed.
func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home directory: %w", err)
	}
	dir := filepath.Join(home, ".config", configDirName)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("create config dir: %w", err)
	}
	return dir, nil
}

// LoadConfig reads the config from disk, returning defaults if not found.
func LoadConfig() (Config, error) {
	dir, err := ConfigDir()
	if err != nil {
		return DefaultConfig(), err
	}

	path := filepath.Join(dir, configFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return DefaultConfig(), fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return DefaultConfig(), fmt.Errorf("parse config: %w", err)
	}

	// Apply defaults for zero values.
	if cfg.DaemonPollInterval <= 0 {
		cfg.DaemonPollInterval = 5
	}

	return cfg, nil
}

// SaveConfig writes the config to disk.
func SaveConfig(cfg Config) error {
	dir, err := ConfigDir()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	return os.WriteFile(filepath.Join(dir, configFile), data, 0600)
}
