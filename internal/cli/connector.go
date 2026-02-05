package cli

import (
	"fmt"
	"log"

	"github.com/eugeniofciuvasile/ssh-x-term/internal/config"
	"github.com/eugeniofciuvasile/ssh-x-term/internal/ssh"
)

const keyringService = "ssh-x-term"

// ConnectDirect opens a direct SSH connection using the golang SSH client
// This replaces external tools like passh, plink, and native ssh command
func ConnectDirect(conn config.SSHConnection) error {
	log.Printf("Direct SSH connection to %s@%s:%d", conn.Username, conn.Host, conn.Port)

	// Use golang SSH client for interactive session
	if err := ssh.ConnectInteractive(conn); err != nil {
		return fmt.Errorf("SSH connection failed: %w", err)
	}

	return nil
}
