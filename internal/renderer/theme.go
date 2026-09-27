package renderer

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// RGB represents an 8-bit per channel color.
type RGB struct {
	R, G, B uint8
}

// Escape returns the 24-bit TrueColor ANSI foreground escape sequence.
func (c RGB) Escape() string {
	return fmt.Sprintf("\033[38;2;%d;%d;%dm", c.R, c.G, c.B)
}

// EscapeBg returns the 24-bit TrueColor ANSI background escape sequence.
func (c RGB) EscapeBg() string {
	return fmt.Sprintf("\033[48;2;%d;%d;%dm", c.R, c.G, c.B)
}

// ThemeDefinition defines metadata and color palette for a named theme.
type ThemeDefinition struct {
	ID        string
	Name      string
	Bg        *RGB // nil = transparent terminal (kitty background), non-nil = solid theme background
	Accent    RGB  // Cyan / BrightCyan / Primary highlight
	Secondary RGB  // Blue / BrightBlue / Secondary borders
	Text      RGB  // BrightWhite / Main text
	Muted     RGB  // Gray / Inactive borders
	Comment   RGB  // Dim text
	Green     RGB  // BrightGreen / User processes / Preserved
	Red       RGB  // Red / Termination / Error / Root
	Yellow    RGB  // BrightYellow / Ports / Search highlight
	Magenta   RGB  // BrightMagenta / Badges / Origins
}

// BuildTheme constructs a Theme instance from this definition.
func (d ThemeDefinition) BuildTheme(enabled bool) *Theme {
	if !enabled {
		return &Theme{
			ID:      d.ID,
			Name:    d.Name,
			Enabled: false,
		}
	}

	bgEsc := ""
	bgReset := ""
	reset := "\033[0m"
	if d.Bg != nil {
		bgEsc = d.Bg.EscapeBg()
		bgReset = "\033[49m"
		reset = "\033[0m" + bgEsc
	}

	if d.ID == "default" {
		return &Theme{
			ID:            "default",
			Name:          "Default",
			Enabled:       true,
			Bg:            nil,
			BgEscape:      "",
			BgReset:       "",
			Reset:         "\033[0m",
			Bold:          "\033[1m",
			Dim:           "\033[2m",
			Italic:        "\033[3m",
			Cyan:          "\033[36m",
			BrightCyan:    "\033[96m",
			Green:         "\033[32m",
			BrightGreen:   "\033[92m",
			Yellow:        "\033[33m",
			BrightYellow:  "\033[93m",
			Red:           "\033[31m",
			BrightWhite:   "\033[97m",
			Gray:          "\033[90m",
			Blue:          "\033[34m",
			BrightBlue:    "\033[94m",
			Magenta:       "\033[35m",
			BrightMagenta: "\033[95m",
		}
	}

	return &Theme{
		ID:            d.ID,
		Name:          d.Name,
		Enabled:       true,
		Bg:            d.Bg,
		BgEscape:      bgEsc,
		BgReset:       bgReset,
		Reset:         reset,
		Bold:          "\033[1m",
		Dim:           "\033[2m",
		Italic:        "\033[3m",
		Cyan:          d.Accent.Escape(),
		BrightCyan:    d.Accent.Escape(),
		Green:         d.Green.Escape(),
		BrightGreen:   d.Green.Escape(),
		Yellow:        d.Yellow.Escape(),
		BrightYellow:  d.Yellow.Escape(),
		Red:           d.Red.Escape(),
		BrightWhite:   d.Text.Escape(),
		Gray:          d.Muted.Escape(),
		Blue:          d.Secondary.Escape(),
		BrightBlue:    d.Secondary.Escape(),
		Magenta:       d.Magenta.Escape(),
		BrightMagenta: d.Magenta.Escape(),
	}
}

