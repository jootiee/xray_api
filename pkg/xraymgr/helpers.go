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

func (m *Manager) createClientConfig(username, id, email, shortID string) ([]byte, error) {
	template, err := m.loadClient(m.clientTemplatePath)
	if err != nil {
		return nil, err
	}
	// adjust first proxy outbound
	for i := range template.Outbounds {
		ob := &template.Outbounds[i]
		if ob.Tag == "proxy" && len(ob.Settings.Vnext) > 0 {
			vn := &ob.Settings.Vnext[0]
			vn.Address = m.cfg.Server.Address
			vn.Port = m.cfg.Server.Port
			if len(vn.Users) == 0 {
				vn.Users = []VnextUser{{}}
			}
			vn.Users[0].ID = id
			vn.Users[0].Email = email
			vn.Users[0].Encryption = "none"
			if ob.StreamSettings != nil && ob.StreamSettings.RealitySettings != nil {
				r := ob.StreamSettings.RealitySettings
				r.PublicKey = m.cfg.Server.PublicKey
				r.ShortID = shortID
				if len(m.cfg.Server.ServerNames) > 0 {
					r.ServerName = m.cfg.Server.ServerNames[0]
				}
			}
		}
	}
	outPath := filepath.Join(m.cfg.Xray.ConfigDir, fmt.Sprintf("config_client_%s.json", username))
	data, err := json.MarshalIndent(template, "", "  ")
	if err != nil {
		return nil, err
	}
	return data, os.WriteFile(outPath, data, 0644)
}
