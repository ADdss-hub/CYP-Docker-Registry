// Package updater provides auto-update functionality for CYP-Docker-Registry.
package updater

import (
	"context"
	"cyp-docker-registry/internal/version"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// VersionInfo represents version and update information.
type VersionInfo struct {
	Current     string    `json:"current"`
	Latest      string    `json:"latest"`
	HasUpdate   bool      `json:"has_update"`
	ReleaseAt   time.Time `json:"release_at"`
	Changelog   string    `json:"changelog"`
	DownloadURL string    `json:"download_url,omitempty"`
	DockerImage string    `json:"docker_image,omitempty"`
	IsDocker    bool      `json:"is_docker"`
	AutoUpdate  bool      `json:"auto_update_enabled"`
}

// UpdateStatus represents the current update status.
type UpdateStatus struct {
	State       string    `json:"state"` // idle, checking, downloading, applying, restarting, error
	Progress    int       `json:"progress"`
	Message     string    `json:"message"`
	LastChecked time.Time `json:"last_checked"`
	Error       string    `json:"error,omitempty"`
}

// UpdateConfig represents update configuration.
type UpdateConfig struct {
	Enabled            bool          `json:"enabled"`
	AutoUpdate         bool          `json:"auto_update"`
	CheckInterval      time.Duration `json:"check_interval"`
	UpdateChannel      string        `json:"update_channel"` // stable, beta, dev
	BackupBeforeUpdate bool          `json:"backup_before_update"`
	NotifyOnUpdate     bool          `json:"notify_on_update"`
	DockerImage        string        `json:"docker_image"`
	GitHubRepo         string        `json:"github_repo"`
}

// GitHubRelease represents a GitHub release response.
type GitHubRelease struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Body        string    `json:"body"`
	Prerelease  bool      `json:"prerelease"`
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
	config       UpdateConfig
	downloadPath string
	status       UpdateStatus
	lastVersion  *VersionInfo
	httpClient   *http.Client
	stopChan     chan struct{}
	isDocker     bool
}

// DefaultConfig returns the default update configuration.
func DefaultConfig() UpdateConfig {
	return UpdateConfig{
		Enabled:            true,
		AutoUpdate:         false,
		CheckInterval:      time.Hour,
		UpdateChannel:      "stable",
		BackupBeforeUpdate: true,
		NotifyOnUpdate:     true,
		DockerImage:        "cyp/docker-registry",
		GitHubRepo:         "CYP/cyp-docker-registry",
	}
}

// NewUpdaterService creates a new updater service.
func NewUpdaterService(config UpdateConfig, downloadPath string) *UpdaterService {
	if config.CheckInterval == 0 {
		config.CheckInterval = time.Hour
	}

	u := &UpdaterService{
		config:       config,
		downloadPath: downloadPath,
		status: UpdateStatus{
			State: "idle",
		},
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		stopChan: make(chan struct{}),
		isDocker: isRunningInDocker(),
	}

	return u
}

// isRunningInDocker checks if the application is running inside a Docker container.
func isRunningInDocker() bool {
	// Check for .dockerenv file
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	// Check cgroup
	if data, err := os.ReadFile("/proc/1/cgroup"); err == nil {
		if strings.Contains(string(data), "docker") {
			return true
		}
	}

	return false
}

// Start starts the background update checker.
func (u *UpdaterService) Start() {
	if !u.config.Enabled {
		return
	}

	go u.backgroundChecker()
}

// Stop stops the background update checker.
func (u *UpdaterService) Stop() {
	close(u.stopChan)
}

// backgroundChecker periodically checks for updates.
func (u *UpdaterService) backgroundChecker() {
	// Initial check after 1 minute
	time.Sleep(time.Minute)

	ticker := time.NewTicker(u.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-u.stopChan:
			return
		case <-ticker.C:
			info, err := u.CheckUpdate()
			if err == nil && info.HasUpdate && u.config.AutoUpdate {
				// Auto update if enabled
				u.performAutoUpdate(info)
			}
		}
	}
}

// CheckUpdate checks for available updates.
func (u *UpdaterService) CheckUpdate() (*VersionInfo, error) {
	u.mu.Lock()
	u.status.State = "checking"
	u.status.Message = "正在检查更新..."
	u.status.Error = ""
	u.mu.Unlock()

	defer func() {
		u.mu.Lock()
		if u.status.State == "checking" {
			u.status.State = "idle"
		}
		u.status.LastChecked = time.Now()
		u.mu.Unlock()
	}()

	currentVersion := version.GetVersion()

	// Fetch latest release from GitHub
	latestVersion, releaseAt, changelog, downloadURL, err := u.fetchLatestRelease()
	if err != nil {
		u.setError(err.Error())
		return nil, err
	}

	hasUpdate := CompareVersions(latestVersion, currentVersion) > 0

	info := &VersionInfo{
		Current:     currentVersion,
		Latest:      latestVersion,
		HasUpdate:   hasUpdate,
		ReleaseAt:   releaseAt,
		Changelog:   changelog,
		DownloadURL: downloadURL,
		DockerImage: fmt.Sprintf("%s:v%s", u.config.DockerImage, latestVersion),
		IsDocker:    u.isDocker,
		AutoUpdate:  u.config.AutoUpdate,
	}

	u.mu.Lock()
	u.lastVersion = info
	u.status.Message = ""
	u.mu.Unlock()

	return info, nil
}

