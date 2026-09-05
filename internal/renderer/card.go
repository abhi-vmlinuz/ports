package renderer

import (
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"ports/internal/model"
	"ports/internal/proc"
)

// CardRenderer formats detailed single-port inspection.
type CardRenderer struct {
	theme       *Theme
	currentUser string
	homeDir     string
}

// NewCardRenderer creates a CardRenderer.
func NewCardRenderer(theme *Theme) *CardRenderer {
	current := os.Getenv("SUDO_USER")
	home := ""

	if current != "" {
		// When run under sudo, look up home directory of the original user
		if u, err := user.Lookup(current); err == nil {
			home = u.HomeDir
		}
	}

	if current == "" {
		if u, err := user.Current(); err == nil {
			current = u.Username
			home = u.HomeDir
		} else {
			current = os.Getenv("USER")
			home = os.Getenv("HOME")
		}
	}

	return &CardRenderer{
		theme:       theme,
		currentUser: current,
		homeDir:     home,
	}
}

// Render writes detailed cards for the given port records to w.
func (cr *CardRenderer) Render(w io.Writer, records []model.PortRecord) error {
	if len(records) == 0 {
		return nil
	}

	th := cr.theme

	for i, r := range records {
		if i > 0 {
			fmt.Fprintln(w)
		}

		protoUpper := strings.ToUpper(r.Protocol)

		// Header for card
		if th.Enabled {
			badge := fmt.Sprintf("%s%s[%s]%s", th.Bold, th.BrightMagenta, protoUpper, th.Reset)
			fmt.Fprintf(w, "%s●%s %sPort %d%s  %s\n",
				th.BrightCyan, th.Reset,
				th.Bold+th.BrightWhite, r.Port, th.Reset,
				badge,
			)
		} else {
			fmt.Fprintf(w, "Port:       %d\nProtocol:   %s\n", r.Port, protoUpper)
		}

		// Interface
		var ifaceVal string
		if th.Enabled {
			if r.Address == "0.0.0.0" || r.Address == "::" {
				ifaceVal = fmt.Sprintf("%s%s%s %s(all interfaces)%s", th.Yellow, r.Address, th.Reset, th.Dim, th.Reset)
			} else if r.Address == "127.0.0.1" || r.Address == "::1" {
				ifaceVal = fmt.Sprintf("%s%s%s %s(localhost)%s", th.BrightCyan, r.Address, th.Reset, th.Dim, th.Reset)
			} else {
				ifaceVal = fmt.Sprintf("%s%s%s", th.BrightWhite, r.Address, th.Reset)
			}
		} else {
			ifaceVal = r.Address
		}
		cr.printField(w, "Interface", ifaceVal)

		// Process
		procVal := r.Process
		if procVal == "" {
			procVal = "<unknown>"
		}
		if th.Enabled {
			if strings.Contains(procVal, "permission denied") {
				procVal = fmt.Sprintf("%s%s%s%s", th.Red, th.Italic, procVal, th.Reset)
			} else {
				procVal = fmt.Sprintf("%s%s%s", th.Bold+th.BrightWhite, procVal, th.Reset)
			}
		}
		cr.printField(w, "Process", procVal)

		// PID
		var pidVal string
		if r.PID > 0 {
			if th.Enabled {
				pidVal = fmt.Sprintf("%s%d%s", th.Yellow, r.PID, th.Reset)
			} else {
				pidVal = strconv.Itoa(r.PID)
			}
		} else {
			if th.Enabled {
				pidVal = fmt.Sprintf("%s<unavailable>%s", th.Gray, th.Reset)
			} else {
				pidVal = "<unavailable>"
			}
		}
		cr.printField(w, "PID", pidVal)

		// User
		if r.User != nil && *r.User != "" {
			usrRaw := *r.User
			var usrVal string
			if cr.currentUser != "" && usrRaw == cr.currentUser {
				if th.Enabled {
					usrVal = fmt.Sprintf("%s%s%s%s %s(current user)%s", th.Bold, th.BrightGreen, usrRaw, th.Reset, th.Dim, th.Reset)
				} else {
					usrVal = fmt.Sprintf("%s (current user)", usrRaw)
				}
			} else if usrRaw == "root" {
				if th.Enabled {
					usrVal = fmt.Sprintf("%s%s%s %s(system)%s", th.Yellow, usrRaw, th.Reset, th.Dim, th.Reset)
				} else {
					usrVal = fmt.Sprintf("%s (system)", usrRaw)
				}
			} else {
				if th.Enabled {
					usrVal = fmt.Sprintf("%s%s%s", th.BrightWhite, usrRaw, th.Reset)
				} else {
					usrVal = usrRaw
				}
			}
			cr.printField(w, "User", usrVal)
		} else if r.UID != nil {
			cr.printField(w, "UID", strconv.Itoa(*r.UID))
		}

		// CWD
		if r.CWD != nil && *r.CWD != "" {
			cwdRaw := *r.CWD
			if cr.homeDir != "" && strings.HasPrefix(cwdRaw, cr.homeDir) {
				cwdRaw = "~" + strings.TrimPrefix(cwdRaw, cr.homeDir)
			}
			var cwdVal string
			if th.Enabled {
				cwdVal = fmt.Sprintf("%s%s%s", th.BrightBlue, cwdRaw, th.Reset)
			} else {
				cwdVal = cwdRaw
			}
			cr.printField(w, "CWD", cwdVal)
		}

		// Command
		if r.Command != nil && *r.Command != "" {
			var cmdVal string
			if th.Enabled {
				cmdVal = fmt.Sprintf("%s%s%s", th.BrightWhite, *r.Command, th.Reset)
			} else {
				cmdVal = *r.Command
			}
			cr.printField(w, "Command", cmdVal)
		}

		// Uptime
		if r.UptimeSeconds != nil {
			upStr := proc.FormatDuration(*r.UptimeSeconds)
			var upVal string
			if th.Enabled {
				upVal = fmt.Sprintf("%s%s%s", th.BrightGreen, upStr, th.Reset)
			} else {
				upVal = upStr
			}
			cr.printField(w, "Uptime", upVal)
		}
	}

	return nil
}

func (cr *CardRenderer) printField(w io.Writer, label, formattedValue string) {
	th := cr.theme
	if th.Enabled {
		fmt.Fprintf(w, "  %s%-10s%s %s\n", th.Cyan, label+":", th.Reset, formattedValue)
	} else {
		fmt.Fprintf(w, "%-12s%s\n", label+":", formattedValue)
	}
}

// AbbreviateHome replaces $HOME with ~ if present.
func AbbreviateHome(path, home string) string {
	if home != "" && strings.HasPrefix(path, home) {
		return filepath.Join("~", strings.TrimPrefix(path, home))
	}
	return path
}
