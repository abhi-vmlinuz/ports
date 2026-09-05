package proc

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"ports/internal/model"
)

// ProcessScanner extracts and caches process metadata from /proc/<pid>/*.
type ProcessScanner struct {
	procPath  string
	sysUptime float64
	cache     map[int]*model.ProcessInfo
}

// NewProcessScanner creates a new process metadata scanner.
func NewProcessScanner(procPath string) *ProcessScanner {
	if procPath == "" {
		procPath = "/proc"
	}
	uptime, _ := GetSystemUptime()
	return &ProcessScanner{
		procPath:  procPath,
		sysUptime: uptime,
		cache:     make(map[int]*model.ProcessInfo),
	}
}

// GetProcess returns the ProcessInfo for a given PID, caching results.
func (ps *ProcessScanner) GetProcess(pid int) *model.ProcessInfo {
	if info, ok := ps.cache[pid]; ok {
		return info
	}

	info := ps.readProcess(pid)
	ps.cache[pid] = info
	return info
}

// readProcess reads /proc/<pid>/ files to construct ProcessInfo.
func (ps *ProcessScanner) readProcess(pid int) *model.ProcessInfo {
	info := &model.ProcessInfo{
		PID: pid,
		UID: -1,
	}

	pidDir := fmt.Sprintf("%s/%d", ps.procPath, pid)
	if _, err := os.Stat(pidDir); err != nil {
		info.Err = err
		return info
	}

	// 1. Process Name from /proc/<pid>/comm
	commPath := filepath.Join(pidDir, "comm")
	if commBytes, err := os.ReadFile(commPath); err == nil {
		info.Name = strings.TrimSpace(string(commBytes))
	}

	// Fallback to /proc/<pid>/exe if comm is empty
	if info.Name == "" {
		exePath := filepath.Join(pidDir, "exe")
		if exeTarget, err := os.Readlink(exePath); err == nil {
			info.Name = filepath.Base(exeTarget)
		}
	}

	// 2. Command Line from /proc/<pid>/cmdline
	cmdlinePath := filepath.Join(pidDir, "cmdline")
	if cmdBytes, err := os.ReadFile(cmdlinePath); err == nil && len(cmdBytes) > 0 {
		// Replace NUL bytes with spaces
		cleaned := bytes.ReplaceAll(cmdBytes, []byte{0}, []byte{' '})
		info.Cmdline = strings.TrimSpace(string(cleaned))
	}
	if info.Cmdline == "" && info.Name != "" {
		// Fallback for processes that clear cmdline
		info.Cmdline = info.Name
	}

	// 3. Current Working Directory from /proc/<pid>/cwd
	cwdPath := filepath.Join(pidDir, "cwd")
	if cwdTarget, err := os.Readlink(cwdPath); err == nil {
		info.CWD = cwdTarget
	}

	// 4. UID and Username from /proc/<pid>/status
	statusPath := filepath.Join(pidDir, "status")
	if statusFile, err := os.Open(statusPath); err == nil {
		scanner := bufio.NewScanner(statusFile)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "Uid:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					if uid, err := strconv.Atoi(fields[1]); err == nil {
						info.UID = uid
					}
				}
				break
			}
		}
		statusFile.Close()
	}

	// Resolve username
	if info.UID >= 0 {
		info.User = ResolveUsername(info.UID)
	}

	// 5. Process Uptime
	if ps.sysUptime > 0 {
		if uptime, err := GetProcessUptime(pid, ps.sysUptime); err == nil {
			info.UptimeSeconds = uptime
		}
	}

	return info
}

// ResolveUsername converts a numeric UID into a username with fallback to "UID:<id>".
func ResolveUsername(uid int) string {
	if u, err := user.LookupId(strconv.Itoa(uid)); err == nil && u.Username != "" {
		return u.Username
	}
	// Fallback to UID string
	return fmt.Sprintf("UID:%d", uid)
}
