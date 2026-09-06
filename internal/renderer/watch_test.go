package renderer

import (
	"strings"
	"testing"
)

func TestVisibleLength(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"hello", 5},
		{"\033[1;32mhello\033[0m", 5},
		{"\033[38;5;214m● Port 8080\033[0m", 11},
		{"plain text with spaces", 22},
		{"", 0},
	}

	for _, tt := range tests {
		got := visibleLength(tt.input)
		if got != tt.want {
			t.Errorf("visibleLength(%q) = %d; want %d", tt.input, got, tt.want)
		}
	}
}

func TestTruncateANSI(t *testing.T) {
	tests := []struct {
		input    string
		maxW     int
		expected string
	}{
		{"hello world", 5, "hello\033[0m"},
		{"hello world", 20, "hello world\033[0m"},
		{"\033[1mhello\033[0m world", 5, "\033[1mhello\033[0m\033[0m"},
		{"abc", 0, ""},
	}

	for _, tt := range tests {
		got := truncateANSI(tt.input, tt.maxW)
		if tt.maxW == 0 {
			if got != "" {
				t.Errorf("truncateANSI(%q, %d) = %q; want empty string", tt.input, tt.maxW, got)
			}
			continue
		}
		vis := visibleLength(got)
		if vis > tt.maxW {
			t.Errorf("truncateANSI(%q, %d) visible length = %d > %d", tt.input, tt.maxW, vis, tt.maxW)
		}
		if !strings.HasSuffix(got, "\033[0m") {
			t.Errorf("truncateANSI(%q, %d) missing reset suffix: %q", tt.input, tt.maxW, got)
		}
	}
}

func TestCopyToClipboard(t *testing.T) {
	// Should execute without panicking or erroring
	err := copyToClipboard("test clipboard content")
	if err != nil {
		t.Errorf("copyToClipboard returned error: %v", err)
	}
}
