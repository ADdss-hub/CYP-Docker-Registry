// Package service 提供全局服务管理
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// GlobalServiceManager 全局服务管理器
// 负责镜像加速、DNS、P2P等服务的全局应用和配置
type GlobalServiceManager struct {
	logger      *zap.Logger
	mu          sync.RWMutex
	initialized bool

	// 服务状态
	acceleratorApplied bool
	dnsApplied         bool
	p2pApplied         bool

	// 配置路径
	dataPath   string
	configPath string

	// DNS解析器（使用自定义DNS服务器）
	customResolver *net.Resolver
	dnsServers     []string

	// 镜像加速源
	acceleratorMirrors []string
}

// GlobalServiceConfig 全局服务配置
type GlobalServiceConfig struct {
	DataPath   string
	ConfigPath string

	// 镜像加速配置
	AcceleratorEnabled bool
	AcceleratorMirrors []string

	// DNS配置
	DNSEnabled bool
	DNSServers []string

	// P2P配置
	P2PEnabled    bool
	P2PListenPort int
}

// NewGlobalServiceManager 创建全局服务管理器
func NewGlobalServiceManager(logger *zap.Logger) *GlobalServiceManager {
	return &GlobalServiceManager{
		logger:     logger,
		dataPath:   "./data",
		configPath: "./configs",
	}
}

// Initialize 初始化全局服务
func (m *GlobalServiceManager) Initialize(config *GlobalServiceConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.initialized {
		return nil
	}

	m.logger.Info("开始初始化全局服务...")

	if config != nil {
		if config.DataPath != "" {
			m.dataPath = config.DataPath
		}
		if config.ConfigPath != "" {
			m.configPath = config.ConfigPath
		}
	}

	// 应用镜像加速配置
	if config != nil && config.AcceleratorEnabled {
		if err := m.applyAcceleratorConfig(config.AcceleratorMirrors); err != nil {
			m.logger.Warn("应用镜像加速配置失败", zap.Error(err))
		} else {
			m.acceleratorApplied = true
			m.logger.Info("镜像加速配置已应用到系统")
		}
	}

	// 应用DNS配置
	if config != nil && config.DNSEnabled {
		if err := m.applyDNSConfig(config.DNSServers); err != nil {
			m.logger.Warn("应用DNS配置失败", zap.Error(err))
		} else {
			m.dnsApplied = true
			m.logger.Info("DNS配置已应用到系统")
		}
	}

	// 应用P2P配置
	if config != nil && config.P2PEnabled {
		if err := m.applyP2PConfig(config.P2PListenPort); err != nil {
			m.logger.Warn("应用P2P配置失败", zap.Error(err))
		} else {
			m.p2pApplied = true
			m.logger.Info("P2P配置已应用到系统")
		}
	}

	m.initialized = true
	m.logger.Info("全局服务初始化完成",
		zap.Bool("accelerator", m.acceleratorApplied),
		zap.Bool("dns", m.dnsApplied),
		zap.Bool("p2p", m.p2pApplied),
	)

	return nil
}

