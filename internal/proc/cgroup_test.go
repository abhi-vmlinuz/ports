package proc

import (
	"testing"
)

func TestParseCgroupPaths(t *testing.T) {
	tests := []struct {
		name     string
		paths    []string
		expected string
	}{
		{
			name:     "Empty paths",
			paths:    []string{},
			expected: "",
		},
		{
			name:     "Root path",
			paths:    []string{"/"},
			expected: "",
		},
		{
			name:     "cgroup v2 systemd service",
			paths:    []string{"/system.slice/docker.service"},
			expected: "systemd (docker.service)",
		},
		{
			name:     "cgroup v2 systemd sshd service",
			paths:    []string{"/system.slice/sshd.service"},
			expected: "systemd (sshd.service)",
		},
		{
			name:     "cgroup v2 docker container scope",
			paths:    []string{"/system.slice/docker-3b3b6fe5f770b2cf0fa23bcff2a08161f54f9d109e4454ac62d032caa8bc8ddc.scope"},
			expected: "docker (3b3b6fe5f770)",
		},
		{
			name:     "cgroup v1 docker path",
			paths:    []string{"/docker/3b3b6fe5f770b2cf0fa23bcff2a08161f54f9d109e4454ac62d032caa8bc8ddc"},
			expected: "docker (3b3b6fe5f770)",
		},
		{
			name:     "cgroup v2 podman container scope",
			paths:    []string{"/machine.slice/libpod-a1b2c3d4e5f67890abcdef1234567890abcdef12.scope"},
			expected: "podman (a1b2c3d4e5f6)",
		},
		{
			name:     "cgroup v2 k8s pod",
			paths:    []string{"/kubepods.slice/kubepods-burstable.slice/kubepods-burstable-pod66b96e95_c220_41ef_8dfc_305d04c44f0b.slice"},
			expected: "k8s (pod-66b96e95)",
		},
		{
			name:     "cgroup v2 systemd user service",
			paths:    []string{"/user.slice/user-1000.slice/user@1000.service/app.slice/mpd.service"},
			expected: "systemd-user (mpd.service)",
		},
		{
			name:     "cgroup v2 desktop app scope",
			paths:    []string{"/user.slice/user-1000.slice/user@1000.service/app.slice/app-antigravity-6887.scope"},
			expected: "app (antigravity)",
		},
		{
			name:     "cgroup v2 interactive terminal session",
			paths:    []string{"/user.slice/user-1000.slice/session-3.scope"},
			expected: "interactive",
		},
		{
			name:     "cgroup v2 init scope",
			paths:    []string{"/init.scope"},
			expected: "init",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := ParseCgroupPaths(tc.paths)
			if actual != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, actual)
			}
		})
	}
}
