package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"ssh-x-term/internal/config"
	"ssh-x-term/internal/ui"
)

func main() {
	// Initialize config manager
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
