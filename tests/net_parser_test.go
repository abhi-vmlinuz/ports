package tests

import (
	"strings"
	"testing"

	"ports/internal/proc"
)

func TestParseAddressIPv4(t *testing.T) {
	// Sample line from /proc/net/tcp for 127.0.0.1:45678 (hex: 0100007F:B26E)
	tcpContent := `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 0100007F:B26E 00000000:0000 0A 00000000:00000000 00:00000000 00000000  1000        0 424693 1 000000004352b9d4 100 0 0 10 0
   1: 00000000:0050 00000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0  12345 1 0000000000000000 100 0 0 10 0
   2: 0100007F:1F90 0100007F:B26E 01 00000000:00000000 00:00000000 00000000  1000        0  99999 1 0000000000000000 100 0 0 10 0
`

	sockets, err := proc.ParseNetReader(strings.NewReader(tcpContent), "tcp", true, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sockets) != 2 {
		t.Fatalf("expected 2 LISTEN sockets, got %d", len(sockets))
	}

	// First socket: 127.0.0.1:45678
	s1 := sockets[0]
	if s1.LocalIP != "127.0.0.1" {
		t.Errorf("expected IP 127.0.0.1, got %s", s1.LocalIP)
	}
	if s1.LocalPort != 45678 {
		t.Errorf("expected port 45678, got %d", s1.LocalPort)
	}
	if s1.Inode != 424693 {
		t.Errorf("expected inode 424693, got %d", s1.Inode)
	}
	if s1.SocketUID != 1000 {
		t.Errorf("expected UID 1000, got %d", s1.SocketUID)
	}

	// Second socket: 0.0.0.0:80
	s2 := sockets[1]
	if s2.LocalIP != "0.0.0.0" {
		t.Errorf("expected IP 0.0.0.0, got %s", s2.LocalIP)
	}
	if s2.LocalPort != 80 {
		t.Errorf("expected port 80, got %d", s2.LocalPort)
	}
	if s2.Inode != 12345 {
		t.Errorf("expected inode 12345, got %d", s2.Inode)
	}
	if s2.SocketUID != 0 {
		t.Errorf("expected UID 0, got %d", s2.SocketUID)
	}
}

func TestParseAddressIPv6(t *testing.T) {
	// Sample line from /proc/net/tcp6 for [::1]:45680 and [::]:3000
	tcp6Content := `  sl  local_address                         remote_address                        st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 00000000000000000000000001000000:B270 00000000000000000000000000000000:0000 0A 00000000:00000000 00:00000000 00000000  1000        0 424917 1 000000009fa97139 100 0 0 10 0
   1: 00000000000000000000000000000000:0BB8 00000000000000000000000000000000:0000 0A 00000000:00000000 00:00000000 00000000  1000        0 424918 1 000000009fa97139 100 0 0 10 0
`

	sockets, err := proc.ParseNetReader(strings.NewReader(tcp6Content), "tcp6", true, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sockets) != 2 {
		t.Fatalf("expected 2 LISTEN sockets, got %d", len(sockets))
	}

	s1 := sockets[0]
	if s1.LocalIP != "::1" {
		t.Errorf("expected IP ::1, got %s", s1.LocalIP)
	}
	if s1.LocalPort != 45680 {
		t.Errorf("expected port 45680, got %d", s1.LocalPort)
	}

	s2 := sockets[1]
	if s2.LocalIP != "::" {
		t.Errorf("expected IP ::, got %s", s2.LocalIP)
	}
	if s2.LocalPort != 3000 {
		t.Errorf("expected port 3000, got %d", s2.LocalPort)
	}
}

func TestParseUDPBound(t *testing.T) {
	// Sample line from /proc/net/udp: bound to 0.0.0.0:5353, and a connected one
	udpContent := `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode ref pointer drops
   0: 00000000:14E9 00000000:0000 07 00000000:00000000 00:00000000 00000000  1000        0 555555 1 0000000000000000 0
   1: 0100007F:1234 0100007F:5678 01 00000000:00000000 00:00000000 00000000  1000        0 666666 1 0000000000000000 0
`

	sockets, err := proc.ParseNetReader(strings.NewReader(udpContent), "udp", false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sockets) != 1 {
		t.Fatalf("expected 1 bound UDP socket, got %d", len(sockets))
	}

	s := sockets[0]
	if s.LocalPort != 5353 {
		t.Errorf("expected port 5353, got %d", s.LocalPort)
	}
	if s.State != "BOUND" {
		t.Errorf("expected state BOUND, got %s", s.State)
	}
}
