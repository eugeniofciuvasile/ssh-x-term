package ssh

import (
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/eugeniofciuvasile/ssh-x-term/internal/config"
	"github.com/zalando/go-keyring"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

const keyringService = "ssh-x-term"

// PassphraseRequiredError indicates that a passphrase is needed to decrypt the SSH key
type PassphraseRequiredError struct {
	KeyFile string
}

func (e *PassphraseRequiredError) Error() string {
	return fmt.Sprintf("passphrase required for encrypted key: %s", e.KeyFile)
}

// Client represents an SSH client connection
type Client struct {
	conn *ssh.Client
}

// NewClient creates a new SSH client from a connection configuration
func NewClient(connConfig config.SSHConnection) (*Client, error) {
	log.Printf("[NewClient] Starting connection for user=%s host=%s port=%d keyFile=%q",
		connConfig.Username, connConfig.Host, connConfig.Port, connConfig.KeyFile)

	// If password-based authentication is enabled, retrieve the password from the keyring
	if connConfig.UsePassword && connConfig.Password == "" {
		password, err := keyring.Get(keyringService, connConfig.ID)
		if err != nil {
			log.Printf("Failed to retrieve password from keyring for connection ID %s: %v", connConfig.ID, err)
			return nil, fmt.Errorf("failed to retrieve password: %w", err)
		}
		connConfig.Password = password
	}

	// Prepare Auth Methods
	var authMethods []ssh.AuthMethod

	// 1. SSH Agent Support (Attempt this first for keys)
	if socket := os.Getenv("SSH_AUTH_SOCK"); socket != "" {
		log.Printf("[NewClient] SSH_AUTH_SOCK found: %s", socket)
		if conn, err := net.Dial("unix", socket); err == nil {
			agentClient := agent.NewClient(conn)
			authMethods = append(authMethods, ssh.PublicKeysCallback(agentClient.Signers))
			log.Printf("[NewClient] Added SSH agent auth method")
		}
	} else {
		log.Printf("[NewClient] No SSH_AUTH_SOCK found")
	}

	// 2. Identity File (Key File) Support
	keyFile := connConfig.KeyFile

	// Expand tilde in key file path
	if keyFile != "" && keyFile[0] == '~' {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			keyFile = filepath.Join(homeDir, keyFile[2:]) // Skip "~/"
		}
	}

	// If no key file specified and not using password, try default key location
	if keyFile == "" && !connConfig.UsePassword {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			keyFile = filepath.Join(homeDir, ".ssh", "id_rsa")
			log.Printf("[NewClient] Using default key file: %s", keyFile)
		}
	}

	if keyFile != "" {
		log.Printf("[NewClient] Reading key file: %s", keyFile)
		keyBytes, err := os.ReadFile(keyFile)
		if err != nil {
			log.Printf("[NewClient] Failed to read key file %s: %v", keyFile, err)
		} else {
			// Try standard key parsing
			signer, err := ssh.ParsePrivateKey(keyBytes)

			// If that fails (e.g., encrypted key), try with passphrase if provided in Password field
			if err != nil {
				log.Printf("[NewClient] ParsePrivateKey failed: %v (type: %T)", err, err)
				if _, ok := err.(*ssh.PassphraseMissingError); ok {
					log.Printf("[NewClient] Key is encrypted (PassphraseMissingError)")
					// Key is encrypted. Do we have a "password" (acting as passphrase)?
					if connConfig.Password != "" {
						log.Printf("[NewClient] Attempting to parse with provided passphrase")
						signer, err = ssh.ParsePrivateKeyWithPassphrase(keyBytes, []byte(connConfig.Password))
						if err != nil {
							log.Printf("[NewClient] Failed to parse encrypted key with provided passphrase: %v", err)
							// Return PassphraseRequiredError so UI can prompt for correct passphrase
							return nil, &PassphraseRequiredError{KeyFile: keyFile}
						}
						log.Printf("[NewClient] Successfully parsed key with passphrase")
					} else {
						// No passphrase provided - return error so UI can prompt
						log.Printf("[NewClient] No passphrase provided, returning PassphraseRequiredError")
						return nil, &PassphraseRequiredError{KeyFile: keyFile}
					}
				} else {
					log.Printf("[NewClient] Failed to parse private key (not a passphrase issue): %v", err)
				}
			} else {
				log.Printf("[NewClient] Key parsed successfully (no passphrase needed)")
			}

			if signer != nil {
				authMethods = append(authMethods, ssh.PublicKeys(signer))
				log.Printf("[NewClient] Added public key auth method")
			}
		}
	}

	// 3. Password Authentication
	if connConfig.UsePassword && connConfig.Password != "" {
		authMethods = append(authMethods, ssh.Password(connConfig.Password))
		log.Printf("[NewClient] Added password auth method")
	}

	log.Printf("[NewClient] Total auth methods: %d", len(authMethods))

	// Create SSH client configuration
	sshConfig := &ssh.ClientConfig{
		User:            connConfig.Username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Note: In production, consider using a more secure approach
		Timeout:         10 * time.Second,
	}

	// Connect to the SSH server
	addr := fmt.Sprintf("%s:%d", connConfig.Host, connConfig.Port)
	log.Printf("[NewClient] Attempting to connect to %s", addr)
	conn, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		log.Printf("[NewClient] Failed to connect to SSH server %s: %v", addr, err)
		return nil, fmt.Errorf("failed to connect to SSH server: %w", err)
	}

	log.Printf("[NewClient] Successfully connected to %s", addr)
	return &Client{conn: conn}, nil
}

// Close closes the SSH client connection
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// NewSession creates a new SSH session
func (c *Client) NewSession() (*ssh.Session, error) {
	if c.conn == nil {
		log.Print("Attempted to create session on nil SSH connection")
		return nil, fmt.Errorf("SSH client not connected")
	}

	return c.conn.NewSession()
}