// AvailableThemes contains all 16 palettes inspired by rinode.
var AvailableThemes = []ThemeDefinition{
	{
		ID:        "default",
		Name:      "Default (Terminal)",
		Bg:        nil,
		Accent:    RGB{0, 215, 255},
		Secondary: RGB{0, 175, 255},
		Text:      RGB{255, 255, 255},
		Muted:     RGB{128, 128, 128},
		Comment:   RGB{100, 100, 100},
		Green:     RGB{0, 255, 128},
		Red:       RGB{255, 85, 85},
		Yellow:    RGB{255, 215, 0},
		Magenta:   RGB{215, 135, 255},
	},
	{
		ID:        "catppuccin",
		Name:      "Catppuccin Mocha",
		Bg:        &RGB{30, 30, 46},
		Accent:    RGB{203, 166, 247}, // Mauve
		Secondary: RGB{137, 220, 235}, // Sky
		Text:      RGB{205, 214, 244}, // Text
		Muted:     RGB{88, 91, 112},   // Surface2
		Comment:   RGB{147, 153, 178}, // Overlay2
		Green:     RGB{166, 227, 161}, // Green
		Red:       RGB{243, 139, 168}, // Red
		Yellow:    RGB{249, 226, 175}, // Yellow
		Magenta:   RGB{245, 194, 231}, // Pink
	},
	{
		ID:        "nord",
		Name:      "Nord",
		Bg:        &RGB{46, 52, 64},
		Accent:    RGB{136, 192, 208}, // Frost Cyan
		Secondary: RGB{129, 161, 193}, // Frost Blue
		Text:      RGB{236, 239, 244}, // Snow White
		Muted:     RGB{67, 76, 94},    // Nord2
		Comment:   RGB{94, 129, 172},  // Nord3
		Green:     RGB{163, 190, 140}, // Green
		Red:       RGB{191, 97, 106},  // Red
		Yellow:    RGB{235, 203, 139}, // Yellow
		Magenta:   RGB{180, 142, 173}, // Purple
	},
	{
		ID:        "dracula",
		Name:      "Dracula",
		Bg:        &RGB{40, 42, 54},
		Accent:    RGB{189, 147, 249}, // Purple
		Secondary: RGB{139, 233, 253}, // Cyan
		Text:      RGB{248, 248, 242}, // Foreground
		Muted:     RGB{98, 114, 164},  // Comment
		Comment:   RGB{98, 114, 164},
		Green:     RGB{80, 250, 123},  // Green
		Red:       RGB{255, 85, 85},   // Red
		Yellow:    RGB{241, 250, 140}, // Yellow
		Magenta:   RGB{255, 121, 198}, // Pink
	},
	{
		ID:        "gruvbox",
		Name:      "Gruvbox Dark",
		Bg:        &RGB{40, 40, 40},
		Accent:    RGB{254, 128, 25},  // Bright Orange
		Secondary: RGB{131, 165, 152}, // Blue
		Text:      RGB{235, 219, 178}, // Foreground
		Muted:     RGB{102, 92, 84},   // Dark Gray
		Comment:   RGB{146, 131, 116}, // Gray
		Green:     RGB{184, 187, 38},  // Green
		Red:       RGB{251, 73, 52},   // Red
		Yellow:    RGB{250, 189, 47},  // Yellow
		Magenta:   RGB{211, 134, 155}, // Purple
	},
	{
		ID:        "tokyo_night",
		Name:      "Tokyo Night",
		Bg:        &RGB{26, 27, 38},
		Accent:    RGB{187, 154, 247}, // Magenta
		Secondary: RGB{125, 207, 255}, // Cyan
		Text:      RGB{192, 202, 245}, // Text
		Muted:     RGB{65, 72, 104},   // Border
		Comment:   RGB{86, 95, 137},   // Comment
		Green:     RGB{158, 206, 106}, // Green
		Red:       RGB{247, 118, 142}, // Red
		Yellow:    RGB{224, 175, 104}, // Yellow
		Magenta:   RGB{187, 154, 247}, // Magenta
	},
	{
		ID:        "rose_pine",
		Name:      "Rosé Pine",
		Bg:        &RGB{25, 23, 36},
		Accent:    RGB{235, 111, 146}, // Rose
		Secondary: RGB{156, 207, 216}, // Foam
		Text:      RGB{224, 222, 244}, // Text
		Muted:     RGB{110, 106, 134}, // Muted
		Comment:   RGB{144, 140, 170}, // Subtle
		Green:     RGB{49, 116, 143},  // Pine
		Red:       RGB{235, 111, 146}, // Love
		Yellow:    RGB{246, 193, 119}, // Gold
		Magenta:   RGB{196, 167, 231}, // Iris
	},
	{
		ID:        "one_dark",
		Name:      "One Dark",
		Bg:        &RGB{40, 44, 52},
		Accent:    RGB{97, 175, 239},  // Blue
		Secondary: RGB{198, 120, 221}, // Purple
		Text:      RGB{171, 178, 191}, // Text
		Muted:     RGB{75, 82, 99},    // Muted
		Comment:   RGB{92, 99, 112},   // Comment
		Green:     RGB{152, 195, 121}, // Green
		Red:       RGB{224, 108, 117}, // Red
		Yellow:    RGB{229, 192, 123}, // Yellow
		Magenta:   RGB{198, 120, 221}, // Purple
	},
	{
		ID:        "monokai",
		Name:      "Monokai Pro",
		Bg:        &RGB{45, 42, 46},
		Accent:    RGB{255, 97, 136},  // Pink
		Secondary: RGB{120, 220, 232}, // Cyan
		Text:      RGB{252, 252, 250}, // Text
		Muted:     RGB{114, 112, 114}, // Muted
		Comment:   RGB{147, 146, 147}, // Comment
		Green:     RGB{169, 220, 106}, // Green
		Red:       RGB{255, 97, 136},  // Red
		Yellow:    RGB{255, 216, 102}, // Yellow
		Magenta:   RGB{171, 157, 242}, // Purple
	},
	{
		ID:        "kanagawa",
		Name:      "Kanagawa",
		Bg:        &RGB{31, 31, 40},
		Accent:    RGB{155, 206, 180}, // Wave Aqua
		Secondary: RGB{126, 156, 216}, // Crystal Blue
		Text:      RGB{220, 223, 228}, // Fuji White
		Muted:     RGB{84, 84, 109},   // Sumi Ink
		Comment:   RGB{114, 113, 133}, // Fuji Gray
		Green:     RGB{152, 187, 108}, // Spring Green
		Red:       RGB{232, 134, 138}, // Autumn Red
		Yellow:    RGB{230, 195, 132}, // Carp Yellow
		Magenta:   RGB{149, 127, 184}, // Oni Violet
	},
	{
		ID:        "cyberpunk",
		Name:      "Cyberpunk Neon",
		Bg:        &RGB{20, 16, 38},
		Accent:    RGB{255, 0, 127},   // Neon Pink
		Secondary: RGB{0, 240, 255},   // Neon Cyan
		Text:      RGB{240, 240, 255}, // Bright Text
		Muted:     RGB{70, 50, 95},    // Muted
		Comment:   RGB{115, 80, 155},  // Comment
		Green:     RGB{57, 255, 20},   // Neon Lime
		Red:       RGB{255, 49, 49},   // Neon Red
		Yellow:    RGB{255, 238, 0},   // Neon Yellow
		Magenta:   RGB{255, 0, 127},   // Neon Pink
	},
	{
		ID:        "everforest",
		Name:      "Everforest",
		Bg:        &RGB{43, 51, 57},
		Accent:    RGB{167, 192, 128}, // Green
		Secondary: RGB{127, 187, 179}, // Aqua
		Text:      RGB{211, 198, 170}, // Fg
		Muted:     RGB{79, 88, 94},    // Bg4
		Comment:   RGB{133, 146, 137}, // Grey
		Green:     RGB{167, 192, 128}, // Green
		Red:       RGB{230, 126, 128}, // Red
		Yellow:    RGB{219, 188, 127}, // Yellow
		Magenta:   RGB{211, 134, 155}, // Purple
	},
	{
		ID:        "ayu",
		Name:      "Ayu Dark",
		Bg:        &RGB{15, 20, 25},
		Accent:    RGB{255, 180, 84}, // Gold
		Secondary: RGB{57, 186, 230},  // Cyan
		Text:      RGB{203, 204, 198}, // Fg
		Muted:     RGB{36, 42, 54},    // Bg
		Comment:   RGB{92, 103, 115},  // Comment
		Green:     RGB{170, 217, 76},  // Green
		Red:       RGB{240, 113, 120}, // Red
		Yellow:    RGB{255, 180, 84},  // Yellow
		Magenta:   RGB{210, 166, 255}, // Purple
	},
	{
		ID:        "synthwave",
		Name:      "Synthwave '84",
		Bg:        &RGB{38, 25, 60},
		Accent:    RGB{254, 68, 153},  // Neon Pink
		Secondary: RGB{249, 126, 240}, // Neon Violet
		Text:      RGB{240, 238, 255}, // Bright Text
		Muted:     RGB{63, 40, 97},    // Muted
		Comment:   RGB{110, 80, 150},  // Comment
		Green:     RGB{114, 241, 184}, // Neon Teal
		Red:       RGB{254, 68, 68},   // Red
		Yellow:    RGB{254, 235, 100}, // Yellow
		Magenta:   RGB{249, 126, 240}, // Neon Violet
	},
	{
		ID:        "solarized",
		Name:      "Solarized Dark",
		Bg:        &RGB{0, 43, 54},
		Accent:    RGB{42, 161, 152},  // Cyan
		Secondary: RGB{38, 139, 210},  // Blue
		Text:      RGB{238, 232, 213}, // Base2
		Muted:     RGB{88, 110, 117},  // Base01
		Comment:   RGB{101, 123, 131}, // Base00
		Green:     RGB{133, 153, 0},   // Green
		Red:       RGB{220, 50, 47},   // Red
		Yellow:    RGB{181, 137, 0},   // Yellow
		Magenta:   RGB{211, 54, 130},  // Magenta
	},
	{
		ID:        "matrix",
		Name:      "Matrix",
		Bg:        &RGB{10, 15, 10},
		Accent:    RGB{0, 255, 65},    // Phosphor Green
		Secondary: RGB{0, 204, 51},    // Medium Green
		Text:      RGB{180, 255, 180}, // Text
		Muted:     RGB{20, 35, 20},    // Muted
		Comment:   RGB{40, 80, 40},    // Comment
		Green:     RGB{0, 255, 65},    // Green
		Red:       RGB{255, 65, 65},   // Red
		Yellow:    RGB{200, 255, 0},   // Yellow-Green
		Magenta:   RGB{0, 255, 65},    // Green
	},
}

