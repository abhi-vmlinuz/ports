package tests

import (
	"testing"

	"ports/internal/proc"
)

func TestFormatDuration(t *testing.T) {
	cases := []struct {
		seconds  int64
		expected string
	}{
		{0, "0s"},
		{45, "45s"},
		{60, "1m 0s"},
		{751, "12m 31s"},
		{3600, "1h"},
		{3665, "1h 1m"},
		{86400, "1d"},
		{90000, "1d 1h"},
	}

	for _, c := range cases {
		actual := proc.FormatDuration(c.seconds)
		if actual != c.expected {
			t.Errorf("FormatDuration(%d): expected %s, got %s", c.seconds, c.expected, actual)
		}
	}
}
