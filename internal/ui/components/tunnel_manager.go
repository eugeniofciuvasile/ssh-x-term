package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/eugeniofciuvasile/ssh-x-term/internal/config"
	"github.com/eugeniofciuvasile/ssh-x-term/internal/tunnel"
)

// TunnelAddRequestMsg requests opening the tunnel creation form
type TunnelAddRequestMsg struct {
	Connection config.SSHConnection
}

// TunnelEditRequestMsg requests opening the tunnel edit form for an existing tunnel
type TunnelEditRequestMsg struct {
	Connection config.SSHConnection
	Tunnel     config.TunnelConfig
}

// TunnelDeleteRequestMsg requests deleting a tunnel configuration
type TunnelDeleteRequestMsg struct {
	Connection config.SSHConnection
	TunnelID   string
}

// TunnelToggleMsg requests activating or deactivating a tunnel
type TunnelToggleMsg struct {
	Connection config.SSHConnection
	Tunnel     config.TunnelConfig
}

// TunnelOpenGraphMsg requests opening the interactive topology diagram view
type TunnelOpenGraphMsg struct {
	Connection  config.SSHConnection
	SelectedIdx int
}

type tunnelLayout struct {
	statusWidth  int
	nameWidth    int
	typeWidth    int
	routeWidth   int
	trafficWidth int
	autoWidth    int
}

// TunnelManager provides a TUI view to manage and control tunnels for an SSH connection
type TunnelManager struct {
	Connection    config.SSHConnection
	Tunnels       []config.TunnelConfig
	selectedIndex int
	width         int
	height        int
	engine        *tunnel.Engine
	errorMessage  string
	layout        tunnelLayout
}

// NewTunnelManager creates a new TunnelManager view for the given SSH connection
func NewTunnelManager(conn config.SSHConnection) *TunnelManager {
	tm := &TunnelManager{
		Connection:    conn,
		Tunnels:       conn.Tunnels,
		selectedIndex: 0,
		width:         80,
		height:        20,
		engine:        tunnel.GetEngine(),
	}
	tm.recalculateLayout(80)
	return tm
}

// Init initializes the component lifecycle
func (tm *TunnelManager) Init() tea.Cmd {
	return nil
}

// SetSize updates the display dimensions and recalculates responsive table column widths
func (tm *TunnelManager) SetSize(width, height int) {
	tm.width = width
	tm.height = height
	tm.recalculateLayout(width)
}

func (tm *TunnelManager) recalculateLayout(totalWidth int) {
	availableWidth := max(totalWidth-4, 0)

	statusW := 12
	typeW := 14
	trafficW := 20
	autoW := 6

	fixedW := statusW + typeW + trafficW + autoW
	remaining := availableWidth - fixedW
	if remaining < 20 {
		remaining = 20
	}

	nameW := int(float64(remaining) * 0.38)
	routeW := remaining - nameW

	if nameW < 10 {
		nameW = 10
	}
	if routeW < 14 {
		routeW = 14
	}

	tm.layout = tunnelLayout{
		statusWidth:  statusW,
		nameWidth:    nameW,
		typeWidth:    typeW,
		routeWidth:   routeW,
		trafficWidth: trafficW,
		autoWidth:    autoW,
	}
}

// SelectedTunnel returns the currently highlighted tunnel configuration, or nil if none exist
func (tm *TunnelManager) SelectedTunnel() *config.TunnelConfig {
	if len(tm.Tunnels) > 0 && tm.selectedIndex < len(tm.Tunnels) {
		return &tm.Tunnels[tm.selectedIndex]
	}
	return nil
}

// SelectedIndex returns the index of the currently highlighted tunnel
func (tm *TunnelManager) SelectedIndex() int {
	return tm.selectedIndex
}

// Update handles user key events and window resize messages
func (tm *TunnelManager) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		tm.SetSize(msg.Width, msg.Height)
		return tm, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
			if tm.selectedIndex > 0 {
				tm.selectedIndex--
			}
			return tm, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
			if tm.selectedIndex < len(tm.Tunnels)-1 {
				tm.selectedIndex++
			}
			return tm, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("a", "A"))):
			return tm, func() tea.Msg {
				return TunnelAddRequestMsg{Connection: tm.Connection}
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("e", "E"))):
			if tun := tm.SelectedTunnel(); tun != nil {
				return tm, func() tea.Msg {
					return TunnelEditRequestMsg{Connection: tm.Connection, Tunnel: *tun}
				}
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("d", "D"))):
			if tun := tm.SelectedTunnel(); tun != nil {
				tunID := tun.ID
				return tm, func() tea.Msg {
					return TunnelDeleteRequestMsg{Connection: tm.Connection, TunnelID: tunID}
				}
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys(" ", "enter"))):
			if tun := tm.SelectedTunnel(); tun != nil {
				tunCopy := *tun
				return tm, func() tea.Msg {
					return TunnelToggleMsg{Connection: tm.Connection, Tunnel: tunCopy}
				}
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("v", "g", "V", "G"))):
			if len(tm.Tunnels) > 0 {
				idx := tm.selectedIndex
				return tm, func() tea.Msg {
					return TunnelOpenGraphMsg{Connection: tm.Connection, SelectedIdx: idx}
				}
			}
		}
	}
	return tm, nil
}

