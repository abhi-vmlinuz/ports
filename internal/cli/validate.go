package cli

import (
	"fmt"
	"strconv"
	"strings"
)

// ParsePortArgument parses and validates a port string (accepting optional leading colon, e.g. ":3000" or "3000").
func ParsePortArgument(arg string) (uint16, error) {
	cleaned := strings.TrimPrefix(strings.TrimSpace(arg), ":")
	if cleaned == "" {
		return 0, fmt.Errorf("port cannot be empty")
	}

	val, err := strconv.ParseInt(cleaned, 10, 64)
	if err != nil || val < 1 || val > 65535 {
		return 0, fmt.Errorf("invalid port %q: must be between 1 and 65535", arg)
	}

	return uint16(val), nil
}
