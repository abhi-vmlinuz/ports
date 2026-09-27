package renderer

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"ports/internal/config"
	"ports/internal/model"
	"ports/internal/proc"

	"golang.org/x/term"
)

// WatchTUI runs an interactive, split-pane terminal dashboard with live navigation,
// process inspection, mouse clicking, popup action menus, clipboard export, and safe killing.
func WatchTUI(filterPort uint16, interval time.Duration, themeID ...string) error {
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

	discoverer := proc.NewDiscoverer("/proc")

	var initialTheme string
	if len(themeID) > 0 && themeID[0] != "" {
		initialTheme = themeID[0]
	} else {
		cfg := config.Load()
		initialTheme = cfg.Theme
	}
	currentThemeDef := FindTheme(initialTheme)
	theme := currentThemeDef.BuildTheme(true)

	// Enter alternate screen, apply theme background, clear screen, hide cursor, enable SGR mouse tracking
	if theme.BgEscape != "" {
		fmt.Printf("\033[?1049h%s\033[2J\033[H\033[?25l\033[?1000h\033[?1006h", theme.BgEscape)
	} else {
		fmt.Printf("\033[?1049h\033[49m\033[2J\033[H\033[?25l\033[?1000h\033[?1006h")
	}
	defer fmt.Print("\033[?1000l\033[?1006l\033[0m\033[49m\033[2J\033[?25h\033[?1049l")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGWINCH)
	defer signal.Stop(sigChan)

	if interval <= 0 {
		interval = 1 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	currentUser := os.Getenv("SUDO_USER")
	homeDir := ""
	if currentUser != "" {
		homeDir = "/home/" + currentUser
	} else {
		currentUser = os.Getenv("USER")
		homeDir = os.Getenv("HOME")
	}

	var records []model.PortRecord
	var filteredRecords []model.PortRecord
	searchQuery := ""
	searchActive := false

	selectedIndex := 0
	scrollOffset := 0

	// Menu & Kill states
	showActionMenu := false
	menuSelection := 0
	confirmingKill := false
	killSignal := syscall.SIGTERM

	statusMsg := ""
	statusMsgTimer := time.Time{}

	applyFilter := func() {
		filteredRecords = FilterRecords(records, searchQuery)
		if selectedIndex >= len(filteredRecords) {
			selectedIndex = len(filteredRecords) - 1
		}
		if selectedIndex < 0 {
			selectedIndex = 0
		}
		if selectedIndex < scrollOffset {
			scrollOffset = selectedIndex
		}
	}

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
		applyFilter()
	}

	refreshData()

	menuItems := []string{
		"1. Kill process (SIGTERM)",
		"2. Force kill (SIGKILL)",
		"3. Copy JSON to clipboard",
		"4. Copy PID",
		"5. Copy Command",
		"6. Copy Address (IP:Port)",
	}

	// Layout coordinates for mouse hit-testing
	boxW := 50
	boxH := len(menuItems) + 3
	menuStartY := 5
	menuStartX := 15
	contentHeight := 15
	splitWidth := 50
	isSplit := true

	draw := func() {
		width, height, err := term.GetSize(stdoutFd)
		if err != nil || width < 40 || height < 10 {
			width = 80
			height = 24
		}

		userCount := 0
		for _, r := range records {
			if r.User != nil && *r.User == currentUser {
				userCount++
			}
		}
		systemCount := len(records) - userCount

		// Non-content rows: Header (1) + Divider (1) + SearchBox (3) + TableHeader (1) + Divider (1) + Footer (1) = 8
		contentHeight = height - 8
		if contentHeight < 2 {
			contentHeight = 2
		}

		splitWidth = width * 54 / 100
		if splitWidth < 46 {
			splitWidth = 46
		}
		rightWidth := width - splitWidth - 3
		isSplit = width >= 80 && rightWidth >= 22

		if !isSplit {
			splitWidth = width
			rightWidth = 0
		}

		// Responsive columns adapting to available splitWidth
		colPort := 7
		colProto := 6
		colPID := 7
		rem := splitWidth - (colPort + colProto + colPID + 4)
		if rem < 20 {
			rem = 20
		}
		colAddr := rem * 44 / 100
		if colAddr < 15 {
			colAddr = 15
		}
		colProc := rem - colAddr
		if colProc < 12 {
			colProc = 12
		}

		divider := strings.Repeat("─", width)
		now := time.Now().Format("15:04:05")

		var buf bytes.Buffer
		buf.WriteString("\033[H") // Reset cursor to (1,1)

		currentLine := 0
		writeRow := func(content string) {
			if currentLine >= height {
				return
			}
			vis := visibleLength(content)
			pad := width - vis
			var lineStr string
			if pad < 0 {
				lineStr = truncateANSI(content, width, theme.Reset)
			} else {
				lineStr = content + strings.Repeat(" ", pad)
			}

			if theme.BgEscape != "" {
				buf.WriteString(theme.BgEscape)
			}
			buf.WriteString(lineStr)
			buf.WriteString("\033[K")
			if theme.BgEscape != "" {
				buf.WriteString(theme.BgReset)
			}
			if currentLine < height-1 {
				buf.WriteString("\r\n")
			}
			currentLine++
		}

		// 1. Header Bar with Theme indicator and user/system counts
		title := "ports live"
		if filterPort > 0 {
			title = fmt.Sprintf("ports live :%d", filterPort)
		}
		var headerLine string
		if theme.Enabled {
			headerLine = fmt.Sprintf("%s%s● %s%s  %s[%s]%s  %s%d listening%s  %s(%d user, %d system)%s  %s[t] Theme: %s%s",
				theme.Bold, theme.BrightCyan, title, theme.Reset,
				theme.Dim, now, theme.Reset,
				theme.Bold, len(records), theme.Reset,
				theme.Dim, userCount, systemCount, theme.Reset,
				theme.Bold+theme.BrightMagenta, theme.Name, theme.Reset,
			)
		} else {
			headerLine = fmt.Sprintf("● %s  [%s]  %d listening (%d user, %d system)  [t] Theme: %s",
				title, now, len(records), userCount, systemCount, theme.Name)
		}
		writeRow(headerLine)

		// 2. Top Divider
		if theme.Enabled {
			writeRow(theme.Dim + divider + theme.Reset)
		} else {
			writeRow(divider)
		}

		// 2.5 Search Bar Box
		searchBoxTitle := " Search [/] "
		if searchActive {
			searchBoxTitle = " Search "
		}
		matchBadge := ""
		if searchQuery != "" {
			matchBadge = fmt.Sprintf(" (%d matches) ", len(filteredRecords))
		}
		tLen := visibleLength(searchBoxTitle)
		mLen := visibleLength(matchBadge)
		fillLen := width - 2 - tLen - mLen
		if fillLen < 0 {
			fillLen = 0
		}
		sbTop := "┌" + searchBoxTitle + strings.Repeat("─", fillLen) + matchBadge + "┐"
		if theme.Enabled {
			if searchActive {
				writeRow(theme.Bold + theme.BrightCyan + sbTop + theme.Reset)
			} else {
				writeRow(theme.Dim + sbTop + theme.Reset)
			}
		} else {
			writeRow(sbTop)
		}

		var promptText string
		if searchActive {
			promptText = fmt.Sprintf("> %s█", searchQuery)
		} else if searchQuery != "" {
			promptText = fmt.Sprintf("> %s", searchQuery)
		} else {
			promptText = "Press / to search by port or process name..."
		}
		pVis := visibleLength(promptText)
		pPad := width - 4 - pVis
		if pPad < 0 {
			pPad = 0
		}
		if theme.Enabled {
			bColor := theme.Dim
			if searchActive {
				bColor = theme.BrightCyan
			}
			var pStyled string
			if searchActive {
				pStyled = theme.Bold + theme.BrightGreen + "> " + theme.Reset + theme.Bold + theme.BrightWhite + searchQuery + theme.BrightCyan + "█" + theme.Reset
			} else if searchQuery != "" {
				pStyled = theme.BrightCyan + "> " + theme.BrightWhite + searchQuery + theme.Reset
			} else {
				pStyled = theme.Dim + promptText + theme.Reset
			}
			writeRow(fmt.Sprintf("%s│%s %s%s %s│%s", bColor, theme.Reset, pStyled, strings.Repeat(" ", pPad), bColor, theme.Reset))
		} else {
			writeRow(fmt.Sprintf("│ %s%s │", promptText, strings.Repeat(" ", pPad)))
		}

		sbBot := "└" + strings.Repeat("─", width-2) + "┘"
		if theme.Enabled {
			if searchActive {
				writeRow(theme.Bold + theme.BrightCyan + sbBot + theme.Reset)
			} else {
				writeRow(theme.Dim + sbBot + theme.Reset)
			}
		} else {
			writeRow(sbBot)
		}

		// 3. Table Header
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
			writeRow(thLine)
		} else {
			scrollInfo := ""
			if len(filteredRecords) > contentHeight {
				scrollInfo = fmt.Sprintf(" [%d/%d]", selectedIndex+1, len(filteredRecords))
			}
			if theme.Enabled {
				writeRow(theme.Dim + thLeft + theme.Reset + theme.Bold + theme.BrightCyan + scrollInfo + theme.Reset)
			} else {
				writeRow(thLeft + scrollInfo)
			}
		}

		// Robust scroll calculation & clamping
		if selectedIndex < 0 {
			selectedIndex = 0
		}
		if selectedIndex >= len(filteredRecords) && len(filteredRecords) > 0 {
			selectedIndex = len(filteredRecords) - 1
		}

		if len(filteredRecords) <= contentHeight {
			scrollOffset = 0
		} else {
			if selectedIndex < scrollOffset {
				scrollOffset = selectedIndex
			}
			if selectedIndex >= scrollOffset+contentHeight {
				scrollOffset = selectedIndex - contentHeight + 1
			}
			maxScroll := len(filteredRecords) - contentHeight
			if scrollOffset > maxScroll {
				scrollOffset = maxScroll
			}
			if scrollOffset < 0 {
				scrollOffset = 0
			}
		}

		// Prepare side panel detail lines for selected item
		var detailLines []string
		if len(filteredRecords) > 0 && selectedIndex >= 0 && selectedIndex < len(filteredRecords) {
			sel := filteredRecords[selectedIndex]
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

			if sel.Origin != nil && *sel.Origin != "" {
				detailLines = append(detailLines, formatDetailField(theme, "Origin", *sel.Origin))
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

		// Scrollbar metrics on vertical divider
		thumbH := 1
		thumbY := 0
		if len(filteredRecords) > contentHeight {
			thumbH = (contentHeight * contentHeight) / len(filteredRecords)
			if thumbH < 1 {
				thumbH = 1
			}
			thumbY = (scrollOffset * (contentHeight - thumbH)) / (len(filteredRecords) - contentHeight)
		}

		// 4. Render Content Rows
		for row := 0; row < contentHeight; row++ {
			recordIdx := scrollOffset + row
			leftText := ""

			if recordIdx < len(filteredRecords) {
				r := filteredRecords[recordIdx]
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

						portHighlighted := HighlightMatches(portStr, searchQuery, theme, theme.BrightCyan)
						procHighlighted := HighlightMatches(procStr, searchQuery, theme, pColor)

						portPad := colPort - visibleLength(portHighlighted)
						if portPad < 0 {
							portPad = 0
						}
						procPad := colProc - visibleLength(procHighlighted)
						if procPad < 0 {
							procPad = 0
						}

						leftText = fmt.Sprintf("  %s%s %s%-*s%s %-*s %s%s %s%-*s%s",
							portHighlighted, strings.Repeat(" ", portPad),
							theme.Gray, colProto, r.Protocol, theme.Reset,
							colAddr, addrStr,
							procHighlighted, strings.Repeat(" ", procPad),
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
			} else if len(filteredRecords) == 0 && row == 0 {
				if searchQuery != "" {
					if theme.Enabled {
						leftText = fmt.Sprintf("  %sNo matching ports or processes for '%s'%s", theme.Dim, searchQuery, theme.Reset)
					} else {
						leftText = fmt.Sprintf("  No matching ports or processes for '%s'", searchQuery)
					}
				} else {
					if theme.Enabled {
						leftText = fmt.Sprintf("  %sNo listening ports%s", theme.Dim, theme.Reset)
					} else {
						leftText = "  No listening ports"
					}
				}
			}

			leftVis := visibleLength(leftText)
			pad := splitWidth - leftVis
			if pad < 0 {
				leftText = truncateANSI(leftText, splitWidth, theme.Reset)
				pad = 0
			}
			leftPadded := leftText + strings.Repeat(" ", pad)

			if isSplit {
				rightText := ""
				if row < len(detailLines) {
					rightText = truncateANSI(detailLines[row], rightWidth, theme.Reset)
				}
				divChar := "│"
				if len(filteredRecords) > contentHeight {
					if row >= thumbY && row < thumbY+thumbH {
						if theme.Enabled {
							divChar = theme.Bold + theme.BrightCyan + "█" + theme.Reset
						} else {
							divChar = "█"
						}
					} else {
						if theme.Enabled {
							divChar = theme.Dim + "│" + theme.Reset
						} else {
							divChar = "│"
						}
					}
				} else {
					if theme.Enabled {
						divChar = theme.Dim + "│" + theme.Reset
					} else {
						divChar = "│"
					}
				}
				rowLine := fmt.Sprintf("%s %s %s", leftPadded, divChar, rightText)
				writeRow(rowLine)
			} else {
				writeRow(leftPadded)
			}
		}

		// 5. Bottom Divider
		if theme.Enabled {
			writeRow(theme.Dim + divider + theme.Reset)
		} else {
			writeRow(divider)
		}

		// 6. Footer Status Bar
		var helpLine string
		if confirmingKill && len(filteredRecords) > 0 && selectedIndex < len(filteredRecords) {
			target := filteredRecords[selectedIndex]
			sigName := "SIGTERM"
			if killSignal == syscall.SIGKILL {
				sigName = "SIGKILL"
			}
			prompt := fmt.Sprintf("⚠️  Kill '%s' (PID %d) with %s?  Press 'y' to confirm, 'n' to cancel",
				target.Process, target.PID, sigName)
			if theme.Enabled {
				helpLine = fmt.Sprintf("%s%s%s%s", theme.Bold, theme.Yellow, prompt, theme.Reset)
			} else {
				helpLine = prompt
			}
		} else if statusMsg != "" && time.Now().Before(statusMsgTimer) {
			if theme.Enabled {
				helpLine = fmt.Sprintf("%s✓ %s%s", theme.Bold+theme.BrightGreen, statusMsg, theme.Reset)
			} else {
				helpLine = "✓ " + statusMsg
			}
		} else if showActionMenu {
			if theme.Enabled {
				helpLine = fmt.Sprintf("%s[↑/↓/1-6]%s Choose  %s│%s  %s[Enter]%s Execute  %s│%s  %s[Esc/q]%s Close",
					theme.Bold+theme.BrightCyan, theme.Reset,
					theme.Dim, theme.Reset,
					theme.Bold+theme.BrightWhite, theme.Reset,
					theme.Dim, theme.Reset,
					theme.Red, theme.Reset,
				)
			} else {
				helpLine = "[↑/↓/1-6] Choose  │  [Enter] Execute  │  [Esc/q] Close"
			}
		} else if searchActive {
			if theme.Enabled {
				helpLine = fmt.Sprintf("%s[Type]%s Filter  %s│%s  %s[Enter/↓]%s Table  %s│%s  %s[Ctrl+U]%s Clear  %s│%s  %s[Esc]%s Cancel",
					theme.Bold+theme.BrightGreen, theme.Reset,
					theme.Dim, theme.Reset,
					theme.Bold+theme.BrightCyan, theme.Reset,
					theme.Dim, theme.Reset,
					theme.Yellow, theme.Reset,
					theme.Dim, theme.Reset,
					theme.Red, theme.Reset,
				)
			} else {
				helpLine = "[Type] Filter  │  [Enter/↓] Table  │  [Ctrl+U] Clear  │  [Esc] Cancel"
			}
		} else if searchQuery != "" {
			if theme.Enabled {
				helpLine = fmt.Sprintf("%s[↑/↓/j/k]%s Navigate  %s│%s  %s[/]%s Search  %s│%s  %s[Esc]%s Clear  %s│%s  %s[Enter]%s Menu  %s│%s  %s[t]%s Theme  %s│%s  %s[q]%s Quit",
					theme.Bold+theme.BrightCyan, theme.Reset,
					theme.Dim, theme.Reset,
					theme.Bold+theme.BrightYellow, theme.Reset,
					theme.Dim, theme.Reset,
					theme.Yellow, theme.Reset,
					theme.Dim, theme.Reset,
					theme.Bold+theme.BrightGreen, theme.Reset,
					theme.Dim, theme.Reset,
					theme.Bold+theme.BrightMagenta, theme.Reset,
					theme.Dim, theme.Reset,
					theme.Red, theme.Reset,
				)
			} else {
				helpLine = "[↑/↓/j/k] Navigate  │  [/] Search  │  [Esc] Clear  │  [Enter] Menu  │  [t] Theme  │  [q] Quit"
			}
		} else {
			if theme.Enabled {
				helpLine = fmt.Sprintf("%s[↑/↓/j/k]%s Navigate  %s│%s  %s[/]%s Search  %s│%s  %s[Enter]%s Menu  %s│%s  %s[t]%s Theme  %s│%s  %s[x]%s Kill  %s│%s  %s[r]%s Refresh  %s│%s  %s[q]%s Quit",
					theme.Bold+theme.BrightCyan, theme.Reset,
					theme.Dim, theme.Reset,
					theme.Bold+theme.BrightYellow, theme.Reset,
					theme.Dim, theme.Reset,
					theme.Bold+theme.BrightGreen, theme.Reset,
					theme.Dim, theme.Reset,
					theme.Bold+theme.BrightMagenta, theme.Reset,
					theme.Dim, theme.Reset,
					theme.Red, theme.Reset,
					theme.Dim, theme.Reset,
					theme.Bold+theme.BrightGreen, theme.Reset,
					theme.Dim, theme.Reset,
					theme.Dim+theme.Red, theme.Reset,
				)
			} else {
				helpLine = "[↑/↓/j/k] Navigate  │  [/] Search  │  [Enter] Menu  │  [t] Theme  │  [x] Kill  │  [r] Refresh  │  [q] Quit"
			}
		}
		writeRow(helpLine)

		// Fill remaining rows down to the bottom with the theme background color
		for currentLine < height {
			writeRow("")
		}

		// 7. Floating Action Menu Modal
		if showActionMenu && len(filteredRecords) > 0 && selectedIndex < len(filteredRecords) {
			target := filteredRecords[selectedIndex]
			boxW = 50
			if boxW > width-2 {
				boxW = width - 2
			}
			innerWidth := boxW - 4
			if innerWidth < 20 {
				innerWidth = 20
			}
			menuStartY = (height - boxH) / 2
			if menuStartY < 2 {
				menuStartY = 2
			}
			menuStartX = (width - boxW) / 2
			if menuStartX < 1 {
				menuStartX = 1
			}

			// Top border with title
			titleStr := fmt.Sprintf(" Actions: :%d (%s) ", target.Port, target.Process)
			if target.Origin != nil && *target.Origin != "" {
				titleStr = fmt.Sprintf(" Actions: :%d (%s • %s) ", target.Port, target.Process, *target.Origin)
			}
			titleLen := visibleLength(titleStr)
			if titleLen > innerWidth {
				titleStr = truncateANSI(titleStr, innerWidth, theme.Reset)
				titleLen = visibleLength(titleStr)
			}
			sideLen := (boxW - 2 - titleLen) / 2
			if sideLen < 0 {
				sideLen = 0
			}
			rightLen := boxW - 2 - sideLen - titleLen
			if rightLen < 0 {
				rightLen = 0
			}
			topBorder := "┌" + strings.Repeat("─", sideLen) + titleStr + strings.Repeat("─", rightLen) + "┐"
			if theme.Enabled {
				fmt.Fprintf(&buf, "\033[%d;%dH%s%s%s%s", menuStartY, menuStartX, theme.BgEscape, theme.Bold+theme.BrightCyan, topBorder, theme.Reset)
			} else {
				fmt.Fprintf(&buf, "\033[%d;%dH%s", menuStartY, menuStartX, topBorder)
			}

			for i, item := range menuItems {
				rowY := menuStartY + 1 + i
				isItemSel := i == menuSelection
				content := item
				if i == 3 && target.PID > 0 {
					content = fmt.Sprintf("4. Copy PID (%d)", target.PID)
				}
				avail := innerWidth - 2
				if avail < 10 {
					avail = 10
				}
				if visibleLength(content) > avail {
					content = truncateANSI(content, avail, theme.Reset)
				}
				pad := avail - visibleLength(content)
				if pad < 0 {
					pad = 0
				}

				var lineText string
				if isItemSel {
					if theme.Enabled {
						lineText = fmt.Sprintf("%s▶ %s%s%s", theme.Bold+theme.BrightCyan, content, strings.Repeat(" ", pad), theme.Reset)
					} else {
						lineText = fmt.Sprintf("▶ %s%s", content, strings.Repeat(" ", pad))
					}
				} else {
					lineText = fmt.Sprintf("  %s%s", content, strings.Repeat(" ", pad))
				}

				if theme.Enabled {
					fmt.Fprintf(&buf, "\033[%d;%dH%s%s│%s %s %s│%s", rowY, menuStartX, theme.BgEscape, theme.BrightCyan, theme.Reset, lineText, theme.BrightCyan, theme.Reset)
				} else {
					fmt.Fprintf(&buf, "\033[%d;%dH│ %s │", rowY, menuStartX, lineText)
				}
			}

			// Hint row
			hintY := menuStartY + 1 + len(menuItems)
			hintText := "[1-6] Choose  │  [Esc] Close"
			if visibleLength(hintText) > innerWidth {
				hintText = truncateANSI(hintText, innerWidth, theme.Reset)
			}
			hPad := innerWidth - visibleLength(hintText)
			if hPad < 0 {
				hPad = 0
			}
			if theme.Enabled {
				fmt.Fprintf(&buf, "\033[%d;%dH%s%s│%s %s%s%s%s %s│%s", hintY, menuStartX, theme.BgEscape, theme.BrightCyan, theme.Reset, theme.Dim, hintText, strings.Repeat(" ", hPad), theme.Reset, theme.BrightCyan, theme.Reset)
			} else {
				fmt.Fprintf(&buf, "\033[%d;%dH│ %s%s │", hintY, menuStartX, hintText, strings.Repeat(" ", hPad))
			}

			// Bottom border
			botBorder := "└" + strings.Repeat("─", boxW-2) + "┘"
			if theme.Enabled {
				fmt.Fprintf(&buf, "\033[%d;%dH%s%s%s%s", menuStartY+boxH-1, menuStartX, theme.BgEscape, theme.Bold+theme.BrightCyan, botBorder, theme.Reset)
			} else {
				fmt.Fprintf(&buf, "\033[%d;%dH%s", menuStartY+boxH-1, menuStartX, botBorder)
			}
		}

		os.Stdout.Write(buf.Bytes())
	}

	draw()

	// Action executor helper
	executeAction := func(idx int) {
		if selectedIndex < 0 || selectedIndex >= len(filteredRecords) {
			showActionMenu = false
			draw()
			return
		}
		target := filteredRecords[selectedIndex]
		showActionMenu = false

		switch idx {
		case 0, 1: // Kill SIGTERM or SIGKILL
			if target.PID <= 0 {
				statusMsg = "Cannot kill: PID is unavailable (permission denied or kernel socket)"
				statusMsgTimer = time.Now().Add(3 * time.Second)
				break
			}
			confirmingKill = true
			if idx == 0 {
				killSignal = syscall.SIGTERM
			} else {
				killSignal = syscall.SIGKILL
			}
		case 2: // Copy JSON
			jsonBytes, err := json.MarshalIndent(target, "", "  ")
			if err == nil {
				_ = copyToClipboard(string(jsonBytes))
				statusMsg = fmt.Sprintf("✓ Copied JSON for port %d to clipboard", target.Port)
			} else {
				statusMsg = fmt.Sprintf("Error generating JSON: %v", err)
			}
			statusMsgTimer = time.Now().Add(3 * time.Second)
		case 3: // Copy PID
			if target.PID > 0 {
				_ = copyToClipboard(strconv.Itoa(target.PID))
				statusMsg = fmt.Sprintf("✓ Copied PID %d to clipboard", target.PID)
			} else {
				statusMsg = "PID is unavailable (permission denied or kernel socket)"
			}
			statusMsgTimer = time.Now().Add(3 * time.Second)
		case 4: // Copy Command
			if target.Command != nil && *target.Command != "" {
				_ = copyToClipboard(*target.Command)
				statusMsg = fmt.Sprintf("✓ Copied command for '%s' to clipboard", target.Process)
			} else {
				statusMsg = "Command line is unavailable"
			}
			statusMsgTimer = time.Now().Add(3 * time.Second)
		case 5: // Copy Address
			addr := fmt.Sprintf("%s:%d", target.Address, target.Port)
			_ = copyToClipboard(addr)
			statusMsg = fmt.Sprintf("✓ Copied '%s' to clipboard", addr)
			statusMsgTimer = time.Now().Add(3 * time.Second)
		}
		draw()
	}

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
		case sig := <-sigChan:
			if sig == syscall.SIGWINCH {
				if theme.BgEscape != "" {
					fmt.Printf("%s\033[2J\033[H", theme.BgEscape)
				} else {
					fmt.Printf("\033[49m\033[2J\033[H")
				}
				draw()
				continue
			}
			return nil

		case <-ticker.C:
			if !confirmingKill && !showActionMenu {
				refreshData()
				draw()
			}

		case input := <-keyChan:
			if len(input) == 0 {
				continue
			}

			// Mouse event: \x1b[<b;col;rowM
			if len(input) >= 6 && input[0] == 0x1b && input[1] == '[' && input[2] == '<' {
				str := string(input[3:])
				if endIdx := strings.IndexAny(str, "Mm"); endIdx != -1 {
					params := strings.Split(str[:endIdx], ";")
					if len(params) == 3 && str[endIdx] == 'M' {
						btn := params[0]
						col, _ := strconv.Atoi(params[1])
						row, _ := strconv.Atoi(params[2])

						if btn == "64" { // Mouse wheel up
							if showActionMenu {
								menuSelection = (menuSelection - 1 + len(menuItems)) % len(menuItems)
							} else if searchActive {
								// In search bar: keep
							} else if selectedIndex > 0 {
								selectedIndex -= 2
								if selectedIndex < 0 {
									selectedIndex = 0
								}
							}
							draw()
							continue
						} else if btn == "65" { // Mouse wheel down
							if showActionMenu {
								menuSelection = (menuSelection + 1) % len(menuItems)
							} else if searchActive {
								// In search bar: keep
							} else if selectedIndex < len(filteredRecords)-1 {
								selectedIndex += 2
								if selectedIndex >= len(filteredRecords) {
									selectedIndex = len(filteredRecords) - 1
								}
							}
							draw()
							continue
						}

						if showActionMenu {
							// Check if click was on a popup menu item
							if row >= menuStartY+1 && row <= menuStartY+len(menuItems) &&
								col >= menuStartX && col <= menuStartX+boxW {
								actionIdx := row - (menuStartY + 1)
								executeAction(actionIdx)
								continue
							}
							// Clicked outside popup menu: close it
							showActionMenu = false
							draw()
							continue
						}

						// Click on search box (rows 3, 4, 5)
						if row >= 3 && row <= 5 {
							searchActive = true
							draw()
							continue
						}

						// Main row click: select, unfocus search, and open action menu
						clickedRecord := scrollOffset + (row - 7)
						if row >= 7 && row < 7+contentHeight && clickedRecord >= 0 && clickedRecord < len(filteredRecords) {
							selectedIndex = clickedRecord
							searchActive = false
							showActionMenu = true
							menuSelection = 0
							draw()
							continue
						}
					}
				}
			}

			// Escape sequences: arrows, page up/down, home/end
			if len(input) >= 3 && input[0] == 0x1b && input[1] == '[' {
				switch input[2] {
				case 'A': // Up
					if showActionMenu {
						menuSelection = (menuSelection - 1 + len(menuItems)) % len(menuItems)
						draw()
					} else if searchActive {
						// In search bar: stay
					} else if selectedIndex > 0 {
						selectedIndex--
						draw()
					} else if selectedIndex == 0 {
						// Up from row 0 enters search box
						searchActive = true
						draw()
					}
					continue
				case 'B': // Down
					if showActionMenu {
						menuSelection = (menuSelection + 1) % len(menuItems)
						draw()
					} else if searchActive {
						// Down from search box enters table
						searchActive = false
						if len(filteredRecords) > 0 && selectedIndex < 0 {
							selectedIndex = 0
						}
						draw()
					} else if selectedIndex < len(filteredRecords)-1 {
						selectedIndex++
						draw()
					}
					continue
				case 'C', 'D': // Right, Left
					continue
				case 'H', '1', '7': // Home (\x1b[H, \x1b[1~, \x1b[7~)
					if !searchActive && len(filteredRecords) > 0 {
						selectedIndex = 0
						draw()
					}
					continue
				case 'F', '4', '8': // End (\x1b[F, \x1b[4~, \x1b[8~)
					if !searchActive && len(filteredRecords) > 0 {
						selectedIndex = len(filteredRecords) - 1
						draw()
					}
					continue
				case '5': // Page Up (\x1b[5~)
					if showActionMenu {
						menuSelection = 0
						draw()
					} else if !searchActive && len(filteredRecords) > 0 {
						selectedIndex -= contentHeight
						if selectedIndex < 0 {
							selectedIndex = 0
						}
						draw()
					}
					continue
				case '6': // Page Down (\x1b[6~)
					if showActionMenu {
						menuSelection = len(menuItems) - 1
						draw()
					} else if !searchActive && len(filteredRecords) > 0 {
						selectedIndex += contentHeight
						if selectedIndex >= len(filteredRecords) {
							selectedIndex = len(filteredRecords) - 1
						}
						draw()
					}
					continue
				}
			}

			// SS3 sequences: \x1bOH (Home), \x1bOF (End)
			if len(input) >= 3 && input[0] == 0x1b && input[1] == 'O' {
				if input[2] == 'H' && !searchActive && len(filteredRecords) > 0 {
					selectedIndex = 0
					draw()
					continue
				} else if input[2] == 'F' && !searchActive && len(filteredRecords) > 0 {
					selectedIndex = len(filteredRecords) - 1
					draw()
					continue
				}
			}

			// Any unhandled multi-byte escape sequence should be ignored (not treated as standalone Esc)
			if len(input) > 1 && input[0] == 0x1b {
				continue
			}

			ch := input[0]

			// Kill confirmation mode
			if confirmingKill {
				if ch == 3 { // Ctrl+C
					return nil
				}
				if ch == 'y' || ch == 'Y' {
					if selectedIndex >= 0 && selectedIndex < len(filteredRecords) {
						target := filteredRecords[selectedIndex]
						if target.PID > 0 {
							if err := syscall.Kill(target.PID, killSignal); err != nil {
								statusMsg = fmt.Sprintf("Error killing PID %d: %v", target.PID, err)
							} else {
								sigName := "SIGTERM"
								if killSignal == syscall.SIGKILL {
									sigName = "SIGKILL"
								}
								statusMsg = fmt.Sprintf("✓ Sent %s to '%s' (PID %d)", sigName, target.Process, target.PID)
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

			// Action menu mode
			if showActionMenu {
				if ch == 3 { // Ctrl+C
					return nil
				}
				if ch == 'q' || ch == 'Q' || ch == 27 { // Esc or q: close menu
					showActionMenu = false
					draw()
					continue
				}
				if ch >= '1' && ch <= '6' {
					idx := int(ch - '1')
					executeAction(idx)
					continue
				}
				if ch == 13 || ch == 10 || ch == ' ' { // Enter or Space: execute selected
					executeAction(menuSelection)
					continue
				}
				if ch == 'k' || ch == 'K' {
					menuSelection = (menuSelection - 1 + len(menuItems)) % len(menuItems)
					draw()
					continue
				}
				if ch == 'j' || ch == 'J' {
					menuSelection = (menuSelection + 1) % len(menuItems)
					draw()
					continue
				}
				continue
			}

			// Search bar mode: live fzf-style filtering on every key
			if searchActive {
				if ch == 3 { // Ctrl+C
					return nil
				}
				if ch == 27 { // Esc
					if searchQuery != "" {
						searchQuery = ""
						applyFilter()
					}
					searchActive = false
					draw()
					continue
				}
				if ch == 13 || ch == 10 { // Enter: finish typing, focus table
					searchActive = false
					draw()
					continue
				}
				if ch == 0x7f || ch == 8 { // Backspace
					if len(searchQuery) > 0 {
						runes := []rune(searchQuery)
						searchQuery = string(runes[:len(runes)-1])
						applyFilter()
						draw()
					}
					continue
				}
				if ch == 21 || ch == 23 { // Ctrl+U or Ctrl+W: clear search
					searchQuery = ""
					applyFilter()
					draw()
					continue
				}
				if ch >= 32 && ch <= 126 { // Printable character: live filter!
					searchQuery += string(ch)
					applyFilter()
					draw()
					continue
				}
				continue
			}

			// Normal mode
			switch ch {
			case 'q', 'Q', 3:
				return nil

			case 27: // Esc: clear search filter if active, or exit
				if searchQuery != "" {
					searchQuery = ""
					applyFilter()
					draw()
					continue
				}
				return nil

			case '/': // Enter search mode
				searchActive = true
				draw()
				continue

			case 13, 10, ' ', 'm', 'M', 'a', 'A': // Enter, Space, m, a: open action popup menu
				if len(filteredRecords) > 0 {
					showActionMenu = true
					menuSelection = 0
					draw()
				}

			case 'k', 'K':
				if selectedIndex > 0 {
					selectedIndex--
					draw()
				} else if selectedIndex == 0 {
					searchActive = true
					draw()
				}

			case 'j', 'J':
				if selectedIndex < len(filteredRecords)-1 {
					selectedIndex++
					draw()
				}

			case 'x', 'X', 'd', 'D': // Direct kill hotkey
				if len(filteredRecords) > 0 && selectedIndex < len(filteredRecords) {
					target := filteredRecords[selectedIndex]
					if target.PID <= 0 {
						statusMsg = "Cannot kill: PID is unavailable (permission denied or kernel socket)"
						statusMsgTimer = time.Now().Add(3 * time.Second)
					} else {
						confirmingKill = true
						killSignal = syscall.SIGTERM
					}
					draw()
				}

			case 'g': // Jump to top
				if len(filteredRecords) > 0 {
					selectedIndex = 0
					draw()
				}

			case 'G': // Jump to bottom
				if len(filteredRecords) > 0 {
					selectedIndex = len(filteredRecords) - 1
					draw()
				}

			case 4, 6: // Ctrl+D (4) or Ctrl+F (6): Page Down
				if len(filteredRecords) > 0 {
					selectedIndex += contentHeight / 2
					if selectedIndex >= len(filteredRecords) {
						selectedIndex = len(filteredRecords) - 1
					}
					draw()
				}

			case 21, 2: // Ctrl+U (21) or Ctrl+B (2): Page Up
				if len(filteredRecords) > 0 {
					selectedIndex -= contentHeight / 2
					if selectedIndex < 0 {
						selectedIndex = 0
					}
					draw()
				}

			case 't', 'T':
				currentThemeDef = CycleTheme(currentThemeDef.ID)
				theme = currentThemeDef.BuildTheme(true)
				_ = config.Save(&config.Config{Theme: currentThemeDef.ID})
				statusMsg = fmt.Sprintf("Theme: %s", currentThemeDef.Name)
				statusMsgTimer = time.Now().Add(2 * time.Second)
				if theme.BgEscape != "" {
					fmt.Printf("%s\033[2J\033[H", theme.BgEscape)
				} else {
					fmt.Printf("\033[49m\033[2J\033[H")
				}
				draw()

			case 'r', 'R':
				refreshData()
				statusMsg = "Refreshed"
				statusMsgTimer = time.Now().Add(1 * time.Second)
				draw()
			}
		}
	}
}

// copyToClipboard writes text to the clipboard using OSC 52 and local utilities as fallback.
func copyToClipboard(text string) error {
	// 1. OSC 52 sequence (universal across modern terminal emulators)
	b64 := base64.StdEncoding.EncodeToString([]byte(text))
	fmt.Printf("\033]52;c;%s\007", b64)

	// 2. Best-effort desktop clipboard fallback
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		if path, err := exec.LookPath("wl-copy"); err == nil {
			cmd := exec.Command(path)
			cmd.Stdin = strings.NewReader(text)
			_ = cmd.Run()
		}
	} else if os.Getenv("DISPLAY") != "" {
		if path, err := exec.LookPath("xclip"); err == nil {
			cmd := exec.Command(path, "-selection", "clipboard")
			cmd.Stdin = strings.NewReader(text)
			_ = cmd.Run()
		} else if path, err := exec.LookPath("xsel"); err == nil {
			cmd := exec.Command(path, "--clipboard", "--input")
			cmd.Stdin = strings.NewReader(text)
			_ = cmd.Run()
		}
	}
	return nil
}

func formatDetailField(theme *Theme, label, val string) string {
	if theme.Enabled {
		return fmt.Sprintf("  %s%-10s%s %s", theme.Cyan, label+":", theme.Reset, val)
	}
	return fmt.Sprintf("  %-10s %s", label+":", val)
}

// truncateANSI truncates a string to at most maxWidth visible columns without cutting
// through ANSI escape sequences or leaving unterminated styles.
func truncateANSI(s string, maxWidth int, resetCode ...string) string {
	if maxWidth <= 0 {
		return ""
	}
	reset := "\033[0m"
	if len(resetCode) > 0 && resetCode[0] != "" {
		reset = resetCode[0]
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
	res.WriteString(reset)
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
