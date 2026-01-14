// Package service provides business logic services for CYP-Docker-Registry.
package service

import (
	"os"
	"runtime"
	"sync"
	"time"

	"go.uber.org/zap"
)

// SystemService provides system information and management services.
type SystemService struct {
	startTime time.Time
	logger    *zap.Logger
	mu        sync.RWMutex
}

// SystemInfo represents system information.
type SystemInfo struct {
	Version     string          `json:"version"`
	BuildTime   string          `json:"build_time"`
	GoVersion   string          `json:"go_version"`
	OS          string          `json:"os"`
	Arch        string          `json:"arch"`
	NumCPU      int             `json:"num_cpu"`
	Hostname    string          `json:"hostname"`
	Uptime      string          `json:"uptime"`
	StartTime   time.Time       `json:"start_time"`
	Environment string          `json:"environment"`
	Features    map[string]bool `json:"features"`
}

// SystemStats represents system statistics.
type SystemStats struct {
	MemoryUsage    MemoryStats   `json:"memory_usage"`
	GoroutineCount int           `json:"goroutine_count"`
	CPUUsage       float64       `json:"cpu_usage"`
	DiskUsage      DiskStats     `json:"disk_usage"`
	Uptime         time.Duration `json:"uptime"`
}

// MemoryStats represents memory statistics.
type MemoryStats struct {
	Alloc      uint64 `json:"alloc"`
	TotalAlloc uint64 `json:"total_alloc"`
	Sys        uint64 `json:"sys"`
	NumGC      uint32 `json:"num_gc"`
	HeapAlloc  uint64 `json:"heap_alloc"`
	HeapSys    uint64 `json:"heap_sys"`
}

// DiskStats represents disk statistics.
type DiskStats struct {
	Total   uint64  `json:"total"`
	Used    uint64  `json:"used"`
	Free    uint64  `json:"free"`
	UsedPct float64 `json:"used_pct"`
}

// HealthStatus represents system health status.
type HealthStatus struct {
	Status    string        `json:"status"`
	Checks    []HealthCheck `json:"checks"`
	Timestamp time.Time     `json:"timestamp"`
}

// HealthCheck represents a single health check result.
type HealthCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// NewSystemService creates a new SystemService instance.
func NewSystemService(logger *zap.Logger) *SystemService {
	return &SystemService{
		startTime: time.Now(),
		logger:    logger,
	}
}

// GetSystemInfo returns system information.
func (s *SystemService) GetSystemInfo() *SystemInfo {
	hostname, _ := os.Hostname()

	return &SystemInfo{
		Version:     "1.0.6",
		BuildTime:   "2026-01-14",
		GoVersion:   runtime.Version(),
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
		NumCPU:      runtime.NumCPU(),
		Hostname:    hostname,
		Uptime:      s.formatDuration(time.Since(s.startTime)),
		StartTime:   s.startTime,
		Environment: s.detectEnvironment(),
		Features: map[string]bool{
			"accelerator": true,
			"p2p":         false,
			"signature":   true,
			"sbom":        true,
			"audit":       true,
			"auto_lock":   true,
		},
	}
}

// GetSystemStats returns system statistics.
func (s *SystemService) GetSystemStats() *SystemStats {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return &SystemStats{
		MemoryUsage: MemoryStats{
			Alloc:      memStats.Alloc,
			TotalAlloc: memStats.TotalAlloc,
			Sys:        memStats.Sys,
			NumGC:      memStats.NumGC,
			HeapAlloc:  memStats.HeapAlloc,
			HeapSys:    memStats.HeapSys,
		},
		GoroutineCount: runtime.NumGoroutine(),
		Uptime:         time.Since(s.startTime),
		DiskUsage:      s.getDiskUsage(),
	}
}

