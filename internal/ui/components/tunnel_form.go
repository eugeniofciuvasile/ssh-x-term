package components

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/eugeniofciuvasile/ssh-x-term/internal/config"
	"github.com/eugeniofciuvasile/ssh-x-term/internal/tunnel"
)

// TunnelForm is a form for adding or editing a tunnel configuration
type TunnelForm struct {
	Connection   config.SSHConnection
	editing      bool
	tunnel       config.TunnelConfig
	inputs       []textinput.Model
	focusIndex   int
	tunnelType   config.TunnelType
	autoStart    bool
	submitted    bool
	canceled     bool
	errorMessage string
	portStatus   string
	width        int
	height       int
}

// NewTunnelForm initializes a new tunnel form
func NewTunnelForm(conn config.SSHConnection, existing *config.TunnelConfig) *TunnelForm {
	editing := existing != nil
	var tun config.TunnelConfig
	if editing {
		tun = *existing
	} else {
		tun = config.TunnelConfig{
			BindHost:   "127.0.0.1",
			BindPort:   tunnel.FindNextAvailablePort("127.0.0.1", 8080),
			Type:       config.TunnelTypeLocal,
			TargetHost: "127.0.0.1",
			TargetPort: 8080,
			Enabled:    true,
			AutoStart:  false,
		}
	}

	// 0: Name, 1: BindHost, 2: BindPort, 3: TargetHost, 4: TargetPort, 5: Notes
	inputs := make([]textinput.Model, 6)

	initInput := func(i int, placeholder string, width int) {
		inputs[i] = textinput.New()
		inputs[i].Placeholder = placeholder
		inputs[i].Width = width
		inputs[i].Prompt = "> "
		inputs[i].PromptStyle = blurredStyle
		inputs[i].TextStyle = blurredStyle
	}

	initInput(0, "Tunnel Name (e.g. Postgres DB)", 40)
	initInput(1, "Bind Host (default: 127.0.0.1)", 30)
	initInput(2, "Bind Port (e.g. 5432)", 20)
	initInput(3, "Target Host (e.g. 10.0.0.15 or localhost)", 40)
	initInput(4, "Target Port (e.g. 5432)", 20)
	initInput(5, "Optional Notes", 40)

	inputs[0].SetValue(tun.Name)
	inputs[1].SetValue(tun.BindHost)
	if tun.BindPort > 0 {
		inputs[2].SetValue(strconv.Itoa(tun.BindPort))
	}
	inputs[3].SetValue(tun.TargetHost)
	if tun.TargetPort > 0 {
		inputs[4].SetValue(strconv.Itoa(tun.TargetPort))
	}
	inputs[5].SetValue(tun.Notes)

	inputs[0].Focus()
	inputs[0].PromptStyle = focusedStyle
	inputs[0].TextStyle = focusedStyle

	tf := &TunnelForm{
		Connection: conn,
		editing:    editing,
		tunnel:     tun,
		inputs:     inputs,
		focusIndex: 0,
		tunnelType: tun.Type,
		autoStart:  tun.AutoStart,
		width:      80,
		height:     24,
	}

	tf.updatePortStatus()
	return tf
}

// Init initializes the form inputs and cursor blinking
func (tf *TunnelForm) Init() tea.Cmd {
	return textinput.Blink
}

// SetSize updates the form display dimensions
func (tf *TunnelForm) SetSize(width, height int) {
	tf.width = width
	tf.height = height
}

// IsSubmitted reports whether the form was submitted by the user
func (tf *TunnelForm) IsSubmitted() bool { return tf.submitted }

// IsCanceled reports whether the form was dismissed by the user
func (tf *TunnelForm) IsCanceled() bool { return tf.canceled }

// Tunnel returns the configured TunnelConfig constructed from form inputs
func (tf *TunnelForm) Tunnel() config.TunnelConfig {
	bindPort, _ := strconv.Atoi(tf.inputs[2].Value())
	targetPort, _ := strconv.Atoi(tf.inputs[4].Value())
	bindHost := tf.inputs[1].Value()
	if bindHost == "" {
		bindHost = "127.0.0.1"
	}
	targetHost := tf.inputs[3].Value()
	if tf.tunnelType != config.TunnelTypeDynamic && targetHost == "" {
		targetHost = "127.0.0.1"
	}

	id := tf.tunnel.ID
	if id == "" {
		id = fmt.Sprintf("tun_%d", time.Now().UnixNano())
	}

	return config.TunnelConfig{
		ID:         id,
		Name:       tf.inputs[0].Value(),
		Type:       tf.tunnelType,
		BindHost:   bindHost,
		BindPort:   bindPort,
		TargetHost: targetHost,
		TargetPort: targetPort,
		AutoStart:  tf.autoStart,
		Enabled:    true,
		Notes:      tf.inputs[5].Value(),
	}
}

