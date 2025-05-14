package main

import (
	"fmt"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
	"ssh-x-term/internal/config"
	"ssh-x-term/internal/ui"
)

func main() {
	if os.Getenv("TMUX") == "" {
		fmt.Println("Not in tmux session. Attempting to launch inside tmux...")

		execPath, err := os.Executable()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting executable path: %v\n", err)
			os.Exit(1)
		}

		args := os.Args[1:]
		tmuxArgs := append([]string{"new-session", execPath}, args...)
		cmd := exec.Command("tmux", tmuxArgs...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to start tmux session: %v\nFalling back to normal execution...\n", err)
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
	configManager, err := config.NewConfigManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing config: %v\n", err)
		os.Exit(1)
	}

	// Load configuration
	if err := configManager.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Create UI model
	model := ui.NewModel(configManager)

	// Initialize the Bubble Tea program
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	// Run the program
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}
