package proc

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// InodeToPID maps a socket inode to its owning PID.
type InodeToPID map[uint64]int

// ScanInodesToPIDs iterates through /proc/<pid>/fd/ to map socket inodes to PIDs.
// targetInodes is the set of socket inodes we care about.
func ScanInodesToPIDs(procPath string, targetInodes map[uint64]struct{}) (InodeToPID, error) {
	if procPath == "" {
		procPath = "/proc"
	}

	result := make(InodeToPID)
	if len(targetInodes) == 0 {
		return result, nil
	}

	entries, err := os.ReadDir(procPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", procPath, err)
	}

	// Remaining inodes to find
	remaining := len(targetInodes)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		// Check if the directory name is a PID (all digits)
		pid, err := strconv.Atoi(name)
		if err != nil || pid <= 0 {
			continue
		}

		fdDir := filepath.Join(procPath, name, "fd")
		fdEntries, err := os.ReadDir(fdDir)
		if err != nil {
			// Permission denied (e.g. process belongs to another user) or exited
			continue
		}

		for _, fdEntry := range fdEntries {
			fdPath := filepath.Join(fdDir, fdEntry.Name())
			linkTarget, err := os.Readlink(fdPath)
			if err != nil {
				// Races: fd closed between ReadDir and Readlink
				continue
			}

			inode, ok := parseSocketInode(linkTarget)
			if !ok {
				continue
			}

			if _, exists := targetInodes[inode]; exists {
				result[inode] = pid
				delete(targetInodes, inode)
				remaining--
				if remaining == 0 {
					return result, nil
				}
			}
		}
	}

	return result, nil
}

// parseSocketInode extracts the inode from "socket:[12345]" or "[0000]:12345".
func parseSocketInode(target string) (uint64, bool) {
	if strings.HasPrefix(target, "socket:[") && strings.HasSuffix(target, "]") {
		inodeStr := target[8 : len(target)-1]
		inode, err := strconv.ParseUint(inodeStr, 10, 64)
		if err == nil && inode > 0 {
			return inode, true
		}
	}
	return 0, false
}
