package xraymgr

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
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

// AddUser creates a new user in server config and a dedicated client config.
func (m *Manager) AddUser(username string) (*UserInfo, error) {
	if username == "" {
		return nil, errors.New("username empty")
	}
	email := usernameToEmail(username)
	sj, err := m.loadServer()
	if err != nil {
		return nil, err
	}

	for _, ib := range sj.Inbounds {
		for _, c := range ib.Settings.Clients {
			if c.Email == email {
				return nil, fmt.Errorf("user %s already exists", username)
			}
		}
	}
	id := newUUID()
	shortID, err := newShortID()
	if err != nil {
		return nil, fmt.Errorf("shortID: %w", err)
	}
	// Insert into all VLESS REALITY inbounds (skip api / non vless).
	for i := range sj.Inbounds {
		ib := &sj.Inbounds[i]
		if ib.Protocol != "vless" {
			continue
		}
		if ib.Tag == "api" {
			continue
		}
		// add client
		client := ServerClient{ID: id, Email: email}
		if ib.StreamSettings != nil && ib.StreamSettings.Security == "reality" && ib.Tag != "grpc" {
			client.Flow = "xtls-rprx-vision"
		}
		ib.Settings.Clients = append(ib.Settings.Clients, client)
		if ib.StreamSettings != nil && ib.StreamSettings.RealitySettings != nil {
			ib.StreamSettings.RealitySettings.ShortIds = append(ib.StreamSettings.RealitySettings.ShortIds, shortID)
		}
	}
	if err := m.saveServer(sj); err != nil {
		return nil, err
	}
	if err := m.createClientConfig(username, id, email, shortID); err != nil {
		return nil, err
	}
	return &UserInfo{Username: username, ID: id}, nil
}

// DeleteUser removes a user from server and deletes its client config file.
func (m *Manager) DeleteUser(username string) error {
	if username == "" {
		return errors.New("username empty")
	}
	email := usernameToEmail(username)
	sj, err := m.loadServer()
	if err != nil {
		return err
	}
	found := false
	for i := range sj.Inbounds {
		ib := &sj.Inbounds[i]
		if len(ib.Settings.Clients) == 0 {
			continue
		}
		filtered := ib.Settings.Clients[:0]
		for _, c := range ib.Settings.Clients {
			if c.Email == email {
				found = true
				continue
			}
			filtered = append(filtered, c)
		}
		ib.Settings.Clients = filtered
	}
	if !found {
		return fmt.Errorf("user %s not found", username)
	}
	if err := m.saveServer(sj); err != nil {
		return err
	}
	clientPath := filepath.Join(m.cfg.Xray.ConfigDir, fmt.Sprintf("config_client_%s.json", username))
	_ = os.Remove(clientPath)
	return nil
}

// SuspendUser removes user from server without deleting its client config.
func (m *Manager) SuspendUser(username string) error {
	if username == "" {
		return errors.New("username empty")
	}
	email := usernameToEmail(username)
	sj, err := m.loadServer()
	if err != nil {
		return err
	}
	found := false
	for i := range sj.Inbounds {
		ib := &sj.Inbounds[i]
		filtered := ib.Settings.Clients[:0]
		for _, c := range ib.Settings.Clients {
			if c.Email == email {
				found = true
				continue
			}
			filtered = append(filtered, c)
		}
		ib.Settings.Clients = filtered
	}
	if !found {
		return fmt.Errorf("user %s not found", username)
	}
	return m.saveServer(sj)
}

