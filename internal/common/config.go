package common

import (
	"github.com/spf13/viper"
)

// Config represents the application configuration.
type Config struct {
	Server      ServerConfig      `mapstructure:"server"`
	Storage     StorageConfig     `mapstructure:"storage"`
	Accelerator AcceleratorConfig `mapstructure:"accelerator"`
	Update      UpdateConfig      `mapstructure:"update"`
	Auth        AuthConfig        `mapstructure:"auth"`
}

// ServerConfig represents server configuration.
type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Host string `mapstructure:"host"`
}

// StorageConfig represents storage configuration.
type StorageConfig struct {
	BlobPath     string `mapstructure:"blob_path"`
	MetaPath     string `mapstructure:"meta_path"`
	CachePath    string `mapstructure:"cache_path"`
	MaxCacheSize string `mapstructure:"max_cache_size"`
}

// AcceleratorConfig represents accelerator configuration.
type AcceleratorConfig struct {
	Enabled   bool             `mapstructure:"enabled"`
	Upstreams []UpstreamConfig `mapstructure:"upstreams"`
}

// UpstreamConfig represents upstream source configuration.
type UpstreamConfig struct {
	Name     string `mapstructure:"name"`
	URL      string `mapstructure:"url"`
	Priority int    `mapstructure:"priority"`
}

// UpdateConfig represents update configuration.
type UpdateConfig struct {
	CheckInterval string `mapstructure:"check_interval"`
	AutoUpdate    bool   `mapstructure:"auto_update"`
	UpdateURL     string `mapstructure:"update_url"`
}

// AuthConfig represents authentication configuration.
type AuthConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

// LoadConfig loads configuration from file and environment.
func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Set config file
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath("./configs")
		v.AddConfigPath(".")
	}

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		// Config file not found is not an error, use defaults
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	// Unmarshal config
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// setDefaults sets default configuration values.
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.host", "0.0.0.0")

	// Storage defaults
	v.SetDefault("storage.blob_path", "./data/blobs")
	v.SetDefault("storage.meta_path", "./data/meta")
	v.SetDefault("storage.cache_path", "./data/cache")
	v.SetDefault("storage.max_cache_size", "10GB")

	// Accelerator defaults
	v.SetDefault("accelerator.enabled", true)
	v.SetDefault("accelerator.upstreams", []map[string]interface{}{
		{"name": "Docker Hub", "url": "https://registry-1.docker.io", "priority": 1},
		{"name": "阿里云", "url": "https://registry.cn-hangzhou.aliyuncs.com", "priority": 2},
	})

	// Update defaults
	v.SetDefault("update.check_interval", "24h")
	v.SetDefault("update.auto_update", false)
	v.SetDefault("update.update_url", "https://api.github.com/repos/CYP/container-registry/releases/latest")

	// Auth defaults
	v.SetDefault("auth.enabled", false)
	v.SetDefault("auth.username", "")
	v.SetDefault("auth.password", "")
}
