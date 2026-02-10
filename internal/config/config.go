package config

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"slices"

	keyring "github.com/zalando/go-keyring"
)

const (
	defaultConfigFileName = "ssh-x-term.json"
	keyringService        = "ssh-x-term" // Keyring service name
)

type ConfigManager struct {
	ConfigPath string
	Config     *Config
}

var IsTmuxAvailable bool = false

func NewConfigManager() (*ConfigManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Failed to get user home directory: %v", err)
		return nil, err
	}

	configDir := filepath.Join(homeDir, ".config", "ssh-x-term")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		log.Printf("Failed to create config directory: %v", err)
		return nil, err
	}

	configPath := filepath.Join(configDir, defaultConfigFileName)

	return &ConfigManager{
		ConfigPath: configPath,
		Config:     NewConfig(),
	}, nil
}

// AddConnection stores an SSH connection in the configuration.
func (cm *ConfigManager) AddConnection(conn SSHConnection) error {
	// Handle password securely using keyring
	if conn.Password != "" {
		if err := keyring.Set(keyringService, conn.ID, conn.Password); err != nil {
			log.Fatalf("Failed to store password in keyring: %v", err)
			return err
		}
		// Unset password in plaintext config
		conn.Password = ""
	}

	for i, existing := range cm.Config.Connections {
		if existing.ID == conn.ID {
			cm.Config.Connections[i] = conn
			return cm.Save()
		}
	}
	cm.Config.Connections = append(cm.Config.Connections, conn)
	return cm.Save()
}

// EditConnection updates an existing SSH connection in the configuration.
func (cm *ConfigManager) EditConnection(conn SSHConnection) error {
	// Handle password securely using keyring
	if conn.Password != "" {
		if err := keyring.Set(keyringService, conn.ID, conn.Password); err != nil {
			log.Printf("Failed to store password in keyring: %v", err)
			return err
		}

		// Unset password in plaintext config
		conn.Password = ""
	}

	for i, existing := range cm.Config.Connections {
		if existing.ID == conn.ID {
			cm.Config.Connections[i] = conn
			return cm.Save()
		}
	}
	log.Printf("Connection with ID %s not found for edit", conn.ID)
	return errors.New("connection with ID " + conn.ID + " not found")
}

// DeleteConnection removes an SSH connection from the configuration and keyring.
func (cm *ConfigManager) DeleteConnection(id string) error {
	// Remove password from keyring
	if err := keyring.Delete(keyringService, id); err != nil {
		log.Printf("Failed to delete password from keyring (may not exist): %v", err)
	}

	for i, conn := range cm.Config.Connections {
		if conn.ID == id {
			cm.Config.Connections = slices.Delete(cm.Config.Connections, i, i+1)
			return cm.Save()
		}
	}
	log.Printf("Connection with ID %s not found for deletion", id)
	return errors.New("connection with ID " + id + " not found")
}

// GetConnection retrieves an SSH connection, including its password if stored.
func (cm *ConfigManager) GetConnection(id string) (SSHConnection, bool) {
	for _, conn := range cm.Config.Connections {
		if conn.ID == id {
			// Retrieve password or passphrase from keyring
			// For key-based auth, use "passphrase:" prefix; for password auth, use ID directly
			var keyringKey string
			if conn.UsePassword {
				keyringKey = id
			} else {
				keyringKey = "passphrase:" + id
			}
			
			password, err := keyring.Get(keyringService, keyringKey)
			if err != nil {
				log.Printf("Failed to retrieve password from keyring (key: %s): %v", keyringKey, err)
			} else {
				conn.Password = password
				log.Printf("Retrieved password from keyring for connection ID: %s (key: %s)", id, keyringKey)
			}
			return conn, true
		}
	}
	return SSHConnection{}, false
}

// ListConnections retrieves all SSH connections, excluding passwords for security.
func (cm *ConfigManager) ListConnections() []SSHConnection {
	return cm.Config.Connections
}

// Load loads the SSH connections from the config file.
func (cm *ConfigManager) Load() error {
	data, err := os.ReadFile(cm.ConfigPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		log.Printf("Failed to read config file: %v", err)
		return err
	}
	if err := json.Unmarshal(data, &cm.Config); err != nil {
		log.Printf("Failed to parse config file: %v", err)
		return err
	}
	return nil
}

// Save saves the SSH connections to the config file.
func (cm *ConfigManager) Save() error {
	data, err := json.MarshalIndent(cm.Config, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal config: %v", err)
		return err
	}
	if err := os.WriteFile(cm.ConfigPath, data, 0644); err != nil {
		log.Printf("Failed to write config file: %v", err)
		return err
	}
	return nil
}