// applyAcceleratorConfig 应用镜像加速配置到Docker daemon
func (m *GlobalServiceManager) applyAcceleratorConfig(mirrors []string) error {
	if len(mirrors) == 0 {
		// 使用默认镜像源
		mirrors = []string{
			"https://registry.cn-hangzhou.aliyuncs.com",
			"https://mirror.ccs.tencentyun.com",
		}
	}

	m.acceleratorMirrors = mirrors

	// 检测Docker daemon配置文件路径
	daemonConfigPath := m.getDockerDaemonConfigPath()
	if daemonConfigPath == "" {
		return fmt.Errorf("无法确定Docker daemon配置文件路径")
	}

	// 生成镜像加速配置
	configContent := m.generateDockerDaemonConfig(mirrors)

	// 保存配置到本地（供用户参考）
	localConfigPath := filepath.Join(m.dataPath, "docker-daemon-config.json")
	if err := os.MkdirAll(filepath.Dir(localConfigPath), 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}
	if err := os.WriteFile(localConfigPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("保存配置文件失败: %w", err)
	}

	// 保存镜像加速源列表到环境变量文件
	envFilePath := filepath.Join(m.dataPath, "accelerator.env")
	envContent := fmt.Sprintf("REGISTRY_MIRRORS=%s\n", strings.Join(mirrors, ","))
	if err := os.WriteFile(envFilePath, []byte(envContent), 0644); err != nil {
		m.logger.Warn("保存环境变量文件失败", zap.Error(err))
	}

	m.logger.Info("镜像加速配置已生成并应用",
		zap.String("config_path", localConfigPath),
		zap.Strings("mirrors", mirrors),
	)

	// 尝试自动应用配置（仅在有权限时）
	if m.canModifyDockerConfig() {
		if err := m.applyDockerConfig(daemonConfigPath, configContent); err != nil {
			m.logger.Warn("自动应用Docker配置失败，请手动配置", zap.Error(err))
		} else {
			m.logger.Info("Docker镜像加速配置已自动应用到daemon")
		}
	}

	return nil
}

// applyDNSConfig 应用DNS配置到系统
func (m *GlobalServiceManager) applyDNSConfig(servers []string) error {
	if len(servers) == 0 {
		// 使用默认DNS服务器
		servers = []string{
			"8.8.8.8",
			"8.8.4.4",
			"114.114.114.114",
		}
	}

	m.dnsServers = servers

	// 创建自定义DNS解析器
	m.customResolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: 10 * time.Second,
			}
			// 使用配置的DNS服务器
			for _, server := range servers {
				conn, err := d.DialContext(ctx, "udp", server+":53")
				if err == nil {
					return conn, nil
				}
			}
			// 回退到默认
			return d.DialContext(ctx, network, address)
		},
	}

	// 保存DNS配置到本地
	dnsConfigPath := filepath.Join(m.dataPath, "dns-config.txt")
	if err := os.MkdirAll(filepath.Dir(dnsConfigPath), 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	var content strings.Builder
	content.WriteString("# CYP-Docker-Registry DNS Configuration\n")
	content.WriteString("# DNS服务器已应用到系统内部解析器\n")
	content.WriteString("# 如需修改宿主机DNS，请将以下内容添加到 /etc/resolv.conf\n\n")
	for _, server := range servers {
		fmt.Fprintf(&content, "nameserver %s\n", server)
	}

	if err := os.WriteFile(dnsConfigPath, []byte(content.String()), 0644); err != nil {
		return fmt.Errorf("保存DNS配置失败: %w", err)
	}

	m.logger.Info("DNS配置已应用到系统内部解析器",
		zap.String("config_path", dnsConfigPath),
		zap.Strings("servers", servers),
	)

	// 在Docker容器中尝试修改/etc/resolv.conf
	if m.isRunningInDocker() {
		if err := m.applyDNSToResolvConf(servers); err != nil {
			m.logger.Warn("修改/etc/resolv.conf失败，使用内部解析器", zap.Error(err))
		} else {
			m.logger.Info("DNS配置已写入/etc/resolv.conf")
		}
	}

	return nil
}

