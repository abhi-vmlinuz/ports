package renderer

import (
	"fmt"
	"io"
	"os"
	"os/user"
	"strconv"
	"strings"

	"ports/internal/model"
)

// TableRenderer formats PortRecords into a beautiful terminal table.
type TableRenderer struct {
	theme       *Theme
	currentUser string
}

// NewTableRenderer creates a TableRenderer.
func NewTableRenderer(theme *Theme) *TableRenderer {
	current := os.Getenv("SUDO_USER")
	if current == "" {
		if u, err := user.Current(); err == nil && u.Username != "" {
			current = u.Username
		} else {
			current = os.Getenv("USER")
		}
	}

	return &TableRenderer{
		theme:       theme,
		currentUser: current,
	}
}

// Render writes the formatted table to w.
func (tr *TableRenderer) Render(w io.Writer, records []model.PortRecord) error {
	if len(records) == 0 {
		if tr.theme.Enabled {
			fmt.Fprintln(w, tr.theme.Dim+"No listening ports found."+tr.theme.Reset)
		} else {
			fmt.Fprintln(w, "No listening ports found.")
		}
		return nil
	}

	headers := []string{"PORT", "PROTO", "ADDRESS", "PROCESS", "PID", "USER"}
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}

	// Calculate column widths based on unformatted content
	type rowData struct {
		port    string
		proto   string
		addr    string
		proc    string
		pid     string
		usr     string
		isUser  bool
		isRoot  bool
		isPerm  bool
	}

	rows := make([]rowData, len(records))
	userCount := 0
	systemCount := 0
	hasPermissionDenied := false

	for i, r := range records {
		portStr := strconv.Itoa(int(r.Port))
		protoStr := r.Protocol
		addrStr := r.Address
		procStr := r.Process
		if procStr == "" {
			procStr = "<unknown>"
		}

		pidStr := "-"
		if r.PID > 0 {
			pidStr = strconv.Itoa(r.PID)
		}

		usrStr := "-"
		if r.User != nil && *r.User != "" {
			usrStr = *r.User
		}

		isUser := tr.currentUser != "" && usrStr == tr.currentUser
		isRoot := usrStr == "root"
		isPerm := strings.Contains(procStr, "permission denied")

		if isPerm {
			hasPermissionDenied = true
		}

		if isUser {
			userCount++
		} else {
			systemCount++
		}

		rows[i] = rowData{
			port:   portStr,
			proto:  protoStr,
			addr:   addrStr,
			proc:   procStr,
			pid:    pidStr,
			usr:    usrStr,
			isUser: isUser,
			isRoot: isRoot,
			isPerm: isPerm,
		}

		if len(portStr) > widths[0] {
			widths[0] = len(portStr)
		}
		if len(protoStr) > widths[1] {
			widths[1] = len(protoStr)
		}
		if len(addrStr) > widths[2] {
			widths[2] = len(addrStr)
		}
		if len(procStr) > widths[3] {
			widths[3] = len(procStr)
		}
		if len(pidStr) > widths[4] {
			widths[4] = len(pidStr)
		}
		if len(usrStr) > widths[5] {
			widths[5] = len(usrStr)
		}
	}

	// Print Header
	th := tr.theme
	if th.Enabled {
		fmt.Fprintf(w, "%s%-*s  %-*s  %-*s  %-*s  %-*s  %-*s%s\n",
			th.Dim,
			widths[0], headers[0],
			widths[1], headers[1],
			widths[2], headers[2],
			widths[3], headers[3],
			widths[4], headers[4],
			widths[5], headers[5],
			th.Reset,
		)
	} else {
		fmt.Fprintf(w, "%-*s  %-*s  %-*s  %-*s  %-*s  %-*s\n",
			widths[0], headers[0],
			widths[1], headers[1],
			widths[2], headers[2],
			widths[3], headers[3],
			widths[4], headers[4],
			widths[5], headers[5],
		)
	}

	// Print Rows
	for _, row := range rows {
		if !th.Enabled {
			fmt.Fprintf(w, "%-*s  %-*s  %-*s  %-*s  %-*s  %-*s\n",
				widths[0], row.port,
				widths[1], row.proto,
				widths[2], row.addr,
				widths[3], row.proc,
				widths[4], row.pid,
				widths[5], row.usr,
			)
			continue
		}

		// Styled output
		portFormatted := th.BrightCyan + fmt.Sprintf("%-*s", widths[0], row.port) + th.Reset
		protoFormatted := th.Gray + fmt.Sprintf("%-*s", widths[1], row.proto) + th.Reset

		var addrFormatted string
		if row.addr == "127.0.0.1" || row.addr == "::1" {
			addrFormatted = fmt.Sprintf("%-*s", widths[2], row.addr)
		} else {
			addrFormatted = th.Gray + fmt.Sprintf("%-*s", widths[2], row.addr) + th.Reset
		}

		var procFormatted string
		if row.isPerm {
			procFormatted = th.Red + th.Italic + fmt.Sprintf("%-*s", widths[3], row.proc) + th.Reset
		} else {
			procFormatted = th.BrightWhite + fmt.Sprintf("%-*s", widths[3], row.proc) + th.Reset
		}

		var pidFormatted string
		if row.pid == "-" {
			pidFormatted = th.Gray + fmt.Sprintf("%-*s", widths[4], row.pid) + th.Reset
		} else {
			pidFormatted = fmt.Sprintf("%-*s", widths[4], row.pid)
		}

		var usrFormatted string
		if row.isUser {
			usrFormatted = th.BrightGreen + fmt.Sprintf("%-*s", widths[5], row.usr) + th.Reset
		} else if row.isRoot {
			usrFormatted = th.Yellow + fmt.Sprintf("%-*s", widths[5], row.usr) + th.Reset
		} else if row.usr == "-" {
			usrFormatted = th.Gray + fmt.Sprintf("%-*s", widths[5], row.usr) + th.Reset
		} else {
			usrFormatted = fmt.Sprintf("%-*s", widths[5], row.usr)
		}

		fmt.Fprintf(w, "%s  %s  %s  %s  %s  %s\n",
			portFormatted, protoFormatted, addrFormatted, procFormatted, pidFormatted, usrFormatted)
	}

	// Bottom Context Summary (only on TTY)
	if th.Enabled {
		portWord := "ports"
		if len(records) == 1 {
			portWord = "port"
		}
		fmt.Fprintf(w, "\n%s● %d listening %s (%d user, %d system)%s\n",
			th.Dim, len(records), portWord, userCount, systemCount, th.Reset)

		if hasPermissionDenied {
			fmt.Fprintf(os.Stderr, "%shint: some process metadata could not be read; run with sudo to inspect all processes%s\n",
				th.Dim, th.Reset)
		}
	}

	return nil
}
