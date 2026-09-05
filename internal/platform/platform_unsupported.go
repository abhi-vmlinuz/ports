//go:build !linux

package platform

import "fmt"

// CheckPlatform verifies that the current operating system is supported.
func CheckPlatform() error {
	return fmt.Errorf("ports is currently only supported on Linux")
}
