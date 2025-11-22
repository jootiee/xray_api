package xraymgr

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func (m *Manager) loadServer() (*ServerJSON, error) {
	data, err := os.ReadFile(m.serverConfigPath)
	if err != nil {
		return nil, fmt.Errorf("read server config: %w", err)
	}
	var sj ServerJSON
	if err := json.Unmarshal(data, &sj); err != nil {
		return nil, fmt.Errorf("unmarshal server: %w", err)
	}
	return &sj, nil
}

func (m *Manager) saveServer(sj *ServerJSON) error {
	tmp := m.serverConfigPath + ".tmp"
	data, err := json.MarshalIndent(sj, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return copyWithBackup(tmp, m.serverConfigPath)
}

func (m *Manager) loadClient(path string) (*ClientJSON, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read client config: %w", err)
	}
	var cj ClientJSON
	if err := json.Unmarshal(data, &cj); err != nil {
		return nil, fmt.Errorf("unmarshal client: %w", err)
	}
	return &cj, nil
}

func (m *Manager) createClientConfig(username, id, shortID string) error {
	template, err := m.loadClient(m.clientTemplatePath)
	if err != nil {
		return err
	}

	ob := &template.Outbounds[0]

	ob.StreamSettings.RealitySettings.ShortID = shortID

	users := &ob.Settings.Vnext[0].Users[0]
	users.ID = id
	users.Email = usernameToEmail(username)

	outPath := filepath.Join(m.cfg.Xray.ConfigDir, fmt.Sprintf("config_client_%s.json", username))
	data, err := json.MarshalIndent(template, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(outPath, data, 0644)
}
