package components

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/eugeniofciuvasile/ssh-x-term/internal/config"
	"github.com/eugeniofciuvasile/ssh-x-term/internal/ssh"
	"github.com/hinshun/vt10x"
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

// TerminalOutputMsg is sent when there's new terminal output
type TerminalOutputMsg struct{}

// PTYSessionMsg is a message containing a PTY SSH session
type PTYSessionMsg struct {
	Session *ssh.PTYSession
	Error   error
}

// startPTYSessionCmd starts a PTY SSH session and returns a message with the result
func startPTYSessionCmd(connConfig config.SSHConnection, width, height int) tea.Cmd {
	return func() tea.Msg {
		// Reserve space for header and footer
		contentHeight := height - 4
		if contentHeight < 1 {
			contentHeight = 1
		}
		session, err := ssh.NewPTYSession(connConfig, width, contentHeight)
		if err != nil {
			return PTYSessionMsg{nil, err}
		}
		return PTYSessionMsg{session, nil}
	}
}

// PTYTerminalComponent represents an integrated PTY terminal for SSH sessions within Bubble Tea
type PTYTerminalComponent struct {
	connection    config.SSHConnection
	session       *ssh.PTYSession
	terminal      vt10x.Terminal
	status        string
	error         error
	loading       bool
	width         int
	height        int
	escapePressed bool
	mutex         sync.Mutex
	outputChan    chan []byte
	done          chan struct{}
	started       bool
}

// NewPTYTerminalComponent creates a new PTY terminal component
func NewPTYTerminalComponent(conn config.SSHConnection) *PTYTerminalComponent {
	return &PTYTerminalComponent{
		connection: conn,
		status:     "Connecting...",
		loading:    true,
		outputChan: make(chan []byte, 100),
		done:       make(chan struct{}),
	}
}

// Init initializes the component
func (t *PTYTerminalComponent) Init() tea.Cmd {
	// Start with default dimensions, will be updated on first WindowSizeMsg
	if t.width == 0 {
		t.width = 80
	}
	if t.height == 0 {
		t.height = 24
	}
	return startPTYSessionCmd(t.connection, t.width, t.height)
}

// startOutputReader starts reading output from the terminal in a goroutine
func (t *PTYTerminalComponent) startOutputReader() tea.Cmd {
	return func() tea.Msg {
		// This goroutine reads from the output channel and sends update messages
		go func() {
			for {
				select {
				case <-t.done:
					return
				case data := <-t.outputChan:
					if len(data) > 0 {
						t.mutex.Lock()
						if t.terminal != nil {
							t.terminal.Write(data)
						}
						t.mutex.Unlock()
					}
				}
			}
		}()
		return TerminalOutputMsg{}
	}
}

// pollOutput creates a command that continuously checks for terminal output
func (t *PTYTerminalComponent) pollOutput() tea.Cmd {
	return func() tea.Msg {
		// Use a small delay to avoid busy-waiting
		select {
		case <-t.done:
			return nil
		default:
			// Return a message to trigger re-render
			return TerminalOutputMsg{}
		}
	}
}

// Update handles component updates
func (t *PTYTerminalComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		t.width = msg.Width
		// Reserve space for header and footer
		contentHeight := msg.Height - 4
		if contentHeight < 1 {
			contentHeight = 1
		}
		t.height = contentHeight
		
		// Resize the terminal emulator if it exists
		if t.terminal != nil {
			t.terminal.Resize(t.width, contentHeight)
		}
		
		// Resize the SSH session PTY if started
		if t.session != nil && t.started {
			go func() {
				if err := t.session.Resize(t.width, contentHeight); err != nil {
					log.Printf("Failed to resize PTY session: %v", err)
				}
			}()
		}
		return t, nil

	case PTYSessionMsg:
		t.loading = false
		if msg.Error != nil {
			t.error = msg.Error
			t.status = fmt.Sprintf("Error: %s", msg.Error)
			return t, nil
		}
		t.session = msg.Session
		t.status = "Connected"

		// Create terminal emulator
		contentHeight := t.height - 4
		if contentHeight < 1 {
			contentHeight = 1
		}
		t.terminal = vt10x.New(vt10x.WithSize(t.width, contentHeight))

		// Start the SSH session with PTY terminal integration
		go func() {
			// Start reading from session and writing to terminal emulator
			go t.readFromSession()
			
			// Mark as started
			t.mutex.Lock()
			t.started = true
			t.mutex.Unlock()
		}()

		return t, t.pollOutput()

	case TerminalOutputMsg:
		if t.started && !t.escapePressed {
			// Continue polling for output updates
			// We need to throttle this to avoid excessive CPU usage
			return t, tea.Tick(16*time.Millisecond, func(time.Time) tea.Msg {
				return TerminalOutputMsg{}
			})
		}
		return t, nil

	case tea.KeyMsg:
		if msg.String() == "esc" {
			t.escapePressed = true
			close(t.done)
			// Close the session
			if t.session != nil {
				go t.session.Close()
			}
			return t, nil
		}
		
		// Forward all other key events to the SSH session
		if t.session != nil && t.started && !t.escapePressed {
			go t.writeToSession(msg)
		}
		return t, nil

	case tea.MouseMsg:
		// Handle mouse events (scroll, click, etc.)
		if t.session != nil && t.started && !t.escapePressed {
			// Mouse events could be forwarded if the terminal supports them
			// For now, we'll handle scroll locally by updating the view
		}
		return t, nil
	}

	return t, nil
}

