package proc

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// UserHZ is the clock ticks per second used by the Linux kernel for /proc/<pid>/stat starttime.
// This is always 100 on Linux regardless of CONFIG_HZ.
const UserHZ = 100.0

// GetSystemUptime returns system uptime in seconds from /proc/uptime.
func GetSystemUptime() (float64, error) {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0, err
	}

	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return 0, fmt.Errorf("empty /proc/uptime")
	}

	uptime, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid /proc/uptime value %q: %w", fields[0], err)
	}

	return uptime, nil
}

// GetProcessUptime calculates process uptime in seconds from /proc/<pid>/stat and sysUptime.
func GetProcessUptime(pid int, sysUptime float64) (int64, error) {
	statPath := fmt.Sprintf("/proc/%d/stat", pid)
	data, err := os.ReadFile(statPath)
	if err != nil {
		return 0, err
	}

	content := string(data)
	lastParen := strings.LastIndex(content, ")")
	if lastParen == -1 || lastParen >= len(content)-1 {
		return 0, fmt.Errorf("malformed stat file for pid %d", pid)
	}

	// Fields after the closing paren:
	// Index 0: state (field 3)
	// Index 19: starttime (field 22)
	fields := strings.Fields(content[lastParen+1:])
	if len(fields) < 20 {
		return 0, fmt.Errorf("not enough fields in stat file for pid %d", pid)
	}

	starttimeTicks, err := strconv.ParseUint(fields[19], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid starttime %q for pid %d: %w", fields[19], pid, err)
	}

	starttimeSec := float64(starttimeTicks) / UserHZ
	uptimeSec := int64(sysUptime - starttimeSec)
	if uptimeSec < 0 {
		uptimeSec = 0
	}

	return uptimeSec, nil
}

// FormatDuration formats seconds into a human-friendly string (e.g. "12m 31s", "2h 5m", "3d 12h").
func FormatDuration(seconds int64) string {
	if seconds < 0 {
		return "0s"
	}

	d := time.Duration(seconds) * time.Second
	days := int64(d / (24 * time.Hour))
	hours := int64((d % (24 * time.Hour)) / time.Hour)
	minutes := int64((d % time.Hour) / time.Minute)
	secs := int64((d % time.Minute) / time.Second)

	if days > 0 {
		if hours > 0 {
			return fmt.Sprintf("%dd %dh", days, hours)
		}
		return fmt.Sprintf("%dd", days)
	}
	if hours > 0 {
		if minutes > 0 {
			return fmt.Sprintf("%dh %dm", hours, minutes)
		}
		return fmt.Sprintf("%dh", hours)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, secs)
	}
	return fmt.Sprintf("%ds", secs)
}
