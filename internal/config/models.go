package config

// SSHConnection represents a saved SSH connection configuration
type SSHConnection struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Username    string `json:"username"`
	Password    string `json:"password,omitempty"`
	KeyFile     string `json:"key_file,omitempty"`
	UsePassword bool   `json:"use_password"`
}

// Config represents the application configuration
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
