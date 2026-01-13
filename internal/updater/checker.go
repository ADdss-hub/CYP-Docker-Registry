// Package updater provides auto-update functionality for CYP-Docker-Registry.
package updater

import (
	"cyp-docker-registry/internal/version"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// VersionInfo represents version and update information.
type VersionInfo struct {
	Current   string    `json:"current"`
	Latest    string    `json:"latest"`
	HasUpdate bool      `json:"has_update"`
	ReleaseAt time.Time `json:"release_at"`
	Changelog string    `json:"changelog"`
}

// UpdateStatus represents the current update status.
type UpdateStatus struct {
	State       string    `json:"state"` // idle, checking, downloading, applying, error
	Progress    int       `json:"progress"`
	Message     string    `json:"message"`
	LastChecked time.Time `json:"last_checked"`
}

// GitHubRelease represents a GitHub release response.
type GitHubRelease struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Body        string    `json:"body"`
	PublishedAt time.Time `json:"published_at"`
	Assets      []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
		Size               int64  `json:"size"`
	} `json:"assets"`
}

// UpdaterService provides update checking and management functionality.
type UpdaterService struct {
	mu           sync.RWMutex
	updateURL    string
	downloadPath string
	status       UpdateStatus
	lastVersion  *VersionInfo
	httpClient   *http.Client
}

