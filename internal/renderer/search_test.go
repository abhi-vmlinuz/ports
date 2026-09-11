package renderer

import (
	"testing"

	"ports/internal/model"
)

func TestFilterRecords(t *testing.T) {
	u1 := "elish4h"
	u2 := "root"
	sample := []model.PortRecord{
		{Port: 22, Protocol: "tcp", Address: "0.0.0.0", Process: "sshd", PID: 812, User: &u2},
		{Port: 80, Protocol: "tcp", Address: "0.0.0.0", Process: "nginx", PID: 1200, User: &u2},
		{Port: 5353, Protocol: "udp", Address: "224.0.0.251", Process: "brave", PID: 15272, User: &u1},
		{Port: 8080, Protocol: "tcp", Address: "127.0.0.1", Process: "node", PID: 9121, User: &u1},
		{Port: 5432, Protocol: "tcp", Address: "127.0.0.1", Process: "postgres", PID: 1932, User: &u2},
	}

	tests := []struct {
		name      string
		query     string
		wantPorts []uint16
	}{
		{
			name:      "Empty query returns all",
			query:     "",
			wantPorts: []uint16{22, 80, 5353, 8080, 5432},
		},
		{
			name:      "Whitespace query returns all",
			query:     "   ",
			wantPorts: []uint16{22, 80, 5353, 8080, 5432},
		},
		{
			name:      "Exact port search",
			query:     "22",
			wantPorts: []uint16{22},
		},
		{
			name:      "Port with colon alias",
			query:     ":8080",
			wantPorts: []uint16{8080},
		},
		{
			name:      "Port substring matches multiple",
			query:     "80",
			wantPorts: []uint16{80, 8080},
		},
		{
			name:      "Process name exact substring",
			query:     "brave",
			wantPorts: []uint16{5353},
		},
		{
			name:      "Process name case insensitive",
			query:     "POSTGRES",
			wantPorts: []uint16{5432},
		},
		{
			name:      "Fuzzy subsequence match",
			query:     "brv",
			wantPorts: []uint16{5353},
		},
		{
			name:      "Multi-token query (process and port)",
			query:     "node 8080",
			wantPorts: []uint16{8080},
		},
		{
			name:      "Multi-token query (protocol and port)",
			query:     "udp 5353",
			wantPorts: []uint16{5353},
		},
		{
			name:      "Filter by user",
			query:     "elish4h",
			wantPorts: []uint16{5353, 8080},
		},
		{
			name:      "Non-matching query",
			query:     "nonexistent",
			wantPorts: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterRecords(sample, tt.query)
			var gotPorts []uint16
			for _, r := range got {
				gotPorts = append(gotPorts, r.Port)
			}
			if len(gotPorts) != len(tt.wantPorts) {
				t.Fatalf("FilterRecords(q=%q) returned %d items (%v), want %d items (%v)",
					tt.query, len(gotPorts), gotPorts, len(tt.wantPorts), tt.wantPorts)
			}
			for i := range gotPorts {
				if gotPorts[i] != tt.wantPorts[i] {
					t.Errorf("item [%d] = %d, want %d", i, gotPorts[i], tt.wantPorts[i])
				}
			}
		})
	}
}

func TestHighlightMatches(t *testing.T) {
	theme := NewTheme(false) // colors enabled
	res := HighlightMatches("brave", "brv", theme, theme.BrightWhite)
	vis := visibleLength(res)
	if vis != 5 {
		t.Errorf("visibleLength(HighlightMatches) = %d, want 5", vis)
	}

	themeNoColor := NewTheme(true) // NO_COLOR
	resNoColor := HighlightMatches("brave", "brv", themeNoColor, "")
	if resNoColor != "brave" {
		t.Errorf("HighlightMatches with NO_COLOR = %q, want 'brave'", resNoColor)
	}
}
