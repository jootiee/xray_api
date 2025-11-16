package xraymgr

// ServerJSON describes the subset of server config we manipulate.
type ServerJSON struct {
	Inbounds  []Inbound  `json:"inbounds"`
	Outbounds []Outbound `json:"outbounds"`
}

// ---- Shared structures (used by both server and client where applicable) ----

type Inbound struct {
	Tag            string          `json:"tag,omitempty"`
	Listen         string          `json:"listen,omitempty"`
	Port           int             `json:"port,omitempty"`
	Protocol       string          `json:"protocol,omitempty"`
	Settings       InboundSettings `json:"settings"`
	StreamSettings *StreamSettings `json:"streamSettings,omitempty"`
	Sniffing       *Sniffing       `json:"sniffing,omitempty"`
}

type InboundSettings struct {
	Clients []ServerClient `json:"clients,omitempty"`
	Address string         `json:"address,omitempty"`
	Auth    string         `json:"auth,omitempty"`
	UDP     bool           `json:"udp,omitempty"`
}

type ServerClient struct {
	ID    string `json:"id"`
	Email string `json:"email,omitempty"`
	Flow  string `json:"flow,omitempty"`
}

type Outbound struct {
	Tag            string            `json:"tag,omitempty"`
	Protocol       string            `json:"protocol,omitempty"`
	Settings       *OutboundSettings `json:"settings,omitempty"`
	StreamSettings *StreamSettings   `json:"streamSettings,omitempty"`
}

type OutboundSettings struct {
	Vnext []Vnext `json:"vnext,omitempty"`
}

type Vnext struct {
	Address string      `json:"address"`
	Port    int         `json:"port"`
	Users   []VnextUser `json:"users"`
}

type VnextUser struct {
	ID         string `json:"id"`
	Email      string `json:"email,omitempty"`
	Encryption string `json:"encryption,omitempty"`
	Flow       string `json:"flow,omitempty"`
}

type StreamSettings struct {
	Network         string           `json:"network,omitempty"`
	Security        string           `json:"security,omitempty"`
	RealitySettings *RealitySettings `json:"realitySettings,omitempty"`
	GRPCSettings    *GRPCSettings    `json:"grpcSettings,omitempty"`
}

type RealitySettings struct {
	Show        bool     `json:"show"`
	PrivateKey  string   `json:"privateKey,omitempty"`
	PublicKey   string   `json:"publicKey,omitempty"`
	Dest        string   `json:"dest,omitempty"`
	ServerNames []string `json:"serverNames,omitempty"`
	ShortIds    []string `json:"shortIds,omitempty"`
	Fingerprint string   `json:"fingerprint,omitempty"`
	// client side
	ServerName string `json:"serverName,omitempty"`
	ShortID    string `json:"shortId,omitempty"`
}

type GRPCSettings struct {
	ServiceName string `json:"serviceName,omitempty"`
}

type Sniffing struct {
	Enabled      bool     `json:"enabled"`
	DestOverride []string `json:"destOverride,omitempty"`
	RouteOnly    bool     `json:"routeOnly,omitempty"`
}

// ---- Client-only top-level sections ----

type Log struct {
	Access   string `json:"access,omitempty"`
	Error    string `json:"error,omitempty"`
	Loglevel string `json:"loglevel,omitempty"`
	DNSLog   bool   `json:"dnsLog,omitempty"`
}

type Stats struct{}

type Policy struct {
	Levels map[string]PolicyLevel `json:"levels,omitempty"`
	System *SystemPolicy          `json:"system,omitempty"`
}

type PolicyLevel struct {
	StatsUserUplink   bool `json:"statsUserUplink,omitempty"`
	StatsUserDownlink bool `json:"statsUserDownlink,omitempty"`
}

type SystemPolicy struct {
	StatsOutboundUplink   bool `json:"statsOutboundUplink,omitempty"`
	StatsOutboundDownlink bool `json:"statsOutboundDownlink,omitempty"`
}

type API struct {
	Tag      string   `json:"tag"`
	Services []string `json:"services"`
}

type Routing struct {
	DomainStrategy string        `json:"domainStrategy,omitempty"`
	Rules          []RoutingRule `json:"rules,omitempty"`
}

type RoutingRule struct {
	Type        string   `json:"type"`
	InboundTag  []string `json:"inboundTag,omitempty"`
	OutboundTag string   `json:"outboundTag,omitempty"`
	IP          []string `json:"ip,omitempty"`
	Domain      []string `json:"domain,omitempty"`
	Protocol    []string `json:"protocol,omitempty"`
	Port        string   `json:"port,omitempty"`
	SourcePort  string   `json:"sourcePort,omitempty"`
}

// ClientJSON represents the template client config we alter per user and must keep extra sections.
type ClientJSON struct {
	Log       *Log       `json:"log,omitempty"`
	Stats     *Stats     `json:"stats,omitempty"`
	Policy    *Policy    `json:"policy,omitempty"`
	API       *API       `json:"api,omitempty"`
	Inbounds  []Inbound  `json:"inbounds,omitempty"`
	Outbounds []Outbound `json:"outbounds"`
	Routing   *Routing   `json:"routing,omitempty"`
}

// UserInfo returned for listing.
type UserInfo struct {
	Username string
	ID       string
}