// fetchLatestRelease fetches the latest release information from GitHub.
func (u *UpdaterService) fetchLatestRelease() (ver string, releaseAt time.Time, changelog, downloadURL string, err error) {
	if u.config.GitHubRepo == "" {
		return "", time.Time{}, "", "", fmt.Errorf("GitHub 仓库未配置")
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", u.config.GitHubRepo)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", time.Time{}, "", "", err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "CYP-Docker-Registry-Updater")

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return "", time.Time{}, "", "", fmt.Errorf("无法连接 GitHub: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", time.Time{}, "", "", fmt.Errorf("未找到发布版本")
	}

	if resp.StatusCode != http.StatusOK {
		return "", time.Time{}, "", "", fmt.Errorf("GitHub API 返回错误: %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", time.Time{}, "", "", fmt.Errorf("解析发布信息失败: %w", err)
	}

	// Skip pre-release if channel is stable
	if u.config.UpdateChannel == "stable" && release.Prerelease {
		return "", time.Time{}, "", "", fmt.Errorf("最新版本为预发布版本")
	}

	// Remove 'v' prefix if present
	ver = strings.TrimPrefix(release.TagName, "v")

	// Find download URL for current platform
	downloadURL = u.findAssetURL(release.Assets)

	return ver, release.PublishedAt, release.Body, downloadURL, nil
}

// findAssetURL finds the appropriate download URL for the current platform.
func (u *UpdaterService) findAssetURL(assets []struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}) string {
	platform := fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)

	for _, asset := range assets {
		if strings.Contains(strings.ToLower(asset.Name), platform) {
			return asset.BrowserDownloadURL
		}
	}

	return ""
}

// CompareVersions compares two semantic version strings.
func CompareVersions(v1, v2 string) int {
	v1 = strings.TrimPrefix(v1, "v")
	v2 = strings.TrimPrefix(v2, "v")

	parts1 := parseVersion(v1)
	parts2 := parseVersion(v2)

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

func parseVersion(v string) [3]int {
	var parts [3]int
	v = strings.Split(v, "-")[0]
	segments := strings.Split(v, ".")

	for i := 0; i < len(segments) && i < 3; i++ {
		parts[i], _ = strconv.Atoi(segments[i])
	}

	return parts
}

// performAutoUpdate performs automatic update.
func (u *UpdaterService) performAutoUpdate(info *VersionInfo) {
	if u.isDocker {
		// For Docker, we can't auto-update the container itself
		// Instead, notify the user or trigger external update mechanism
		u.mu.Lock()
		u.status.State = "idle"
		u.status.Message = fmt.Sprintf("发现新版本 v%s，请手动更新 Docker 镜像或使用 Watchtower", info.Latest)
		u.mu.Unlock()
		return
	}

	// For binary deployment, download and apply update
	if err := u.DownloadUpdate(info.Latest); err != nil {
		return
	}

	if err := u.ApplyUpdate(); err != nil {
		return
	}
}

// DownloadUpdate downloads the update package.
func (u *UpdaterService) DownloadUpdate(targetVersion string) error {
	u.mu.Lock()
	u.status.State = "downloading"
	u.status.Progress = 0
	u.status.Message = "正在下载更新..."
	u.status.Error = ""
	u.mu.Unlock()

	defer func() {
		u.mu.Lock()
		if u.status.State == "downloading" {
			u.status.State = "idle"
		}
		u.mu.Unlock()
	}()

	// Get download URL
	info := u.GetLastVersionInfo()
	if info == nil || info.DownloadURL == "" {
		err := fmt.Errorf("未找到下载链接，请先检查更新")
		u.setError(err.Error())
		return err
	}

	// Create download directory
	if err := os.MkdirAll(u.downloadPath, 0755); err != nil {
		u.setError("创建下载目录失败: " + err.Error())
		return err
	}

	// Download the file
	resp, err := u.httpClient.Get(info.DownloadURL)
	if err != nil {
		u.setError("下载失败: " + err.Error())
		return err
	}
	defer resp.Body.Close()

	// Create destination file
	filename := filepath.Base(info.DownloadURL)
	destPath := filepath.Join(u.downloadPath, filename)
	destFile, err := os.Create(destPath)
	if err != nil {
		u.setError("创建文件失败: " + err.Error())
		return err
	}
	defer destFile.Close()

	// Copy with progress
	totalSize := resp.ContentLength
	var downloaded int64
	buf := make([]byte, 32*1024)

	for {
		n, err := resp.Body.Read(buf)
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
			u.setError("下载中断: " + err.Error())
			return err
		}
	}

	u.mu.Lock()
	u.status.Progress = 100
	u.status.Message = "下载完成"
	u.mu.Unlock()

	return nil
}

