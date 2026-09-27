package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Theme != "default" {
		t.Errorf("expected default theme to be 'default', got %q", cfg.Theme)
	}
}

func TestLoadSave(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("SUDO_USER", "")

	cfg := Load()
	if cfg.Theme != "default" {
		t.Errorf("expected default theme 'default', got %q", cfg.Theme)
	}

	cfg.Theme = "catppuccin"
	if err := Save(cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	configFile := filepath.Join(tmpDir, ".config", "ports", "config.json")
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		t.Fatalf("expected config file at %s, but does not exist", configFile)
	}

	loaded := Load()
	if loaded.Theme != "catppuccin" {
		t.Errorf("expected loaded theme 'catppuccin', got %q", loaded.Theme)
	}
}
