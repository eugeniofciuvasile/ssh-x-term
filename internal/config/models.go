package config

// SSHConnection represents a saved SSH connection configuration
type SSHConnection struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Host           string   `json:"host"`
	Port           int      `json:"port"`
	Username       string   `json:"username"`
	Password       string   `json:"password,omitempty"`
	PublicKey      string   `json:"public_key,omitempty"`
	UsePassword    bool     `json:"use_password"`
	KeyFile        string   `json:"key_file,omitempty"`
	Notes          string   `json:"notes,omitempty"`
	OrganizationID string   `json:"organizationId"`
	CollectionIds  []string `json:"collectionIds,omitempty"`
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
