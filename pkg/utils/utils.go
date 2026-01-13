// Package utils provides utility functions for CYP-Registry.
package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// GenerateID generates a random ID.
func GenerateID(prefix string, length int) string {
	bytes := make([]byte, length/2)
	rand.Read(bytes)
	return prefix + hex.EncodeToString(bytes)
}

// GenerateToken generates a random token.
func GenerateToken(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// HashPassword hashes a password using SHA256.
func HashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

// VerifyPassword verifies a password against a hash.
func VerifyPassword(password, hash string) bool {
	return HashPassword(password) == hash
}

// ParseSize parses a size string like "10GB" into bytes.
func ParseSize(s string) int64 {
	if s == "" {
		return 0
	}

	s = strings.TrimSpace(strings.ToUpper(s))

	var multiplier int64 = 1
	numStr := s

	if strings.HasSuffix(s, "GB") {
		multiplier = 1024 * 1024 * 1024
		numStr = s[:len(s)-2]
	} else if strings.HasSuffix(s, "MB") {
		multiplier = 1024 * 1024
		numStr = s[:len(s)-2]
	} else if strings.HasSuffix(s, "KB") {
		multiplier = 1024
		numStr = s[:len(s)-2]
	} else if strings.HasSuffix(s, "B") {
		numStr = s[:len(s)-1]
	}

	num, _ := strconv.ParseInt(strings.TrimSpace(numStr), 10, 64)
	return num * multiplier
}

// FormatSize formats bytes into a human-readable string.
func FormatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// ParseDuration parses a duration string like "1h30m" into time.Duration.
func ParseDuration(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}

// FormatDuration formats a duration into a human-readable string.
func FormatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

// IsValidEmail validates an email address.
func IsValidEmail(email string) bool {
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	matched, _ := regexp.MatchString(pattern, email)
	return matched
}

// IsValidUsername validates a username.
func IsValidUsername(username string) bool {
	pattern := `^[a-zA-Z][a-zA-Z0-9_-]{2,31}$`
	matched, _ := regexp.MatchString(pattern, username)
	return matched
}

// IsValidImageRef validates a container image reference.
func IsValidImageRef(ref string) bool {
	pattern := `^[a-z0-9]+([._-][a-z0-9]+)*(/[a-z0-9]+([._-][a-z0-9]+)*)*(:[\w][\w.-]{0,127})?(@sha256:[a-f0-9]{64})?$`
	matched, _ := regexp.MatchString(pattern, ref)
	return matched
}

// SanitizeFilename sanitizes a filename for safe storage.
func SanitizeFilename(name string) string {
	// Remove or replace unsafe characters
	safe := ""
	for _, c := range name {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' {
			safe += string(c)
		} else {
			safe += "_"
		}
	}
	return safe
}

// FileExists checks if a file exists.
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// DirExists checks if a directory exists.
func DirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// EnsureDir ensures a directory exists.
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// GetEnv gets an environment variable with a default value.
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetEnvInt gets an environment variable as an integer with a default value.
func GetEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

// GetEnvBool gets an environment variable as a boolean with a default value.
func GetEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return defaultValue
}

// TruncateString truncates a string to a maximum length.
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// ContainsString checks if a slice contains a string.
func ContainsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// UniqueStrings returns unique strings from a slice.
func UniqueStrings(slice []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)
	for _, s := range slice {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
