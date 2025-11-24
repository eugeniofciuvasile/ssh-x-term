package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/eugeniofciuvasile/ssh-x-term/internal/config"
	"github.com/eugeniofciuvasile/ssh-x-term/internal/ui"
)

func main() {
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
