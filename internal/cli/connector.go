package cli

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/eugeniofciuvasile/ssh-x-term/internal/config"
	"github.com/zalando/go-keyring"
)

const keyringService = "ssh-x-term"

// ConnectDirect opens a direct SSH connection using the native SSH client
func ConnectDirect(conn config.SSHConnection) error {
	log.Printf("Direct SSH connection to %s@%s:%d", conn.Username, conn.Host, conn.Port)

	// Prepare SSH arguments
	args := []string{}

	// Add port if not default
	if conn.Port != 22 {
		args = append(args, "-p", strconv.Itoa(conn.Port))
	}

	// Handle key file authentication
	if !conn.UsePassword && conn.KeyFile != "" {
		keyFile := conn.KeyFile
		if keyFile[0] == '~' {
			homeDir, err := os.UserHomeDir()
			if err == nil {
				keyFile = filepath.Join(homeDir, keyFile[2:])
			}
		}
		args = append(args, "-i", keyFile)
	}

	// Handle password authentication
	var password string
	if conn.UsePassword {
		// Try to get password from keyring first
		if conn.Password == "" {
			pw, err := keyring.Get(keyringService, conn.ID)
			if err != nil {
				log.Printf("Failed to retrieve password from keyring: %v", err)
				return fmt.Errorf("password required but not found in keyring")
			}
			password = pw
		} else {
			password = conn.Password
		}
	}

	userHost := fmt.Sprintf("%s@%s", conn.Username, conn.Host)

	// Determine which SSH command to use based on OS and authentication method
	var cmd *exec.Cmd

	if runtime.GOOS == "windows" {
		// On Windows, use plink for password authentication
		if conn.UsePassword && password != "" {
			plinkArgs := []string{"-ssh"}
			if conn.Port != 22 {
				plinkArgs = append(plinkArgs, "-P", strconv.Itoa(conn.Port))
			}
			plinkArgs = append(plinkArgs, "-pw", password, userHost)
			cmd = exec.Command("plink.exe", plinkArgs...)
		} else {
			// Use standard ssh for key auth
			args = append(args, userHost)
			cmd = exec.Command("ssh", args...)
		}
	} else {
		// On Unix, use passh for password authentication
		if conn.UsePassword && password != "" {
			passhArgs := []string{"-p", password, "ssh"}
			// Add SSH options after "ssh"
			if conn.Port != 22 {
				passhArgs = append(passhArgs, "-p", strconv.Itoa(conn.Port))
			}
			// Add key file if specified
			if conn.KeyFile != "" {
				keyFile := conn.KeyFile
				if keyFile[0] == '~' {
					homeDir, _ := os.UserHomeDir()
					keyFile = filepath.Join(homeDir, keyFile[2:])
				}
				passhArgs = append(passhArgs, "-i", keyFile)
			}
			passhArgs = append(passhArgs, userHost)
			cmd = exec.Command("passh", passhArgs...)
		} else {
			// Use standard ssh for key auth
			args = append(args, userHost)
			cmd = exec.Command("ssh", args...)
		}
	}

	// Connect stdin/stdout/stderr to current terminal
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	// Set TERM to a widely compatible value to avoid terminal type issues
	cmd.Env = os.Environ()
	termSet := false
	for i, env := range cmd.Env {
		if len(env) >= 5 && env[:5] == "TERM=" {
			cmd.Env[i] = "TERM=xterm-256color"
			termSet = true
			break
		}
	}
	if !termSet {
		cmd.Env = append(cmd.Env, "TERM=xterm-256color")
	}

	log.Printf("Executing: %s %v", cmd.Path, cmd.Args)

	// Run the command
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("SSH connection failed: %w", err)
	}

	return nil
}

func RunDirectConnect(connectionID string) {
	// Load SSH config
	sshConfigManager, err := config.NewSSHConfigManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading SSH config: %v\n", err)
		os.Exit(1)
	}

	if err := sshConfigManager.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading connections: %v\n", err)
		os.Exit(1)
	}

	// Get the connection by ID
	conn, found := sshConfigManager.GetConnection(connectionID)
	if !found {
		fmt.Fprintf(os.Stderr, "Error: Connection with ID '%s' not found.\n", connectionID)
		fmt.Fprintln(os.Stderr, "\nAvailable connections:")

		connections := sshConfigManager.ListConnections()
		if len(connections) == 0 {
			fmt.Fprintln(os.Stderr, "  (none)")
		} else {
			for _, c := range connections {
				fmt.Fprintf(os.Stderr, "  • %s (%s) - %s@%s:%d\n", c.Name, c.ID, c.Username, c.Host, c.Port)
			}
		}
		os.Exit(1)
	}

	// Connect directly using the same method as sxt -l
	fmt.Printf("Connecting to %s...\n", conn.Name)
	if err := ConnectDirect(conn); err != nil {
		fmt.Fprintf(os.Stderr, "Connection failed: %v\n", err)
		os.Exit(1)
	}
}