// View renders the tunnel manager table and status rows
func (tm *TunnelManager) View() string {
	if len(tm.Tunnels) == 0 {
		return lipgloss.Place(
			tm.width,
			tm.height,
			lipgloss.Center,
			lipgloss.Center,
			lipgloss.NewStyle().Foreground(colorSubText).Render(
				fmt.Sprintf("No SSH tunnels configured for '%s'.\n\nPress 'a' to add a port forwarding tunnel.\nPress 'esc' to return.", tm.Connection.Name),
			),
		)
	}

	var sb strings.Builder

	// Table Headers using dynamic layout widths
	headers := lipgloss.NewStyle().PaddingLeft(2).Render(
		lipgloss.JoinHorizontal(lipgloss.Left,
			headerStyle.Width(tm.layout.statusWidth).Render("Status"),
			headerStyle.Width(tm.layout.nameWidth).Render("Name"),
			headerStyle.Width(tm.layout.typeWidth).Render("Type"),
			headerStyle.Width(tm.layout.routeWidth).Render("Port Mapping"),
			headerStyle.Width(tm.layout.trafficWidth).Render("Traffic (▲/▼)"),
			headerStyle.Width(tm.layout.autoWidth).Render("Auto"),
		),
	)
	sb.WriteString(headers)
	sb.WriteString("\n")

	// Render rows
	for idx, tun := range tm.Tunnels {
		rt := tm.engine.GetRuntimeInfo(tun.ID)

		statusStr := lipgloss.NewStyle().Foreground(colorSubText).Render("○ Stopped")
		if rt.Status == tunnel.StatusActive {
			statusStr = lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B")).Bold(true).Render("● Active")
		} else if rt.Status == tunnel.StatusError {
			statusStr = lipgloss.NewStyle().Foreground(colorError).Bold(true).Render("✖ Error")
		}

		typeStr := string(tun.Type)
		switch tun.Type {
		case config.TunnelTypeLocal:
			typeStr = "Local (-L)"
		case config.TunnelTypeRemote:
			typeStr = "Remote (-R)"
		case config.TunnelTypeDynamic:
			typeStr = "Dynamic (-D)"
		}

		bindHost := tun.BindHost
		if bindHost == "" {
			bindHost = "127.0.0.1"
		}
		var routeStr string
		switch tun.Type {
		case config.TunnelTypeLocal:
			targetHost := tun.TargetHost
			if targetHost == "" {
				targetHost = "127.0.0.1"
			}
			routeStr = fmt.Sprintf("%s:%d ► %s:%d", bindHost, tun.BindPort, targetHost, tun.TargetPort)
		case config.TunnelTypeRemote:
			targetHost := tun.TargetHost
			if targetHost == "" {
				targetHost = "127.0.0.1"
			}
			routeStr = fmt.Sprintf("%s:%d ◄ %s:%d", bindHost, tun.BindPort, targetHost, tun.TargetPort)
		case config.TunnelTypeDynamic:
			routeStr = fmt.Sprintf("SOCKS5 on %s:%d", bindHost, tun.BindPort)
		}

		trafficStr := fmt.Sprintf("▲ %s  ▼ %s", formatBytes(rt.BytesOut), formatBytes(rt.BytesIn))

		autoStr := "-"
		if tun.AutoStart {
			autoStr = "✓"
		}

		row := lipgloss.JoinHorizontal(lipgloss.Left,
			lipgloss.NewStyle().Width(tm.layout.statusWidth).Render(statusStr),
			lipgloss.NewStyle().Width(tm.layout.nameWidth).Render(truncate(tun.Name, tm.layout.nameWidth)),
			lipgloss.NewStyle().Width(tm.layout.typeWidth).Render(truncate(typeStr, tm.layout.typeWidth)),
			lipgloss.NewStyle().Width(tm.layout.routeWidth).Render(truncate(routeStr, tm.layout.routeWidth)),
			lipgloss.NewStyle().Width(tm.layout.trafficWidth).Render(truncate(trafficStr, tm.layout.trafficWidth)),
			lipgloss.NewStyle().Width(tm.layout.autoWidth).Render(autoStr),
		)

		if idx == tm.selectedIndex {
			sb.WriteString(selectedItemStyle.Render(row))
		} else {
			sb.WriteString(itemStyle.Render(row))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
