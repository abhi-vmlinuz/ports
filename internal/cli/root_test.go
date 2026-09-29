package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCmdFlags(t *testing.T) {
	cmd := NewRootCmd("test")

	// Snapshot flag
	snapshotFlag := cmd.Flags().Lookup("snapshot")
	if snapshotFlag == nil {
		t.Fatal("expected --snapshot flag to exist")
	}
	if snapshotFlag.Shorthand != "s" {
		t.Errorf("expected shorthand 's' for snapshot, got %q", snapshotFlag.Shorthand)
	}

	// Table alias flag
	tableFlag := cmd.Flags().Lookup("table")
	if tableFlag == nil {
		t.Fatal("expected --table flag to exist")
	}
	if !tableFlag.Hidden {
		t.Errorf("expected --table flag to be hidden alias")
	}

	// Theme flag
	themeFlag := cmd.Flags().Lookup("theme")
	if themeFlag == nil {
		t.Fatal("expected --theme flag to exist")
	}
	if themeFlag.Shorthand != "t" {
		t.Errorf("expected shorthand 't' for theme, got %q", themeFlag.Shorthand)
	}

	// Watch flag
	watchFlag := cmd.Flags().Lookup("watch")
	if watchFlag == nil {
		t.Fatal("expected --watch flag to exist")
	}
	if watchFlag.Shorthand != "w" {
		t.Errorf("expected shorthand 'w' for watch, got %q", watchFlag.Shorthand)
	}

	// PID flag
	pidFlag := cmd.Flags().Lookup("pid")
	if pidFlag == nil {
		t.Fatal("expected --pid flag to exist")
	}
	if pidFlag.Shorthand != "p" {
		t.Errorf("expected shorthand 'p' for pid, got %q", pidFlag.Shorthand)
	}
}

func TestThemeListExecution(t *testing.T) {
	cmd := NewRootCmd("test")
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--theme", "list"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected --theme list to succeed, got %v", err)
	}
}

func TestSnapshotExecution(t *testing.T) {
	cmd := NewRootCmd("test")
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--snapshot", "--no-color"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected --snapshot to succeed, got %v", err)
	}
}

func TestJSONOutputExecution(t *testing.T) {
	cmd := NewRootCmd("test")
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--json"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected --json to succeed, got %v", err)
	}
	output := buf.String()
	if !strings.HasPrefix(strings.TrimSpace(output), "[") {
		t.Errorf("expected JSON array output, got: %s", output)
	}
}

func TestPIDOutputFlag(t *testing.T) {
	cmd := NewRootCmd("test")
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"99999", "--pid"})

	// Port 99999 should not exist (exit code 1)
	err := cmd.Execute()
	if err == nil {
		t.Errorf("expected error for non-existent port with --pid")
	}
}
