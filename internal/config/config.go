package config

import (
	"encoding/json"
	"errors"
	"log"
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

// Storage interface methods
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

func (cm *ConfigManager) EditConnection(conn SSHConnection) error {
	for i, existing := range cm.Config.Connections {
		if existing.ID == conn.ID {
			cm.Config.Connections[i] = conn
			return cm.Save()
		}
	}
	log.Printf("Connection with ID %s not found for edit", conn.ID)
	return errors.New("connection with ID " + conn.ID + " not found")
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
	log.Printf("Connection with ID %s not found for deletion", id)
	return errors.New("connection with ID " + id + " not found")
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
