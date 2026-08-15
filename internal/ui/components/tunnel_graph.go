package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/eugeniofciuvasile/ssh-x-term/internal/config"
	"github.com/eugeniofciuvasile/ssh-x-term/internal/tunnel"
	"github.com/zalando/go-keyring"
)

type GraphViewMode int

const (
	ViewModeTopology GraphViewMode = iota
	ViewModeHexDump
	ViewModeASCII
)

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(250*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

type resetGraphCopiedMsg struct{}

// TunnelGraphView provides an interactive visual topology diagram and live packet inspector
type TunnelGraphView struct {
	Connection   config.SSHConnection
	Tunnels      []config.TunnelConfig
	selectedIdx  int
	mode         GraphViewMode
	viewport     viewport.Model
	width        int
	height       int
	finished     bool
	engine       *tunnel.Engine
	paused       bool
	lastPacketCt int
	errorMessage string
	copied       bool
}

// NewTunnelGraphView creates a new interactive topology and packet inspector view
func NewTunnelGraphView(conn config.SSHConnection, selectedIdx int) *TunnelGraphView {
	if selectedIdx < 0 || selectedIdx >= len(conn.Tunnels) {
		selectedIdx = 0
	}
	vp := viewport.New(84, 12)
	return &TunnelGraphView{
		Connection:  conn,
		Tunnels:     conn.Tunnels,
		selectedIdx: selectedIdx,
		mode:        ViewModeTopology,
		viewport:    vp,
		width:       90,
		height:      22,
		engine:      tunnel.GetEngine(),
	}
}

// Init starts the live statistics polling ticker
func (tg *TunnelGraphView) Init() tea.Cmd {
	return tickCmd()
}

// Update processes key events, live packet updates, and view mode switching
func (tg *TunnelGraphView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case resetGraphCopiedMsg:
		tg.copied = false
		return tg, nil

	case tickMsg:
		if len(tg.Tunnels) > 0 {
			tun := tg.Tunnels[tg.selectedIdx]
			pkts := tg.engine.GetRecentPackets(tun.ID)
			if tg.mode != ViewModeTopology && !tg.paused {
				if len(pkts) != tg.lastPacketCt {
					tg.lastPacketCt = len(pkts)
					tg.updateViewportContent(pkts)
					tg.viewport.GotoBottom()
				}
			}
		}
		return tg, tickCmd()

	case tea.WindowSizeMsg:
		tg.SetSize(msg.Width, msg.Height)
		return tg, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc", "q", "v", "g"))):
			tg.finished = true
			return tg, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("x", "X", "d", "D"))):
			// Toggle between Topology View and Live Hex/Data Stream Inspector
			if tg.mode == ViewModeTopology {
				tg.mode = ViewModeHexDump
				if len(tg.Tunnels) > 0 {
					pkts := tg.engine.GetRecentPackets(tg.Tunnels[tg.selectedIdx].ID)
					tg.updateViewportContent(pkts)
					tg.viewport.GotoBottom()
				}
			} else {
				tg.mode = ViewModeTopology
			}
			return tg, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("y", "Y", "ctrl+y"))):
			// Copy all raw captured packets (Hex or ASCII) to system clipboard
			if len(tg.Tunnels) > 0 {
				tun := tg.Tunnels[tg.selectedIdx]
				pkts := tg.engine.GetRecentPackets(tun.ID)
				if len(pkts) > 0 {
					var sb strings.Builder
					for _, p := range pkts {
						timeStr := p.Timestamp.Format("15:04:05.000")
						dirIcon := "──►"
						if p.Direction == tunnel.DirRemoteToLocal {
							dirIcon = "◄──"
						}
						fmt.Fprintf(&sb, "%s [%s] %s (%d bytes)\n", dirIcon, timeStr, p.Direction, p.Length)
						if tg.mode == ViewModeASCII {
							sb.WriteString(formatASCIIStream(p.Data))
						} else {
							sb.WriteString(formatHexDump(p.Data))
						}
						sb.WriteString("\n")
					}
					_ = clipboard.WriteAll(sb.String())
					tg.copied = true
					return tg, func() tea.Msg {
						time.Sleep(2 * time.Second)
						return resetGraphCopiedMsg{}
					}
				}
			}
			return tg, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("m", "M"))):
			// Cycle inspection mode (Hex Dump <-> ASCII Text)
			if tg.mode == ViewModeHexDump {
				tg.mode = ViewModeASCII
			} else if tg.mode == ViewModeASCII {
				tg.mode = ViewModeHexDump
			}
			if len(tg.Tunnels) > 0 {
				pkts := tg.engine.GetRecentPackets(tg.Tunnels[tg.selectedIdx].ID)
				tg.updateViewportContent(pkts)
			}
			return tg, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("c", "C"))):
			// Clear capture buffer
			if len(tg.Tunnels) > 0 {
				tg.engine.ClearPackets(tg.Tunnels[tg.selectedIdx].ID)
				tg.updateViewportContent(nil)
			}
			return tg, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("f", "F"))):
			// Toggle freeze / auto-scroll
			tg.paused = !tg.paused
			return tg, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("left", "h"))):
			if len(tg.Tunnels) > 0 {
				tg.selectedIdx = (tg.selectedIdx - 1 + len(tg.Tunnels)) % len(tg.Tunnels)
				if tg.mode != ViewModeTopology {
					pkts := tg.engine.GetRecentPackets(tg.Tunnels[tg.selectedIdx].ID)
					tg.updateViewportContent(pkts)
					tg.viewport.GotoBottom()
				}
			}
			return tg, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("right", "l", "tab"))):
			if len(tg.Tunnels) > 0 {
				tg.selectedIdx = (tg.selectedIdx + 1) % len(tg.Tunnels)
				if tg.mode != ViewModeTopology {
					pkts := tg.engine.GetRecentPackets(tg.Tunnels[tg.selectedIdx].ID)
					tg.updateViewportContent(pkts)
					tg.viewport.GotoBottom()
				}
			}
			return tg, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys(" ", "enter"))):
			if len(tg.Tunnels) > 0 && tg.selectedIdx < len(tg.Tunnels) {
				tun := tg.Tunnels[tg.selectedIdx]
				conn := tg.Connection
				if conn.UsePassword && conn.Password == "" {
					if pass, err := keyring.Get("ssh-x-term", conn.ID); err == nil {
						conn.Password = pass
					}
				}
				_, err := tg.engine.ToggleTunnel(conn, tun)
				if err != nil {
					tg.errorMessage = err.Error()
				} else {
					tg.errorMessage = ""
				}
			}
			return tg, nil
		}

		if tg.mode != ViewModeTopology {
			var cmd tea.Cmd
			tg.viewport, cmd = tg.viewport.Update(msg)
			return tg, cmd
		}
	}
	return tg, nil
}

