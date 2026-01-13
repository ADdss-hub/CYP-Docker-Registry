// Package config provides configuration management for CYP-Registry.
package config

import (
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration.
type Config struct {
	App         AppConfig         `yaml:"app"`
	Server      ServerConfig      `yaml:"server"`
	Storage     StorageConfig     `yaml:"storage"`
	Security    SecurityConfig    `yaml:"security"`
	Accelerator AcceleratorConfig `yaml:"accelerator"`
	P2P         P2PConfig         `yaml:"p2p"`
	Compression CompressionConfig `yaml:"compression"`
	Signature   SignatureConfig   `yaml:"signature"`
	SBOM        SBOMConfig        `yaml:"sbom"`
	Sync        SyncConfig        `yaml:"sync"`
	Notify      NotifyConfig      `yaml:"notify"`
	Environment EnvironmentConfig `yaml:"environment"`
}

// AppConfig holds application settings.
type AppConfig struct {
	Name     string `yaml:"name"`
	Version  string `yaml:"version"`
	LogLevel string `yaml:"log_level"`
}

// ServerConfig holds server settings.
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
	Mode string `yaml:"mode"`
}

// StorageConfig holds storage settings.
type StorageConfig struct {
	BlobPath     string `yaml:"blob_path"`
	MetaPath     string `yaml:"meta_path"`
	CachePath    string `yaml:"cache_path"`
	MaxCacheSize string `yaml:"max_cache_size"`
}

// SecurityConfig holds security settings.
type SecurityConfig struct {
	ForceLogin         ForceLoginConfig         `yaml:"force_login"`
	FailedAttempts     FailedAttemptsConfig     `yaml:"failed_attempts"`
	AutoLock           AutoLockConfig           `yaml:"auto_lock"`
	IntrusionDetection IntrusionDetectionConfig `yaml:"intrusion_detection"`
	Audit              AuditConfig              `yaml:"audit"`
	JWTSecret          string                   `yaml:"jwt_secret"`
}

// ForceLoginConfig holds force login settings.
type ForceLoginConfig struct {
	Enabled bool   `yaml:"enabled"`
	Mode    string `yaml:"mode"`
}

// FailedAttemptsConfig holds failed attempts settings.
type FailedAttemptsConfig struct {
	MaxLoginAttempts int    `yaml:"max_login_attempts"`
	MaxTokenAttempts int    `yaml:"max_token_attempts"`
	MaxAPIAttempts   int    `yaml:"max_api_attempts"`
	LockDuration     string `yaml:"lock_duration"`
}

// AutoLockConfig holds auto lock settings.
type AutoLockConfig struct {
	Enabled             bool                `yaml:"enabled"`
	LockOnBypassAttempt bool                `yaml:"lock_on_bypass_attempt"`
	Hardware            HardwareLockConfig  `yaml:"hardware"`
	Network             NetworkLockConfig   `yaml:"network"`
	Service             ServiceLockConfig   `yaml:"service"`
}

// HardwareLockConfig holds hardware lock settings.
type HardwareLockConfig struct {
	LockCPUPercent    int `yaml:"lock_cpu_percent"`
	LockMemoryPercent int `yaml:"lock_memory_percent"`
}

// NetworkLockConfig holds network lock settings.
type NetworkLockConfig struct {
	BlockIncoming bool `yaml:"block_incoming"`
	BlockOutgoing bool `yaml:"block_outgoing"`
}

// ServiceLockConfig holds service lock settings.
type ServiceLockConfig struct {
	PauseAllWorkflows bool `yaml:"pause_all_workflows"`
	AllowReadOnlyMode bool `yaml:"allow_readonly_mode"`
}

// IntrusionDetectionConfig holds intrusion detection settings.
type IntrusionDetectionConfig struct {
	Enabled            bool                   `yaml:"enabled"`
	Rules              []IntrusionRule        `yaml:"rules"`
	RealTimeMonitoring bool                   `yaml:"real_time_monitoring"`
	NotifyOnLock       bool                   `yaml:"notify_on_lock"`
}

