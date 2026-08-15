package config

// SSHConnection represents a saved SSH connection configuration
type SSHConnection struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	HostPattern    string         `json:"host_pattern,omitempty"` // SSH config Host pattern
	Host           string         `json:"host"`                   // Actual hostname
	Port           int            `json:"port"`
	Username       string         `json:"username"`
	Password       string         `json:"password,omitempty"`
	SudoPassword   string         `json:"sudo_password,omitempty"`
	PublicKey      string         `json:"public_key,omitempty"`
	UsePassword    bool           `json:"use_password"`
	KeyFile        string         `json:"key_file,omitempty"`
	Notes          string         `json:"notes,omitempty"`
	OrganizationID string         `json:"organizationId"`
	CollectionIds  []string       `json:"collectionIds,omitempty"`
	Pinned         bool           `json:"pinned"`
	Order          int            `json:"order"`
	Tunnels        []TunnelConfig `json:"tunnels,omitempty"`
}

// TunnelType represents the forwarding mode
type TunnelType string

const (
	TunnelTypeLocal   TunnelType = "local"   // Local forwarding (-L)
	TunnelTypeRemote  TunnelType = "remote"  // Remote forwarding (-R)
	TunnelTypeDynamic TunnelType = "dynamic" // Dynamic SOCKS5 forwarding (-D)
)

// TunnelConfig represents a port forwarding tunnel configuration
type TunnelConfig struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Type       TunnelType `json:"type"`        // local, remote, dynamic
	BindHost   string     `json:"bind_host"`   // e.g. "127.0.0.1" or "0.0.0.0"
	BindPort   int        `json:"bind_port"`   // Local listening port (or remote port for -R)
	TargetHost string     `json:"target_host"` // Destination host (e.g. "10.0.0.12" or "localhost")
	TargetPort int        `json:"target_port"` // Destination port (e.g. 5432)
	AutoStart  bool       `json:"auto_start"`  // Auto start when connecting
	Enabled    bool       `json:"enabled"`     // Enabled toggle state
	Notes      string     `json:"notes,omitempty"`
}

// Organization represents the user's organization
type Organization struct {
	Object  string `json:"object"`
	ID      string `json:"id"`
	Name    string `json:"name"`
	Status  int    `json:"status"`
	Type    int    `json:"type"`
	Enabled bool   `json:"enabled"`
}

// Collection represents the organization's collection of SSH connections
type Collection struct {
	Object         string  `json:"object"`
	ID             string  `json:"id"`
	OrganizationID string  `json:"organizationId"`
	Name           string  `json:"name"`
	ExternalID     *string `json:"externalId"`
}

type Config struct {
	Connections []SSHConnection `json:"connections"`
	LastUsed    string          `json:"last_used,omitempty"`
}

// NewConfig creates a new default configuration
func NewConfig() *Config {
	return &Config{
		Connections: []SSHConnection{},
	}
}
