//go:build windows

package ssh

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"github.com/eugeniofciuvasile/ssh-x-term/internal/config"
	"github.com/zalando/go-keyring"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

// ConnectInteractive creates an interactive SSH session using golang SSH client
// This replaces the need for external tools like passh, plink, or native ssh command
func ConnectInteractive(connConfig config.SSHConnection) error {
	log.Printf("[ConnectInteractive] Starting interactive session for %s@%s:%d",
		connConfig.Username, connConfig.Host, connConfig.Port)

	// Create SSH client (this handles keyring password retrieval and SSH agent)
	client, err := NewClient(connConfig)
	if err != nil {
		// Check if passphrase is required for the key
		var passphraseErr *PassphraseRequiredError
		if errors.As(err, &passphraseErr) {
			// Key requires passphrase - prompt user
			fmt.Fprintf(os.Stderr, "Key file %s requires a passphrase.\n", passphraseErr.KeyFile)
			fmt.Fprintf(os.Stderr, "Enter passphrase: ")

			passphrase, err := term.ReadPassword(int(os.Stdin.Fd()))
			fmt.Fprintf(os.Stderr, "\n")
			if err != nil {
				return fmt.Errorf("failed to read passphrase: %w", err)
			}

			// Retry connection with passphrase
			connConfig.Password = string(passphrase)
			client, err = NewClient(connConfig)
			if err != nil {
				return fmt.Errorf("failed to create SSH client: %w", err)
			}

			// Save passphrase to keyring for future use
			savePassphraseToKeyring(connConfig.ID, string(passphrase))
		} else {
			return fmt.Errorf("failed to create SSH client: %w", err)
		}
	}
	defer client.Close()

	// Create SSH session
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	// Get current terminal file descriptor
	fd := int(os.Stdin.Fd())

	// Check if stdin is a terminal
	if !term.IsTerminal(fd) {
		return fmt.Errorf("stdin is not a terminal")
	}

	// Get current terminal state
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return fmt.Errorf("failed to set terminal to raw mode: %w", err)
	}
	defer term.Restore(fd, oldState)

	// Get terminal size
	width, height, err := term.GetSize(fd)
	if err != nil {
		log.Printf("[ConnectInteractive] Failed to get terminal size: %v, using defaults", err)
		width, height = 80, 24
	}

	// Set up terminal modes
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
		ssh.ICANON:        0,
		ssh.ISIG:          1,
	}

	// Request PTY with xterm-256color
	if err := session.RequestPty("xterm-256color", height, width, modes); err != nil {
		return fmt.Errorf("failed to request PTY: %w", err)
	}

	// Set up pipes
	stdin, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to setup stdin pipe: %w", err)
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to setup stdout pipe: %w", err)
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to setup stderr pipe: %w", err)
	}

	// Note: Windows doesn't support SIGWINCH, so window resize is not handled

	// Start shell
	if err := session.Shell(); err != nil {
		return fmt.Errorf("failed to start shell: %w", err)
	}

	// Copy I/O
	var wg sync.WaitGroup
	wg.Add(2)

	// Copy stdin to remote
	go func() {
		defer wg.Done()
		io.Copy(stdin, os.Stdin)
	}()

	// Copy remote output to stdout/stderr
	go func() {
		defer wg.Done()
		io.Copy(os.Stdout, stdout)
	}()

	// Copy stderr
	go func() {
		io.Copy(os.Stderr, stderr)
	}()

	// Wait for session to complete
	if err := session.Wait(); err != nil {
		log.Printf("[ConnectInteractive] Session ended with error: %v", err)
	}

	// Wait for I/O to complete
	wg.Wait()

	log.Printf("[ConnectInteractive] Session closed")
	return nil
}

// savePassphraseToKeyring saves the passphrase to the system keyring
func savePassphraseToKeyring(connectionID, passphrase string) {
	keyID := "passphrase:" + connectionID
	err := keyring.Set("ssh-x-term", keyID, passphrase)
	if err != nil {
		log.Printf("[savePassphraseToKeyring] Failed to save passphrase to keyring: %v", err)
		fmt.Fprintf(os.Stderr, "Note: Could not save passphrase to keyring (will be prompted again next time)\n")
	} else {
		log.Printf("[savePassphraseToKeyring] Saved passphrase for connection %s to keyring", connectionID)
		fmt.Fprintf(os.Stderr, "Passphrase saved to keyring.\n")
	}
}
