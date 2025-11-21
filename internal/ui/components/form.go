package components

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/eugeniofciuvasile/ssh-x-term/internal/config"
	"github.com/eugeniofciuvasile/ssh-x-term/pkg/sshutil"

	"github.com/charmbracelet/bubbles/list"
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

	// Dropdown for SSH key selection (only used when usePassword == false)
	dropdownOpen bool
	keyList      list.Model
	allKeys      []string // scanned keys from ~/.ssh
}

// list item type for key paths
type keyItem string

func (k keyItem) Title() string       { return string(k) }
func (k keyItem) Description() string { return "" }
func (k keyItem) FilterValue() string { return string(k) }

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

	// Scan ~/.ssh for private keys (simple scan)
	keys := sshutil.ScanSSHKeys()

	// Prepare list items
	items := make([]list.Item, 0, len(keys))
	for _, k := range keys {
		items = append(items, keyItem(k))
	}

	l := list.New(items, list.NewDefaultDelegate(), 50, 6)
	l.SetShowTitle(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)
	l.DisableQuitKeybindings()

	return &ConnectionForm{
		inputs:       inputs,
		focusIndex:   0,
		editing:      editing,
		connection:   initialConn,
		usePassword:  initialConn.UsePassword,
		dropdownOpen: false,
		keyList:      l,
		allKeys:      keys,
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
		// Global Cancel
		if msg.String() == "ctrl+c" {
			m.canceled = true
			return m, nil
		}

		// Dropdown navigation logic
		if m.focusIndex == 5 && !m.usePassword && m.dropdownOpen {
			switch msg.String() {
			case "esc":
				// Close dropdown, stay on field
				m.dropdownOpen = false
				return m, nil

			case "enter":
				// Select current item
				selected := m.keyList.SelectedItem()
				if selected != nil {
					if s, ok := selected.(keyItem); ok {
						m.inputs[5].SetValue(string(s))
					}
				}
				m.dropdownOpen = false
				return m, nil

			// Trap navigation keys strictly for the list
			case "up", "down", "left", "right", "j", "k", "h", "l":
				var cmd tea.Cmd
				m.keyList, cmd = m.keyList.Update(msg)
				return m, cmd

			case "tab", "shift+tab":
				// Tab closes the dropdown and proceeds to Standard Navigation below
				m.dropdownOpen = false
				// Fallthrough is not native in Go switches like this,
				// so we continue execution after this block.
			}

			// Handle typing in the field while dropdown is open (Filtering)
			if isPrintableKey(msg) {
				newTi, cmd := m.inputs[5].Update(msg)
				m.inputs[5] = newTi
				cmds = append(cmds, cmd)

				cur := strings.TrimSpace(m.inputs[5].Value())

				// If field is empty after backspace, reopen dropdown with all keys
				if cur == "" {
					items := make([]list.Item, 0, len(m.allKeys))
					for _, k := range m.allKeys {
						items = append(items, keyItem(k))
					}
					m.keyList.SetItems(items)
					m.keyList.ResetSelected()
					m.dropdownOpen = true
					return m, tea.Batch(cmds...)
				}

				// Manual entry trigger: if user types /, ~, or ., close dropdown immediately
				if strings.Contains(cur, "/") || strings.HasPrefix(cur, "~") || strings.HasPrefix(cur, ".") {
					m.dropdownOpen = false
					return m, tea.Batch(cmds...)
				}

				// Filter the list
				filtered := filterKeys(m.allKeys, cur)
				items := make([]list.Item, 0, len(filtered))
				for _, k := range filtered {
					items = append(items, keyItem(k))
				}
				m.keyList.SetItems(items)
				m.keyList.ResetSelected()

				return m, tea.Batch(cmds...)
			}
			// If key was not handled above (e.g. Tab), we fall through to Standard Navigation
		}

		// Handle backspace
		if m.focusIndex == 5 && !m.usePassword && !m.dropdownOpen {
			if msg.String() == "backspace" || msg.String() == "delete" {
				newTi, cmd := m.inputs[5].Update(msg)
				m.inputs[5] = newTi
				cmds = append(cmds, cmd)

				cur := strings.TrimSpace(m.inputs[5].Value())

				// If field is empty after backspace, reopen dropdown with all keys
				if cur == "" {
					items := make([]list.Item, 0, len(m.allKeys))
					for _, k := range m.allKeys {
						items = append(items, keyItem(k))
					}
					m.keyList.SetItems(items)
					m.keyList.ResetSelected()
					m.dropdownOpen = true
					return m, tea.Batch(cmds...)
				}

				return m, tea.Batch(cmds...)
			}
		}

		// Standard navigation logic
		switch msg.String() {
		case "esc":
			// If not in dropdown (handled above), Esc cancels the form
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
			if m.focusIndex > 7 {
				m.focusIndex = 0
			} else if m.focusIndex < 0 {
				m.focusIndex = 7
			}

			// Re-check skip logic after wrap-around
			if m.focusIndex == 6 {
				m.focusIndex += step
			}
			if m.usePassword && m.focusIndex == 5 {
				m.focusIndex += step
			}
			if !m.usePassword && m.focusIndex == 4 {
				m.focusIndex += step
			}

			if m.focusIndex == 5 && !m.usePassword {
				m.dropdownOpen = true
				// Reset list to full view initially or filtered by existing text
				cur := m.inputs[5].Value()
				filtered := filterKeys(m.allKeys, cur)
				items := make([]list.Item, 0, len(filtered))
				for _, k := range filtered {
					items = append(items, keyItem(k))
				}
				m.keyList.SetItems(items)
				m.keyList.ResetSelected()
			} else {
				m.dropdownOpen = false
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
				// Move to next field on enter
				return m, func() tea.Msg { return tea.KeyMsg{Type: tea.KeyTab} }
			}

		case "ctrl+p":
			// Toggle between password and key authentication
			m.usePassword = !m.usePassword
			m.dropdownOpen = false

			// Adjust focus if currently on the toggleable field
			if m.usePassword && m.focusIndex == 5 {
				m.focusIndex = 4
				m.inputs[5].Blur()
				m.inputs[4].Focus()
			} else if !m.usePassword && m.focusIndex == 4 {
				m.focusIndex = 5
				m.inputs[4].Blur()
				m.inputs[5].Focus()
				// Auto-open if we switched into key field
				m.dropdownOpen = true
			}
		}
	}

	// Handle character input for standard fields
	// (Key field handled separately in dropdown block if open, but we update it here if closed or fallback)
	if m.focusIndex < len(m.inputs) {
		// Avoid double update for index 5 if we already handled it in dropdown block
		if kmsg, ok := msg.(tea.KeyMsg); ok && m.focusIndex == 5 && m.dropdownOpen && isPrintableKey(kmsg) {
			// already handled
		} else {
			newInput, cmd := m.inputs[m.focusIndex].Update(msg)
			m.inputs[m.focusIndex] = newInput
			cmds = append(cmds, cmd)
		}
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
		// SSH Key input
		b.WriteString(m.inputs[5].View())

		// Render dropdown under SSH key input
		if m.dropdownOpen {
			dropdownBox := lipgloss.NewStyle().
				MarginLeft(2).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("63")).
				Padding(0, 1).
				Render(m.keyList.View())
			b.WriteString("\n" + dropdownBox)
		}
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
	availableHeight := max(m.height-3, 0)
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

	// If using key authentication, key path must not be empty
	if !m.usePassword && strings.TrimSpace(m.inputs[5].Value()) == "" {
		return false, "SSH key path is required for key authentication"
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
		Password:    strings.TrimSpace(m.inputs[4].Value()),
		KeyFile:     strings.TrimSpace(m.inputs[5].Value()),
		UsePassword: m.usePassword,
	}
}

// ---------- Helper functions ----------

// filterKeys returns keys that contain the filter substring (case-insensitive)
func filterKeys(keys []string, filter string) []string {
	if filter == "" {
		return keys
	}
	filter = strings.ToLower(filter)
	out := make([]string, 0)
	for _, k := range keys {
		if strings.Contains(strings.ToLower(k), filter) {
			out = append(out, k)
		}
	}
	return out
}

// isPrintableKey checks if the key message represents a character that modifies text input.
// It returns true for standard characters, spaces, backspace, delete, and paste events.
func isPrintableKey(msg tea.KeyMsg) bool {
	// Check for standard text input types
	switch msg.Type {
	case tea.KeyRunes, tea.KeySpace, tea.KeyBackspace, tea.KeyDelete:
		return true
	}

	// Allow paste events (which usually contain printable text)
	if msg.Paste {
		return true
	}

	return false
}