func (tf *TunnelForm) updatePortStatus() {
	portStr := tf.inputs[2].Value()
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 || port > 65535 {
		tf.portStatus = lipgloss.NewStyle().Foreground(colorSubText).Render("Enter a valid port (1-65535)")
		return
	}

	host := tf.inputs[1].Value()
	if host == "" {
		host = "127.0.0.1"
	}

	res := tunnel.CheckLocalPort(host, port)
	if res.Available {
		tf.portStatus = lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B")).Render(fmt.Sprintf("✓ Port %d is available", port))
	} else {
		next := tunnel.FindNextAvailablePort(host, port+1)
		tf.portStatus = lipgloss.NewStyle().Foreground(colorError).Render(fmt.Sprintf("⚠ Port %d is in use! Next free: %d", port, next))
	}
}

// Update handles user key input, cycling options, and form submission
func (tf *TunnelForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		tf.SetSize(msg.Width, msg.Height)
		return tf, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			tf.canceled = true
			return tf, nil

		case "ctrl+t":
			// Cycle tunnel type: Local -> Remote -> Dynamic -> Local
			switch tf.tunnelType {
			case config.TunnelTypeLocal:
				tf.tunnelType = config.TunnelTypeRemote
			case config.TunnelTypeRemote:
				tf.tunnelType = config.TunnelTypeDynamic
			case config.TunnelTypeDynamic:
				tf.tunnelType = config.TunnelTypeLocal
			}
			return tf, nil

		case "ctrl+a":
			// Toggle auto-start
			tf.autoStart = !tf.autoStart
			return tf, nil

		case "tab", "down":
			tf.focusIndex = (tf.focusIndex + 1) % (len(tf.inputs) + 2) // +2 for Type and Submit button
			tf.updateFocus()
			return tf, nil

		case "shift+tab", "up":
			tf.focusIndex = (tf.focusIndex - 1 + len(tf.inputs) + 2) % (len(tf.inputs) + 2)
			tf.updateFocus()
			return tf, nil

		case "enter":
			// If on submit button or last input, submit form
			if tf.focusIndex == len(tf.inputs)+1 || tf.focusIndex == len(tf.inputs) {
				if err := tf.validate(); err != nil {
					tf.errorMessage = err.Error()
					return tf, nil
				}
				tf.submitted = true
				return tf, nil
			}
			// Move to next field
			tf.focusIndex++
			tf.updateFocus()
			return tf, nil
		}
	}

	// Update the focused textinput
	if tf.focusIndex < len(tf.inputs) {
		var cmd tea.Cmd
		tf.inputs[tf.focusIndex], cmd = tf.inputs[tf.focusIndex].Update(msg)
		if tf.focusIndex == 1 || tf.focusIndex == 2 {
			tf.updatePortStatus()
		}
		return tf, cmd
	}

	return tf, nil
}

func (tf *TunnelForm) updateFocus() {
	for i := range tf.inputs {
		if i == tf.focusIndex {
			tf.inputs[i].Focus()
			tf.inputs[i].PromptStyle = focusedStyle
			tf.inputs[i].TextStyle = focusedStyle
		} else {
			tf.inputs[i].Blur()
			tf.inputs[i].PromptStyle = blurredStyle
			tf.inputs[i].TextStyle = blurredStyle
		}
	}
}

func (tf *TunnelForm) validate() error {
	if strings.TrimSpace(tf.inputs[0].Value()) == "" {
		return fmt.Errorf("tunnel name is required")
	}

	bindPort, err := strconv.Atoi(tf.inputs[2].Value())
	if err != nil || bindPort <= 0 || bindPort > 65535 {
		return fmt.Errorf("invalid bind port number (1-65535)")
	}

	if tf.tunnelType != config.TunnelTypeDynamic {
		targetPort, err := strconv.Atoi(tf.inputs[4].Value())
		if err != nil || targetPort <= 0 || targetPort > 65535 {
			return fmt.Errorf("invalid target port number (1-65535)")
		}
	}

	return nil
}

