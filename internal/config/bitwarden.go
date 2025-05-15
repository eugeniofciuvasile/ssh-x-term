package config

import ()

// BitwardenConfig holds config for authentication (add fields as needed)
type BitwardenConfig struct {
	ServerURL string
	Email     string
	// Add OAuth tokens/password fields as needed
	AccessToken string
}

// BitwardenManager implements Storage backed by Bitwarden API.
type BitwardenManager struct {
	Config *BitwardenConfig
	// Add internal cache/mapping if needed
}

// NewBitwardenManager returns a Bitwarden storage backend.
func NewBitwardenManager(cfg *BitwardenConfig) (*BitwardenManager, error) {
	// Authenticate here (OAuth or password fallback)
	// Save access token to cfg.AccessToken
	return &BitwardenManager{
		Config: cfg,
	}, nil
}

func (b *BitwardenManager) Load() error {
	// Optionally cache/fetch all items on startup
	// Or always use API for live listing
	return nil
}

func (b *BitwardenManager) Save() error {
	// Not needed for Bitwarden – CRUD is live.
	return nil
}

func (b *BitwardenManager) AddConnection(conn SSHConnection) error {
	// Use Bitwarden API to create/update an item.
	// return errors.New("Bitwarden AddConnection not implemented")
	return nil
}

func (b *BitwardenManager) DeleteConnection(id string) error {
	// Use Bitwarden API to delete the item by ID.
	// return errors.New("Bitwarden DeleteConnection not implemented")
	return nil
}

func (b *BitwardenManager) GetConnection(id string) (SSHConnection, bool) {
	// Use Bitwarden API to fetch item by ID.
	return SSHConnection{}, false
}

func (b *BitwardenManager) ListConnections() []SSHConnection {
	// Use Bitwarden API to list and map items to SSHConnection.
	return nil
}