// ApplyUpdate applies the downloaded update.
func (u *UpdaterService) ApplyUpdate() error {
	u.mu.Lock()
	u.status.State = "applying"
	u.status.Message = "正在应用更新..."
	u.mu.Unlock()

	if u.isDocker {
		// Docker container cannot update itself
		// Provide instructions for manual update
		u.mu.Lock()
		u.status.State = "idle"
		u.status.Message = "Docker 容器无法自动更新，请使用以下方式更新:\n" +
			"1. 使用 Watchtower 自动更新\n" +
			"2. 手动执行: docker pull " + u.config.DockerImage + ":latest && docker-compose up -d"
		u.mu.Unlock()
		return nil
	}

	// For binary deployment
	// 1. Backup current binary
	execPath, err := os.Executable()
	if err != nil {
		u.setError("获取程序路径失败")
		return err
	}

	backupPath := execPath + ".backup"
	if u.config.BackupBeforeUpdate {
		if err := copyFile(execPath, backupPath); err != nil {
			u.setError("备份失败: " + err.Error())
			return err
		}
	}

	// 2. Find downloaded update file
	files, err := filepath.Glob(filepath.Join(u.downloadPath, "*"))
	if err != nil || len(files) == 0 {
		u.setError("未找到更新文件")
		return fmt.Errorf("未找到更新文件")
	}

	updateFile := files[0]

	// 3. Replace binary
	if err := os.Rename(updateFile, execPath); err != nil {
		// Try copy instead
		if err := copyFile(updateFile, execPath); err != nil {
			u.setError("替换程序失败: " + err.Error())
			return err
		}
	}

	// 4. Set executable permission
	if err := os.Chmod(execPath, 0755); err != nil {
		u.setError("设置权限失败")
		return err
	}

	u.mu.Lock()
	u.status.State = "idle"
	u.status.Message = "更新已应用，请重启服务"
	u.mu.Unlock()

	return nil
}

// Rollback rolls back to the previous version.
func (u *UpdaterService) Rollback() error {
	u.mu.Lock()
	defer u.mu.Unlock()

	if u.isDocker {
		u.status.Message = "Docker 容器请使用: docker pull " + u.config.DockerImage + ":<previous-version>"
		return nil
	}

	execPath, err := os.Executable()
	if err != nil {
		return err
	}

	backupPath := execPath + ".backup"
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("未找到备份文件")
	}

	if err := os.Rename(backupPath, execPath); err != nil {
		return err
	}

	u.status.State = "idle"
	u.status.Message = "已回滚到之前版本，请重启服务"

	return nil
}

// GetDockerUpdateCommand returns the Docker update command.
func (u *UpdaterService) GetDockerUpdateCommand() string {
	if !u.isDocker {
		return ""
	}

	info := u.GetLastVersionInfo()
	if info == nil {
		return fmt.Sprintf("docker pull %s:latest && docker-compose up -d", u.config.DockerImage)
	}

	return fmt.Sprintf("docker pull %s:v%s && docker-compose up -d", u.config.DockerImage, info.Latest)
}

// GetWatchtowerConfig returns Watchtower configuration for auto-update.
func (u *UpdaterService) GetWatchtowerConfig() string {
	return `# 添加 Watchtower 服务到 docker-compose.yaml 实现自动更新:
watchtower:
  image: containrrr/watchtower
  container_name: watchtower
  restart: unless-stopped
  volumes:
    - /var/run/docker.sock:/var/run/docker.sock
  environment:
    - WATCHTOWER_CLEANUP=true
    - WATCHTOWER_POLL_INTERVAL=86400  # 每24小时检查一次
    - WATCHTOWER_INCLUDE_STOPPED=false
    - WATCHTOWER_NOTIFICATIONS=email  # 可选: 邮件通知
  command: cyp-docker-registry  # 只监控此容器`
}

// RestartService restarts the service (for systemd deployments).
func (u *UpdaterService) RestartService() error {
	if u.isDocker {
		return fmt.Errorf("Docker 容器请使用 docker-compose restart")
	}

	// Try systemctl restart
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "systemctl", "restart", "cyp-docker-registry")
	return cmd.Run()
}

// setError sets the error state.
func (u *UpdaterService) setError(message string) {
	u.mu.Lock()
	u.status.State = "error"
	u.status.Message = message
	u.status.Error = message
	u.mu.Unlock()
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

// GetConfig returns the current configuration.
func (u *UpdaterService) GetConfig() UpdateConfig {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.config
}

// SetConfig updates the configuration.
func (u *UpdaterService) SetConfig(config UpdateConfig) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.config = config
}

// IsDocker returns whether running in Docker.
func (u *UpdaterService) IsDocker() bool {
	return u.isDocker
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}
