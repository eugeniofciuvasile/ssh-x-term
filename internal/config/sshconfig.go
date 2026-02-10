package config

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	keyring "github.com/zalando/go-keyring"
)

const (
	sshConfigFileName   = "config"
	sshKeyringService   = "ssh-x-term"
	sxtCommentPrefix    = "#sxt:"
	migrationMarkerFile = ".migration_done"
)

type SSHConfigManager struct {
	ConfigPath string
	Config     *Config
}

func NewSSHConfigManager() (*SSHConfigManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Failed to get user home directory: %v", err)
		return nil, err
	}

	sshDir := filepath.Join(homeDir, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		log.Printf("Failed to create .ssh directory: %v", err)
		return nil, err
	}

	configPath := filepath.Join(sshDir, sshConfigFileName)

	return &SSHConfigManager{
		ConfigPath: configPath,
		Config:     NewConfig(),
	}, nil
}

// parseSSHConfig parses the SSH config file and extracts connections
func (scm *SSHConfigManager) parseSSHConfig() error {
	file, err := os.Open(scm.ConfigPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var currentConn *SSHConnection
	var sxtMetadata map[string]string
	connections := []SSHConnection{}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Parse sxt metadata comments
		if strings.HasPrefix(line, sxtCommentPrefix) {
			if sxtMetadata == nil {
				sxtMetadata = make(map[string]string)
			}
			metadata := strings.TrimPrefix(line, sxtCommentPrefix)
			parts := strings.SplitN(metadata, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				sxtMetadata[key] = value
			}
			continue
		}

		// Skip empty lines and regular comments
		if line == "" || strings.HasPrefix(line, "#") {
			// Reset metadata on empty line or regular comment if we're not in a Host block
			if currentConn == nil {
				sxtMetadata = nil
			}
			continue
		}

		// Parse SSH config directives
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		keyword := strings.ToLower(fields[0])
		value := strings.Join(fields[1:], " ")

		if keyword == "host" {
			// Save previous connection if exists
			if currentConn != nil {
				// If no explicit metadata about use_password
				if sxtMetadata == nil || sxtMetadata["use_password"] == "" {
					// Only force password auth if no key file AND UsePassword wasn't explicitly set to false
					if currentConn.KeyFile == "" && currentConn.UsePassword {
						currentConn.UsePassword = true
					}
				}
				connections = append(connections, *currentConn)
			}

			// Start new connection - default to password auth unless key file is found
			currentConn = &SSHConnection{
				Port:        22,    // Default SSH port
				HostPattern: value, // Store the Host pattern
				UsePassword: true,  // Default to password auth
			}

			// Apply metadata to new connection
			if sxtMetadata != nil {
				if id, ok := sxtMetadata["id"]; ok {
					currentConn.ID = id
				}
				if name, ok := sxtMetadata["name"]; ok {
					currentConn.Name = name
				}
				if notes, ok := sxtMetadata["notes"]; ok {
					currentConn.Notes = notes
				}
				if usePassword, ok := sxtMetadata["use_password"]; ok {
					currentConn.UsePassword = usePassword == "true"
				}
				if publicKey, ok := sxtMetadata["public_key"]; ok {
					currentConn.PublicKey = publicKey
				}
				if orgID, ok := sxtMetadata["organization_id"]; ok {
					currentConn.OrganizationID = orgID
				}
			}

			// Generate ID if not set
			if currentConn.ID == "" {
				currentConn.ID = generateID()
			}

			// Set name from Host pattern if not set
			if currentConn.Name == "" {
				currentConn.Name = value
			}

			// Reset metadata for next host
			sxtMetadata = nil

		} else if currentConn != nil {
			switch keyword {
			case "hostname":
				currentConn.Host = value
			case "port":
				if port, err := strconv.Atoi(value); err == nil {
					currentConn.Port = port
				}
			case "user":
				currentConn.Username = value
			case "identityfile":
				currentConn.KeyFile = value
				currentConn.UsePassword = false // Has key file, not password auth
			case "identitiesonly", "pubkeyauthentication":
				// These options indicate key-based authentication
				if value == "yes" {
					currentConn.UsePassword = false
				}
			case "preferredauthentications":
				// Check if password is preferred over publickey
				if strings.Contains(strings.ToLower(value), "publickey") {
					currentConn.UsePassword = false
				}
			}
		}
	}

	// Save last connection
	if currentConn != nil {
		// If no explicit metadata about use_password
		if sxtMetadata == nil || sxtMetadata["use_password"] == "" {
			// If no key file AND UsePassword wasn't explicitly set to false by config options
			// then default to password auth
			if currentConn.KeyFile == "" && currentConn.UsePassword {
				currentConn.UsePassword = true
			}
		}
		connections = append(connections, *currentConn)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	scm.Config.Connections = connections
	return nil
}

// writeSSHConfig writes connections to SSH config file
func (scm *SSHConfigManager) writeSSHConfig() error {
	// Write new config with all entries properly tagged
	file, err := os.OpenFile(scm.ConfigPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	// Write all connections with sxt metadata
	for i := range scm.Config.Connections {
		conn := &scm.Config.Connections[i]

		// Ensure ID exists
		if conn.ID == "" {
			conn.ID = generateID()
		}

		// Store password if it exists
		if conn.Password != "" {
			keyring.Set(sshKeyringService, conn.ID, conn.Password)
			conn.Password = "" // Don't keep in memory
		}

		// Write metadata comments
		fmt.Fprintf(writer, "%sid=%s\n", sxtCommentPrefix, conn.ID)
		if conn.Name != "" {
			fmt.Fprintf(writer, "%sname=%s\n", sxtCommentPrefix, conn.Name)
		}
		if conn.Notes != "" {
			fmt.Fprintf(writer, "%snotes=%s\n", sxtCommentPrefix, conn.Notes)
		}
		fmt.Fprintf(writer, "%suse_password=%t\n", sxtCommentPrefix, conn.UsePassword)
		if conn.PublicKey != "" {
			fmt.Fprintf(writer, "%spublic_key=%s\n", sxtCommentPrefix, conn.PublicKey)
		}
		if conn.OrganizationID != "" {
			fmt.Fprintf(writer, "%sorganization_id=%s\n", sxtCommentPrefix, conn.OrganizationID)
		}

		// Write SSH config
		hostPattern := conn.HostPattern
		if hostPattern == "" {
			// Fallback to Name or Host if HostPattern not set
			hostPattern = conn.Name
			if hostPattern == "" {
				hostPattern = conn.Host
			}
		}
		fmt.Fprintf(writer, "Host %s\n", hostPattern)

		if conn.Host != "" {
			fmt.Fprintf(writer, "    HostName %s\n", conn.Host)
		}
		if conn.Port != 0 && conn.Port != 22 {
			fmt.Fprintf(writer, "    Port %d\n", conn.Port)
		}
		if conn.Username != "" {
			fmt.Fprintf(writer, "    User %s\n", conn.Username)
		}
		if conn.KeyFile != "" {
			fmt.Fprintf(writer, "    IdentityFile %s\n", conn.KeyFile)
		}
		fmt.Fprintf(writer, "\n")
	}

	return writer.Flush()
}

// AddConnection stores an SSH connection in the SSH config.
func (scm *SSHConfigManager) AddConnection(conn SSHConnection) error {
	// Generate ID if not set
	if conn.ID == "" {
		conn.ID = generateID()
	}

	// Handle password securely using keyring
	if conn.Password != "" {
		if err := keyring.Set(sshKeyringService, conn.ID, conn.Password); err != nil {
			log.Printf("Failed to store password in keyring: %v", err)
			return err
		}
		conn.Password = ""
	}

	// Check if connection already exists
	for i, existing := range scm.Config.Connections {
		if existing.ID == conn.ID {
			scm.Config.Connections[i] = conn
			return scm.Save()
		}
	}

	scm.Config.Connections = append(scm.Config.Connections, conn)
	return scm.Save()
}

// EditConnection updates an existing SSH connection in the configuration.
func (scm *SSHConfigManager) EditConnection(conn SSHConnection) error {
	// Handle password securely using keyring
	if conn.Password != "" {
		if err := keyring.Set(sshKeyringService, conn.ID, conn.Password); err != nil {
			log.Printf("Failed to store password in keyring: %v", err)
			return err
		}
		conn.Password = ""
	}

	for i, existing := range scm.Config.Connections {
		if existing.ID == conn.ID {
			scm.Config.Connections[i] = conn
			return scm.Save()
		}
	}

	log.Printf("Connection with ID %s not found for edit", conn.ID)
	return errors.New("connection with ID " + conn.ID + " not found")
}

// DeleteConnection removes an SSH connection from the configuration and keyring.
func (scm *SSHConfigManager) DeleteConnection(id string) error {
	// Remove password from keyring
	if err := keyring.Delete(sshKeyringService, id); err != nil {
		log.Printf("Failed to delete password from keyring (may not exist): %v", err)
	}

	for i, conn := range scm.Config.Connections {
		if conn.ID == id {
			scm.Config.Connections = append(scm.Config.Connections[:i], scm.Config.Connections[i+1:]...)
			return scm.Save()
		}
	}

	log.Printf("Connection with ID %s not found for deletion", id)
	return errors.New("connection with ID " + id + " not found")
}

// GetConnection retrieves an SSH connection, including its password if stored.
func (scm *SSHConfigManager) GetConnection(id string) (SSHConnection, bool) {
	for _, conn := range scm.Config.Connections {
		if conn.ID == id {
			// Retrieve password or passphrase from keyring
			// For key-based auth, use "passphrase:" prefix; for password auth, use ID directly
			var keyringKey string
			if conn.UsePassword {
				keyringKey = id
			} else {
				keyringKey = "passphrase:" + id
			}
			
			password, err := keyring.Get(sshKeyringService, keyringKey)
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
func (scm *SSHConfigManager) ListConnections() []SSHConnection {
	return scm.Config.Connections
}

// Load loads the SSH connections from the SSH config file.
func (scm *SSHConfigManager) Load() error {
	err := scm.parseSSHConfig()
	if err != nil {
		return err
	}

	// Check if migration is needed (no marker file exists)
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	sxtConfigDir := filepath.Join(homeDir, ".config", "ssh-x-term")
	migrationMarkerPath := filepath.Join(sxtConfigDir, migrationMarkerFile)

	if _, err := os.Stat(migrationMarkerPath); os.IsNotExist(err) {
		// First load - perform migration
		log.Println("First load detected, performing initial migration...")

		// Create backup before migration
		if _, err := os.Stat(scm.ConfigPath); err == nil && len(scm.Config.Connections) > 0 {
			backupPath := fmt.Sprintf("%s.backup.%s", scm.ConfigPath, getTimestamp())
			if err := copyFile(scm.ConfigPath, backupPath); err != nil {
				log.Printf("Warning: failed to create backup: %v", err)
			} else {
				log.Printf("Created initial migration backup at: %s", backupPath)
			}
		}

		// Try to recover passwords for all connections
		for i := range scm.Config.Connections {
			conn := &scm.Config.Connections[i]

			// Ensure ID exists
			if conn.ID == "" {
				conn.ID = generateID()
			}

			// Try to recover password if not set
			if conn.UsePassword && conn.Password == "" {
				recoveredPassword := tryRecoverPassword(*conn)
				if recoveredPassword != "" {
					// Store password with proper ID
					keyring.Set(sshKeyringService, conn.ID, recoveredPassword)
					log.Printf("Recovered password for connection: %s", conn.Name)
				}
			}
		}

		// Save the migrated config
		if err := scm.Save(); err != nil {
			log.Printf("Warning: failed to save migrated config: %v", err)
			return err
		}

		// Create the marker file
		if err := os.MkdirAll(sxtConfigDir, 0755); err != nil {
			log.Printf("Warning: failed to create config directory: %v", err)
		} else {
			markerContent := fmt.Sprintf("SSH config migration completed at: %s\n", time.Now().Format(time.RFC3339))
			if err := os.WriteFile(migrationMarkerPath, []byte(markerContent), 0644); err != nil {
				log.Printf("Warning: failed to create migration marker: %v", err)
			} else {
				log.Printf("Created migration marker at: %s", migrationMarkerPath)
			}
		}

		log.Println("Initial migration completed successfully")
	}

	return nil
}

// Save saves the SSH connections to the SSH config file.
func (scm *SSHConfigManager) Save() error {
	return scm.writeSSHConfig()
}

// Helper function to generate unique ID
func generateID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based ID if random fails
		return fmt.Sprintf("sxt-%d", os.Getpid())
	}
	return fmt.Sprintf("sxt-%s", hex.EncodeToString(b))
}

// Helper function to get timestamp for backup files
func getTimestamp() string {
	// Format: YYYYMMDD-HHMMSS
	return time.Now().Format("20060102-150405")
}

// tryRecoverPassword attempts to recover password from keyring using various strategies
func tryRecoverPassword(conn SSHConnection) string {
	// Try strategies in order of likelihood
	strategies := []string{
		conn.ID,          // Original ID
		conn.HostPattern, // Host pattern
		conn.Host,        // Hostname
		fmt.Sprintf("%s@%s", conn.Username, conn.Host), // user@host
	}

	for _, id := range strategies {
		if id == "" {
			continue
		}
		password, err := keyring.Get(sshKeyringService, id)
		if err == nil && password != "" {
			log.Printf("Recovered password for connection using ID: %s", id)
			// Store with proper ID for future use
			if conn.ID != "" && id != conn.ID {
				keyring.Set(sshKeyringService, conn.ID, password)
			}
			return password
		}
	}

	return ""
}

// Helper function to copy file
func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, input, 0600)
}
