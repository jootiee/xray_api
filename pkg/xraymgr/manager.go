package xraymgr

import (
	"fmt"
	"os/exec"
	"path/filepath"
)

// Manager is the main entry point for operating on xray configs.
type Manager struct {
	cfg                *Config
	serverConfigPath   string
	clientTemplatePath string
}

// New creates a new Manager with configuration loaded from YAML path.
func New(configPath string) (*Manager, error) {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return nil, err
	}
	m := &Manager{
		cfg:                cfg,
		serverConfigPath:   filepath.Join(cfg.Xray.ConfigDir, "config_server.json"),
		clientTemplatePath: filepath.Join(cfg.Xray.ConfigDir, "config_client.json"),
	}
	return m, nil
}

// PushServerConfig copies server config to system path and restarts service.
func (m *Manager) PushServerConfig() error {
	if err := copyWithBackup(m.serverConfigPath, m.cfg.Xray.SystemConfigPath); err != nil {
		return err
	}
	cmd := exec.Command("systemctl", "restart", m.cfg.Xray.ServiceName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("restart service: %w", err)
	}
	return nil
}
