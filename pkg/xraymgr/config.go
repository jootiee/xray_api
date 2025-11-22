package xraymgr

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents high-level manager configuration loaded from YAML.
type Config struct {
	BinaryPath       string `yaml:"binary_path"`
	ConfigDir        string `yaml:"config_dir"`
	SystemConfigPath string `yaml:"system_config_path"`
	ServiceName      string `yaml:"service_name"`
}

// LoadConfig loads YAML configuration from the provided path and applies defaults.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	// Defaults
	if cfg.BinaryPath == "" {
		cfg.BinaryPath = "/usr/local/bin/xray"
	}
	if cfg.ConfigDir == "" {
		cfg.ConfigDir = "./_conf"
	}
	if cfg.SystemConfigPath == "" {
		cfg.SystemConfigPath = "/usr/local/etc/xray/config.json"
	}
	if cfg.ServiceName == "" {
		cfg.ServiceName = "xray"
	}

	// Ensure directory exists.
	if err := os.MkdirAll(cfg.ConfigDir, 0755); err != nil {
		return nil, fmt.Errorf("mkdir config dir: %w", err)
	}

	// Normalize config paths (allow relative).
	cfg.ConfigDir, _ = filepath.Abs(cfg.ConfigDir)
	return &cfg, nil
}