// IntrusionRule represents an intrusion detection rule.
type IntrusionRule struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Action      string `yaml:"action"`
	Threshold   int    `yaml:"threshold"`
}

// AuditConfig holds audit settings.
type AuditConfig struct {
	LogAllRequests bool   `yaml:"log_all_requests"`
	LogFailedAuth  bool   `yaml:"log_failed_auth"`
	LogLockEvents  bool   `yaml:"log_lock_events"`
	BlockchainHash bool   `yaml:"blockchain_hash"`
	Retention      string `yaml:"retention"`
}

// AcceleratorConfig holds accelerator settings.
type AcceleratorConfig struct {
	Enabled   bool             `yaml:"enabled"`
	CacheSize string           `yaml:"cache_size"`
	TTL       int              `yaml:"ttl"`
	Upstreams []UpstreamConfig `yaml:"upstreams"`
}

// UpstreamConfig holds upstream mirror settings.
type UpstreamConfig struct {
	Name     string `yaml:"name"`
	URL      string `yaml:"url"`
	Priority int    `yaml:"priority"`
}

// P2PConfig holds P2P settings.
type P2PConfig struct {
	Enabled          bool     `yaml:"enabled"`
	MaxConnections   int      `yaml:"max_connections"`
	EnableRelay      bool     `yaml:"enable_relay"`
	EnableNATPortMap bool     `yaml:"enable_nat_port_map"`
	BandwidthLimit   string   `yaml:"bandwidth_limit"`
	BootstrapPeers   []string `yaml:"bootstrap_peers"`
}

// CompressionConfig holds compression settings.
type CompressionConfig struct {
	Enabled   bool   `yaml:"enabled"`
	Algorithm string `yaml:"algorithm"`
	Level     int    `yaml:"level"`
	Parallel  bool   `yaml:"parallel"`
}

// SignatureConfig holds signature settings.
type SignatureConfig struct {
	Enabled          bool   `yaml:"enabled"`
	Mode             string `yaml:"mode"`
	KeyPath          string `yaml:"key_path"`
	AutoSign         bool   `yaml:"auto_sign"`
	RequireSignature bool   `yaml:"require_signature"`
}

// SBOMConfig holds SBOM settings.
type SBOMConfig struct {
	Enabled       bool   `yaml:"enabled"`
	Generator     string `yaml:"generator"`
	Format        string `yaml:"format"`
	StoragePath   string `yaml:"storage_path"`
	VulnScan      bool   `yaml:"vuln_scan"`
	VulnScanner   string `yaml:"vuln_scanner"`
	AutoGenerate  bool   `yaml:"auto_generate"`
}

// SyncConfig holds sync settings.
type SyncConfig struct {
	Enabled  bool         `yaml:"enabled"`
	Interval string       `yaml:"interval"`
	Parallel int          `yaml:"parallel"`
	Targets  []SyncTarget `yaml:"targets"`
}

