package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/eugeniofciuvasile/ssh-x-term/internal/cli"
	"github.com/eugeniofciuvasile/ssh-x-term/internal/config"
	"github.com/eugeniofciuvasile/ssh-x-term/internal/ui"
)

func main() {
	// Parse command-line flags
	listFlag := flag.Bool("l", false, "List and select from saved SSH connections")
	initFlag := flag.Bool("i", false, "Initialize SSH config and perform first-time migration")
	connectFlag := flag.String("c", "", "Connect directly to a saved connection by ID using golang SSH client")
	flag.Parse()

	logfilePath := os.Getenv("SSH_X_TERM_LOG")
	if logfilePath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("Unable to get user home directory: %v", err)
		}
		configDir := filepath.Join(homeDir, ".config", "ssh-x-term")
		if err := os.MkdirAll(configDir, 0700); err != nil {
			log.Fatalf("Unable to create config directory: %v", err)
		}
		logfilePath = filepath.Join(configDir, "sxt.log")
	}

	logCloser, err := tea.LogToFile(logfilePath, "")
	if err != nil {
		log.Fatalf("Unable to set Bubble Tea log file: %v", err)
	}
	defer func() {
		if logCloser != nil {
			logCloser.Close()
		}
	}()

	// Handle -i flag for initialization
	if *initFlag {
		runInitialization()
		return
	}

	// Handle -c flag for direct connection by ID
	if *connectFlag != "" {
		// Check if initialization has been done
		if !isInitialized() {
			fmt.Fprintln(os.Stderr, "Error: SSH-X-Term not initialized.")
			fmt.Fprintln(os.Stderr, "Please run 'sxt -i' first to initialize and migrate your configuration.")
			os.Exit(1)
		}
		runDirectConnect(*connectFlag)
		return
	}

	// Handle -l flag for quick connection selection
	if *listFlag {
		// Check if initialization has been done
		if !isInitialized() {
			fmt.Fprintln(os.Stderr, "Error: SSH-X-Term not initialized.")
			fmt.Fprintln(os.Stderr, "Please run 'sxt -i' first to initialize and migrate your configuration.")
			os.Exit(1)
		}
		runQuickConnect()
		return
	}

	if os.Getenv("TMUX") == "" {
		log.Println("Not in tmux session. Attempting to launch inside tmux...")

		execPath, err := os.Executable()
		if err != nil {
			log.Printf("Error getting executable path: %v\n", err)
			os.Exit(1)
		}

		args := os.Args[1:]
		tmuxArgs := append([]string{"new-session", execPath}, args...)
		cmd := exec.Command("tmux", tmuxArgs...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		if err := cmd.Run(); err != nil {
			log.Printf("Failed to start tmux session: %v\nFalling back to normal execution...\n", err)
			config.IsTmuxAvailable = false
			runApp()
			return
		}
		return
	}

	config.IsTmuxAvailable = true
	runApp()
}

func runApp() {
	// Check and migrate from old JSON config if needed
	if err := config.CheckAndMigrate(); err != nil {
		log.Printf("Warning: migration failed: %v\n", err)
		// Continue anyway - user can manually migrate
	}

	// Create UI model
	model := ui.NewModel()

	// Initialize the Bubble Tea program
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	// Run the program
	if _, err := p.Run(); err != nil {
		log.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}

func runQuickConnect() {
	// Load SSH config directly
	sshConfigManager, err := config.NewSSHConfigManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading SSH config: %v\n", err)
		os.Exit(1)
	}

	if err := sshConfigManager.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading connections: %v\n", err)
		os.Exit(1)
	}

	connections := sshConfigManager.ListConnections()
	if len(connections) == 0 {
		fmt.Println("No saved connections found.")
		os.Exit(0)
	}

	// Show connection selector
	selector := cli.NewSelector(connections)
	p := tea.NewProgram(selector)

	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running selector: %v\n", err)
		os.Exit(1)
	}

	// Get the selected connection
	selectorModel := finalModel.(*cli.SelectorModel)
	choice := selectorModel.Choice()

	if choice == nil {
		// User canceled
		fmt.Println("Connection canceled.")
		os.Exit(0)
	}

	// Connect directly using native SSH client
	fmt.Printf("Connecting to %s...\n", choice.Name)
	if err := cli.ConnectDirect(*choice); err != nil {
		fmt.Fprintf(os.Stderr, "Connection failed: %v\n", err)
		os.Exit(1)
	}
}

func isInitialized() bool {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	migrationMarkerPath := filepath.Join(homeDir, ".config", "ssh-x-term", ".migration_done")
	_, err = os.Stat(migrationMarkerPath)
	return err == nil
}

func runInitialization() {
	fmt.Println("Initializing SSH-X-Term...")
	fmt.Println()

	// Check and perform migration
	if err := config.CheckAndMigrate(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: migration check failed: %v\n", err)
	}

	// Load SSH config manager (this will trigger first-time migration)
	sshConfigManager, err := config.NewSSHConfigManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing SSH config: %v\n", err)
		os.Exit(1)
	}

	if err := sshConfigManager.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading SSH config: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("✓ Initialization complete!")
	fmt.Println()

	connections := sshConfigManager.ListConnections()
	if len(connections) > 0 {
		fmt.Printf("✓ Found %d connection(s)\n", len(connections))
	} else {
		fmt.Println("No connections found. Use 'sxt' to add connections via the full TUI.")
	}

	fmt.Println()
	fmt.Println("You can now use:")
	fmt.Println("  • sxt         - Launch full TUI")
	fmt.Println("  • sxt -l      - Quick connect mode")
	fmt.Println("  • sxt -c <id> - Direct connect by ID")
}

func runDirectConnect(connectionID string) {
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

	// Connect using golang SSH client (cli.ConnectDirect uses ssh.ConnectInteractive)
	fmt.Printf("Connecting to %s...\n", conn.Name)
	if err := cli.ConnectDirect(conn); err != nil {
		fmt.Fprintf(os.Stderr, "Connection failed: %v\n", err)
		os.Exit(1)
	}
}
