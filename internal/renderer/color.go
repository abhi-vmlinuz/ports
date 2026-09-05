package renderer

import (
	"os"
)

// Theme defines ANSI escape codes for styling.
type Theme struct {
	Enabled     bool
	Reset       string
	Bold        string
	Dim         string
	Italic      string
	Cyan        string
	BrightCyan  string
	Green       string
	BrightGreen string
	Yellow      string
	Red         string
	BrightWhite string
	Gray        string
	Blue        string
	BrightBlue  string
	Magenta     string
	BrightMagenta string
}

// NewTheme creates a Theme based on TTY presence and NO_COLOR.
func NewTheme(forceDisable bool) *Theme {
	if forceDisable || !isTerminal() || os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return &Theme{Enabled: false}
	}

	return &Theme{
		Enabled:     true,
		Reset:       "\033[0m",
		Bold:        "\033[1m",
		Dim:         "\033[2m",
		Italic:      "\033[3m",
		Cyan:        "\033[36m",
		BrightCyan:  "\033[96m",
		Green:       "\033[32m",
		BrightGreen: "\033[92m",
		Yellow:      "\033[33m",
		Red:         "\033[31m",
		BrightWhite:   "\033[97m",
		Gray:          "\033[90m",
		Blue:          "\033[34m",
		BrightBlue:    "\033[94m",
		Magenta:       "\033[35m",
		BrightMagenta: "\033[95m",
	}
}

// isTerminal checks if standard output is a character device (TTY).
func isTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
