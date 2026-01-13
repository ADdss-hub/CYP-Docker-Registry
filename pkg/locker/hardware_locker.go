// Package locker provides system locking mechanisms for security enforcement.
package locker

import (
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"sync"
)

// HardwareLocker implements hardware resource limiting for security lockdown.
type HardwareLocker struct {
	originalMemoryLimit int64
	originalCPUQuota    int64
	isLocked            bool
	lockCPUPercent      int
	lockMemoryPercent   int
	containerID         string
	mu                  sync.Mutex
}

// HardwareLockerConfig holds configuration for hardware locking.
type HardwareLockerConfig struct {
	LockCPUPercent    int
	LockMemoryPercent int
	ContainerID       string
}

// NewHardwareLocker creates a new HardwareLocker instance.
func NewHardwareLocker(config *HardwareLockerConfig) *HardwareLocker {
	if config == nil {
		config = &HardwareLockerConfig{
			LockCPUPercent:    10,
			LockMemoryPercent: 10,
		}
	}

	return &HardwareLocker{
		lockCPUPercent:    config.LockCPUPercent,
		lockMemoryPercent: config.LockMemoryPercent,
		containerID:       config.ContainerID,
	}
}

// Lock restricts hardware resources.
func (l *HardwareLocker) Lock() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.isLocked {
		return nil
	}

	// Linux: Use cgroup to limit resources
	if runtime.GOOS == "linux" {
		if err := l.lockLinux(); err != nil {
			return err
		}
	}

	// Docker: Use docker update command
	if l.isDocker() {
		if err := l.lockDocker(); err != nil {
			return err
		}
	}

	l.isLocked = true
	return nil
}

// Unlock restores hardware resources.
func (l *HardwareLocker) Unlock() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.isLocked {
		return nil
	}

	// Linux: Restore cgroup settings
	if runtime.GOOS == "linux" {
		if err := l.unlockLinux(); err != nil {
			return err
		}
	}

	// Docker: Restore container resources
	if l.isDocker() {
		if err := l.unlockDocker(); err != nil {
			return err
		}
	}

	l.isLocked = false
	return nil
}

// IsLocked returns the current lock status.
func (l *HardwareLocker) IsLocked() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.isLocked
}

// lockLinux applies cgroup limits on Linux.
func (l *HardwareLocker) lockLinux() error {
	// CPU limit using cgroup v1
	cgroupCPUPath := "/sys/fs/cgroup/cpu/cpu.cfs_quota_us"
	if _, err := os.Stat(cgroupCPUPath); err == nil {
		// Read current quota
		if data, err := os.ReadFile(cgroupCPUPath); err == nil {
			l.originalCPUQuota, _ = strconv.ParseInt(string(data), 10, 64)
		}

		// Set limited quota (100000 = 100% of one CPU)
		quota := int64(100000) * int64(l.lockCPUPercent) / 100
		os.WriteFile(cgroupCPUPath, []byte(strconv.FormatInt(quota, 10)), 0644)
	}

	// Memory limit using cgroup v1
	cgroupMemPath := "/sys/fs/cgroup/memory/memory.limit_in_bytes"
	if _, err := os.Stat(cgroupMemPath); err == nil {
		if data, err := os.ReadFile(cgroupMemPath); err == nil {
			total, _ := strconv.ParseInt(string(data), 10, 64)
			l.originalMemoryLimit = total

			// Set limited memory
			memLimit := total * int64(l.lockMemoryPercent) / 100
			os.WriteFile(cgroupMemPath, []byte(strconv.FormatInt(memLimit, 10)), 0644)
		}
	}

	// Try cgroup v2 if v1 not available
	cgroupV2Path := "/sys/fs/cgroup/cgroup.controllers"
	if _, err := os.Stat(cgroupV2Path); err == nil {
		l.lockCgroupV2()
	}

	return nil
}

