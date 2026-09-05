package tests

import (
	"bytes"
	"encoding/json"
	"testing"

	"ports/internal/model"
	"ports/internal/renderer"
)

func TestJSONRenderer(t *testing.T) {
	jr := renderer.NewJSONRenderer()

	uid := 1000
	user := "abhi"
	cwd := "/home/abhi/projects/api"
	cmd := "node server.js"
	var uptime int64 = 751

	records := []model.PortRecord{
		{
			Port:          3000,
			Protocol:      "tcp",
			Address:       "0.0.0.0",
			PID:           18421,
			Process:       "node",
			UID:           &uid,
			User:          &user,
			CWD:           &cwd,
			Command:       &cmd,
			UptimeSeconds: &uptime,
		},
		{
			Port:     8080,
			Protocol: "tcp",
			Address:  "127.0.0.1",
			PID:      0,
			Process:  "<permission denied>",
			// Null pointers
		},
	}

	var buf bytes.Buffer
	if err := jr.Render(&buf, records); err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	var unmarshaled []map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &unmarshaled); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if len(unmarshaled) != 2 {
		t.Fatalf("expected 2 records, got %d", len(unmarshaled))
	}

	// First record checks
	r1 := unmarshaled[0]
	if r1["port"].(float64) != 3000 {
		t.Errorf("expected port 3000, got %v", r1["port"])
	}
	if r1["user"].(string) != "abhi" {
		t.Errorf("expected user abhi, got %v", r1["user"])
	}

	// Second record checks (null fields)
	r2 := unmarshaled[1]
	if r2["user"] != nil {
		t.Errorf("expected null user, got %v", r2["user"])
	}
	if r2["cwd"] != nil {
		t.Errorf("expected null cwd, got %v", r2["cwd"])
	}
}

func TestJSONEmptyList(t *testing.T) {
	jr := renderer.NewJSONRenderer()
	var buf bytes.Buffer
	if err := jr.Render(&buf, nil); err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	var unmarshaled []map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &unmarshaled); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if len(unmarshaled) != 0 {
		t.Fatalf("expected empty array, got len %d", len(unmarshaled))
	}
}