// GetHealthStatus returns system health status.
func (s *SystemService) GetHealthStatus() *HealthStatus {
	checks := []HealthCheck{
		s.checkMemory(),
		s.checkDisk(),
		s.checkGoroutines(),
	}

	status := "healthy"
	for _, check := range checks {
		if check.Status == "unhealthy" {
			status = "unhealthy"
			break
		} else if check.Status == "degraded" && status == "healthy" {
			status = "degraded"
		}
	}

	return &HealthStatus{
		Status:    status,
		Checks:    checks,
		Timestamp: time.Now(),
	}
}

// checkMemory checks memory health.
func (s *SystemService) checkMemory() HealthCheck {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Check if memory usage is too high (> 80% of sys)
	usedPct := float64(memStats.Alloc) / float64(memStats.Sys) * 100

	check := HealthCheck{Name: "memory"}
	if usedPct > 90 {
		check.Status = "unhealthy"
		check.Message = "Memory usage critical"
	} else if usedPct > 80 {
		check.Status = "degraded"
		check.Message = "Memory usage high"
	} else {
		check.Status = "healthy"
	}

	return check
}

// checkDisk checks disk health.
func (s *SystemService) checkDisk() HealthCheck {
	disk := s.getDiskUsage()

	check := HealthCheck{Name: "disk"}
	if disk.UsedPct > 95 {
		check.Status = "unhealthy"
		check.Message = "Disk space critical"
	} else if disk.UsedPct > 85 {
		check.Status = "degraded"
		check.Message = "Disk space low"
	} else {
		check.Status = "healthy"
	}

	return check
}

// checkGoroutines checks goroutine health.
func (s *SystemService) checkGoroutines() HealthCheck {
	count := runtime.NumGoroutine()

	check := HealthCheck{Name: "goroutines"}
	if count > 10000 {
		check.Status = "unhealthy"
		check.Message = "Too many goroutines"
	} else if count > 5000 {
		check.Status = "degraded"
		check.Message = "High goroutine count"
	} else {
		check.Status = "healthy"
	}

	return check
}

// getDiskUsage returns disk usage statistics.
func (s *SystemService) getDiskUsage() DiskStats {
	// Simplified disk usage - in production use syscall or external library
	return DiskStats{
		Total:   100 * 1024 * 1024 * 1024, // 100GB placeholder
		Used:    50 * 1024 * 1024 * 1024,  // 50GB placeholder
		Free:    50 * 1024 * 1024 * 1024,  // 50GB placeholder
		UsedPct: 50.0,
	}
}

// detectEnvironment detects the running environment.
func (s *SystemService) detectEnvironment() string {
	// Check for Docker
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return "docker"
	}

	// Check for Kubernetes
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		return "kubernetes"
	}

	// Check for cloud providers
	if os.Getenv("AWS_REGION") != "" {
		return "cloud-aws"
	}
	if os.Getenv("ALIBABA_CLOUD_REGION") != "" {
		return "cloud-aliyun"
	}

	// Default to physical
	return "physical"
}

// formatDuration formats a duration as a human-readable string.
func (s *SystemService) formatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return formatDurationString(days, hours, minutes)
	} else if hours > 0 {
		return formatHoursMinutes(hours, minutes)
	}
	return formatMinutes(minutes)
}

func formatDurationString(days, hours, minutes int) string {
	return string(rune(days)) + "d " + string(rune(hours)) + "h " + string(rune(minutes)) + "m"
}

func formatHoursMinutes(hours, minutes int) string {
	return string(rune(hours)) + "h " + string(rune(minutes)) + "m"
}

func formatMinutes(minutes int) string {
	return string(rune(minutes)) + "m"
}

// GetUptime returns the system uptime.
func (s *SystemService) GetUptime() time.Duration {
	return time.Since(s.startTime)
}

// TriggerGC triggers garbage collection.
func (s *SystemService) TriggerGC() {
	runtime.GC()
	if s.logger != nil {
		s.logger.Info("Garbage collection triggered")
	}
}

// GetVersion returns the system version.
func (s *SystemService) GetVersion() string {
	return "1.0.3"
}
