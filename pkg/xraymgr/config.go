package xraymgr

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents high-level manager configuration loaded from YAML.
type Config struct {
	Xray   XraySection   `yaml:"xray"`
	Server ServerSection `yaml:"server"`
}

// XraySection holds paths and service details.
type XraySection struct {
	BinaryPath       string `yaml:"binary_path"`
	ConfigDir        string `yaml:"config_dir"`
	SystemConfigPath string `yaml:"system_config_path"`
	ServiceName      string `yaml:"service_name"`
}

// ServerSection holds server runtime details used to build client configs.
type ServerSection struct {
	Address       string   `yaml:"address"`
	Port          int      `yaml:"port"`
	Port80        int      `yaml:"port_80"`
	GRPCPort      int      `yaml:"grpc_port"`
	FakeSite      string   `yaml:"fake_site"`
	PrivateKey    string   `yaml:"private_key"`
	PublicKey     string   `yaml:"public_key"`
	ServerNames   []string `yaml:"server_names"`
	DefaultUserID string   `yaml:"default_user_id"`
	DefaultEmail  string   `yaml:"default_email"`
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
	if cfg.Xray.BinaryPath == "" {
		cfg.Xray.BinaryPath = "/usr/local/bin/xray"
	}
	if cfg.Xray.ConfigDir == "" {
		cfg.Xray.ConfigDir = "./_conf"
	}
	if cfg.Xray.SystemConfigPath == "" {
		cfg.Xray.SystemConfigPath = "/usr/local/etc/xray/config.json"
	}
	if cfg.Xray.ServiceName == "" {
		cfg.Xray.ServiceName = "xray"
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 443
	}
	if cfg.Server.Port80 == 0 {
		cfg.Server.Port80 = 80
	}
	if cfg.Server.GRPCPort == 0 {
		cfg.Server.GRPCPort = 50051
	}
	if len(cfg.Server.ServerNames) == 0 && cfg.Server.FakeSite != "" {
		cfg.Server.ServerNames = []string{cfg.Server.FakeSite}
	}
	// Ensure directory exists.
	if err := os.MkdirAll(cfg.Xray.ConfigDir, 0755); err != nil {
		return nil, fmt.Errorf("mkdir config dir: %w", err)
	}
	// Normalize config paths (allow relative).
	cfg.Xray.ConfigDir, _ = filepath.Abs(cfg.Xray.ConfigDir)
	return &cfg, nil
}