// SetSize updates dimensions and scales the viewport for the active view mode
func (tg *TunnelGraphView) SetSize(width, height int) {
	tg.width = width
	tg.height = height

	boxW := tg.calculateBoxWidth()
	vpHeight := max(height-6, 6)
	tg.viewport.Width = max(boxW-4, 50)
	tg.viewport.Height = vpHeight
}

func (tg *TunnelGraphView) calculateBoxWidth() int {
	w := tg.width - 6
	if w > 94 {
		w = 94
	}
	if w < 78 {
		w = 78
	}
	return w
}

// IsFinished reports whether the user exited the graph view
func (tg *TunnelGraphView) IsFinished() bool {
	return tg.finished
}

// SelectedIndex returns the currently focused tunnel index
func (tg *TunnelGraphView) SelectedIndex() int {
	return tg.selectedIdx
}

// View renders either the 2D Topology Diagram or Live Traffic Inspector
func (tg *TunnelGraphView) View() string {
	if len(tg.Tunnels) == 0 {
		return lipgloss.Place(
			tg.width,
			tg.height,
			lipgloss.Center,
			lipgloss.Center,
			lipgloss.NewStyle().Foreground(colorSubText).Render("No tunnels configured for this host.\nPress 'a' in the tunnel list to create one."),
		)
	}

	tun := tg.Tunnels[tg.selectedIdx]
	runtime := tg.engine.GetRuntimeInfo(tun.ID)
	boxW := tg.calculateBoxWidth()

	var sb strings.Builder

	// Top navigation bar
	navText := fmt.Sprintf("◄  Tunnel %d of %d: %s  ►", tg.selectedIdx+1, len(tg.Tunnels), tun.Name)
	navStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(colorAccent).
		Align(lipgloss.Center).
		Width(boxW)
	sb.WriteString(navStyle.Render(navText))
	sb.WriteString("\n")

	if tg.mode == ViewModeTopology {
		// Render Visual Topology Canvas
		graphContent := tg.renderTopologyCanvas(tun, runtime, boxW)
		sb.WriteString(graphContent)
		sb.WriteString("\n")

		// Render Metrics Dashboard Card
		metricsCard := tg.renderMetricsDashboard(tun, runtime, boxW)
		sb.WriteString(metricsCard)
	} else {
		// Render Live Packet / Hex Dump Inspector
		inspectorContent := tg.renderTrafficInspector(tun, runtime, boxW)
		sb.WriteString(inspectorContent)
	}

	return lipgloss.Place(
		tg.width,
		tg.height,
		lipgloss.Center,
		lipgloss.Center,
		sb.String(),
	)
}

