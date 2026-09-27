package config

import (
	"encoding/json"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
)

// Config represents persistent user settings for ports.
type Config struct {
	Theme string `json:"theme"`
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Theme: "default",
	}
}

// GetConfigPath returns the absolute path to config.json.
// When running under sudo, it resolves to the actual invoking user's home directory.
func GetConfigPath() string {
	dir := GetConfigDir()
	return filepath.Join(dir, "config.json")
}

// GetConfigDir returns the directory path for ports configuration.
func GetConfigDir() string {
	sudoUser := os.Getenv("SUDO_USER")
	if sudoUser != "" {
		if u, err := user.Lookup(sudoUser); err == nil && u.HomeDir != "" {
			return filepath.Join(u.HomeDir, ".config", "ports")
		}
	}

	if home := os.Getenv("HOME"); home != "" {
		return filepath.Join(home, ".config", "ports")
	}

	if u, err := user.Current(); err == nil && u.HomeDir != "" {
		return filepath.Join(u.HomeDir, ".config", "ports")
	}

	return filepath.Join(".", ".config", "ports")
}

// Load loads the configuration from disk, returning defaults if not found.
func Load() *Config {
	cfg := DefaultConfig()
	filePath := GetConfigPath()

	data, err := os.ReadFile(filePath)
	if err != nil {
		return cfg
	}

	var parsed Config
	if err := json.Unmarshal(data, &parsed); err == nil {
		if parsed.Theme != "" {
			cfg.Theme = parsed.Theme
		}
	}

	return cfg
}

// Save writes the configuration to disk.
func Save(cfg *Config) error {
	dir := GetConfigDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	filePath := filepath.Join(dir, "config.json")
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return err
	}

	// If running under sudo, chown config directory and file back to SUDO_USER
	sudoUser := os.Getenv("SUDO_USER")
	if sudoUser != "" {
		if u, err := user.Lookup(sudoUser); err == nil {
			if uid, err := strconv.Atoi(u.Uid); err == nil {
				if gid, err := strconv.Atoi(u.Gid); err == nil {
					_ = os.Chown(dir, uid, gid)
					_ = os.Chown(filePath, uid, gid)
				}
			}
		}
	}

	return nil
}
