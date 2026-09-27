package renderer

import (
	"bytes"
	"strings"
	"testing"
)

func TestAvailableThemes(t *testing.T) {
	if len(AvailableThemes) != 16 {
		t.Fatalf("expected 16 themes, got %d", len(AvailableThemes))
	}

	expectedIDs := []string{
		"default",
		"catppuccin",
		"nord",
		"dracula",
		"gruvbox",
		"tokyo_night",
		"rose_pine",
		"one_dark",
		"monokai",
		"kanagawa",
		"cyberpunk",
		"everforest",
		"ayu",
		"synthwave",
		"solarized",
		"matrix",
	}

	for i, id := range expectedIDs {
		if AvailableThemes[i].ID != id {
			t.Errorf("theme index %d ID = %q; want %q", i, AvailableThemes[i].ID, id)
		}
		if AvailableThemes[i].Name == "" {
			t.Errorf("theme %s has empty Name", id)
		}
	}
}

func TestFindTheme(t *testing.T) {
	tests := []struct {
		input  string
		wantID string
	}{
		{"catppuccin", "catppuccin"},
		{"Catppuccin", "catppuccin"},
		{"CATPPUCCIN", "catppuccin"},
		{"tokyo-night", "tokyo_night"},
		{"Tokyo Night", "tokyo_night"},
		{"rose_pine", "rose_pine"},
		{"Rosé Pine", "rose_pine"},
		{"non-existent-theme-xyz", "default"},
		{"", "default"},
	}

	for _, tt := range tests {
		got := FindTheme(tt.input)
		if got.ID != tt.wantID {
			t.Errorf("FindTheme(%q) = %q; want %q", tt.input, got.ID, tt.wantID)
		}
	}
}

func TestIsValidTheme(t *testing.T) {
	if !IsValidTheme("catppuccin") {
		t.Errorf("expected catppuccin to be valid")
	}
	if !IsValidTheme("Catppuccin Mocha") {
		t.Errorf("expected Catppuccin Mocha to be valid")
	}
	if !IsValidTheme("nord") {
		t.Errorf("expected nord to be valid")
	}
	if !IsValidTheme("cyberpunk") {
		t.Errorf("expected cyberpunk to be valid")
	}
	if IsValidTheme("random_nonexistent") {
		t.Errorf("expected random_nonexistent to be invalid")
	}
}

func TestCycleTheme(t *testing.T) {
	current := "default"
	visited := make(map[string]bool)

	for i := 0; i < len(AvailableThemes); i++ {
		next := CycleTheme(current)
		if visited[next.ID] {
			t.Fatalf("cycle visited %s multiple times in one round", next.ID)
		}
		visited[next.ID] = true
		current = next.ID
	}

	// Cycling once more should wrap around back to the first theme after default
	firstAfterDefault := AvailableThemes[1].ID
	if current != AvailableThemes[0].ID {
		t.Errorf("expected cycle after len(AvailableThemes) to reach index 0, got %s", current)
	}
	next := CycleTheme(current)
	if next.ID != firstAfterDefault {
		t.Errorf("expected next after default to be %s, got %s", firstAfterDefault, next.ID)
	}
}

func TestBuildTheme(t *testing.T) {
	def := FindTheme("dracula")
	enabled := def.BuildTheme(true)
	if !enabled.Enabled {
		t.Errorf("expected theme to be enabled")
	}
	if enabled.ID != "dracula" {
		t.Errorf("expected theme ID dracula, got %s", enabled.ID)
	}
	if enabled.Cyan == "" || enabled.Reset == "" {
		t.Errorf("expected ANSI escape strings to be non-empty")
	}
	if enabled.Bg == nil || !strings.HasPrefix(enabled.BgEscape, "\033[48;2;") {
		t.Errorf("expected dracula to have 24-bit background escape, got %q", enabled.BgEscape)
	}
	if !strings.Contains(enabled.Reset, enabled.BgEscape) {
		t.Errorf("expected Reset to maintain background escape, got %q", enabled.Reset)
	}

	defaultDef := FindTheme("default")
	defaultTheme := defaultDef.BuildTheme(true)
	if defaultTheme.Bg != nil || defaultTheme.BgEscape != "" {
		t.Errorf("expected default theme to have transparent/nil background, got %q", defaultTheme.BgEscape)
	}
	if defaultTheme.Reset != "\033[0m" {
		t.Errorf("expected default theme Reset to be \\033[0m, got %q", defaultTheme.Reset)
	}

	disabled := def.BuildTheme(false)
	if disabled.Enabled {
		t.Errorf("expected theme to be disabled")
	}
}

func TestPrintAvailableThemes(t *testing.T) {
	var buf bytes.Buffer
	PrintAvailableThemes(&buf, "catppuccin")
	output := buf.String()

	if !strings.Contains(output, "Available Themes:") {
		t.Errorf("expected output to contain 'Available Themes:'")
	}
	if !strings.Contains(output, "catppuccin") {
		t.Errorf("expected output to contain 'catppuccin'")
	}
	if !strings.Contains(output, "Catppuccin Mocha") {
		t.Errorf("expected output to contain 'Catppuccin Mocha'")
	}
	if !strings.Contains(output, "(active)") {
		t.Errorf("expected output to mark active theme")
	}
}