// applyP2PConfig 应用P2P配置
func (m *GlobalServiceManager) applyP2PConfig(listenPort int) error {
	if listenPort == 0 {
		listenPort = 4001
	}

	// 保存P2P配置
	p2pConfigPath := filepath.Join(m.dataPath, "p2p-config.json")
	if err := os.MkdirAll(filepath.Dir(p2pConfigPath), 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	p2pConfig := map[string]any{
		"enabled":             true,
		"listen_port":         listenPort,
		"share_mode":          "selective",
		"enable_mdns":         true,
		"enable_relay":        true,
		"enable_nat_port_map": true,
		"max_connections":     50,
		"applied_at":          time.Now().Format(time.RFC3339),
	}

	configContent, err := json.MarshalIndent(p2pConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化P2P配置失败: %w", err)
	}

	if err := os.WriteFile(p2pConfigPath, configContent, 0644); err != nil {
		return fmt.Errorf("保存P2P配置失败: %w", err)
	}

	m.logger.Info("P2P配置已生成并应用",
		zap.String("config_path", p2pConfigPath),
		zap.Int("listen_port", listenPort),
	)

	// 检查端口是否可用
	if err := m.checkPortAvailable(listenPort); err != nil {
		m.logger.Warn("P2P端口可能被占用", zap.Int("port", listenPort), zap.Error(err))
	}

	return nil
}

// getDockerDaemonConfigPath 获取Docker daemon配置文件路径
func (m *GlobalServiceManager) getDockerDaemonConfigPath() string {
	switch runtime.GOOS {
	case "linux":
		return "/etc/docker/daemon.json"
	case "darwin":
		return filepath.Join(os.Getenv("HOME"), ".docker", "daemon.json")
	case "windows":
		return filepath.Join(os.Getenv("PROGRAMDATA"), "docker", "config", "daemon.json")
	default:
		return ""
	}
}

// generateDockerDaemonConfig 生成Docker daemon配置
func (m *GlobalServiceManager) generateDockerDaemonConfig(mirrors []string) string {
	var mirrorsJSON strings.Builder
	mirrorsJSON.WriteString("[")
	for i, mirror := range mirrors {
		if i > 0 {
			mirrorsJSON.WriteString(", ")
		}
		fmt.Fprintf(&mirrorsJSON, `"%s"`, mirror)
	}
	mirrorsJSON.WriteString("]")

	return fmt.Sprintf(`{
  "registry-mirrors": %s,
  "insecure-registries": [],
  "debug": false,
  "experimental": false
}`, mirrorsJSON.String())
}

// canModifyDockerConfig 检查是否有权限修改Docker配置
func (m *GlobalServiceManager) canModifyDockerConfig() bool {
	// 检查是否以root运行
	if os.Geteuid() == 0 {
		return true
	}
	return false
}

// applyDockerConfig 应用Docker配置
func (m *GlobalServiceManager) applyDockerConfig(configPath, content string) error {
	// 备份现有配置
	if _, err := os.Stat(configPath); err == nil {
		backupPath := configPath + ".backup"
		if err := os.Rename(configPath, backupPath); err != nil {
			return fmt.Errorf("备份配置失败: %w", err)
		}
	}

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 写入新配置
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("写入配置失败: %w", err)
	}

	// 尝试重启Docker服务
	if err := m.restartDockerService(); err != nil {
		m.logger.Warn("重启Docker服务失败，请手动重启", zap.Error(err))
	}

	return nil
}

// restartDockerService 重启Docker服务
func (m *GlobalServiceManager) restartDockerService() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*1000000000) // 30秒
	defer cancel()

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.CommandContext(ctx, "systemctl", "restart", "docker")
	default:
		return fmt.Errorf("不支持在 %s 上自动重启Docker", runtime.GOOS)
	}

	return cmd.Run()
}

// isRunningInDocker 检查是否在Docker容器中运行
func (m *GlobalServiceManager) isRunningInDocker() bool {
	// 检查 /.dockerenv 文件
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	// 检查 cgroup
	if data, err := os.ReadFile("/proc/1/cgroup"); err == nil {
		return strings.Contains(string(data), "docker")
	}
	return false
}

