package renderer

import (
	"bytes"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"ports/internal/model"
	"ports/internal/proc"

	"golang.org/x/term"
)

// WatchTUI runs an interactive, split-pane terminal dashboard with live navigation,
// process inspection, mouse clicking, and safe process killing without screen flickering or wrapping.
func WatchTUI(filterPort uint16, interval time.Duration) error {
	stdoutFd := int(os.Stdout.Fd())
	stdinFd := int(os.Stdin.Fd())

	if !term.IsTerminal(stdoutFd) || !term.IsTerminal(stdinFd) {
		return fmt.Errorf("watch mode requires an interactive terminal")
	}

	oldState, err := term.MakeRaw(stdinFd)
	if err != nil {
		return fmt.Errorf("failed to enable raw terminal mode: %w", err)
	}
	defer term.Restore(stdinFd, oldState)

	// Enter alternate screen, hide cursor, enable SGR mouse tracking
	fmt.Print("\033[?1049h\033[?25l\033[?1000h\033[?1006h")
	defer fmt.Print("\033[?1000l\033[?1006l\033[?25h\033[?1049l")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	if interval <= 0 {
		interval = 1 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	discoverer := proc.NewDiscoverer("/proc")
	theme := NewTheme(false)

	currentUser := os.Getenv("SUDO_USER")
	homeDir := ""
	if currentUser != "" {
		homeDir = "/home/" + currentUser
	} else {
		currentUser = os.Getenv("USER")
		homeDir = os.Getenv("HOME")
	}

	var records []model.PortRecord
	selectedIndex := 0
	scrollOffset := 0
	confirmingKill := false
	statusMsg := ""
	statusMsgTimer := time.Time{}

	refreshData := func() {
		var err error
		if filterPort > 0 {
			records, err = discoverer.DiscoverPort(filterPort)
		} else {
			records, err = discoverer.DiscoverAll()
		}
		if err != nil {
			statusMsg = fmt.Sprintf("Error: %v", err)
			statusMsgTimer = time.Now().Add(3 * time.Second)
		}
		if selectedIndex >= len(records) {
			selectedIndex = len(records) - 1
		}
		if selectedIndex < 0 {
			selectedIndex = 0
		}
	}

	refreshData()

	draw := func() {
		width, height, err := term.GetSize(stdoutFd)
		if err != nil || width < 40 || height < 10 {
			width = 80
			height = 24
		}

		// Keep visible lines strictly within bounds to prevent auto-wrapping
		maxW := width - 1
		if maxW < 30 {
			maxW = 30
		}

		// Content rows = total height minus (header, divider, table-header, divider, footer) = height - 5
		contentHeight := height - 5
		if contentHeight < 2 {
			contentHeight = 2
		}

		splitWidth := 50
		if width > 105 {
			splitWidth = 54
		}
		rightWidth := maxW - splitWidth - 3
		isSplit := width >= 80 && rightWidth >= 22

		divider := strings.Repeat("─", maxW)
		now := time.Now().Format("15:04:05")

		var buf bytes.Buffer
		buf.WriteString("\033[H") // Reset cursor to (1,1)

		// 1. Header Bar
		title := "ports live"
		if filterPort > 0 {
			title = fmt.Sprintf("ports live :%d", filterPort)
		}
		headerRaw := fmt.Sprintf("● %s  [%s]  %d listening ports", title, now, len(records))
		var headerLine string
		if theme.Enabled {
			headerLine = fmt.Sprintf("%s%s● %s%s  %s[%s]%s  %s%d listening ports%s",
				theme.Bold, theme.BrightCyan, title, theme.Reset,
				theme.Dim, now, theme.Reset,
				theme.Dim, len(records), theme.Reset,
			)
		} else {
			headerLine = headerRaw
		}
		buf.WriteString(truncateANSI(headerLine, maxW) + "\033[K\r\n")

		// 2. Top Divider
		if theme.Enabled {
			buf.WriteString(theme.Dim + divider + theme.Reset + "\033[K\r\n")
		} else {
			buf.WriteString(divider + "\033[K\r\n")
		}

		// 3. Table Header
		colPort := 7
		colProto := 6
		colAddr := 17
		colProc := 15
		colPID := 6

		thLeft := fmt.Sprintf("  %-*s %-*s %-*s %-*s %-*s",
			colPort, "PORT",
			colProto, "PROTO",
			colAddr, "ADDRESS",
			colProc, "PROCESS",
			colPID, "PID",
		)
		if len(thLeft) > splitWidth {
			thLeft = thLeft[:splitWidth]
		}

		if isSplit {
			rightHdr := "PROCESS DETAILS"
			var thLine string
			if theme.Enabled {
				thLine = fmt.Sprintf("%s%-*s%s %s│%s %s%s%s",
					theme.Dim, splitWidth, thLeft, theme.Reset,
					theme.Dim, theme.Reset,
					theme.Bold, rightHdr, theme.Reset,
				)
			} else {
				thLine = fmt.Sprintf("%-*s │ %s", splitWidth, thLeft, rightHdr)
			}
			buf.WriteString(truncateANSI(thLine, maxW) + "\033[K\r\n")
		} else {
			if theme.Enabled {
				buf.WriteString(truncateANSI(theme.Dim+thLeft+theme.Reset, maxW) + "\033[K\r\n")
			} else {
				buf.WriteString(truncateANSI(thLeft, maxW) + "\033[K\r\n")
			}
		}

		// Scroll calculation
		if selectedIndex < scrollOffset {
			scrollOffset = selectedIndex
		}
		if selectedIndex >= scrollOffset+contentHeight {
			scrollOffset = selectedIndex - contentHeight + 1
		}
		if scrollOffset < 0 {
			scrollOffset = 0
		}

		// Prepare side panel detail lines for selected item
		var detailLines []string
		if len(records) > 0 && selectedIndex >= 0 && selectedIndex < len(records) {
			sel := records[selectedIndex]
			protoUpper := strings.ToUpper(sel.Protocol)

			if theme.Enabled {
				detailLines = append(detailLines, fmt.Sprintf("%s● Port %d%s  %s[%s]%s",
					theme.Bold+theme.BrightCyan, sel.Port, theme.Reset,
					theme.Bold+theme.BrightMagenta, protoUpper, theme.Reset))
			} else {
				detailLines = append(detailLines, fmt.Sprintf("● Port %d [%s]", sel.Port, protoUpper))
			}
			detailLines = append(detailLines, "")

			detailLines = append(detailLines, formatDetailField(theme, "Process", sel.Process))
			if sel.PID > 0 {
				detailLines = append(detailLines, formatDetailField(theme, "PID", strconv.Itoa(sel.PID)))
			} else {
				detailLines = append(detailLines, formatDetailField(theme, "PID", "<unavailable>"))
			}

			if sel.User != nil && *sel.User != "" {
				u := *sel.User
				if u == currentUser {
					u += " (you)"
				}
				detailLines = append(detailLines, formatDetailField(theme, "User", u))
			} else if sel.UID != nil {
				detailLines = append(detailLines, formatDetailField(theme, "UID", strconv.Itoa(*sel.UID)))
			}

			detailLines = append(detailLines, formatDetailField(theme, "Interface", sel.Address))

			if sel.CWD != nil && *sel.CWD != "" {
				cwd := *sel.CWD
				if homeDir != "" && strings.HasPrefix(cwd, homeDir) {
					cwd = "~" + strings.TrimPrefix(cwd, homeDir)
				}
				detailLines = append(detailLines, formatDetailField(theme, "CWD", cwd))
			}

			if sel.Command != nil && *sel.Command != "" {
				detailLines = append(detailLines, formatDetailField(theme, "Command", *sel.Command))
			}

			if sel.UptimeSeconds != nil {
				detailLines = append(detailLines, formatDetailField(theme, "Uptime", proc.FormatDuration(*sel.UptimeSeconds)))
			}
		}

		// 4. Render Content Rows (strictly contentHeight rows)
		for row := 0; row < contentHeight; row++ {
			recordIdx := scrollOffset + row
			leftText := ""

			if recordIdx < len(records) {
				r := records[recordIdx]
				isSelected := recordIdx == selectedIndex

				prefix := "  "
				if isSelected {
					prefix = "▶ "
				}

				portStr := strconv.Itoa(int(r.Port))
				addrStr := r.Address
				if len(addrStr) > colAddr {
					addrStr = addrStr[:colAddr-1] + "…"
				}
				procStr := r.Process
				if len(procStr) > colProc {
					procStr = procStr[:colProc-1] + "…"
				}
				pidStr := "-"
				if r.PID > 0 {
					pidStr = strconv.Itoa(r.PID)
				}

				if theme.Enabled {
					if isSelected {
						leftText = fmt.Sprintf("%s%s%-*s %-*s %-*s %-*s %-*s%s",
							theme.Bold+theme.BrightCyan, prefix,
							colPort, portStr,
							colProto, r.Protocol,
							colAddr, addrStr,
							colProc, procStr,
							colPID, pidStr,
							theme.Reset,
						)
					} else {
						pColor := theme.BrightWhite
						if strings.Contains(procStr, "permission denied") {
							pColor = theme.Red
						}
						uColor := theme.Reset
						if r.User != nil && *r.User == currentUser {
							uColor = theme.BrightGreen
						}

						leftText = fmt.Sprintf("  %s%-*s%s %s%-*s%s %-*s %s%-*s%s %s%-*s%s",
							theme.BrightCyan, colPort, portStr, theme.Reset,
							theme.Gray, colProto, r.Protocol, theme.Reset,
							colAddr, addrStr,
							pColor, colProc, procStr, theme.Reset,
							uColor, colPID, pidStr, theme.Reset,
						)
					}
				} else {
					leftText = fmt.Sprintf("%s%-*s %-*s %-*s %-*s %-*s",
						prefix,
						colPort, portStr,
						colProto, r.Protocol,
						colAddr, addrStr,
						colProc, procStr,
						colPID, pidStr,
					)
				}
			}

			// Format row with right pane
			leftVis := visibleLength(leftText)
			pad := splitWidth - leftVis
			if pad < 0 {
				leftText = truncateANSI(leftText, splitWidth)
				pad = 0
			}
			leftPadded := leftText + strings.Repeat(" ", pad)

			if isSplit {
				rightText := ""
				if row < len(detailLines) {
					rightText = truncateANSI(detailLines[row], rightWidth)
				}
				rowLine := fmt.Sprintf("%s │ %s", leftPadded, rightText)
				buf.WriteString(truncateANSI(rowLine, maxW) + "\033[K\r\n")
			} else {
				buf.WriteString(truncateANSI(leftPadded, maxW) + "\033[K\r\n")
			}
		}

		// 5. Bottom Divider
		if theme.Enabled {
			buf.WriteString(theme.Dim + divider + theme.Reset + "\033[K\r\n")
		} else {
			buf.WriteString(divider + "\033[K\r\n")
		}

		// 6. Footer Status Bar (Final line: NO trailing \r\n to prevent terminal scroll!)
		if confirmingKill && len(records) > 0 && selectedIndex < len(records) {
			target := records[selectedIndex]
			prompt := fmt.Sprintf("⚠️  Kill '%s' (PID %d) on port %d?  Press 'y' to confirm, 'n' to cancel",
				target.Process, target.PID, target.Port)
			if theme.Enabled {
				buf.WriteString(truncateANSI(fmt.Sprintf("%s%s%s%s", theme.Bold, theme.Yellow, prompt, theme.Reset), maxW) + "\033[K")
			} else {
				buf.WriteString(truncateANSI(prompt, maxW) + "\033[K")
			}
		} else if statusMsg != "" && time.Now().Before(statusMsgTimer) {
			if theme.Enabled {
				buf.WriteString(truncateANSI(fmt.Sprintf("%s%s%s", theme.BrightGreen, statusMsg, theme.Reset), maxW) + "\033[K")
			} else {
				buf.WriteString(truncateANSI(statusMsg, maxW) + "\033[K")
			}
		} else {
			helpText := "[↑/↓/j/k] Navigate  │  [x] Kill  │  [r] Refresh  │  [q] Quit"
			if theme.Enabled {
				buf.WriteString(truncateANSI(theme.Dim+helpText+theme.Reset, maxW) + "\033[K")
			} else {
				buf.WriteString(truncateANSI(helpText, maxW) + "\033[K")
			}
		}

		os.Stdout.Write(buf.Bytes())
	}

	draw()

	keyChan := make(chan []byte, 32)
	go func() {
		buf := make([]byte, 64)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil || n == 0 {
				return
			}
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			keyChan <- chunk
		}
	}()

	for {
		select {
		case <-sigChan:
			return nil

		case <-ticker.C:
			if !confirmingKill {
				refreshData()
				draw()
			}

		case input := <-keyChan:
			if len(input) == 0 {
				continue
			}

			// Mouse event: \x1b[<0;X;YM
			if len(input) >= 6 && input[0] == 0x1b && input[1] == '[' && input[2] == '<' {
				str := string(input[3:])
				if endIdx := strings.IndexAny(str, "Mm"); endIdx != -1 {
					params := strings.Split(str[:endIdx], ";")
					if len(params) == 3 && str[endIdx] == 'M' {
						row, _ := strconv.Atoi(params[2])
						// Row 1: Header, Row 2: Divider, Row 3: Table Header. First content row is 4.
						clickedRecord := scrollOffset + (row - 4)
						if clickedRecord >= 0 && clickedRecord < len(records) {
							selectedIndex = clickedRecord
							draw()
							continue
						}
					}
				}
			}

			// Arrow keys
			if len(input) >= 3 && input[0] == 0x1b && input[1] == '[' {
				switch input[2] {
				case 'A': // Up
					if selectedIndex > 0 {
						selectedIndex--
						draw()
					}
					continue
				case 'B': // Down
					if selectedIndex < len(records)-1 {
						selectedIndex++
						draw()
					}
					continue
				}
			}

			ch := input[0]

			if confirmingKill {
				if ch == 'y' || ch == 'Y' {
					if selectedIndex >= 0 && selectedIndex < len(records) {
						target := records[selectedIndex]
						if target.PID > 0 {
							if err := syscall.Kill(target.PID, syscall.SIGTERM); err != nil {
								statusMsg = fmt.Sprintf("Error killing PID %d: %v", target.PID, err)
							} else {
								statusMsg = fmt.Sprintf("✓ Sent SIGTERM to '%s' (PID %d)", target.Process, target.PID)
							}
						} else {
							statusMsg = "Cannot kill process: PID is unavailable or permission denied"
						}
						statusMsgTimer = time.Now().Add(3 * time.Second)
					}
					confirmingKill = false
					refreshData()
					draw()
				} else {
					confirmingKill = false
					statusMsg = "Kill cancelled"
					statusMsgTimer = time.Now().Add(2 * time.Second)
					draw()
				}
				continue
			}

			switch ch {
			case 'q', 'Q', 3, 27:
				return nil

			case 'k', 'K':
				if selectedIndex > 0 {
					selectedIndex--
					draw()
				}

			case 'j', 'J':
				if selectedIndex < len(records)-1 {
					selectedIndex++
					draw()
				}

			case 'x', 'X', 'd', 'D':
				if len(records) > 0 && selectedIndex < len(records) {
					confirmingKill = true
					draw()
				}

			case 'r', 'R':
				refreshData()
				statusMsg = "Refreshed"
				statusMsgTimer = time.Now().Add(1 * time.Second)
				draw()
			}
		}
	}
}

