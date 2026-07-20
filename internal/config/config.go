package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config is the persisted pr-agent configuration.
type Config struct {
	Token string `json:"token"`
	User  string `json:"user,omitempty"`
}

// Path returns the config file location.
// Honors PR_AGENT_CONFIG, otherwise uses the user config dir.
func Path() (string, error) {
	if p := os.Getenv("PR_AGENT_CONFIG"); p != "" {
		return p, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve config dir: %w", err)
	}
	return filepath.Join(dir, "pr-agent", "config.json"), nil
}

// Load reads the stored config. Returns (nil, nil) if no config exists yet.
func Load() (*Config, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	return &cfg, nil
}

// Save writes the config with restrictive permissions.
func Save(cfg *Config) (string, error) {
	path, err := Path()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", fmt.Errorf("create config dir: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return "", fmt.Errorf("write config: %w", err)
	}
	return path, nil
}

// Delete removes the stored config. Returns the path and whether it existed.
func Delete() (string, bool, error) {
	path, err := Path()
	if err != nil {
		return "", false, err
	}
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return path, false, nil
		}
		return path, false, fmt.Errorf("remove config: %w", err)
	}
	return path, true, nil
}

// StoredToken returns the token from the config file, or "" if none.
func StoredToken() string {
	cfg, err := Load()
	if err != nil || cfg == nil {
		return ""
	}
	return cfg.Token
}
