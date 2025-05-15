package config

// Storage defines the backend interface for SSH connection storage.
type Storage interface {
	Load() error
	Save() error
	AddConnection(conn SSHConnection) error
	DeleteConnection(id string) error
	GetConnection(id string) (SSHConnection, bool)
	ListConnections() []SSHConnection
	EditConnection(conn SSHConnection) error
}
