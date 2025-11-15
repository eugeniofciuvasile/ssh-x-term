package components

import (
	"fmt"
	"io"
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
				Align(lipgloss.Center).
				Padding(0, 1)

	terminalFooterStyle = lipgloss.NewStyle().
				Bold(true).
				Background(lipgloss.Color("8")).
				Foreground(lipgloss.Color("255")).
				Align(lipgloss.Center).
				Padding(0, 1)

	terminalErrorStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("9")).
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

// TerminalComponent represents a terminal component for SSH sessions
type TerminalComponent struct {
	connection     config.SSHConnection
	session        *ssh.BubbleTeaSession
	vterm          *VTerminal
	status         string
	error          error
	loading        bool
	width          int
	height         int
	finished       bool
	mutex          sync.Mutex
	sessionClosed  bool
	sessionStarted bool // Track if session has been initiated
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
	// Don't start session yet if we don't have dimensions
	// Wait for WindowSizeMsg to arrive first
	return nil
}

// Update handles component updates
func (t *TerminalComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		t.width = msg.Width
		t.height = msg.Height

		// If session not started yet, start it now with proper dimensions
		if !t.sessionStarted && !t.finished && t.error == nil {
			t.sessionStarted = true
			return t, t.startSession(t.connection, t.width, t.height)
		}

		// Otherwise just resize
		t.resizeTerminal()
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

		t.createAndStartVTerminal()
		return t, t.listenForSSHOutput()

	case SSHOutputMsg:
		if len(msg.Data) > 0 {
			t.writeToVTerminal(msg.Data)
		}
		return t, t.listenForSSHOutput() // Continue listening

	case SSHErrorMsg:
		t.handleSessionError(msg.Err)
		return t, nil

	case tea.KeyMsg:
		return t.handleKey(msg)

	case tea.MouseMsg:
		t.handleMouse(msg)
		return t, nil
	}

	return t, nil
}

// View renders the component.
func (t *TerminalComponent) View() string {
	if t.finished {
		return ""
	}

	if t.loading {
		return fmt.Sprintf("\nConnecting to %s@%s:%d...\n", t.connection.Username, t.connection.Host, t.connection.Port)
	}

	if t.error != nil {
		return fmt.Sprintf(
			"\nError connecting to %s@%s:%d\n\n%s\n",
			t.connection.Username, t.connection.Host, t.connection.Port,
			terminalErrorStyle.Render(t.error.Error()),
		)
	}

	// Build terminal header
	headerText := fmt.Sprintf(
		"SSH: %s@%s:%d - %s",
		t.connection.Username, t.connection.Host, t.connection.Port, t.connection.Name,
	)

	// Include scroll indicator if applicable
	if t.vterm != nil && t.vterm.IsScrolledBack() {
		headerText += " [SCROLL]"
	}

	header := terminalHeaderStyle.Width(t.width).Render(headerText)

	// Get terminal content
	content := ""
	if t.vterm != nil {
		content = t.vterm.Render()
	}

	// Combine header and content (footer is now handled by main view)
	return lipgloss.JoinVertical(lipgloss.Left, header, content)
}

// Utility: Calculate content height
func (t *TerminalComponent) contentHeight() int {
	// Terminal now receives the content area size directly from the main view
	// We only need to subtract the terminal's header (1 line)
	// Footer is now handled by the main view
	return t.height - 1
}

// Utility: Resize terminal components dynamically
func (t *TerminalComponent) resizeTerminal() {
	contentHeight := t.contentHeight()
	if t.vterm != nil {
		t.vterm.Resize(t.width, contentHeight)
	}
	if t.session != nil {
		t.session.Resize(t.width, contentHeight)
	}
}

// Utility: Handle session start
func (t *TerminalComponent) startSession(conn config.SSHConnection, width, height int) tea.Cmd {
	return func() tea.Msg {
		if width <= 0 {
			width = 80
		}
		if height <= 0 {
			height = 24
		}
		// Only subtract 1 for the terminal header (footer is now in main view)
		session, err := ssh.NewBubbleTeaSession(conn, width, height-1)
		if err != nil {
			return SSHSessionMsg{nil, err}
		}
		return SSHSessionMsg{session, nil}
	}
}

