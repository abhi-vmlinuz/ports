package model

// PortRecord represents the public, stable data model for a listening port.
// This struct strictly conforms to the JSON specification in ports-spec.md.
type PortRecord struct {
	Port          uint16  `json:"port"`
	Protocol      string  `json:"protocol"`
	Address       string  `json:"address"`
	PID           int     `json:"pid"`
	Process       string  `json:"process"`
	UID           *int    `json:"uid"`
	User          *string `json:"user"`
	CWD           *string `json:"cwd"`
	Command       *string `json:"command"`
	UptimeSeconds *int64  `json:"uptime_seconds"`
}

// SocketInfo represents low-level socket data parsed from /proc/net/*.
type SocketInfo struct {
	Protocol  string // "tcp", "tcp6", "udp", "udp6"
	LocalIP   string // e.g., "127.0.0.1", "::1", "0.0.0.0", "::"
	LocalPort uint16
	State     string // e.g. "LISTEN", "BOUND"
	Inode     uint64
	SocketUID int
}

// ProcessInfo holds cached process metadata retrieved from /proc/<pid>/*.
type ProcessInfo struct {
	PID           int
	Name          string
	Cmdline       string
	CWD           string
	UID           int
	User          string
	UptimeSeconds int64
	Err           error // any permission or read error encountered
}
