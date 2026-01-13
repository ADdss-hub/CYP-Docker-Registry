// Package detector provides host system detection functionality.
package detector

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
)

// SystemInfo represents host system information.
type SystemInfo struct {
	OS            string `json:"os"`
	OSVersion     string `json:"os_version"`
	Arch          string `json:"arch"`
	Hostname      string `json:"hostname"`
	DockerVersion string `json:"docker_version"`
	ContainerdVer string `json:"containerd_version"`
	CPUCores      int    `json:"cpu_cores"`
	MemoryTotal   int64  `json:"memory_total"`
	DiskTotal     int64  `json:"disk_total"`
	DiskFree      int64  `json:"disk_free"`
}

// CompatibilityReport represents system compatibility check results.
type CompatibilityReport struct {
	Compatible bool                `json:"compatible"`
	Warnings   []CompatibilityWarn `json:"warnings,omitempty"`
	Errors     []CompatibilityErr  `json:"errors,omitempty"`
}

// CompatibilityWarn represents a compatibility warning.
type CompatibilityWarn struct {
	Component string `json:"component"`
	Message   string `json:"message"`
}

// CompatibilityErr represents a compatibility error.
type CompatibilityErr struct {
	Component string `json:"component"`
	Message   string `json:"message"`
}

// DetectorService provides system detection functionality.
type DetectorService struct {
	mu         sync.RWMutex
	cachedInfo *SystemInfo
}

// NewDetectorService creates a new detector service.
func NewDetectorService() *DetectorService {
	return &DetectorService{}
}

// GetSystemInfo retrieves current system information.
func (d *DetectorService) GetSystemInfo() (*SystemInfo, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	info := &SystemInfo{
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		CPUCores: runtime.NumCPU(),
	}

	// Get hostname
	if hostname, err := os.Hostname(); err == nil {
		info.Hostname = hostname
	}

	// Get OS version
	info.OSVersion = d.getOSVersion()

	// Get Docker version
	info.DockerVersion = d.getDockerVersion()

	// Get containerd version
	info.ContainerdVer = d.getContainerdVersion()

	// Get memory info
	info.MemoryTotal = d.getMemoryTotal()

	// Get disk info
	info.DiskTotal, info.DiskFree = d.getDiskInfo()

	d.cachedInfo = info
	return info, nil
}

// getOSVersion retrieves the OS version string.
func (d *DetectorService) getOSVersion() string {
	switch runtime.GOOS {
	case "linux":
		return d.getLinuxVersion()
	case "darwin":
		return d.getDarwinVersion()
	case "windows":
		return d.getWindowsVersion()
	default:
		return "unknown"
	}
}

// getLinuxVersion retrieves Linux distribution version.
func (d *DetectorService) getLinuxVersion() string {
	// Try /etc/os-release first
	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		lines := strings.Split(string(data), "\n")
		var prettyName string
		for _, line := range lines {
			if strings.HasPrefix(line, "PRETTY_NAME=") {
				prettyName = strings.Trim(line[12:], "\"")
				break
			}
		}
		if prettyName != "" {
			return prettyName
		}
	}

	// Fallback to uname
	if out, err := exec.Command("uname", "-r").Output(); err == nil {
		return strings.TrimSpace(string(out))
	}

	return "Linux"
}

// getDarwinVersion retrieves macOS version.
func (d *DetectorService) getDarwinVersion() string {
	if out, err := exec.Command("sw_vers", "-productVersion").Output(); err == nil {
		return "macOS " + strings.TrimSpace(string(out))
	}
	return "macOS"
}

// getWindowsVersion retrieves Windows version.
func (d *DetectorService) getWindowsVersion() string {
	if out, err := exec.Command("cmd", "/c", "ver").Output(); err == nil {
		return strings.TrimSpace(string(out))
	}
	return "Windows"
}

// getDockerVersion retrieves Docker version.
func (d *DetectorService) getDockerVersion() string {
	out, err := exec.Command("docker", "version", "--format", "{{.Server.Version}}").Output()
	if err != nil {
		// Try alternative format
		out, err = exec.Command("docker", "--version").Output()
		if err != nil {
			return "not installed"
		}
		// Parse "Docker version X.Y.Z, build abc123"
		version := strings.TrimSpace(string(out))
		if strings.HasPrefix(version, "Docker version ") {
			parts := strings.Split(version[15:], ",")
			if len(parts) > 0 {
				return strings.TrimSpace(parts[0])
			}
		}
		return version
	}
	return strings.TrimSpace(string(out))
}

// getContainerdVersion retrieves containerd version.
func (d *DetectorService) getContainerdVersion() string {
	out, err := exec.Command("containerd", "--version").Output()
	if err != nil {
		return "not installed"
	}
	// Parse "containerd containerd.io X.Y.Z abc123"
	version := strings.TrimSpace(string(out))
	parts := strings.Fields(version)
	if len(parts) >= 3 {
		return parts[2]
	}
	return version
}

