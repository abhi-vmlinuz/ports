package renderer

import (
	"os"

	"ports/internal/config"
)

// Theme defines ANSI escape codes for styling.
type Theme struct {
	ID            string
	Name          string
	Enabled       bool
	Bg            *RGB
	BgEscape      string
	BgReset       string
	Reset         string
	Bold          string
	Dim           string
	Italic        string
	Cyan          string
	BrightCyan    string
	Green         string
	BrightGreen   string
	Yellow        string
	BrightYellow  string
	Red           string
	BrightWhite   string
	Gray          string
	Blue          string
	BrightBlue    string
	Magenta       string
	BrightMagenta string
}

// NewTheme creates a Theme based on TTY presence, NO_COLOR, and user/flag selection.
func NewTheme(forceDisable bool, themeID ...string) *Theme {
	var requestedTheme string
	if len(themeID) > 0 && themeID[0] != "" {
		requestedTheme = themeID[0]
	} else {
		cfg := config.Load()
		requestedTheme = cfg.Theme
	}

	themeDef := FindTheme(requestedTheme)

	if forceDisable || !IsTerminal() || os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return themeDef.BuildTheme(false)
	}

	return themeDef.BuildTheme(true)
}

// IsTerminal checks if standard output is a character device (TTY).
func IsTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
