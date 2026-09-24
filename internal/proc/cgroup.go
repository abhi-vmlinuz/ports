package proc

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	dockerScopeRegex = regexp.MustCompile(`docker-([0-9a-fA-F]{12,64})\.scope`)
	dockerPathRegex  = regexp.MustCompile(`/docker/([0-9a-fA-F]{12,64})`)
	podmanScopeRegex = regexp.MustCompile(`libpod-([0-9a-fA-F]{12,64})\.scope`)
	podmanPathRegex  = regexp.MustCompile(`(?:/libpod|/podman)/([0-9a-fA-F]{12,64})`)
	k8sPodRegex      = regexp.MustCompile(`pod([0-9a-fA-F_-]{8,36})`)
	systemdUserRegex = regexp.MustCompile(`user@\d+\.service/(?:app\.slice/)?([a-zA-Z0-9_\-\.@]+\.service)`)
	systemdSysRegex  = regexp.MustCompile(`(?:system\.slice/)([a-zA-Z0-9_\-\.@]+\.service)`)
	appScopeRegex    = regexp.MustCompile(`app(?:-gnome)?-([a-zA-Z0-9_\-\.]+?)(?:-[0-9]+)?\.scope`)
)

// DetectOrigin reads /proc/<pid>/cgroup and returns a human-readable origin identifier,
// such as "docker (3b3b6fe5f770)", "systemd (nginx.service)", or "interactive".
// Returns empty string if the cgroup cannot be read or categorized.
func DetectOrigin(procPath string, pid int) string {
	cgroupPath := filepath.Join(procPath, strconvItoa(pid), "cgroup")
	f, err := os.Open(cgroupPath)
	if err != nil {
		return ""
	}
	defer f.Close()

	var paths []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		// Line format: hierarchy-ID:controller-list:cgroup-path
		parts := strings.SplitN(line, ":", 3)
		if len(parts) == 3 {
			paths = append(paths, parts[2])
		}
	}

	return ParseCgroupPaths(paths)
}

// ParseCgroupPaths analyzes a slice of cgroup path strings and classifies the lifecycle controller.
func ParseCgroupPaths(paths []string) string {
	for _, p := range paths {
		if orig := classifyCgroupPath(p); orig != "" {
			return orig
		}
	}
	return ""
}

func classifyCgroupPath(path string) string {
	if path == "" || path == "/" {
		return ""
	}

	// 1. Kubernetes
	if strings.Contains(path, "kubepods") || strings.Contains(path, "k8s_") {
		if m := k8sPodRegex.FindStringSubmatch(path); len(m) > 1 {
			podID := m[1]
			if len(podID) > 8 {
				podID = podID[:8]
			}
			return "k8s (pod-" + podID + ")"
		}
		return "k8s"
	}

	// 2. Docker container
	if m := dockerScopeRegex.FindStringSubmatch(path); len(m) > 1 {
		id := m[1]
		if len(id) > 12 {
			id = id[:12]
		}
		return "docker (" + id + ")"
	}
	if m := dockerPathRegex.FindStringSubmatch(path); len(m) > 1 {
		id := m[1]
		if len(id) > 12 {
			id = id[:12]
		}
		return "docker (" + id + ")"
	}

	// 3. Podman container
	if m := podmanScopeRegex.FindStringSubmatch(path); len(m) > 1 {
		id := m[1]
		if len(id) > 12 {
			id = id[:12]
		}
		return "podman (" + id + ")"
	}
	if m := podmanPathRegex.FindStringSubmatch(path); len(m) > 1 {
		id := m[1]
		if len(id) > 12 {
			id = id[:12]
		}
		return "podman (" + id + ")"
	}

	// 4. Containerd direct
	if strings.Contains(path, "/containerd/") || strings.Contains(path, "containerd-") {
		return "containerd"
	}

	// 5. systemd user service
	if m := systemdUserRegex.FindStringSubmatch(path); len(m) > 1 {
		return "systemd-user (" + m[1] + ")"
	}

	// 6. systemd system service
	if m := systemdSysRegex.FindStringSubmatch(path); len(m) > 1 {
		return "systemd (" + m[1] + ")"
	}

	// 7. Desktop App scope
	if m := appScopeRegex.FindStringSubmatch(path); len(m) > 1 {
		name := m[1]
		name = strings.TrimPrefix(name, "app-")
		return "app (" + name + ")"
	}

	// 8. Interactive terminal session
	if strings.Contains(path, "session-") && strings.Contains(path, ".scope") {
		return "interactive"
	}

	// 9. Init system
	if strings.Contains(path, "init.scope") {
		return "init"
	}

	return ""
}

// strconvItoa avoids importing strconv if not needed, but strconv.Itoa is fine.
func strconvItoa(val int) string {
	return strconvFormatInt(int64(val), 10)
}

func strconvFormatInt(i int64, base int) string {
	if i == 0 {
		return "0"
	}
	var b [32]byte
	bp := len(b)
	neg := i < 0
	if neg {
		i = -i
	}
	for i > 0 {
		bp--
		b[bp] = byte('0' + (i % 10))
		i /= 10
	}
	if neg {
		bp--
		b[bp] = '-'
	}
	return string(b[bp:])
}
