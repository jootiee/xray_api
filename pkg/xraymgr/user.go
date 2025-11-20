package xraymgr

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Add creates a new user in server config and a dedicated client config.
func (m *Manager) Add(username string) (*UserInfo, error) {
	if username == "" {
		return nil, errors.New("username empty")
	}
	email := usernameToEmail(username)
	sj, err := m.loadServer()
	if err != nil {
		return nil, err
	}

	if ok, err := m.IsConfigExists(username); err != nil {
		return nil, err
	} else if ok {
		return nil, fmt.Errorf("user %s already exists", username)
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
	} else {
		return &UserInfo{Username: username, ID: id, ShortID: shortID}, nil
	}
}

// Delete removes a user from server and deletes its client config file.
func (m *Manager) Delete(username string) error {
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

// Suspend removes user from server without deleting its client config.
func (m *Manager) Suspend(username string) error {
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

// Resume re-adds user using its client config.
func (m *Manager) Resume(username string) error {
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

// List returns active users aggregated uniquely.
func (m *Manager) List() ([]UserInfo, error) {
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

// GetLink builds a VLESS link from the user's client config.
func (m *Manager) GetLink(username string) (string, error) {
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

func (m *Manager) IsConfigExists(username string) (bool, error) {
	sj, err := m.loadServer()
	if err != nil {
		return false, err
	}
	email := usernameToEmail(username)

	for _, ib := range sj.Inbounds {
		for _, c := range ib.Settings.Clients {
			if c.Email == email {
				return true, nil
			}
		}
	}
	return false, nil
}

func (m *Manager) GetConfigPath(username string) (string, error) {
	if ok, err := m.IsConfigExists(username); err != nil {
		return "", err
	} else if !ok {
		return "", fmt.Errorf("%s's config doesn't exist", username)
	}
	path := filepath.Join(m.cfg.Xray.ConfigDir, fmt.Sprintf("config_client_%s.json", username))

	if !fileExists(path) {
		return "", fmt.Errorf("client config for %s not found", username)
	}
	return path, nil
}
