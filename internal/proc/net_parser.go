package proc

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"

	"ports/internal/model"
)

const (
	// TCPStateListen is the hex state for TCP_LISTEN in /proc/net/tcp
	TCPStateListen = "0A"
	// UDPStateClose is the default state for bound UDP sockets in /proc/net/udp
	UDPStateClose = "07"
)

// NetScanner parses Linux /proc/net files.
type NetScanner struct {
	basePath string
}

// NewNetScanner creates a scanner with the given base path (usually "/proc/net").
func NewNetScanner(basePath string) *NetScanner {
	if basePath == "" {
		basePath = "/proc/net"
	}
	return &NetScanner{basePath: basePath}
}

// ScanSockets discovers all listening TCP and bound UDP sockets from /proc/net/*.
func (s *NetScanner) ScanSockets() ([]model.SocketInfo, error) {
	var sockets []model.SocketInfo

	files := []struct {
		relPath  string
		protocol string
		isTCP    bool
		isIPv6   bool
	}{
		{"tcp", "tcp", true, false},
		{"tcp6", "tcp6", true, true},
		{"udp", "udp", false, false},
		{"udp6", "udp6", false, true},
	}

	for _, f := range files {
		path := fmt.Sprintf("%s/%s", s.basePath, f.relPath)
		fileSockets, err := s.parseFile(path, f.protocol, f.isTCP, f.isIPv6)
		if err != nil {
			if os.IsNotExist(err) || os.IsPermission(err) {
				// Some kernel configs or containers may not have tcp6 or udp6
				continue
			}
			return nil, fmt.Errorf("failed to parse %s: %w", path, err)
		}
		sockets = append(sockets, fileSockets...)
	}

	return sockets, nil
}

// parseFile parses a single /proc/net file.
func (s *NetScanner) parseFile(path, protocol string, isTCP, isIPv6 bool) ([]model.SocketInfo, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return ParseNetReader(file, protocol, isTCP, isIPv6)
}

// ParseNetReader parses lines from any io.Reader matching /proc/net/ format.
func ParseNetReader(r io.Reader, protocol string, isTCP, isIPv6 bool) ([]model.SocketInfo, error) {
	var sockets []model.SocketInfo
	scanner := bufio.NewScanner(r)

	// Line 1 is the header
	if !scanner.Scan() {
		return nil, scanner.Err()
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 10 {
			continue
		}

		localAddr := fields[1]
		remAddr := fields[2]
		state := fields[3]
		uidStr := fields[7]
		inodeStr := fields[9]

		// TCP filtering: Only LISTEN
		if isTCP && strings.ToUpper(state) != TCPStateListen {
			continue
		}

		// UDP filtering: Sockets that are locally bound (not connected to a remote peer)
		if !isTCP {
			// Remote address must be 0:0
			remParts := strings.Split(remAddr, ":")
			if len(remParts) == 2 {
				remPort, _ := strconv.ParseUint(remParts[1], 16, 16)
				if remPort != 0 {
					continue // Connected UDP socket, not a listening/bound endpoint
				}
			}
		}

		// Parse local IP and port
		ipStr, port, err := parseAddress(localAddr, isIPv6)
		if err != nil {
			continue
		}

		inode, err := strconv.ParseUint(inodeStr, 10, 64)
		if err != nil || inode == 0 {
			continue
		}

		uid, _ := strconv.Atoi(uidStr)

		socketProtocol := protocol
		stateStr := "LISTEN"
		if !isTCP {
			stateStr = "BOUND"
		}

		sockets = append(sockets, model.SocketInfo{
			Protocol:  socketProtocol,
			LocalIP:   ipStr,
			LocalPort: port,
			State:     stateStr,
			Inode:     inode,
			SocketUID: uid,
		})
	}

	return sockets, scanner.Err()
}

// parseAddress parses hexadecimal IP:Port strings.
func parseAddress(addrStr string, isIPv6 bool) (string, uint16, error) {
	parts := strings.Split(addrStr, ":")
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid address format: %s", addrStr)
	}

	port64, err := strconv.ParseUint(parts[1], 16, 16)
	if err != nil {
		return "", 0, fmt.Errorf("invalid port hex %s: %w", parts[1], err)
	}
	port := uint16(port64)

	hexIP := parts[0]
	if isIPv6 {
		ip, err := parseIPv6Hex(hexIP)
		if err != nil {
			return "", 0, err
		}
		return ip, port, nil
	}

	ip, err := parseIPv4Hex(hexIP)
	if err != nil {
		return "", 0, err
	}
	return ip, port, nil
}

// parseIPv4Hex parses 8-character little-endian hex into IPv4 string.
func parseIPv4Hex(hexStr string) (string, error) {
	if len(hexStr) != 8 {
		return "", fmt.Errorf("invalid ipv4 hex length: %s", hexStr)
	}

	val, err := strconv.ParseUint(hexStr, 16, 32)
	if err != nil {
		return "", fmt.Errorf("failed to parse ipv4 hex %s: %w", hexStr, err)
	}

	// Linux stores IPv4 in host byte order (little-endian on x86/ARM)
	ip := net.IPv4(
		byte(val),
		byte(val>>8),
		byte(val>>16),
		byte(val>>24),
	)
	return ip.String(), nil
}

// parseIPv6Hex parses 32-character hex into IPv6 string.
// Linux prints IPv6 in /proc/net/tcp6 as four 32-bit words in host endianness.
func parseIPv6Hex(hexStr string) (string, error) {
	if len(hexStr) != 32 {
		return "", fmt.Errorf("invalid ipv6 hex length: %s", hexStr)
	}

	var ipBytes [16]byte
	for i := 0; i < 4; i++ {
		chunk := hexStr[i*8 : (i+1)*8]
		word, err := strconv.ParseUint(chunk, 16, 32)
		if err != nil {
			return "", fmt.Errorf("failed to parse ipv6 chunk %s: %w", chunk, err)
		}
		// Word in host endianness (little-endian)
		ipBytes[i*4] = byte(word)
		ipBytes[i*4+1] = byte(word >> 8)
		ipBytes[i*4+2] = byte(word >> 16)
		ipBytes[i*4+3] = byte(word >> 24)
	}

	ip := net.IP(ipBytes[:])
	return ip.String(), nil
}
