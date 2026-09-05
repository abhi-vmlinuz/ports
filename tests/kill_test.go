package tests

import (
	"bufio"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"ports/internal/proc"
)

func TestSafeKillSubprocess(t *testing.T) {
	// Spawn an isolated child process listening on an ephemeral port
	cmd := exec.Command("python3", "-c", `
import socket, time, sys
s = socket.socket()
s.bind(('127.0.0.1', 0))
s.listen(1)
port = s.getsockname()[1]
print(port, flush=True)
time.sleep(30)
`)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start test listener process: %v", err)
	}

	childPID := cmd.Process.Pid
	defer func() {
		// Guaranteed cleanup
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	// Read ephemeral port from child process
	scanner := bufio.NewScanner(stdout)
	if !scanner.Scan() {
		t.Fatalf("failed to read ephemeral port from child: %v", scanner.Err())
	}
	portStr := strings.TrimSpace(scanner.Text())
	portVal, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("invalid port %q from child: %v", portStr, err)
	}
	port := uint16(portVal)

	// Wait briefly for Linux kernel procfs registration
	time.Sleep(100 * time.Millisecond)

	// 1. Verify discovery identifies childPID as the port owner
	discoverer := proc.NewDiscoverer("/proc")
	records, err := discoverer.DiscoverPort(port)
	if err != nil {
		t.Fatalf("DiscoverPort failed: %v", err)
	}

	if len(records) == 0 {
		t.Fatalf("expected to discover port %d owned by PID %d, got 0 records", port, childPID)
	}

	var foundPID int
	for _, r := range records {
		if r.Port == port && r.PID == childPID {
			foundPID = r.PID
			break
		}
	}

	if foundPID != childPID {
		t.Fatalf("expected port %d to be owned by child PID %d, but found %d", port, childPID, foundPID)
	}

	// 2. Perform safe SIGTERM kill
	if err := syscall.Kill(childPID, syscall.SIGTERM); err != nil {
		t.Fatalf("failed to send SIGTERM to test child PID %d: %v", childPID, err)
	}

	// 3. Verify process terminates cleanly
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-time.After(3 * time.Second):
		t.Fatalf("child PID %d did not terminate within 3s after SIGTERM", childPID)
	case err := <-done:
		// Python exited due to SIGTERM (signal: terminated)
		if err == nil {
			t.Logf("child PID %d exited with code 0", childPID)
		} else {
			t.Logf("child PID %d terminated as expected: %v", childPID, err)
		}
	}
}