// padCenter safely centers text in terminal column width using display width calculation
func padCenter(s string, targetW int) string {
	w := lipgloss.Width(s)
	if w >= targetW {
		return s
	}
	pLeft := (targetW - w) / 2
	pRight := targetW - w - pLeft
	return strings.Repeat(" ", pLeft) + s + strings.Repeat(" ", pRight)
}

// renderTopologyCanvas renders a guaranteed-alignment 2D ASCII/Unicode canvas
func (tg *TunnelGraphView) renderTopologyCanvas(tun config.TunnelConfig, rt tunnel.RuntimeInfo, boxWidth int) string {
	bindHost := tun.BindHost
	if bindHost == "" {
		bindHost = "127.0.0.1"
	}
	sshHost := fmt.Sprintf("%s:%d", tg.Connection.Host, tg.Connection.Port)
	if tg.Connection.Port == 0 {
		sshHost = fmt.Sprintf("%s:22", tg.Connection.Host)
	}

	isActive := rt.Status == tunnel.StatusActive

	// Color definitions
	n1Color := colorSecondary
	if isActive {
		n1Color = lipgloss.Color("#50FA7B") // Active Neon Green
	}
	n2Color := colorPrimary
	n3Color := colorAccent

	arrowColor := colorInactive
	if isActive {
		arrowColor = lipgloss.Color("#50FA7B")
	}

	// Node titles, endpoints, and sublabels based on tunnel type
	var n1Title, n1Addr, n1Sub string
	var n2Title, n2Addr, n2Sub string
	var n3Title, n3Addr, n3Sub string
	var arr1Text, arr2Text string

	switch tun.Type {
	case config.TunnelTypeLocal:
		n1Title, n1Addr, n1Sub = "LOCAL LISTENER", fmt.Sprintf("%s:%d", bindHost, tun.BindPort), "(Your Machine)"
		n2Title, n2Addr, n2Sub = "SSH RELAY", sshHost, "(Bastion / Server)"
		targetHost := tun.TargetHost
		if targetHost == "" {
			targetHost = "127.0.0.1"
		}
		n3Title, n3Addr, n3Sub = "REMOTE TARGET", fmt.Sprintf("%s:%d", targetHost, tun.TargetPort), "(Target Service)"
		if isActive {
			arr1Text = "══(SSH)══►"
			arr2Text = "──(TCP)──►"
		} else {
			arr1Text = "──(SSH)──►"
			arr2Text = "──(TCP)──►"
		}

	case config.TunnelTypeRemote:
		n1Title, n1Addr, n1Sub = "REMOTE LISTENER", fmt.Sprintf("%s:%d", bindHost, tun.BindPort), "(Public / Remote)"
		n2Title, n2Addr, n2Sub = "SSH GATEWAY", sshHost, "(Relay Server)"
		targetHost := tun.TargetHost
		if targetHost == "" {
			targetHost = "127.0.0.1"
		}
		n3Title, n3Addr, n3Sub = "LOCAL SERVICE", fmt.Sprintf("%s:%d", targetHost, tun.TargetPort), "(Your Local Dev)"
		if isActive {
			arr1Text = "◄══(SSH)══"
			arr2Text = "◄──(TCP)──"
		} else {
			arr1Text = "◄──(SSH)══"
			arr2Text = "◄──(TCP)──"
		}

	case config.TunnelTypeDynamic:
		n1Title, n1Addr, n1Sub = "SOCKS5 PROXY", fmt.Sprintf("%s:%d", bindHost, tun.BindPort), "(Browser / Apps)"
		n2Title, n2Addr, n2Sub = "SSH FORWARDER", sshHost, "(Proxy Router)"
		n3Title, n3Addr, n3Sub = "DYNAMIC TARGET", "Dynamic (*:*)", "(VPC Network)"
		if isActive {
			arr1Text = "═(SOCKS5)═►"
			arr2Text = "──(TCP)──►"
		} else {
			arr1Text = "─(SOCKS5)─►"
			arr2Text = "──(TCP)──►"
		}
	}

	// Calculate exact widths for 3 boxes and 2 arrows
	nodeW := 22
	availContent := boxWidth - 4 // inner width without outer borders
	arrowW := (availContent - (nodeW * 3)) / 2
	if arrowW < 10 {
		arrowW = 10
		nodeW = (availContent - (arrowW * 2)) / 3
	}

	// Construct Node Box Lines
	boxTop := func(w int, col lipgloss.Color) string {
		return lipgloss.NewStyle().Foreground(col).Render("╭" + strings.Repeat("─", w-2) + "╮")
	}
	boxBottom := func(w int, col lipgloss.Color) string {
		return lipgloss.NewStyle().Foreground(col).Render("╰" + strings.Repeat("─", w-2) + "╯")
	}
	boxRow := func(text string, w int, borderCol lipgloss.Color, textCol lipgloss.Color, bold bool) string {
		content := padCenter(text, w-2)
		styledContent := lipgloss.NewStyle().Foreground(textCol).Bold(bold).Render(content)
		b := lipgloss.NewStyle().Foreground(borderCol).Render("│")
		return b + styledContent + b
	}

	formatArrow := func(text string, width int, col lipgloss.Color) string {
		padded := padCenter(text, width)
		return lipgloss.NewStyle().Foreground(col).Bold(isActive).Render(padded)
	}

	arrowSpacer := strings.Repeat(" ", arrowW)

	row0 := boxTop(nodeW, n1Color) + arrowSpacer + boxTop(nodeW, n2Color) + arrowSpacer + boxTop(nodeW, n3Color)
	row1 := boxRow(n1Title, nodeW, n1Color, n1Color, true) + arrowSpacer + boxRow(n2Title, nodeW, n2Color, n2Color, true) + arrowSpacer + boxRow(n3Title, nodeW, n3Color, n3Color, true)
	row2 := boxRow(n1Addr, nodeW, n1Color, colorText, true) + formatArrow(arr1Text, arrowW, arrowColor) + boxRow(n2Addr, nodeW, n2Color, colorText, true) + formatArrow(arr2Text, arrowW, arrowColor) + boxRow(n3Addr, nodeW, n3Color, colorText, true)
	row3 := boxRow(n1Sub, nodeW, n1Color, colorSubText, false) + arrowSpacer + boxRow(n2Sub, nodeW, n2Color, colorSubText, false) + arrowSpacer + boxRow(n3Sub, nodeW, n3Color, colorSubText, false)
	row4 := boxBottom(nodeW, n1Color) + arrowSpacer + boxBottom(nodeW, n2Color) + arrowSpacer + boxBottom(nodeW, n3Color)

	diagramCanvas := fmt.Sprintf("%s\n%s\n%s\n%s\n%s", row0, row1, row2, row3, row4)

	// Top Title Pill
	boxTitle := fmt.Sprintf(" TOPOLOGY: %s (%s) ", strings.ToUpper(string(tun.Type)), tun.Name)
	headerPill := lipgloss.NewStyle().
		Background(colorPrimary).
		Foreground(colorText).
		Bold(true).
		Padding(0, 2).
		Render(boxTitle)

	// Outer border box
	outerBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPrimary).
		Padding(0, 1).
		Width(boxWidth).
		Align(lipgloss.Center).
		Render(diagramCanvas)

	return lipgloss.JoinVertical(lipgloss.Center, headerPill, outerBox)
}