// Utility: Create and start the virtual terminal
func (t *TerminalComponent) createAndStartVTerminal() {
	t.vterm = NewVTerminal(t.width, t.contentHeight())
	if err := t.session.Start(); err != nil {
		t.error = err
		t.status = fmt.Sprintf("Error: %s", err)
	}
}

// Utility: Continuously listen for SSH output
func (t *TerminalComponent) listenForSSHOutput() tea.Cmd {
	return func() tea.Msg {
		buf := make([]byte, 32*1024)
		n, err := t.session.Read(buf)
		if err != nil {
			if err == io.EOF {
				return SSHErrorMsg{fmt.Errorf("session closed")}
			}
			return SSHErrorMsg{err}
		}
		return SSHOutputMsg{Data: buf[:n]}
	}
}

// Utility: Write data to virtual terminal
func (t *TerminalComponent) writeToVTerminal(data []byte) {
	t.vterm.Write(data)
	if !t.vterm.IsScrolledBack() {
		t.vterm.ScrollToBottom()
	}
}

// Utility: Render the footer with helper instructions
func (t *TerminalComponent) renderFooter() string {
	if t.sessionClosed {
		return "Session closed - Press ESC to return"
	}

	if t.width < 80 {
		return "ESC: Exit | CTRL+C: Interrupt | CTRL+D: EOF"
	}

	return "ESC: Exit | CTRL+C: Interrupt | CTRL+D: EOF | PgUp/PgDn: Scroll | Mouse: Select/Copy"
}

// Utility: Handle session errors
func (t *TerminalComponent) handleSessionError(err error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.sessionClosed = true
	if err.Error() != "session closed" {
		t.error = err
		t.status = fmt.Sprintf("Error: %s", err)
	}
}

// Utility: Handle key input
func (t *TerminalComponent) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		t.finished = true
		if t.session != nil {
			t.session.Close()
		}
		return t, nil

	case "pgup", "shift+up":
		t.vterm.ScrollUp(10) // Handle scrolling up
		return t, nil

	case "pgdown", "shift+down":
		t.vterm.ScrollDown(10) // Handle scrolling down
		return t, nil

	default:
		t.forwardKeyToSession(msg.String())
	}

	return t, nil
}

