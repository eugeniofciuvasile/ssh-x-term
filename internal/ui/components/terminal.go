package components

import (
	"fmt"
	"io"
	"log"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/eugeniofciuvasile/ssh-x-term/internal/config"
	"github.com/eugeniofciuvasile/ssh-x-term/internal/ssh"
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

// SSHOutputMsg contains output from the SSH session
type SSHOutputMsg struct {
	Data []byte
}

// SSHErrorMsg contains an error from the SSH session
type SSHErrorMsg struct {
	Err error
}

// SSHSessionMsg is a message containing an SSH session
type SSHSessionMsg struct {
	Session *ssh.BubbleTeaSession
	Error   error
}

// startSessionCmd starts an SSH session and returns a message with the result
func startSessionCmd(connConfig config.SSHConnection, width, height int) tea.Cmd {
	return func() tea.Msg {
		// Ensure we have valid dimensions
		if width <= 0 {
			width = 80
		}
		if height <= 0 {
			height = 24
		}

		// Calculate terminal dimensions (leaving room for header/footer)
		// Header and footer take 2 lines total
		termHeight := height - 2
		if termHeight < 10 {
			termHeight = 10
		}

		session, err := ssh.NewBubbleTeaSession(connConfig, width, termHeight)
		if err != nil {
			return SSHSessionMsg{nil, err}
		}
		return SSHSessionMsg{session, nil}
	}
}

// listenForOutput listens for output from the SSH session
func listenForOutput(session *ssh.BubbleTeaSession) tea.Cmd {
	return func() tea.Msg {
		buf := make([]byte, 32*1024)
		n, err := session.Read(buf)
		if err != nil {
			if err == io.EOF {
				return SSHErrorMsg{fmt.Errorf("session closed")}
			}
			return SSHErrorMsg{err}
		}
		if n > 0 {
			return SSHOutputMsg{Data: buf[:n]}
		}
		// If n == 0, we should still continue listening
		// Return a message that tells us to continue
		return SSHOutputMsg{Data: nil}
	}
}

// TerminalComponent represents a terminal component for SSH sessions
type TerminalComponent struct {
	connection    config.SSHConnection
	session       *ssh.BubbleTeaSession
	vterm         *VTerminal
	status        string
	error         error
	loading       bool
	width         int
	height        int
	finished      bool
	mutex         sync.Mutex
	sessionClosed bool
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
	return startSessionCmd(t.connection, t.width, t.height)
}

// Update handles component updates
func (t *TerminalComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		t.width = msg.Width
		t.height = msg.Height

		// Resize virtual terminal
		if t.vterm != nil {
			// Header and footer each take 1 line, plus we need 2 newlines between them
			// So total overhead is 2 lines (header + footer) and we keep content in between
			termHeight := t.height - 2
			if termHeight < 10 {
				termHeight = 10
			}
			t.vterm.Resize(t.width, termHeight)

			// Resize SSH session
			if t.session != nil {
				t.session.Resize(t.width, termHeight)
			}
		}
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

		// Create virtual terminal with proper dimensions
		// Ensure we have valid dimensions
		width := t.width
		if width <= 0 {
			width = 80 // Default width
		}
		// Header and footer take 2 lines total
		termHeight := t.height - 2
		if termHeight < 10 {
			termHeight = 10
		}

		log.Printf("Creating VTerminal with width=%d, height=%d (window: %dx%d)", width, termHeight, t.width, t.height)
		t.vterm = NewVTerminal(width, termHeight)

		// Start the session
		if err := t.session.Start(); err != nil {
			t.error = err
			t.status = fmt.Sprintf("Error: %s", err)
			return t, nil
		}

		// Start listening for output
		return t, listenForOutput(t.session)

	case SSHOutputMsg:
		if t.vterm != nil && msg.Data != nil && len(msg.Data) > 0 {
			// Write data to virtual terminal
			t.vterm.Write(msg.Data)

			// Log for debugging
			log.Printf("Received %d bytes from SSH, wrote to vterm", len(msg.Data))

			// Automatically scroll to bottom when new data arrives (unless manually scrolled)
			if !t.vterm.IsScrolledBack() {
				t.vterm.ScrollToBottom()
			}
		}

		// Continue listening (even if no data this time)
		if t.session != nil && !t.sessionClosed {
			return t, listenForOutput(t.session)
		}
		return t, nil

	case SSHErrorMsg:
		t.mutex.Lock()
		t.sessionClosed = true
		t.mutex.Unlock()

		if msg.Err != nil && msg.Err.Error() != "session closed" {
			t.error = msg.Err
			t.status = fmt.Sprintf("Error: %s", msg.Err)
		}
		return t, nil

	case tea.KeyMsg:
		// Handle escape to exit
		if msg.String() == "esc" {
			t.finished = true
			if t.session != nil {
				t.session.Close()
			}
			return t, nil
		}

		// Handle scrolling
		if t.vterm != nil {
			switch msg.String() {
			case "pgup", "shift+up":
				t.vterm.ScrollUp(10)
				return t, nil
			case "pgdown", "shift+down":
				t.vterm.ScrollDown(10)
				return t, nil
			case "ctrl+home":
				// Scroll to top
				t.vterm.ScrollUp(100000)
				return t, nil
			case "ctrl+end":
				// Scroll to bottom
				t.vterm.ScrollToBottom()
				return t, nil
			case "ctrl+c":
				// Copy selection to clipboard
				if t.vterm.HasSelection() {
					if err := t.vterm.CopySelection(); err != nil {
						log.Printf("Failed to copy: %v", err)
					}
					t.vterm.ClearSelection()
					return t, nil
				}
				// If no selection, send SIGINT to SSH session
				if t.session != nil {
					t.session.Write([]byte{3}) // Send Ctrl+C
				}
				return t, nil
			case "ctrl+shift+c":
				// Force copy
				if t.vterm.HasSelection() {
					if err := t.vterm.CopySelection(); err != nil {
						log.Printf("Failed to copy: %v", err)
					}
					t.vterm.ClearSelection()
				}
				return t, nil
			}
		}

		// Handle CTRL+D - send EOF
		if msg.String() == "ctrl+d" {
			if t.session != nil {
				// Send EOT (End of Transmission)
				t.session.Write([]byte{4})
			}
			return t, nil
		}

		// Forward all other key presses to SSH session
		if t.session != nil && !t.sessionClosed {
			data := []byte(msg.String())

			// Handle special keys
			switch msg.String() {
			case "enter":
				data = []byte{'\r'}
			case "backspace", "delete":
				data = []byte{127}
			case "tab":
				data = []byte{'\t'}
			case "up":
				data = []byte{27, '[', 'A'}
			case "down":
				data = []byte{27, '[', 'B'}
			case "right":
				data = []byte{27, '[', 'C'}
			case "left":
				data = []byte{27, '[', 'D'}
			case "home":
				data = []byte{27, '[', 'H'}
			case "end":
				data = []byte{27, '[', 'F'}
			}

			if _, err := t.session.Write(data); err != nil {
				log.Printf("Error writing to SSH session: %v", err)
			}
		}

		return t, nil

	case tea.MouseMsg:
		// Handle mouse wheel scrolling
		if t.vterm != nil {
			switch msg.Button {
			case tea.MouseButtonWheelUp:
				t.vterm.ScrollUp(3)
				return t, nil
			case tea.MouseButtonWheelDown:
				t.vterm.ScrollDown(3)
				return t, nil
			case tea.MouseButtonLeft:
				// Start selection on press
				if msg.Action == tea.MouseActionPress {
					// Adjust Y coordinate for header (subtract 1 for header line)
					adjustedY := msg.Y - 1
					if adjustedY >= 0 {
						t.vterm.StartSelection(msg.X, adjustedY)
					}
				} else if msg.Action == tea.MouseActionRelease {
					// Try to copy selection on release
					if t.vterm.HasSelection() {
						if err := t.vterm.CopySelection(); err != nil {
							log.Printf("Failed to copy selection: %v", err)
						}
					}
				}
				return t, nil
			case tea.MouseButtonNone:
				// Update selection while dragging
				if msg.Action == tea.MouseActionMotion {
					adjustedY := msg.Y - 1
					if adjustedY >= 0 {
						t.vterm.UpdateSelection(msg.X, adjustedY)
					}
				}
				return t, nil
			}
		}
		return t, nil
	}

	return t, nil
}