// FindTheme finds a theme definition by ID or name (case-insensitive).
func FindTheme(idOrName string) ThemeDefinition {
	cleaned := strings.ToLower(strings.TrimSpace(idOrName))
	cleaned = strings.ReplaceAll(cleaned, "-", "_")
	cleaned = strings.ReplaceAll(cleaned, " ", "_")

	for _, t := range AvailableThemes {
		if t.ID == cleaned || strings.ToLower(t.Name) == strings.ToLower(idOrName) {
			return t
		}
	}
	return AvailableThemes[0] // fallback to default
}

// IsValidTheme checks if a theme exists by ID or name.
func IsValidTheme(idOrName string) bool {
	cleaned := strings.ToLower(strings.TrimSpace(idOrName))
	cleaned = strings.ReplaceAll(cleaned, "-", "_")
	cleaned = strings.ReplaceAll(cleaned, " ", "_")

	for _, t := range AvailableThemes {
		if t.ID == cleaned || strings.ToLower(t.Name) == strings.ToLower(idOrName) {
			return true
		}
	}
	return false
}

// CycleTheme returns the next theme in the palette sequence.
func CycleTheme(currentID string) ThemeDefinition {
	cleaned := strings.ToLower(strings.TrimSpace(currentID))
	cleaned = strings.ReplaceAll(cleaned, "-", "_")
	cleaned = strings.ReplaceAll(cleaned, " ", "_")

	idx := 0
	for i, t := range AvailableThemes {
		if t.ID == cleaned {
			idx = i
			break
		}
	}

	nextIdx := (idx + 1) % len(AvailableThemes)
	return AvailableThemes[nextIdx]
}

