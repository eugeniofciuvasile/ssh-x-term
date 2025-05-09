package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

const (
	defaultConfigFileName = "ssh-x-term.json"
)

// ConfigManager handles loading, saving, and modifying the application configuration
type ConfigManager struct {
	ConfigPath string
	Config     *Config
}

// NewConfigManager creates a new configuration manager
func NewConfigManager() (*ConfigManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", "ssh-x-term")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	configPath := filepath.Join(configDir, defaultConfigFileName)

	return &ConfigManager{
		ConfigPath: configPath,
		Config:     NewConfig(),
	}, nil
}

// Load loads the configuration from the config file
func (cm *ConfigManager) Load() error {
	data, err := os.ReadFile(cm.ConfigPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Config file doesn't exist, use defaults
			return nil
		}
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if err := json.Unmarshal(data, &cm.Config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	return nil
}

// Save saves the configuration to the config file
func (cm *ConfigManager) Save() error {
	data, err := json.MarshalIndent(cm.Config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(cm.ConfigPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// AddConnection adds a new SSH connection to the configuration
func (cm *ConfigManager) AddConnection(conn SSHConnection) error {
	// Check if a connection with the same ID already exists
	for i, existing := range cm.Config.Connections {
		if existing.ID == conn.ID {
			// Update existing connection
			cm.Config.Connections[i] = conn
			return cm.Save()
		}
	}

	// Add new connection
	cm.Config.Connections = append(cm.Config.Connections, conn)
	return cm.Save()
}

// DeleteConnection removes an SSH connection from the configuration
func (cm *ConfigManager) DeleteConnection(id string) error {
	for i, conn := range cm.Config.Connections {
		if conn.ID == id {
			// Remove the connection
			cm.Config.Connections = slices.Delete(cm.Config.Connections, i, i+1)
			return cm.Save()
		}
	}

	return fmt.Errorf("connection with ID %s not found", id)
}

// GetConnection returns a connection by ID
func (cm *ConfigManager) GetConnection(id string) (SSHConnection, bool) {
	for _, conn := range cm.Config.Connections {
		if conn.ID == id {
			return conn, true
		}
	}

	return SSHConnection{}, false
}