// ResumeUser re-adds user using its client config.
func (m *Manager) ResumeUser(username string) error {
	if username == "" {
		return errors.New("username empty")
	}
	path := filepath.Join(m.cfg.Xray.ConfigDir, fmt.Sprintf("config_client_%s.json", username))
	if !fileExists(path) {
		return fmt.Errorf("client config for %s not found", username)
	}
	cj, err := m.loadClient(path)
	if err != nil {
		return err
	}
	if len(cj.Outbounds) == 0 {
		return errors.New("client outbound missing")
	}
	proxy := cj.Outbounds[0]
	if proxy.Settings == nil || len(proxy.Settings.Vnext) == 0 || len(proxy.Settings.Vnext[0].Users) == 0 {
		return errors.New("client vnext user missing")
	}
	user := proxy.Settings.Vnext[0].Users[0]
	shortID := ""
	if proxy.StreamSettings != nil && proxy.StreamSettings.RealitySettings != nil {
		shortID = proxy.StreamSettings.RealitySettings.ShortID
	}
	email := user.Email
	sj, err := m.loadServer()
	if err != nil {
		return err
	}
	for _, ib := range sj.Inbounds { // ensure not duplicated
		for _, c := range ib.Settings.Clients {
			if c.Email == email {
				return fmt.Errorf("user %s already active", username)
			}
		}
	}
	for i := range sj.Inbounds {
		ib := &sj.Inbounds[i]
		if ib.Protocol != "vless" || ib.Tag == "api" {
			continue
		}
		client := ServerClient{ID: user.ID, Email: email}
		if ib.StreamSettings != nil && ib.StreamSettings.Security == "reality" && ib.Tag != "grpc" {
			client.Flow = "xtls-rprx-vision"
		}
		ib.Settings.Clients = append(ib.Settings.Clients, client)
		if shortID != "" && ib.StreamSettings != nil && ib.StreamSettings.RealitySettings != nil {
			// avoid duplicate shortID
			exists := false
			for _, sid := range ib.StreamSettings.RealitySettings.ShortIds {
				if sid == shortID {
					exists = true
					break
				}
			}
			if !exists {
				ib.StreamSettings.RealitySettings.ShortIds = append(ib.StreamSettings.RealitySettings.ShortIds, shortID)
			}
		}
	}
	return m.saveServer(sj)
}

// ListUsers returns active users aggregated uniquely.
func (m *Manager) ListUsers() ([]UserInfo, error) {
	sj, err := m.loadServer()
	if err != nil {
		return nil, err
	}
	seen := map[string]UserInfo{}
	for _, ib := range sj.Inbounds {
		for _, c := range ib.Settings.Clients {
			if c.Email == "" || c.Email == m.cfg.Server.DefaultEmail {
				continue
			}
			u := emailToUsername(c.Email)
			if _, ok := seen[u]; !ok {
				seen[u] = UserInfo{Username: u, ID: c.ID}
			}
		}
	}
	users := make([]UserInfo, 0, len(seen))
	for _, ui := range seen {
		users = append(users, ui)
	}
	sort.Slice(users, func(i, j int) bool { return users[i].Username < users[j].Username })
	return users, nil
}

// GenerateLink builds a VLESS link from the user's client config.
func (m *Manager) GenerateLink(username string) (string, error) {
	path := filepath.Join(m.cfg.Xray.ConfigDir, fmt.Sprintf("config_client_%s.json", username))
	cj, err := m.loadClient(path)
	if err != nil {
		return "", err
	}
	if len(cj.Outbounds) == 0 {
		return "", errors.New("no outbound")
	}
	ob := cj.Outbounds[0]
	if ob.Protocol != "vless" {
		return "", fmt.Errorf("unsupported protocol %s", ob.Protocol)
	}
	if ob.Settings == nil || len(ob.Settings.Vnext) == 0 {
		return "", errors.New("missing vnext")
	}
	vn := ob.Settings.Vnext[0]
	if len(vn.Users) == 0 {
		return "", errors.New("missing users")
	}
	u := vn.Users[0]
	ss := ob.StreamSettings
	params := []string{"encryption=none"}
	if u.Flow != "" {
		params = append(params, "flow="+u.Flow)
	}
	if ss != nil {
		if ss.Security != "" {
			params = append(params, "security="+ss.Security)
		}
		if ss.Network != "" {
			params = append(params, "type="+ss.Network)
		}
		if ss.RealitySettings != nil {
			r := ss.RealitySettings
			if r.PublicKey != "" {
				params = append(params, "pbk="+r.PublicKey)
			}
			if r.Fingerprint != "" {
				params = append(params, "fp="+r.Fingerprint)
			}
			if r.ServerName != "" {
				params = append(params, "sni="+r.ServerName)
			}
			if r.ShortID != "" {
				params = append(params, "sid="+r.ShortID)
			}
		}
	}
	query := strings.Join(params, "&")
	link := fmt.Sprintf("vless://%s@%s:%d?%s#%s", u.ID, vn.Address, vn.Port, query, username)
	return link, nil
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

// ---- internal helpers ----

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

func (m *Manager) createClientConfig(username, id, email, shortID string) error {
	template, err := m.loadClient(m.clientTemplatePath)
	if err != nil {
		return err
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
		return err
	}
	return os.WriteFile(outPath, data, 0644)
}