// Utility: Forward keys to SSH session
func (t *TerminalComponent) forwardKeyToSession(key string) {
	var data []byte
	switch key {
	// Basic keys
	case "tab":
		data = []byte{'\t'}
	case "enter":
		data = []byte{'\r'}
	case "backspace", "delete":
		data = []byte{127}

	// Arrow keys (ANSI escape sequences)
	case "up":
		data = []byte{27, '[', 'A'}
	case "down":
		data = []byte{27, '[', 'B'}
	case "right":
		data = []byte{27, '[', 'C'}
	case "left":
		data = []byte{27, '[', 'D'}

	// Home/End keys
	case "home":
		data = []byte{27, '[', 'H'}
	case "end":
		data = []byte{27, '[', 'F'}

	// Page Up/Page Down (when not used for scrolling)
	case "pgup":
		data = []byte{27, '[', '5', '~'}
	case "pgdown":
		data = []byte{27, '[', '6', '~'}

	// Function keys
	case "f1":
		data = []byte{27, 'O', 'P'}
	case "f2":
		data = []byte{27, 'O', 'Q'}
	case "f3":
		data = []byte{27, 'O', 'R'}
	case "f4":
		data = []byte{27, 'O', 'S'}
	case "f5":
		data = []byte{27, '[', '1', '5', '~'}
	case "f6":
		data = []byte{27, '[', '1', '7', '~'}
	case "f7":
		data = []byte{27, '[', '1', '8', '~'}
	case "f8":
		data = []byte{27, '[', '1', '9', '~'}
	case "f9":
		data = []byte{27, '[', '2', '0', '~'}
	case "f10":
		data = []byte{27, '[', '2', '1', '~'}
	case "f11":
		data = []byte{27, '[', '2', '3', '~'}
	case "f12":
		data = []byte{27, '[', '2', '4', '~'}

	// Control keys - the most important additions for proper terminal interaction
	case "ctrl+@", "ctrl+space":
		data = []byte{0x00} // NUL
	case "ctrl+a":
		data = []byte{0x01} // Start of line (common in shells)
	case "ctrl+b":
		data = []byte{0x02} // Move back one character
	case "ctrl+c":
		data = []byte{0x03} // SIGINT (interrupt signal)
	case "ctrl+d":
		data = []byte{0x04} // EOF (end of file / logout)
	case "ctrl+e":
		data = []byte{0x05} // End of line
	case "ctrl+f":
		data = []byte{0x06} // Move forward one character
	case "ctrl+g":
		data = []byte{0x07} // Bell
	case "ctrl+h":
		data = []byte{0x08} // Backspace
	case "ctrl+i":
		data = []byte{0x09} // Tab
	case "ctrl+j":
		data = []byte{0x0A} // Line feed
	case "ctrl+k":
		data = []byte{0x0B} // Kill line from cursor
	case "ctrl+l":
		data = []byte{0x0C} // Clear screen
	case "ctrl+m":
		data = []byte{0x0D} // Carriage return
	case "ctrl+n":
		data = []byte{0x0E} // Next line in history
	case "ctrl+o":
		data = []byte{0x0F} // Execute command
	case "ctrl+p":
		data = []byte{0x10} // Previous line in history
	case "ctrl+q":
		data = []byte{0x11} // XON (resume transmission)
	case "ctrl+r":
		data = []byte{0x12} // Reverse search
	case "ctrl+s":
		data = []byte{0x13} // XOFF (stop transmission) or forward search
	case "ctrl+t":
		data = []byte{0x14} // Transpose characters
	case "ctrl+u":
		data = []byte{0x15} // Kill line before cursor
	case "ctrl+v":
		data = []byte{0x16} // Literal next character
	case "ctrl+w":
		data = []byte{0x17} // Delete word backwards
	case "ctrl+x":
		data = []byte{0x18} // Various editor commands
	case "ctrl+y":
		data = []byte{0x19} // Yank (paste)
	case "ctrl+z":
		data = []byte{0x1A} // SIGTSTP (suspend)
	case "ctrl+[":
		data = []byte{0x1B} // ESC
	case "ctrl+\\":
		data = []byte{0x1C} // SIGQUIT (quit with core dump)
	case "ctrl+]":
		data = []byte{0x1D} // Telnet escape character
	case "ctrl+^", "ctrl+shift+6":
		data = []byte{0x1E} // Record separator
	case "ctrl+_", "ctrl+/":
		data = []byte{0x1F} // Undo

	// Alt/Meta key combinations (send ESC prefix)
	case "alt+b", "meta+b":
		data = []byte{27, 'b'} // Back one word
	case "alt+f", "meta+f":
		data = []byte{27, 'f'} // Forward one word
	case "alt+d", "meta+d":
		data = []byte{27, 'd'} // Delete word forward
	case "alt+backspace", "meta+backspace":
		data = []byte{27, 0x7F} // Delete word backward

	default:
		// For regular characters, just send them as-is
		data = []byte(key)
	}
	if t.session != nil {
		t.session.Write(data)
	}
}

// Utility: Handle mouse events
func (t *TerminalComponent) handleMouse(msg tea.MouseMsg) {
	if msg.Button == tea.MouseButtonWheelUp {
		t.vterm.ScrollUp(3)
	}
	if msg.Button == tea.MouseButtonWheelDown {
		t.vterm.ScrollDown(3)
	}
}

// IsFinished returns whether the terminal session is finished
func (t *TerminalComponent) IsFinished() bool {
	return t.finished
}

// IsSessionClosed returns whether the SSH session is closed
func (t *TerminalComponent) IsSessionClosed() bool {
	return t.sessionClosed
}

// IsScrolledBack returns whether the terminal is scrolled back in history
func (t *TerminalComponent) IsScrolledBack() bool {
	if t.vterm != nil {
		return t.vterm.IsScrolledBack()
	}
	return false
}

// GetWidth returns the terminal width
func (t *TerminalComponent) GetWidth() int {
	return t.width
}