// applyDNSToResolvConf 在Docker容器中修改/etc/resolv.conf
func (m *GlobalServiceManager) applyDNSToResolvConf(servers []string) error {
	resolvPath := "/etc/resolv.conf"

	// 读取现有配置
	existingContent, _ := os.ReadFile(resolvPath)

	var content strings.Builder
	content.WriteString("# Generated by CYP-Docker-Registry\n")
	content.WriteString("# Original content preserved below\n")
	for _, server := range servers {
		fmt.Fprintf(&content, "nameserver %s\n", server)
	}

	// 保留原有的search和options行
	for _, line := range strings.Split(string(existingContent), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "search ") || strings.HasPrefix(line, "options ") {
			content.WriteString(line + "\n")
		}
	}

	// 尝试写入
	if err := os.WriteFile(resolvPath, []byte(content.String()), 0644); err != nil {
		return fmt.Errorf("写入resolv.conf失败: %w", err)
	}

	return nil
}

// GetCustomResolver 获取自定义DNS解析器
func (m *GlobalServiceManager) GetCustomResolver() *net.Resolver {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.customResolver
}

// GetDNSServers 获取配置的DNS服务器列表
func (m *GlobalServiceManager) GetDNSServers() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.dnsServers
}

// GetAcceleratorMirrors 获取配置的镜像加速源列表
func (m *GlobalServiceManager) GetAcceleratorMirrors() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.acceleratorMirrors
}

// ResolveDomain 使用自定义DNS解析域名
func (m *GlobalServiceManager) ResolveDomain(ctx context.Context, domain string) ([]string, error) {
	m.mu.RLock()
	resolver := m.customResolver
	m.mu.RUnlock()

	if resolver == nil {
		resolver = net.DefaultResolver
	}

	return resolver.LookupHost(ctx, domain)
}

// checkPortAvailable 检查端口是否可用
func (m *GlobalServiceManager) checkPortAvailable(port int) error {
	// 简单检查端口是否被占用
	ctx, cancel := context.WithTimeout(context.Background(), 5*1000000000) // 5秒
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", fmt.Sprintf("netstat -tuln | grep :%d", port))
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		return fmt.Errorf("端口 %d 已被占用", port)
	}
	return nil
}

// GetStatus 获取全局服务状态
func (m *GlobalServiceManager) GetStatus() map[string]any {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := map[string]any{
		"initialized":         m.initialized,
		"accelerator_applied": m.acceleratorApplied,
		"dns_applied":         m.dnsApplied,
		"p2p_applied":         m.p2pApplied,
		"data_path":           m.dataPath,
		"config_path":         m.configPath,
		"running_in_docker":   m.isRunningInDocker(),
	}

	// 添加详细配置信息
	if m.acceleratorApplied && len(m.acceleratorMirrors) > 0 {
		status["accelerator_mirrors"] = m.acceleratorMirrors
	}
	if m.dnsApplied && len(m.dnsServers) > 0 {
		status["dns_servers"] = m.dnsServers
	}

	// 检查配置文件是否存在
	configFiles := map[string]string{
		"docker_daemon_config": filepath.Join(m.dataPath, "docker-daemon-config.json"),
		"dns_config":           filepath.Join(m.dataPath, "dns-config.txt"),
		"p2p_config":           filepath.Join(m.dataPath, "p2p-config.json"),
		"accelerator_env":      filepath.Join(m.dataPath, "accelerator.env"),
	}

	configStatus := make(map[string]bool)
	for name, path := range configFiles {
		_, err := os.Stat(path)
		configStatus[name] = err == nil
	}
	status["config_files"] = configStatus

	return status
}

// ApplyAccelerator 手动应用镜像加速
func (m *GlobalServiceManager) ApplyAccelerator(mirrors []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.applyAcceleratorConfig(mirrors); err != nil {
		return err
	}
	m.acceleratorApplied = true
	return nil
}

// ApplyDNS 手动应用DNS配置
func (m *GlobalServiceManager) ApplyDNS(servers []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.applyDNSConfig(servers); err != nil {
		return err
	}
	m.dnsApplied = true
	return nil
}

// ApplyP2P 手动应用P2P配置
func (m *GlobalServiceManager) ApplyP2P(listenPort int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.applyP2PConfig(listenPort); err != nil {
		return err
	}
	m.p2pApplied = true
	return nil
}