// unlockLinux restores cgroup settings on Linux.
func (l *HardwareLocker) unlockLinux() error {
	// Restore CPU quota
	cgroupCPUPath := "/sys/fs/cgroup/cpu/cpu.cfs_quota_us"
	if l.originalCPUQuota > 0 {
		os.WriteFile(cgroupCPUPath, []byte(strconv.FormatInt(l.originalCPUQuota, 10)), 0644)
	} else {
		os.WriteFile(cgroupCPUPath, []byte("-1"), 0644)
	}

	// Restore memory limit
	cgroupMemPath := "/sys/fs/cgroup/memory/memory.limit_in_bytes"
	if l.originalMemoryLimit > 0 {
		os.WriteFile(cgroupMemPath, []byte(strconv.FormatInt(l.originalMemoryLimit, 10)), 0644)
	}

	return nil
}

// lockCgroupV2 applies limits using cgroup v2.
func (l *HardwareLocker) lockCgroupV2() error {
	// cgroup v2 uses different paths
	cgroupPath := "/sys/fs/cgroup"

	// CPU limit: cpu.max format is "quota period"
	cpuMaxPath := cgroupPath + "/cpu.max"
	if _, err := os.Stat(cpuMaxPath); err == nil {
		quota := int64(100000) * int64(l.lockCPUPercent) / 100
		os.WriteFile(cpuMaxPath, []byte(strconv.FormatInt(quota, 10)+" 100000"), 0644)
	}

	// Memory limit: memory.max
	memMaxPath := cgroupPath + "/memory.max"
	if _, err := os.Stat(memMaxPath); err == nil {
		if data, err := os.ReadFile(memMaxPath); err == nil {
			if string(data) != "max" {
				total, _ := strconv.ParseInt(string(data), 10, 64)
				memLimit := total * int64(l.lockMemoryPercent) / 100
				os.WriteFile(memMaxPath, []byte(strconv.FormatInt(memLimit, 10)), 0644)
			}
		}
	}

	return nil
}

// lockDocker applies limits using docker update command.
func (l *HardwareLocker) lockDocker() error {
	if l.containerID == "" {
		l.containerID = l.detectContainerID()
	}

	if l.containerID == "" {
		return nil
	}

	// Calculate CPU limit (0.1 = 10%)
	cpuLimit := float64(l.lockCPUPercent) / 100.0

	cmd := exec.Command("docker", "update",
		"--cpus", strconv.FormatFloat(cpuLimit, 'f', 2, 64),
		"--memory", "100m",
		l.containerID,
	)
	return cmd.Run()
}

// unlockDocker restores container resources.
func (l *HardwareLocker) unlockDocker() error {
	if l.containerID == "" {
		return nil
	}

	// Remove CPU and memory limits
	cmd := exec.Command("docker", "update",
		"--cpus", "0",
		"--memory", "0",
		l.containerID,
	)
	return cmd.Run()
}

// isDocker checks if running inside a Docker container.
func (l *HardwareLocker) isDocker() bool {
	// Check for .dockerenv file
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	// Check cgroup for docker
	if data, err := os.ReadFile("/proc/1/cgroup"); err == nil {
		return contains(string(data), "docker")
	}

	return false
}

// detectContainerID attempts to detect the current container ID.
func (l *HardwareLocker) detectContainerID() string {
	// Try to read from cgroup
	if data, err := os.ReadFile("/proc/self/cgroup"); err == nil {
		lines := splitLines(string(data))
		for _, line := range lines {
			if contains(line, "docker") {
				parts := splitString(line, "/")
				if len(parts) > 0 {
					return parts[len(parts)-1]
				}
			}
		}
	}

	// Try hostname (often set to container ID)
	if hostname, err := os.Hostname(); err == nil && len(hostname) == 12 {
		return hostname
	}

	return ""
}

// Helper functions
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func splitString(s, sep string) []string {
	var parts []string
	start := 0
	for i := 0; i <= len(s)-len(sep); i++ {
		if s[i:i+len(sep)] == sep {
			parts = append(parts, s[start:i])
			start = i + len(sep)
			i = start - 1
		}
	}
	if start < len(s) {
		parts = append(parts, s[start:])
	}
	return parts
}