// View renders the centered card containing the interactive tunnel form
func (tf *TunnelForm) View() string {
	var sb strings.Builder

	title := "Add Port Forwarding Tunnel"
	if tf.editing {
		title = "Edit Port Forwarding Tunnel"
	}
	sb.WriteString(formTitleStyle.Render(fmt.Sprintf("%s for %s", title, tf.Connection.Name)))
	sb.WriteString("\n\n")

	if tf.errorMessage != "" {
		sb.WriteString(errorStyle.Render("Error: " + tf.errorMessage))
		sb.WriteString("\n\n")
	}

	// 1. Name Input
	sb.WriteString(lipgloss.NewStyle().Bold(true).Render("Tunnel Name:"))
	sb.WriteString("\n")
	sb.WriteString(tf.inputs[0].View())
	sb.WriteString("\n\n")

	// 2. Tunnel Type Selector
	sb.WriteString(lipgloss.NewStyle().Bold(true).Render("Tunnel Type (Press Ctrl+T to cycle):"))
	sb.WriteString("\n")
	typeLocal := "[ Local (-L) ]"
	typeRemote := "[ Remote (-R) ]"
	typeDynamic := "[ Dynamic SOCKS5 (-D) ]"

	switch tf.tunnelType {
	case config.TunnelTypeLocal:
		typeLocal = focusedStyle.Bold(true).Render("► [ Local (-L) ] ◄")
	case config.TunnelTypeRemote:
		typeRemote = focusedStyle.Bold(true).Render("► [ Remote (-R) ] ◄")
	case config.TunnelTypeDynamic:
		typeDynamic = focusedStyle.Bold(true).Render("► [ Dynamic SOCKS5 (-D) ] ◄")
	}
	sb.WriteString(fmt.Sprintf("%s  %s  %s\n\n", typeLocal, typeRemote, typeDynamic))

	// 3. Bind Host & Port
	sb.WriteString(lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Bold(true).Render("Bind Host:"),
			tf.inputs[1].View(),
		),
		"   ",
		lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Bold(true).Render("Bind Port:"),
			tf.inputs[2].View(),
		),
		"   ",
		lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Bold(true).Render("Port Status:"),
			tf.portStatus,
		),
	))
	sb.WriteString("\n\n")

	// 4. Target Host & Port (Disabled for Dynamic SOCKS5)
	if tf.tunnelType != config.TunnelTypeDynamic {
		sb.WriteString(lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.JoinVertical(lipgloss.Left,
				lipgloss.NewStyle().Bold(true).Render("Target Host:"),
				tf.inputs[3].View(),
			),
			"   ",
			lipgloss.JoinVertical(lipgloss.Left,
				lipgloss.NewStyle().Bold(true).Render("Target Port:"),
				tf.inputs[4].View(),
			),
		))
		sb.WriteString("\n\n")
	} else {
		sb.WriteString(lipgloss.NewStyle().Foreground(colorSubText).Render("Target: Dynamic (Clients specify destination via SOCKS5 protocol)\n\n"))
	}

	// 5. Auto Start Checkbox
	autoStartCheck := "[ ] Auto-start when connecting to host"
	if tf.autoStart {
		autoStartCheck = "[✓] Auto-start when connecting to host"
	}
	sb.WriteString(lipgloss.NewStyle().Foreground(colorAccent).Render(autoStartCheck + " (Press Ctrl+A to toggle)"))
	sb.WriteString("\n\n")

	// 6. Notes
	sb.WriteString(lipgloss.NewStyle().Bold(true).Render("Notes:"))
	sb.WriteString("\n")
	sb.WriteString(tf.inputs[5].View())
	sb.WriteString("\n\n")

	// Submit button
	btn := blurredButton
	if tf.focusIndex == len(tf.inputs)+1 || tf.focusIndex == len(tf.inputs) {
		btn = focusedButton
	}
	sb.WriteString("\n")
	sb.WriteString(btn)

	// Wrap content in a bordered box matching other forms
	formBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPrimary).
		Padding(1, 3).
		Width(64).
		Align(lipgloss.Left).
		Render(sb.String())

	return lipgloss.Place(
		tf.width,
		tf.height,
		lipgloss.Center,
		lipgloss.Center,
		formBox,
	)
}
