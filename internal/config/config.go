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

type ConfigManager struct {
	ConfigPath string
	Config     *Config
}

var IsTmuxAvailable bool = false

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

// Storage interface methods
func (cm *ConfigManager) Load() error {
	data, err := os.ReadFile(cm.ConfigPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("failed to read config file: %w", err)
	}
	if err := json.Unmarshal(data, &cm.Config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}
	return nil
}

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

func (cm *ConfigManager) EditConnection(conn SSHConnection) error {
	for i, existing := range cm.Config.Connections {
		if existing.ID == conn.ID {
			cm.Config.Connections[i] = conn
			return cm.Save()
		}
	}
	return fmt.Errorf("connection with ID %s not found", conn.ID)
}

func (cm *ConfigManager) AddConnection(conn SSHConnection) error {
	for i, existing := range cm.Config.Connections {
		if existing.ID == conn.ID {
			cm.Config.Connections[i] = conn
			return cm.Save()
		}
	}
	cm.Config.Connections = append(cm.Config.Connections, conn)
	return cm.Save()
}

func (cm *ConfigManager) DeleteConnection(id string) error {
	for i, conn := range cm.Config.Connections {
		if conn.ID == id {
			cm.Config.Connections = slices.Delete(cm.Config.Connections, i, i+1)
			return cm.Save()
		}
	}
	return fmt.Errorf("connection with ID %s not found", id)
}

func (cm *ConfigManager) GetConnection(id string) (SSHConnection, bool) {
	for _, conn := range cm.Config.Connections {
		if conn.ID == id {
			return conn, true
		}
	}
	return SSHConnection{}, false
}

func (cm *ConfigManager) ListConnections() []SSHConnection {
	return cm.Config.Connections
}