func (tg *TunnelGraphView) renderMetricsDashboard(tun config.TunnelConfig, rt tunnel.RuntimeInfo, boxWidth int) string {
	statusBadge := lipgloss.NewStyle().Foreground(colorSubText).Render("○ INACTIVE")
	if rt.Status == tunnel.StatusActive {
		statusBadge = lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B")).Bold(true).Render("● ACTIVE")
	} else if rt.Status == tunnel.StatusError {
		statusBadge = lipgloss.NewStyle().Foreground(colorError).Bold(true).Render("✖ ERROR")
	}

	uptimeStr := "-"
	if rt.Status == tunnel.StatusActive && !rt.StartedAt.IsZero() {
		dur := time.Since(rt.StartedAt).Round(time.Second)
		uptimeStr = dur.String()
	}

	upArrow := lipgloss.NewStyle().Foreground(colorSecondary).Bold(true).Render("▲")
	downArrow := lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B")).Bold(true).Render("▼")
	trafficStr := fmt.Sprintf("%s %s  %s %s", upArrow, formatBytes(rt.BytesOut), downArrow, formatBytes(rt.BytesIn))
	connsStr := fmt.Sprintf("%d active", rt.ActiveConns)

	// Dynamic column distribution to give ample room to traffic numbers
	availW := boxWidth - 4 // inner width without outer borders
	dividersW := 3         // 3 divider bars '│'
	contentW := availW - dividersW

	colStatusW := 18
	colConnsW := 16
	colUptimeW := 18
	colTrafficW := contentW - colStatusW - colConnsW - colUptimeW
	if colTrafficW < 32 {
		colTrafficW = 32
	}

	formatCell := func(label string, val string, w int) string {
		l := lipgloss.NewStyle().Foreground(colorSubText).Bold(true).Render(label + ":")
		return padCenter(fmt.Sprintf("%s %s", l, val), w)
	}

	divider := lipgloss.NewStyle().Foreground(colorInactive).Render("│")

	metricsRow := formatCell("STATUS", statusBadge, colStatusW) +
		divider +
		formatCell("CONNS", connsStr, colConnsW) +
		divider +
		formatCell("TRAFFIC", trafficStr, colTrafficW) +
		divider +
		formatCell("UPTIME", uptimeStr, colUptimeW)

	// Error message if any
	errorMsg := ""
	if tg.errorMessage != "" {
		errorMsg = "\n" + lipgloss.NewStyle().Foreground(colorError).Bold(true).Render("Error: "+tg.errorMessage)
	} else if rt.Status == tunnel.StatusError && rt.LastError != "" {
		errorMsg = "\n" + lipgloss.NewStyle().Foreground(colorError).Bold(true).Render("Error: "+rt.LastError)
	}

	dashboardContent := fmt.Sprintf("%s%s", metricsRow, errorMsg)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorInactive).
		Background(lipgloss.Color("236")).
		Padding(0, 1).
		Width(boxWidth).
		Align(lipgloss.Center).
		Render(dashboardContent)
}