// getMemoryTotal retrieves total system memory in bytes.
func (d *DetectorService) getMemoryTotal() int64 {
	switch runtime.GOOS {
	case "linux":
		if data, err := os.ReadFile("/proc/meminfo"); err == nil {
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "MemTotal:") {
					fields := strings.Fields(line)
					if len(fields) >= 2 {
						var kb int64
						for _, c := range fields[1] {
							if c >= '0' && c <= '9' {
								kb = kb*10 + int64(c-'0')
							}
						}
						return kb * 1024 // Convert KB to bytes
					}
				}
			}
		}
	case "darwin":
		if out, err := exec.Command("sysctl", "-n", "hw.memsize").Output(); err == nil {
			var mem int64
			for _, c := range strings.TrimSpace(string(out)) {
				if c >= '0' && c <= '9' {
					mem = mem*10 + int64(c-'0')
				}
			}
			return mem
		}
	case "windows":
		if out, err := exec.Command("wmic", "computersystem", "get", "TotalPhysicalMemory").Output(); err == nil {
			lines := strings.Split(string(out), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line != "" && line != "TotalPhysicalMemory" {
					var mem int64
					for _, c := range line {
						if c >= '0' && c <= '9' {
							mem = mem*10 + int64(c-'0')
						}
					}
					return mem
				}
			}
		}
	}
	return 0
}

// getDiskInfo retrieves disk total and free space in bytes.
func (d *DetectorService) getDiskInfo() (total, free int64) {
	switch runtime.GOOS {
	case "linux", "darwin":
		if out, err := exec.Command("df", "-B1", ".").Output(); err == nil {
			lines := strings.Split(string(out), "\n")
			if len(lines) >= 2 {
				fields := strings.Fields(lines[1])
				if len(fields) >= 4 {
					total = parseNumber(fields[1])
					free = parseNumber(fields[3])
				}
			}
		}
	case "windows":
		// Get current drive
		if cwd, err := os.Getwd(); err == nil && len(cwd) >= 2 {
			drive := cwd[:2]
			if out, err := exec.Command("wmic", "logicaldisk", "where", "DeviceID='"+drive+"'", "get", "Size,FreeSpace").Output(); err == nil {
				lines := strings.Split(string(out), "\n")
				for _, line := range lines {
					fields := strings.Fields(line)
					if len(fields) >= 2 {
						// First field is FreeSpace, second is Size
						freeVal := parseNumber(fields[0])
						totalVal := parseNumber(fields[1])
						if totalVal > 0 {
							return totalVal, freeVal
						}
					}
				}
			}
		}
	}
	return
}

// parseNumber parses a numeric string to int64.
func parseNumber(s string) int64 {
	var n int64
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int64(c-'0')
		}
	}
	return n
}

// CheckCompatibility checks system compatibility for running the registry.
func (d *DetectorService) CheckCompatibility() (*CompatibilityReport, error) {
	info, err := d.GetSystemInfo()
	if err != nil {
		return nil, err
	}

	report := &CompatibilityReport{
		Compatible: true,
	}

	// Check Docker installation
	if info.DockerVersion == "not installed" {
		report.Warnings = append(report.Warnings, CompatibilityWarn{
			Component: "Docker",
			Message:   "Docker未安装，部分功能可能不可用",
		})
	}

	// Check minimum memory (recommend at least 1GB)
	minMemory := int64(1024 * 1024 * 1024) // 1GB
	if info.MemoryTotal > 0 && info.MemoryTotal < minMemory {
		report.Warnings = append(report.Warnings, CompatibilityWarn{
			Component: "Memory",
			Message:   "系统内存低于推荐值(1GB)，可能影响性能",
		})
	}

	// Check minimum disk space (recommend at least 10GB free)
	minDisk := int64(10 * 1024 * 1024 * 1024) // 10GB
	if info.DiskFree > 0 && info.DiskFree < minDisk {
		report.Warnings = append(report.Warnings, CompatibilityWarn{
			Component: "Disk",
			Message:   "可用磁盘空间低于推荐值(10GB)，可能影响镜像存储",
		})
	}

	// Check supported OS
	supportedOS := map[string]bool{
		"linux":   true,
		"darwin":  true,
		"windows": true,
	}
	if !supportedOS[info.OS] {
		report.Compatible = false
		report.Errors = append(report.Errors, CompatibilityErr{
			Component: "OS",
			Message:   "不支持的操作系统: " + info.OS,
		})
	}

	// Check supported architecture
	supportedArch := map[string]bool{
		"amd64": true,
		"arm64": true,
		"arm":   true,
	}
	if !supportedArch[info.Arch] {
		report.Warnings = append(report.Warnings, CompatibilityWarn{
			Component: "Architecture",
			Message:   "架构 " + info.Arch + " 可能存在兼容性问题",
		})
	}

	return report, nil
}

// GetCachedInfo returns cached system info if available.
func (d *DetectorService) GetCachedInfo() *SystemInfo {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.cachedInfo
}