// View renders the component
func (t *TerminalComponent) View() string {
	if t.finished {
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

	// Render the terminal
	header := terminalHeaderStyle.Width(t.width).Render(
		fmt.Sprintf("SSH: %s@%s:%d - %s",
			t.connection.Username,
			t.connection.Host,
			t.connection.Port,
			t.connection.Name),
	)

	var scrollIndicator string
	if t.vterm != nil && t.vterm.IsScrolledBack() {
		scrollIndicator = " [SCROLL]"
	}

	// Create status text that adapts to terminal width
	var statusText string
	if t.sessionClosed {
		statusText = "Session closed - Press ESC to return"
	} else if t.width < 80 {
		// Short version for narrow terminals
		statusText = "ESC: Exit | CTRL+D: EOF" + scrollIndicator
	} else {
		// Full version
		statusText = "ESC: Exit | CTRL+D: EOF | PgUp/PgDn: Scroll | Mouse: Select & Copy" + scrollIndicator
	}

	footer := terminalFooterStyle.Width(t.width).Render(statusText)

	var content string
	if t.vterm != nil {
		content = t.vterm.Render()
		// Remove trailing newline if present to control spacing
		content = strings.TrimRight(content, "\n")
	} else {
		content = strings.Repeat("\n", max(t.height-2, 0))
	}

	return fmt.Sprintf("%s\n%s\n%s", header, content, footer)
}

// IsFinished returns whether the terminal session is finished
func (t *TerminalComponent) IsFinished() bool {
	return t.finished
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
