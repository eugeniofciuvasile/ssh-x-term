package components

import (
	"fmt"
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
	// 0: Name, 1: Host, 2: Port, 3: User, 4: Pass, 5: Key, 6: ID
	inputs = make([]textinput.Model, 7)

	// Helper to init standard inputs
	initInput := func(i int, placeholder string, width int) {
		inputs[i] = textinput.New()
		inputs[i].Placeholder = placeholder
		inputs[i].Width = width
		inputs[i].Prompt = "> "
		inputs[i].PromptStyle = blurredStyle
		inputs[i].TextStyle = blurredStyle
	}

	initInput(0, "Connection Name", 40)
	inputs[0].Focus() // Focus first input initially
	inputs[0].PromptStyle = focusedStyle
	inputs[0].TextStyle = focusedStyle

	initInput(1, "Hostname or IP", 40)
	initInput(2, "Port (default: 22)", 40)
	initInput(3, "Username", 30)

	// Password input
	initInput(4, "Password", 40)
	inputs[4].EchoMode = textinput.EchoPassword
	inputs[4].EchoCharacter = '•'

	// Key file input
	initInput(5, "Path to SSH key (example: ~/.ssh/id_rsa)", 50)

	// ID input (hidden from view, used as identifier)
	initInput(6, "ID (auto-generated)", 40)

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
		m.SetSize(msg.Width, msg.Height)

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.canceled = true
			return m, nil

		case "tab", "shift+tab", "up", "down":
			// Calculate move direction
			step := 1
			if msg.String() == "shift+tab" || msg.String() == "up" {
				step = -1
			}

			// Move focus
			m.focusIndex += step

			// Handle skipping logic
			// Indices: 0=Name, 1=Host, 2=Port, 3=User, 4=Pass, 5=Key, 6=ID(Hidden), 7=SubmitButton

			// Skip Password(4) if using Key
			if !m.usePassword && m.focusIndex == 4 {
				m.focusIndex += step
			}
			// Skip Key(5) if using Password
			if m.usePassword && m.focusIndex == 5 {
				m.focusIndex += step
			}
			// Skip ID(6) always
			if m.focusIndex == 6 {
				m.focusIndex += step
			}

			// Handle Wrap-around
			// Max index is 7 (Submit button)
			if m.focusIndex > 7 {
				m.focusIndex = 0
			} else if m.focusIndex < 0 {
				m.focusIndex = 7
			}

			// Re-check skip logic after wrap-around (e.g. going backwards from 0 to 7, then hitting 6)
			if m.focusIndex == 6 {
				m.focusIndex += step
			}
			// Re-check auth fields after wrap/skip
			if m.usePassword && m.focusIndex == 5 {
				m.focusIndex += step
			}
			if !m.usePassword && m.focusIndex == 4 {
				m.focusIndex += step
			}

			// Apply Focus/Blur Styles
			for i := 0; i < len(m.inputs); i++ {
				if i == m.focusIndex {
					cmds = append(cmds, m.inputs[i].Focus())
					m.inputs[i].PromptStyle = focusedStyle
					m.inputs[i].TextStyle = focusedStyle
				} else {
					m.inputs[i].Blur()
					m.inputs[i].PromptStyle = blurredStyle
					m.inputs[i].TextStyle = blurredStyle
				}
			}

		case "enter":
			// Check if we are at the submit button (index 7) OR submitting from a field
			if m.focusIndex == 7 {
				if valid, err := m.validateForm(); valid {
					m.updateConnection()
					m.submitted = true
				} else {
					m.errorMessage = err
				}
			} else {
				// Move to next field on enter, unless it's the last visible field, then submit?
				// For now, standard behavior is just consume enter or move next.
				// Let's treat it like tab for fields
				m.Update(tea.KeyMsg{Type: tea.KeyTab})
				return m, nil
			}

		case "ctrl+p":
			// Toggle between password and key authentication
			m.usePassword = !m.usePassword
			// Adjust focus if currently on the toggleable field
			if m.usePassword && m.focusIndex == 5 {
				m.focusIndex = 4
				m.inputs[5].Blur()
				m.inputs[4].Focus()
			} else if !m.usePassword && m.focusIndex == 4 {
				m.focusIndex = 5
				m.inputs[4].Blur()
				m.inputs[5].Focus()
			}
		}
	}

	// Handle character input only if a specific input field is focused
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
	b.WriteString(sectionTitleStyle.Render(title))
	b.WriteString("\n\n")

	// Helper for rendering labels
	label := func(text string) string {
		return lipgloss.NewStyle().Foreground(colorSubText).Render(text)
	}

	// Render standard inputs
	b.WriteString(label("Name") + "\n")
	b.WriteString(m.inputs[0].View() + "\n\n")

	b.WriteString(label("Host") + "\n")
	b.WriteString(m.inputs[1].View() + "\n\n")

	b.WriteString(label("Port") + "\n")
	b.WriteString(m.inputs[2].View() + "\n\n")

	b.WriteString(label("Username") + "\n")
	b.WriteString(m.inputs[3].View() + "\n\n")

	// Auth method header
	authMethod := "Using Password Authentication"
	if !m.usePassword {
		authMethod = "Using SSH Key Authentication"
	}
	authHint := lipgloss.NewStyle().Foreground(colorInactive).Render("(Ctrl+P to toggle)")
	b.WriteString(fmt.Sprintf("%s %s\n", label(authMethod), authHint))

	// Render conditional input
	if m.usePassword {
		b.WriteString(m.inputs[4].View()) // Password
	} else {
		b.WriteString(m.inputs[5].View()) // SSH Key
	}
	b.WriteString("\n\n")

	// Render submit button (Index 7)
	button := blurredButton
	if m.focusIndex == 7 {
		button = focusedButton
	}
	b.WriteString(button)

	// Show error message if any
	if m.errorMessage != "" {
		b.WriteString("\n\n")
		b.WriteString(errorStyle.Render(m.errorMessage))
	}

	// Wrap content in a bordered box (Left aligned content)
	formBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPrimary).
		Padding(1, 3).
		Width(60).            // Fixed width for the box
		Align(lipgloss.Left). // Align text inside the box to the left
		Render(b.String())

	// Center the box on the screen
	availableHeight := max(m.height-2, 0)
	return lipgloss.Place(
		m.width,
		availableHeight,
		lipgloss.Center, // Horizontal Center
		lipgloss.Center, // Vertical Center
		formBox,
	)
}

func (m *ConnectionForm) SetSize(width, height int) {
	m.width = width
	m.height = height
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