func (tg *TunnelGraphView) updateViewportContent(pkts []tunnel.TrafficPacket) {
	if len(pkts) == 0 {
		tg.viewport.SetContent(lipgloss.NewStyle().Foreground(colorSubText).Render(
			"\n  Waiting for traffic on this tunnel...\n  Connect your client (e.g. VNC, DB, HTTP) to stream raw packets live.\n",
		))
		return
	}

	var sb strings.Builder
	for _, p := range pkts {
		timeStr := p.Timestamp.Format("15:04:05.000")
		dirColor := colorSecondary
		dirIcon := "──►"
		if p.Direction == tunnel.DirRemoteToLocal {
			dirColor = lipgloss.Color("#50FA7B")
			dirIcon = "◄──"
		}

		hdr := lipgloss.NewStyle().Foreground(dirColor).Bold(true).Render(
			fmt.Sprintf("%s [%s] %s (%d bytes)", dirIcon, timeStr, p.Direction, p.Length),
		)
		sb.WriteString(hdr + "\n")

		if tg.mode == ViewModeHexDump {
			sb.WriteString(formatHexDump(p.Data))
		} else {
			sb.WriteString(formatASCIIStream(p.Data))
		}
		sb.WriteString("\n")
	}

	tg.viewport.SetContent(sb.String())
}

