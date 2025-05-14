package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"ssh-x-term/internal/config"
	"ssh-x-term/internal/ui/components"

	tea "github.com/charmbracelet/bubbletea"
)

// AppState represents the current state of the application
type AppState int

const (
	StateConnectionList AppState = iota
	StateAddConnection
	StateEditConnection
	StateSSHTerminal
)

// Model represents the UI model for the application
type Model struct {
	configManager  *config.ConfigManager
	state          AppState
	width          int
	height         int
	connectionList *components.ConnectionList
	connectionForm *components.ConnectionForm
	terminal       *components.TerminalComponent
	errorMessage   string
}

// NewModel creates a new UI model
func NewModel(configManager *config.ConfigManager) *Model {
	connectionList := components.NewConnectionList(configManager.Config.Connections)

	return &Model{
		configManager:  configManager,
		state:          StateConnectionList,
		connectionList: connectionList,
	}
}

// Init initializes the model
func (m *Model) Init() tea.Cmd {
	return nil
}

// LoadConnections loads connections from config and updates the connection list
func (m *Model) LoadConnections() {
	m.connectionList.SetConnections(m.configManager.Config.Connections)
}

// getActiveComponent returns the currently active component
func (m *Model) getActiveComponent() tea.Model {
	switch m.state {
	case StateConnectionList:
		return m.connectionList
	case StateAddConnection, StateEditConnection:
		return m.connectionForm
	case StateSSHTerminal:
		return m.terminal
	default:
		return nil
	}
}

// handleComponentResult handles the result of a component's update function
func (m *Model) handleComponentResult(model tea.Model, cmd tea.Cmd) tea.Cmd {
	switch m.state {
	case StateConnectionList:
		m.connectionList = model.(*components.ConnectionList)
		if conn := m.connectionList.SelectedConnection(); conn != nil {
			if !m.connectionList.OpenInNewTerminal() {
				/*
					SSH session in the same terminal
					A connection was selected, start SSH terminal
				*/
				m.terminal = components.NewTerminalComponent(*conn)
				m.state = StateSSHTerminal
				m.connectionList.Reset()
				return m.terminal.Init()
			} else {
				/*
				   SSH session in another tmux window
				   Build SSH command
				*/
				sshArgs := []string{}
				if conn.KeyFile != "" {
					sshArgs = append(sshArgs, "-i", filepath.Clean(conn.KeyFile))
				}
				if conn.Port != 22 && conn.Port != 0 {
					sshArgs = append(sshArgs, "-p", strconv.Itoa(conn.Port))
				}
				userHost := fmt.Sprintf("%s@%s", conn.Username, conn.Host)
				sshArgs = append(sshArgs, userHost)

				// Build full SSH command
				sshCommand := fmt.Sprintf("ssh %s", strings.Join(sshArgs, " "))

				// Sanitize and build tmux window name: "user@host:port - ConnectionName"
				windowName := fmt.Sprintf("%s@%s:%d - %s", conn.Username, conn.Host, conn.Port, conn.Name)

				// Launch tmux new window with name and SSH command
				cmd := exec.Command("tmux", "new-window", "-n", windowName, sshCommand)
				err := cmd.Start()
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error launching tmux window: %v\n", err)
				}

				// Optional: Warn if password use is enabled
				if conn.UsePassword && conn.Password != "" {
					tea.Println("Password authentication not supported in this mode. Use manual entry or key-based login.")
				}

				// Remain on the connection list view
				m.state = StateConnectionList
				m.connectionList.Reset()
				return nil
			}
		}

	case StateAddConnection, StateEditConnection:
		m.connectionForm = model.(*components.ConnectionForm)
		if m.connectionForm.IsCanceled() {
			// Form was canceled, return to connection list
			m.state = StateConnectionList
			m.connectionList.Reset()
			return nil
		}
		if m.connectionForm.IsSubmitted() {
			// Form was submitted, save connection
			conn := m.connectionForm.Connection()
			err := m.configManager.AddConnection(conn)
			if err != nil {
				m.errorMessage = fmt.Sprintf("Failed to save connection: %s", err)
			} else {
				// Reload connections and return to list
				m.LoadConnections()
				m.state = StateConnectionList
			}
			m.connectionList.Reset()
			return nil
		}

	case StateSSHTerminal:
		m.terminal = model.(*components.TerminalComponent)
		if m.terminal.IsFinished() {
			// Terminal session ended, return to connection list
			m.state = StateConnectionList
			m.connectionList.Reset()
			return nil
		}
	}

	return cmd
}