// SyncTarget represents a sync target.
type SyncTarget struct {
	Name     string `yaml:"name"`
	Type     string `yaml:"type"`
	URL      string `yaml:"url"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// NotifyConfig holds notification settings.
type NotifyConfig struct {
	Channels NotifyChannels `yaml:"channels"`
}

// NotifyChannels holds notification channel settings.
type NotifyChannels struct {
	WebSocket WebSocketConfig `yaml:"websocket"`
	Webhook   WebhookConfig   `yaml:"webhook"`
	Email     EmailConfig     `yaml:"email"`
}

// WebSocketConfig holds WebSocket notification settings.
type WebSocketConfig struct {
	Enabled bool `yaml:"enabled"`
}

// WebhookConfig holds webhook notification settings.
type WebhookConfig struct {
	Enabled bool   `yaml:"enabled"`
	URL     string `yaml:"url"`
	Secret  string `yaml:"secret"`
}

// EmailConfig holds email notification settings.
type EmailConfig struct {
	Enabled  bool     `yaml:"enabled"`
	SMTPHost string   `yaml:"smtp_host"`
	SMTPPort int      `yaml:"smtp_port"`
	Username string   `yaml:"username"`
	Password string   `yaml:"password"`
	To       []string `yaml:"to"`
}

// EnvironmentConfig holds environment detection settings.
type EnvironmentConfig struct {
	Type          string `yaml:"type"`
	AutoDetected  bool   `yaml:"auto_detected"`
	AutoConfigure bool   `yaml:"auto_configure"`
}

var (
	globalConfig *Config
	configMutex  sync.RWMutex
	readOnlyMode bool
)

// Load loads configuration from a file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Expand environment variables
	data = []byte(os.ExpandEnv(string(data)))

	config := &Config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, err
	}

	// Set defaults
	setDefaults(config)

	configMutex.Lock()
	globalConfig = config
	configMutex.Unlock()

	return config, nil
}

// Get returns the global configuration.
func Get() *Config {
	configMutex.RLock()
	defer configMutex.RUnlock()
	return globalConfig
}

// SetReadOnlyMode sets the read-only mode.
func SetReadOnlyMode(enabled bool) {
	configMutex.Lock()
	defer configMutex.Unlock()
	readOnlyMode = enabled
}

// IsReadOnlyMode returns whether read-only mode is enabled.
func IsReadOnlyMode() bool {
	configMutex.RLock()
	defer configMutex.RUnlock()
	return readOnlyMode
}

// setDefaults sets default values for configuration.
func setDefaults(c *Config) {
	if c.App.Name == "" {
		c.App.Name = "CYP-Registry"
	}
	if c.App.Version == "" {
		c.App.Version = "1.0.0"
	}
	if c.App.LogLevel == "" {
		c.App.LogLevel = "info"
	}
	if c.Server.Host == "" {
		c.Server.Host = "0.0.0.0"
	}
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.Server.Mode == "" {
		c.Server.Mode = "release"
	}
	if c.Storage.BlobPath == "" {
		c.Storage.BlobPath = "./data/blobs"
	}
	if c.Storage.MetaPath == "" {
		c.Storage.MetaPath = "./data/meta"
	}
	if c.Storage.CachePath == "" {
		c.Storage.CachePath = "./data/cache"
	}
	if c.Storage.MaxCacheSize == "" {
		c.Storage.MaxCacheSize = "10GB"
	}
	if c.Security.FailedAttempts.MaxLoginAttempts == 0 {
		c.Security.FailedAttempts.MaxLoginAttempts = 3
	}
	if c.Security.FailedAttempts.MaxTokenAttempts == 0 {
		c.Security.FailedAttempts.MaxTokenAttempts = 5
	}
	if c.Security.FailedAttempts.LockDuration == "" {
		c.Security.FailedAttempts.LockDuration = "1h"
	}
	if c.Security.AutoLock.Hardware.LockCPUPercent == 0 {
		c.Security.AutoLock.Hardware.LockCPUPercent = 10
	}
	if c.Security.AutoLock.Hardware.LockMemoryPercent == 0 {
		c.Security.AutoLock.Hardware.LockMemoryPercent = 10
	}
	if c.Signature.Mode == "" {
		c.Signature.Mode = "warn"
	}
	if c.SBOM.Generator == "" {
		c.SBOM.Generator = "syft"
	}
	if c.SBOM.Format == "" {
		c.SBOM.Format = "spdx-json"
	}
}

// LoadTemplate loads a configuration template by environment type.
func LoadTemplate(envType string) (*Config, error) {
	templatePath := filepath.Join("internal/config/templates", envType+".yaml")
	return Load(templatePath)
}

// Save saves the configuration to a file.
func Save(path string, config *Config) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
