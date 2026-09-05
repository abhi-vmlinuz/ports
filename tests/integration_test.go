package tests

import (
	"net"
	"os"
	"testing"
	"time"

	"ports/internal/proc"
)

func TestEphemeralTCPListenerDiscovery(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen on ephemeral port: %v", err)
	}
	defer ln.Close()

	tcpAddr := ln.Addr().(*net.TCPAddr)
	port := uint16(tcpAddr.Port)

	// Wait briefly for Linux kernel procfs entry
	time.Sleep(50 * time.Millisecond)

	discoverer := proc.NewDiscoverer("/proc")
	records, err := discoverer.DiscoverPort(port)
	if err != nil {
		t.Fatalf("DiscoverPort failed: %v", err)
	}

	if len(records) == 0 {
		t.Fatalf("expected to discover port %d, got 0 records", port)
	}

	found := false
	currentPID := os.Getpid()
	for _, r := range records {
		if r.Port == port && r.Protocol == "tcp" {
			found = true
			if r.PID != currentPID {
				t.Errorf("expected PID %d, got %d", currentPID, r.PID)
			}
			if r.Address != "127.0.0.1" {
				t.Errorf("expected address 127.0.0.1, got %s", r.Address)
			}
			break
		}
	}

	if !found {
		t.Errorf("did not find matching TCP record for port %d", port)
	}
}

func TestEphemeralUDPListenerDiscovery(t *testing.T) {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("failed to listen UDP on ephemeral port: %v", err)
	}
	defer conn.Close()

	udpAddr := conn.LocalAddr().(*net.UDPAddr)
	port := uint16(udpAddr.Port)

	time.Sleep(50 * time.Millisecond)

	discoverer := proc.NewDiscoverer("/proc")
	records, err := discoverer.DiscoverPort(port)
	if err != nil {
		t.Fatalf("DiscoverPort failed: %v", err)
	}

	if len(records) == 0 {
		t.Fatalf("expected to discover UDP port %d, got 0 records", port)
	}

	found := false
	currentPID := os.Getpid()
	for _, r := range records {
		if r.Port == port && r.Protocol == "udp" {
			found = true
			if r.PID != currentPID {
				t.Errorf("expected PID %d, got %d", currentPID, r.PID)
			}
			break
		}
	}

	if !found {
		t.Errorf("did not find matching UDP record for port %d", port)
	}
}