func formatDetailField(theme *Theme, label, val string) string {
	if theme.Enabled {
		return fmt.Sprintf("  %s%-10s%s %s", theme.Cyan, label+":", theme.Reset, val)
	}
	return fmt.Sprintf("  %-10s %s", label+":", val)
}

// truncateANSI truncates a string to at most maxWidth visible columns without cutting
// through ANSI escape sequences or leaving unterminated styles.
func truncateANSI(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	var res bytes.Buffer
	visible := 0
	inEsc := false
	var escBuf bytes.Buffer

	for _, r := range s {
		if r == '\033' {
			inEsc = true
			escBuf.WriteRune(r)
			continue
		}
		if inEsc {
			escBuf.WriteRune(r)
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEsc = false
				res.Write(escBuf.Bytes())
				escBuf.Reset()
			}
			continue
		}
		if visible < maxWidth {
			res.WriteRune(r)
			visible++
		}
	}
	res.WriteString("\033[0m")
	return res.String()
}

// visibleLength measures visible display characters, ignoring ANSI escape sequences.
func visibleLength(s string) int {
	inEsc := false
	length := 0
	for _, r := range s {
		if r == '\033' {
			inEsc = true
			continue
		}
		if inEsc {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEsc = false
			}
			continue
		}
		length++
	}
	return length
}
