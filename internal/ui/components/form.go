package components

import (
	"strconv"
	"strings"
	"time"

	"github.com/eugeniofciuvasile/ssh-x-term/internal/config"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ConnectionForm represents a form for creating/editing connections
type ConnectionForm struct {
	inputs       []textinput.Model
	focusIndex   int
	editing      bool
	connection   config.SSHConnection
	usePassword  bool
	submitted    bool
	canceled     bool
	width        int
	height       int
	errorMessage string
}

// NewConnectionForm creates a new connection form
func NewConnectionForm(conn *config.SSHConnection) *ConnectionForm {
	var inputs []textinput.Model
	var initialConn config.SSHConnection
	editing := conn != nil

	if editing {
		initialConn = *conn
	} else {
		initialConn = config.SSHConnection{
			Port:        22, // Default SSH port
			UsePassword: true,
		}
	}

	// Create text inputs
	inputs = make([]textinput.Model, 7)

	// Name input
	inputs[0] = textinput.New()
	inputs[0].Placeholder = "Connection Name"
	inputs[0].Focus()
	inputs[0].Width = 40
	inputs[0].Prompt = RenderPrompt(true)
	inputs[0].PromptStyle = StyleFocused
	inputs[0].TextStyle = StyleFocused

	// Host input
	inputs[1] = textinput.New()
	inputs[1].Placeholder = "Hostname or IP"
	inputs[1].Width = 40
	inputs[1].Prompt = RenderPrompt(false)

	// Port input
	inputs[2] = textinput.New()
	inputs[2].Placeholder = "Port (default: 22)"
	inputs[2].Width = 40
	inputs[2].Prompt = RenderPrompt(false)

	// Username input
	inputs[3] = textinput.New()
	inputs[3].Placeholder = "Username"
	inputs[3].Width = 30
	inputs[3].Prompt = RenderPrompt(false)

	// Password input
	inputs[4] = textinput.New()
	inputs[4].Placeholder = "Password"
	inputs[4].Width = 40
	inputs[4].Prompt = RenderPrompt(false)
	inputs[4].EchoMode = textinput.EchoPassword
	inputs[4].EchoCharacter = '•'

	// Key file input
	inputs[5] = textinput.New()
	inputs[5].Placeholder = "Path to SSH key (example: ~/.ssh/id_rsa)"
	inputs[5].Width = 60
	inputs[5].Prompt = RenderPrompt(false)

	// ID input (hidden, used as identifier)
	inputs[6] = textinput.New()
	inputs[6].Placeholder = "ID (auto-generated)"
	inputs[6].Width = 40
	inputs[6].Prompt = RenderPrompt(false)

	// If editing, fill the fields
	if editing {
		inputs[0].SetValue(initialConn.Name)
		inputs[1].SetValue(initialConn.Host)
		inputs[2].SetValue(strconv.Itoa(initialConn.Port))
		inputs[3].SetValue(initialConn.Username)
		inputs[4].SetValue(initialConn.Password)
		inputs[5].SetValue(initialConn.KeyFile)
		inputs[6].SetValue(initialConn.ID)
	}

	return &ConnectionForm{
		inputs:      inputs,
		focusIndex:  0,
		editing:     editing,
		connection:  initialConn,
		usePassword: initialConn.UsePassword,
	}
}

// Init initializes the form
func (m *ConnectionForm) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles updates to the form
func (m *ConnectionForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.canceled = true
			return m, nil

		case "tab", "shift+tab", "up", "down":
			// Cycle through inputs
			if msg.String() == "shift+tab" || msg.String() == "up" {
				m.focusIndex--
				if m.focusIndex == 5 && m.usePassword {
					m.focusIndex--
				} else if m.focusIndex == 4 && !m.usePassword {
					m.focusIndex--
				}
			} else {
				m.focusIndex++
				if m.focusIndex == 4 && !m.usePassword {
					m.focusIndex++
				} else if m.focusIndex == 5 && m.usePassword {
					m.focusIndex++
				}
			}

			// Wrap around
			if m.focusIndex > len(m.inputs) {
				m.focusIndex = 0
			} else if m.focusIndex < 0 {
				m.focusIndex = len(m.inputs)
			}

			// Update input styles
			for i := 0; i < len(m.inputs); i++ {
				if i == m.focusIndex {
					cmds = append(cmds, m.inputs[i].Focus())
					m.inputs[i].PromptStyle = StyleFocused
					m.inputs[i].TextStyle = StyleFocused
					m.inputs[i].Prompt = RenderPrompt(true)
				} else {
					m.inputs[i].Blur()
					m.inputs[i].PromptStyle = StyleNormal
					m.inputs[i].TextStyle = StyleNormal
					m.inputs[i].Prompt = RenderPrompt(false)
				}
			}

		case "enter":
			if m.focusIndex == len(m.inputs) {
				// Submit button
				if valid, err := m.validateForm(); valid {
					m.updateConnection()
					m.submitted = true
				} else {
					m.errorMessage = err
				}
			}

		case "ctrl+p":
			// Toggle between password and key authentication
			m.usePassword = !m.usePassword
			if m.usePassword {
				m.inputs[4].SetValue("")
			} else {
				m.inputs[5].SetValue("")
			}
		}
	}

	// Handle character input
	if m.focusIndex < len(m.inputs) {
		newInput, cmd := m.inputs[m.focusIndex].Update(msg)
		m.inputs[m.focusIndex] = newInput
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the form
func (m *ConnectionForm) View() string {
	var b strings.Builder

	// Title
	title := "Add SSH Connection"
	if m.editing {
		title = "Edit SSH Connection"
	}
	b.WriteString(StyleTitle.Render(title))
	b.WriteString("\n\n")

	// Form field style
	fieldLabelStyle := lipgloss.NewStyle().Foreground(ColorText).Bold(true)

	// Render inputs
	b.WriteString(fieldLabelStyle.Render("Name:"))
	b.WriteString("\n")
	b.WriteString(m.inputs[0].View())
	b.WriteString("\n\n")

	b.WriteString(fieldLabelStyle.Render("Host:"))
	b.WriteString("\n")
	b.WriteString(m.inputs[1].View())
	b.WriteString("\n\n")

	b.WriteString(fieldLabelStyle.Render("Port:"))
	b.WriteString("\n")
	b.WriteString(m.inputs[2].View())
	b.WriteString("\n\n")

	b.WriteString(fieldLabelStyle.Render("Username:"))
	b.WriteString("\n")
	b.WriteString(m.inputs[3].View())
	b.WriteString("\n\n")

	// Auth method toggle
	authMethod := "Using Password Authentication"
	authStyle := StyleInfo
	if !m.usePassword {
		authMethod = "Using SSH Key Authentication"
		authStyle = StyleSuccess
	}
	b.WriteString(authStyle.Render(authMethod))
	b.WriteString(" ")
	b.WriteString(StyleHelp.Render("(Ctrl+P to toggle)"))
	b.WriteString("\n\n")

	// Render password or key file input based on auth method
	if m.usePassword {
		b.WriteString(fieldLabelStyle.Render("Password:"))
		b.WriteString("\n")
		b.WriteString(m.inputs[4].View())
		b.WriteString("\n\n")
	} else {
		b.WriteString(fieldLabelStyle.Render("SSH Key Path:"))
		b.WriteString("\n")
		b.WriteString(m.inputs[5].View())
		b.WriteString("\n\n")
	}

	// Render submit button
	button := RenderButton("Submit", m.focusIndex == len(m.inputs))
	b.WriteString("\n")
	b.WriteString(button)
	b.WriteString("\n")

	// Show error message if any
	if m.errorMessage != "" {
		b.WriteString("\n")
		b.WriteString(StyleError.Render(m.errorMessage))
		b.WriteString("\n")
	}

	return StyleContainer.Render(b.String())
}

// IsCanceled returns whether the form was canceled
func (m *ConnectionForm) IsCanceled() bool {
	return m.canceled
}

// IsSubmitted returns whether the form was submitted
func (m *ConnectionForm) IsSubmitted() bool {
	return m.submitted
}

// Connection returns the connection from the form
func (m *ConnectionForm) Connection() config.SSHConnection {
	return m.connection
}

// validateForm checks if the form inputs are valid
func (m *ConnectionForm) validateForm() (bool, string) {
	// Check required fields
	if strings.TrimSpace(m.inputs[0].Value()) == "" {
		return false, "Connection name is required"
	}
	if strings.TrimSpace(m.inputs[1].Value()) == "" {
		return false, "Host is required"
	}
	if strings.TrimSpace(m.inputs[3].Value()) == "" {
		return false, "Username is required"
	}

	// Check port
	if strings.TrimSpace(m.inputs[2].Value()) != "" {
		port, err := strconv.Atoi(m.inputs[2].Value())
		if err != nil || port < 1 || port > 65535 {
			return false, "Port must be a number between 1 and 65535"
		}
	}

	return true, ""
}

// updateConnection updates the connection from the form inputs
func (m *ConnectionForm) updateConnection() {
	// Generate ID if not editing
	id := m.inputs[6].Value()
	if id == "" {
		id = strings.ReplaceAll(m.inputs[0].Value(), " ", "_") + "_" +
			strconv.FormatInt(time.Now().UnixNano(), 10)
		m.inputs[6].SetValue(id)
	}

	// Parse port
	port := 22
	if strings.TrimSpace(m.inputs[2].Value()) != "" {
		port, _ = strconv.Atoi(m.inputs[2].Value())
	}

	// Update connection
	m.connection = config.SSHConnection{
		ID:          id,
		Name:        m.inputs[0].Value(),
		Host:        m.inputs[1].Value(),
		Port:        port,
		Username:    m.inputs[3].Value(),
		Password:    m.inputs[4].Value(),
		KeyFile:     m.inputs[5].Value(),
		UsePassword: m.usePassword,
	}
}
