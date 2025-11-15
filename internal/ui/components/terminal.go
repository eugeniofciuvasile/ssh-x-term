package components

import (
	"fmt"
	"io"
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
	return t.startSession(t.connection, t.width, t.height)
}

// Update handles component updates
func (t *TerminalComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		t.width = msg.Width
		t.height = msg.Height
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
		return t.centeredBox(
			fmt.Sprintf("Connecting to %s@%s:%d...",
				t.connection.Username,
				t.connection.Host,
				t.connection.Port),
		)
	}

	if t.error != nil {
		return t.centeredBox(fmt.Sprintf(
			"Error connecting to %s@%s:%d\n\n%s\n\nPress ESC to return",
			t.connection.Username, t.connection.Host, t.connection.Port,
			terminalErrorStyle.Render(t.error.Error()),
		))
	}

	header := terminalHeaderStyle.Width(t.width).Render(fmt.Sprintf(
		"SSH: %s@%s:%d - %s",
		t.connection.Username, t.connection.Host, t.connection.Port, t.connection.Name,
	))

	content := ""
	if t.vterm != nil {
		content = t.vterm.Render()
	} else {
		content = strings.Repeat("\n", t.contentHeight())
	}

	// Include scroll indicator if applicable
	var scrollIndicator string
	if t.vterm != nil && t.vterm.IsScrolledBack() {
		scrollIndicator = " [SCROLL]"
	}

	footer := terminalFooterStyle.Width(t.width).Render(t.renderFooter() + scrollIndicator)

	return strings.Join([]string{header, content, footer}, "\n")
}

// Utility: Dynamically calculate terminal height
func (t *TerminalComponent) contentHeight() int {
	return t.height - 2 // 1 line each reserved for header and footer
}

// Utility: Resize terminal components dynamically
func (t *TerminalComponent) resizeTerminal() {
	if t.contentHeight() < 1 {
		return
	}
	if t.vterm != nil {
		t.vterm.Resize(t.width, t.contentHeight())
	}
	if t.session != nil {
		t.session.Resize(t.width, t.contentHeight())
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
		session, err := ssh.NewBubbleTeaSession(conn, width, height-2)
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

// Utility: Render the footer with helper instructions.
func (t *TerminalComponent) renderFooter() string {
	if t.sessionClosed {
		return "Session closed - Press ESC to return"
	}
	if t.width < 80 {
		// Condensed instructions for narrow terminals
		return "ESC: Exit | CTRL+D: EOF | Scroll: PgUp/PgDn"
	}
	// Full instructions for wider terminals
	return "ESC: Exit | CTRL+D: EOF | PgUp/PgDn: Scroll Vertically | Tab: Complete Command | Mouse: Copy Text"
}

// Utility: Handle key input.
func (t *TerminalComponent) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		t.finished = true
		if t.session != nil {
			t.session.Close()
		}
		return t, nil

	case "pgup", "shift+up":
		t.vterm.ScrollUp(10) // Only scrolls up, doesn't forward this key to the SSH session.
		return t, nil

	case "pgdown", "shift+down":
		t.vterm.ScrollDown(10) // Only scrolls down, doesn't forward this key to the SSH session.
		return t, nil

	default:
		t.forwardKeyToSession(msg.String())
	}

	return t, nil
}

// Utility: Forward keys to the SSH session.
func (t *TerminalComponent) forwardKeyToSession(key string) {
	var data []byte
	switch key {
	case "tab":
		data = []byte{'\t'}
	case "enter":
		data = []byte{'\r'}
	case "backspace", "delete":
		data = []byte{127}
	case "up":
		data = []byte{27, '[', 'A'}
	case "down":
		data = []byte{27, '[', 'B'}
	case "right":
		data = []byte{27, '[', 'C'}
	case "left":
		data = []byte{27, '[', 'D'}
	default:
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

// Utility: Generate a centered box view
func (t *TerminalComponent) centeredBox(content string) string {
	lines := strings.Split(content, "\n")
	topPadding := max((t.height-len(lines))/2, 0)
	builder := strings.Builder{}
	builder.WriteString(strings.Repeat("\n", topPadding))
	for _, line := range lines {
		leftPadding := (t.width - len(line)) / 2
		builder.WriteString(strings.Repeat(" ", leftPadding))
		builder.WriteString(line)
		builder.WriteString("\n")
	}
	return builder.String()
}

// IsFinished returns whether the terminal session is finished
func (t *TerminalComponent) IsFinished() bool {
	return t.finished
}
