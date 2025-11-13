package ssh

import (
	"fmt"
	"log"
	"time"

	"github.com/eugeniofciuvasile/ssh-x-term/internal/config"
	"github.com/eugeniofciuvasile/ssh-x-term/pkg/sshutil"
	"github.com/zalando/go-keyring" // Import the keyring package
	"golang.org/x/crypto/ssh"
)

const keyringService = "ssh-x-term" // Define the keyring service name for consistency

// Client represents an SSH client connection
type Client struct {
	conn *ssh.Client
}

// NewClient creates a new SSH client from a connection configuration
func NewClient(connConfig config.SSHConnection) (*Client, error) {
	// If password-based authentication is enabled, retrieve the password from the keyring
	if connConfig.UsePassword && connConfig.Password == "" {
		password, err := keyring.Get(keyringService, connConfig.ID)
		if err != nil {
			log.Printf("Failed to retrieve password from keyring for connection ID %s: %v", connConfig.ID, err)
			return nil, fmt.Errorf("failed to retrieve password: %w", err)
		}
		connConfig.Password = password // Populate the password in the connection configuration
	}

	// Set up auth method based on configuration
	authMethod, err := sshutil.GetAuthMethod(connConfig.UsePassword, connConfig.Password, connConfig.KeyFile)
	if err != nil {
		log.Printf("Failed to get auth method: %v", err)
		return nil, fmt.Errorf("failed to get auth method: %w", err)
	}

	// Create SSH client configuration
	sshConfig := &ssh.ClientConfig{
		User: connConfig.Username,
		Auth: []ssh.AuthMethod{
			authMethod,
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Note: In production, consider using a more secure approach
		Timeout:         10 * time.Second,
	}

	// Connect to the SSH server
	addr := fmt.Sprintf("%s:%d", connConfig.Host, connConfig.Port)
	conn, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		log.Printf("Failed to connect to SSH server %s: %v", addr, err)
		return nil, fmt.Errorf("failed to connect to SSH server: %w", err)
	}

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