// PrintAvailableThemes lists all available themes with color swatches.
func PrintAvailableThemes(w io.Writer, activeThemeID string) {
	isTerm := IsTerminal() && os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "dumb"
	activeCleaned := strings.ToLower(strings.TrimSpace(activeThemeID))
	activeCleaned = strings.ReplaceAll(activeCleaned, "-", "_")
	activeCleaned = strings.ReplaceAll(activeCleaned, " ", "_")

	fmt.Fprintln(w, "Available Themes:")
	for _, t := range AvailableThemes {
		isActive := t.ID == activeCleaned
		mark := "  "
		suffix := ""
		if isActive {
			mark = "▶ "
			suffix = " (active)"
		}

		if isTerm {
			swatches := fmt.Sprintf("%s■%s %s■%s %s■%s %s■%s %s■%s",
				t.Accent.Escape(), "\033[0m",
				t.Secondary.Escape(), "\033[0m",
				t.Green.Escape(), "\033[0m",
				t.Yellow.Escape(), "\033[0m",
				t.Red.Escape(), "\033[0m",
			)
			if isActive {
				fmt.Fprintf(w, "\033[1m%s%-12s\033[0m  %-20s  %s\033[1;32m%s\033[0m\n",
					mark, t.ID, t.Name, swatches, suffix)
			} else {
				fmt.Fprintf(w, "%s%-12s  %-20s  %s\n",
					mark, t.ID, t.Name, swatches)
			}
		} else {
			fmt.Fprintf(w, "%s%-12s  %-20s%s\n", mark, t.ID, t.Name, suffix)
		}
	}
}