func (tg *TunnelGraphView) renderTrafficInspector(tun config.TunnelConfig, rt tunnel.RuntimeInfo, boxWidth int) string {
	modeLabel := "HEX + ASCII DUMP"
	if tg.mode == ViewModeASCII {
		modeLabel = "ASCII TEXT STREAM"
	}

	pauseIndicator := ""
	if tg.paused {
		pauseIndicator = " [PAUSED]"
	}

	copiedBadge := ""
	if tg.copied {
		copiedBadge = lipgloss.NewStyle().
			Background(lipgloss.Color("#50FA7B")).
			Foreground(lipgloss.Color("#000000")).
			Bold(true).
			Padding(0, 1).
			Render(" [✓ COPIED TO CLIPBOARD] ")
	}

	headerPill := lipgloss.NewStyle().
		Background(colorSecondary).
		Foreground(colorText).
		Bold(true).
		Padding(0, 2).
		Render(fmt.Sprintf(" LIVE TRAFFIC INSPECTOR: %s (%s)%s ", tun.Name, modeLabel, pauseIndicator))

	if copiedBadge != "" {
		headerPill = lipgloss.JoinHorizontal(lipgloss.Center, headerPill, "  ", copiedBadge)
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorSecondary).
		Padding(0, 1).
		Width(boxWidth)

	viewportBox := boxStyle.Render(tg.viewport.View())

	return lipgloss.JoinVertical(lipgloss.Center, headerPill, viewportBox)
}

func formatHexDump(data []byte) string {
	var sb strings.Builder
	for i := 0; i < len(data); i += 16 {
		// Offset
		fmt.Fprintf(&sb, "  %08x  ", i)

		// Hex bytes
		for j := 0; j < 16; j++ {
			if i+j < len(data) {
				fmt.Fprintf(&sb, "%02x ", data[i+j])
			} else {
				sb.WriteString("   ")
			}
			if j == 7 {
				sb.WriteString(" ")
			}
		}

		// ASCII representation
		sb.WriteString(" |")
		for j := 0; j < 16 && i+j < len(data); j++ {
			b := data[i+j]
			if b >= 32 && b <= 126 {
				sb.WriteByte(b)
			} else {
				sb.WriteByte('.')
			}
		}
		sb.WriteString("|\n")
	}
	return sb.String()
}

func formatASCIIStream(data []byte) string {
	var sb strings.Builder
	sb.WriteString("  ")
	for _, b := range data {
		if b == '\n' {
			sb.WriteString("\n  ")
		} else if b == '\r' {
			// skip CR
		} else if b >= 32 && b <= 126 {
			sb.WriteByte(b)
		} else {
			sb.WriteByte('.')
		}
	}
	sb.WriteString("\n")
	return sb.String()
}
