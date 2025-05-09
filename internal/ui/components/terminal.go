package components

import (
	"fmt"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"ssh-x-term/internal/config"
	"ssh-x-term/internal/ssh"
)

var (
	terminalHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Background(lipgloss.Color("4")).
				Foreground(lipgloss.Color("255")).
				Width(100).
				Align(lipgloss.Center).
				Padding(0, 1)

	terminalFooterStyle = lipgloss.NewStyle().
				Bold(true).
				Background(lipgloss.Color("8")).
				Foreground(lipgloss.Color("255")).
				Width(100).
				Align(lipgloss.Center).
				Padding(0, 1)

	terminalErrorStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("9")).
				Width(100).
				Align(lipgloss.Center).
				Padding(1, 0)
)

// SSHSessionMsg is a message containing an SSH session
type SSHSessionMsg struct {
	Session *ssh.Session
	Error   error
}

// startSessionCmd starts an SSH session and returns a message with the result
func startSessionCmd(connConfig config.SSHConnection) tea.Cmd {
	return func() tea.Msg {
		session, err := ssh.NewSession(connConfig)
		if err != nil {
			return SSHSessionMsg{nil, err}
		}
		return SSHSessionMsg{session, nil}
	}
}

// TerminalComponent represents a terminal component for SSH sessions
type TerminalComponent struct {
	connection    config.SSHConnection
	session       *ssh.Session
	status        string
	error         error
	loading       bool
	width         int
	height        int
	escapePressed bool
	mutex         sync.Mutex
}

// NewTerminalComponent creates a new terminal component
func NewTerminalComponent(conn config.SSHConnection) *TerminalComponent {
	return &TerminalComponent{
		connection: conn,
		status:     "Connecting...",
		loading:    true,
	}
}

// Init initializes the component
func (t *TerminalComponent) Init() tea.Cmd {
	return startSessionCmd(t.connection)
}

// Update handles component updates
func (t *TerminalComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		t.width = msg.Width
		t.height = msg.Height
		return t, nil

	case SSHSessionMsg:
		t.loading = false
		if msg.Error != nil {
			t.error = msg.Error
			t.status = fmt.Sprintf("Error: %s", msg.Error)
			return t, nil
		}
		t.session = msg.Session
		t.status = "Connected"

		// Start the session in a goroutine
		go func() {
			err := t.session.Start()
			if err != nil {
				t.mutex.Lock()
				t.error = err
				t.status = fmt.Sprintf("Error: %s", err)
				t.mutex.Unlock()
			}
		}()

		return t, nil

	case tea.KeyMsg:
		if msg.String() == "esc" {
			t.escapePressed = true
			// Close the session
			if t.session != nil {
				t.session.Close()
			}
			return t, nil
		}
	}

	return t, nil
}

// View renders the component
func (t *TerminalComponent) View() string {
	if t.escapePressed {
		return ""
	}

	if t.loading {
		return centeredBox(
			fmt.Sprintf("Connecting to %s@%s:%d...",
				t.connection.Username,
				t.connection.Host,
				t.connection.Port),
			t.width,
			t.height,
		)
	}

	if t.error != nil {
		return centeredBox(
			fmt.Sprintf("Error connecting to %s@%s:%d\n\n%s\n\nPress ESC to return",
				t.connection.Username,
				t.connection.Host,
				t.connection.Port,
				terminalErrorStyle.Render(t.error.Error())),
			t.width,
			t.height,
		)
	}

	// When connected, we don't actually render anything here
	// The SSH session takes over the terminal
	header := terminalHeaderStyle.Width(t.width).Render(
		fmt.Sprintf("Connected to %s@%s:%d",
			t.connection.Username,
			t.connection.Host,
			t.connection.Port),
	)

	footer := terminalFooterStyle.Width(t.width).Render(
		"Press ESC to disconnect",
	)

	content := strings.Repeat("\n", max(t.height-4, 0))

	return fmt.Sprintf("%s\n%s\n%s", header, content, footer)
}

// IsFinished returns whether the terminal session is finished
func (t *TerminalComponent) IsFinished() bool {
	return t.escapePressed
}

// centeredBox creates a centered box with the given content
func centeredBox(content string, width, height int) string {
	lines := strings.Split(content, "\n")

	// Calculate vertical padding
	contentHeight := len(lines)
	topPadding := (height - contentHeight) / 2
	if topPadding < 0 {
		topPadding = 0
	}

	// Add padding
	var result strings.Builder
	result.WriteString(strings.Repeat("\n", topPadding))

	// Add each line centered
	for _, line := range lines {
		// Calculate horizontal padding for this line
		lineLength := len(line)
		leftPadding := (width - lineLength) / 2
		if leftPadding < 0 {
			leftPadding = 0
		}

		result.WriteString(strings.Repeat(" ", leftPadding))
		result.WriteString(line)
		result.WriteString("\n")
	}

	return result.String()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