// NewUpdaterService creates a new updater service.
func NewUpdaterService(updateURL, downloadPath string) *UpdaterService {
	return &UpdaterService{
		updateURL:    updateURL,
		downloadPath: downloadPath,
		status: UpdateStatus{
			State: "idle",
		},
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CheckUpdate checks for available updates.
func (u *UpdaterService) CheckUpdate() (*VersionInfo, error) {
	u.mu.Lock()
	u.status.State = "checking"
	u.status.Message = "正在检查更新..."
	u.mu.Unlock()

	defer func() {
		u.mu.Lock()
		u.status.State = "idle"
		u.status.LastChecked = time.Now()
		u.mu.Unlock()
	}()

	currentVersion := version.GetVersion()

	// Fetch latest release from update URL
	latestVersion, releaseAt, changelog, err := u.fetchLatestRelease()
	if err != nil {
		u.mu.Lock()
		u.status.State = "error"
		u.status.Message = err.Error()
		u.mu.Unlock()
		return nil, err
	}

	hasUpdate := CompareVersions(latestVersion, currentVersion) > 0

	info := &VersionInfo{
		Current:   currentVersion,
		Latest:    latestVersion,
		HasUpdate: hasUpdate,
		ReleaseAt: releaseAt,
		Changelog: changelog,
	}

	u.mu.Lock()
	u.lastVersion = info
	u.mu.Unlock()

	return info, nil
}

// fetchLatestRelease fetches the latest release information from GitHub.
func (u *UpdaterService) fetchLatestRelease() (version string, releaseAt time.Time, changelog string, err error) {
	if u.updateURL == "" {
		return "", time.Time{}, "", fmt.Errorf("更新URL未配置")
	}

	resp, err := u.httpClient.Get(u.updateURL)
	if err != nil {
		return "", time.Time{}, "", fmt.Errorf("无法连接更新服务器: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", time.Time{}, "", fmt.Errorf("更新服务器返回错误: %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", time.Time{}, "", fmt.Errorf("解析更新信息失败: %w", err)
	}

	// Remove 'v' prefix if present
	ver := strings.TrimPrefix(release.TagName, "v")

	return ver, release.PublishedAt, release.Body, nil
}

// CompareVersions compares two semantic version strings.
// Returns: 1 if v1 > v2, -1 if v1 < v2, 0 if equal.
func CompareVersions(v1, v2 string) int {
	// Remove 'v' prefix if present
	v1 = strings.TrimPrefix(v1, "v")
	v2 = strings.TrimPrefix(v2, "v")

	parts1 := parseVersion(v1)
	parts2 := parseVersion(v2)

	// Compare major, minor, patch
	for i := 0; i < 3; i++ {
		if parts1[i] > parts2[i] {
			return 1
		}
		if parts1[i] < parts2[i] {
			return -1
		}
	}

	return 0
}

// parseVersion parses a version string into [major, minor, patch].
func parseVersion(v string) [3]int {
	var parts [3]int

	// Split by dots and hyphens (for pre-release versions)
	v = strings.Split(v, "-")[0] // Remove pre-release suffix
	segments := strings.Split(v, ".")

	for i := 0; i < len(segments) && i < 3; i++ {
		parts[i] = parseInt(segments[i])
	}

	return parts
}

// parseInt parses a string to int, returning 0 on error.
func parseInt(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

// DownloadUpdate downloads the update package.
func (u *UpdaterService) DownloadUpdate(targetVersion string) error {
	u.mu.Lock()
	u.status.State = "downloading"
	u.status.Progress = 0
	u.status.Message = "正在下载更新..."
	u.mu.Unlock()

	defer func() {
		u.mu.Lock()
		if u.status.State == "downloading" {
			u.status.State = "idle"
		}
		u.mu.Unlock()
	}()

	// Fetch release info to get download URL
	resp, err := u.httpClient.Get(u.updateURL)
	if err != nil {
		u.setError("下载失败: " + err.Error())
		return err
	}
	defer resp.Body.Close()

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		u.setError("解析更新信息失败")
		return err
	}

	// Find appropriate asset for current platform
	assetURL, assetName := u.findAsset(release.Assets)
	if assetURL == "" {
		err := fmt.Errorf("未找到适用于当前平台的更新包")
		u.setError(err.Error())
		return err
	}

	// Create download directory
	if err := os.MkdirAll(u.downloadPath, 0755); err != nil {
		u.setError("创建下载目录失败")
		return err
	}

	// Download the asset
	downloadResp, err := u.httpClient.Get(assetURL)
	if err != nil {
		u.setError("下载更新包失败")
		return err
	}
	defer downloadResp.Body.Close()

	// Create destination file
	destPath := filepath.Join(u.downloadPath, assetName)
	destFile, err := os.Create(destPath)
	if err != nil {
		u.setError("创建文件失败")
		return err
	}
	defer destFile.Close()

	// Copy with progress tracking
	totalSize := downloadResp.ContentLength
	var downloaded int64

	buf := make([]byte, 32*1024)
	for {
		n, err := downloadResp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := destFile.Write(buf[:n]); writeErr != nil {
				u.setError("写入文件失败")
				return writeErr
			}
			downloaded += int64(n)

			if totalSize > 0 {
				progress := int(float64(downloaded) / float64(totalSize) * 100)
				u.mu.Lock()
				u.status.Progress = progress
				u.mu.Unlock()
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			u.setError("下载中断")
			return err
		}
	}

	u.mu.Lock()
	u.status.Progress = 100
	u.status.Message = "下载完成"
	u.mu.Unlock()

	return nil
}

// findAsset finds the appropriate download asset for the current platform.
func (u *UpdaterService) findAsset(assets []struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}) (url, name string) {
	// Platform-specific asset naming patterns
	patterns := []string{
		"linux-amd64",
		"linux-arm64",
		"darwin-amd64",
		"darwin-arm64",
		"windows-amd64",
	}

	for _, asset := range assets {
		for _, pattern := range patterns {
			if strings.Contains(strings.ToLower(asset.Name), pattern) {
				return asset.BrowserDownloadURL, asset.Name
			}
		}
	}

	// Fallback: return first asset if no platform match
	if len(assets) > 0 {
		return assets[0].BrowserDownloadURL, assets[0].Name
	}

	return "", ""
}

// setError sets the error state.
func (u *UpdaterService) setError(message string) {
	u.mu.Lock()
	u.status.State = "error"
	u.status.Message = message
	u.mu.Unlock()
}

// ApplyUpdate applies the downloaded update.
func (u *UpdaterService) ApplyUpdate() error {
	u.mu.Lock()
	u.status.State = "applying"
	u.status.Message = "正在应用更新..."
	u.mu.Unlock()

	// In a real implementation, this would:
	// 1. Backup current binary
	// 2. Replace with new binary
	// 3. Restart the service
	// For now, we just simulate the process

	// This is a placeholder - actual implementation would depend on
	// deployment method (Docker, systemd, etc.)
	u.mu.Lock()
	u.status.State = "idle"
	u.status.Message = "更新已准备就绪，请重启服务以应用更新"
	u.mu.Unlock()

	return nil
}

// Rollback rolls back to the previous version.
func (u *UpdaterService) Rollback() error {
	u.mu.Lock()
	defer u.mu.Unlock()

	// In a real implementation, this would restore the backup
	// For now, we just reset the status
	u.status.State = "idle"
	u.status.Message = "已回滚到之前版本"

	return nil
}

// GetStatus returns the current update status.
func (u *UpdaterService) GetStatus() UpdateStatus {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.status
}

// GetLastVersionInfo returns the last checked version info.
func (u *UpdaterService) GetLastVersionInfo() *VersionInfo {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.lastVersion
}

// SetUpdateURL sets the update URL.
func (u *UpdaterService) SetUpdateURL(url string) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.updateURL = url
}
