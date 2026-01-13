// Package version provides version information for the container registry.
package version

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Build-time variables (can be set via ldflags)
var (
	Version   = ""
	BuildTime = "unknown"
	GitCommit = "unknown"
)

var (
	once          sync.Once
	cachedVersion string
)

// GetVersion returns the current version number.
// It reads from the VERSION file if the build-time variable is not set.
func GetVersion() string {
	once.Do(func() {
		if Version != "" {
			cachedVersion = Version
			return
		}
		cachedVersion = readVersionFile()
	})
	return cachedVersion
}

// GetFullVersion returns the full version string including build information.
func GetFullVersion() string {
	ver := GetVersion()
	return ver + " (build: " + BuildTime + ", commit: " + GitCommit + ")"
}

// readVersionFile reads the version from the VERSION file.
func readVersionFile() string {
	// Try multiple paths to find VERSION file
	paths := []string{
		"VERSION",
		"../VERSION",
		"../../VERSION",
	}

	// Also try relative to executable
	if execPath, err := os.Executable(); err == nil {
		execDir := filepath.Dir(execPath)
		paths = append(paths,
			filepath.Join(execDir, "VERSION"),
			filepath.Join(execDir, "..", "VERSION"),
		)
	}

	for _, path := range paths {
		if data, err := os.ReadFile(path); err == nil {
			return strings.TrimSpace(string(data))
		}
	}

	return "0.0.0"
}

// ResetCache resets the cached version (useful for testing).
func ResetCache() {
	once = sync.Once{}
	cachedVersion = ""
}
