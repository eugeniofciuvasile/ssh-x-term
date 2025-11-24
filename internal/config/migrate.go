package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	keyring "github.com/zalando/go-keyring"
)

const (
	legacyConfigFileName = "ssh-x-term.json"
	legacyKeyringService = "ssh-x-term"
)

// MigrateFromJSON migrates connections from the old JSON format to SSH config
func MigrateFromJSON() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	// Check if old JSON config exists
	oldConfigPath := filepath.Join(homeDir, ".config", "ssh-x-term", legacyConfigFileName)
	if _, err := os.Stat(oldConfigPath); os.IsNotExist(err) {
		// No old config to migrate
		return nil
	}

	log.Printf("Found legacy JSON config at %s, migrating to SSH config...", oldConfigPath)

	// Load old config
	data, err := os.ReadFile(oldConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read legacy config: %w", err)
	}

	var oldConfig Config
	if err := json.Unmarshal(data, &oldConfig); err != nil {
		return fmt.Errorf("failed to parse legacy config: %w", err)
	}

	if len(oldConfig.Connections) == 0 {
		log.Println("No connections to migrate")
		return nil
	}

	// Create new SSH config manager
	scm, err := NewSSHConfigManager()
	if err != nil {
		return fmt.Errorf("failed to create SSH config manager: %w", err)
	}

	// Load existing SSH config (if any) to avoid overwriting
	if err := scm.Load(); err != nil {
		log.Printf("Warning: failed to load existing SSH config: %v", err)
	}

	// Migrate each connection
	migratedCount := 0
	for _, conn := range oldConfig.Connections {
		// Try to retrieve password from old keyring
		password, err := keyring.Get(legacyKeyringService, conn.ID)
		if err == nil && password != "" {
			conn.Password = password
		}

		// Add connection to SSH config
		if err := scm.AddConnection(conn); err != nil {
			log.Printf("Warning: failed to migrate connection %s (%s): %v", conn.Name, conn.ID, err)
			continue
		}
		migratedCount++
	}

	log.Printf("Successfully migrated %d/%d connections", migratedCount, len(oldConfig.Connections))

	// Rename old config as backup
	backupPath := oldConfigPath + ".migrated"
	if err := os.Rename(oldConfigPath, backupPath); err != nil {
		log.Printf("Warning: failed to rename old config to %s: %v", backupPath, err)
	} else {
		log.Printf("Old config backed up to %s", backupPath)
	}

	return nil
}

// CheckAndMigrate checks if migration is needed and performs it
func CheckAndMigrate() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	oldConfigPath := filepath.Join(homeDir, ".config", "ssh-x-term", legacyConfigFileName)
	sshConfigPath := filepath.Join(homeDir, ".ssh", "config")

	// Check if old config exists
	if _, err := os.Stat(oldConfigPath); os.IsNotExist(err) {
		// No migration needed
		return nil
	}

	// Check if SSH config has sxt entries already
	if _, err := os.Stat(sshConfigPath); err == nil {
		data, err := os.ReadFile(sshConfigPath)
		if err == nil && len(data) > 0 {
			content := string(data)
			if len(content) > 0 && (contains(content, sxtCommentPrefix) || contains(content, "Host ")) {
				// SSH config already has content, skip auto-migration to avoid conflicts
				log.Println("SSH config already exists with content. Skipping auto-migration.")
				log.Printf("To manually migrate, your old config is at: %s", oldConfigPath)
				return nil
			}
		}
	}

	// Perform migration
	return MigrateFromJSON()
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && 
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		indexOf(s, substr) >= 0))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