// readFromSession reads output from the SSH session and feeds it to the terminal emulator
func (t *PTYTerminalComponent) readFromSession() {
	buf := make([]byte, 4096)
	for {
		select {
		case <-t.done:
			return
		default:
			// This is a blocking read, which is fine since we're in a goroutine
			if t.session == nil {
				return
			}
			
			n, err := t.session.Read(buf)
			if err != nil {
				if err != io.EOF {
					log.Printf("Error reading from session: %v", err)
				}
				return
			}
			
			if n > 0 {
				data := make([]byte, n)
				copy(data, buf[:n])
				
				// Write to terminal emulator
				t.mutex.Lock()
				if t.terminal != nil {
					t.terminal.Write(data)
				}
				t.mutex.Unlock()
			}
		}
	}
}

// writeToSession writes keyboard input to the SSH session
func (t *PTYTerminalComponent) writeToSession(msg tea.KeyMsg) {
	if t.session == nil {
		return
	}
	
	var data []byte
	
	switch msg.Type {
	case tea.KeyEnter:
		data = []byte("\r")
	case tea.KeyBackspace:
		data = []byte("\x7f")
	case tea.KeyTab:
		data = []byte("\t")
	case tea.KeyUp:
		data = []byte("\x1b[A")
	case tea.KeyDown:
		data = []byte("\x1b[B")
	case tea.KeyRight:
		data = []byte("\x1b[C")
	case tea.KeyLeft:
		data = []byte("\x1b[D")
	case tea.KeyHome:
		data = []byte("\x1b[H")
	case tea.KeyEnd:
		data = []byte("\x1b[F")
	case tea.KeyPgUp:
		data = []byte("\x1b[5~")
	case tea.KeyPgDown:
		data = []byte("\x1b[6~")
	case tea.KeyDelete:
		data = []byte("\x1b[3~")
	case tea.KeyCtrlC:
		data = []byte("\x03")
	case tea.KeyCtrlD:
		data = []byte("\x04")
	case tea.KeyCtrlZ:
		data = []byte("\x1a")
	case tea.KeyRunes:
		data = []byte(msg.String())
	default:
		// For other keys, try to use the string representation
		if s := msg.String(); s != "" && s != "esc" {
			data = []byte(s)
		}
	}
	
	if len(data) > 0 {
		t.session.Write(data)
	}
}

// View renders the component
func (t *PTYTerminalComponent) View() string {
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
			t.height+4,
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
			t.height+4,
		)
	}

	// Render the terminal emulator output
	header := terminalHeaderStyle.Width(t.width).Render(
		fmt.Sprintf("Connected to %s@%s:%d",
			t.connection.Username,
			t.connection.Host,
			t.connection.Port),
	)

	footer := terminalFooterStyle.Width(t.width).Render(
		"Press ESC to disconnect | Mouse: scroll, select, copy",
	)

	// Get terminal content
	var content string
	t.mutex.Lock()
	if t.terminal != nil {
		content = t.renderTerminalContent()
	} else {
		content = strings.Repeat("\n", max(t.height-1, 0))
	}
	t.mutex.Unlock()

	return fmt.Sprintf("%s\n%s\n%s", header, content, footer)
}

// renderTerminalContent renders the terminal emulator's screen buffer
func (t *PTYTerminalComponent) renderTerminalContent() string {
	if t.terminal == nil {
		return ""
	}

	var buf bytes.Buffer
	
	// Lock the terminal while reading
	t.terminal.Lock()
	defer t.terminal.Unlock()
	
	cols, rows := t.terminal.Size()
	
	// Render each line of the terminal
	for row := 0; row < rows; row++ {
		if row > 0 {
			buf.WriteString("\n")
		}
		
		for col := 0; col < cols; col++ {
			cell := t.terminal.Cell(col, row)
			// Get the character
			ch := cell.Char
			if ch == 0 {
				ch = ' '
			}
			buf.WriteRune(ch)
		}
	}

	return buf.String()
}

// IsFinished returns whether the terminal session is finished
func (t *PTYTerminalComponent) IsFinished() bool {
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
