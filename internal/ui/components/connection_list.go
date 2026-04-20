package components

import (
	"fmt"
	"io"
	"slices"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/eugeniofciuvasile/ssh-x-term/internal/config"
)

type ToggleOpenInNewTerminalMsg struct{}

type DeleteConnectionMsg struct {
	Connection config.SSHConnection
}

type RenameConnectionMsg struct {
	Connection config.SSHConnection
	NewName    string
}

type TogglePinnedMsg struct {
	Connection config.SSHConnection
}

type MoveConnectionUpMsg struct {
	Connection config.SSHConnection
}

type MoveConnectionDownMsg struct {
	Connection config.SSHConnection
}

type connectionItem struct {
	connection config.SSHConnection
}

func (i connectionItem) FilterValue() string { return i.connection.Name + " " + i.connection.Host }

// connectionDelegate handles the rendering of each list item with dynamic widths
type connectionDelegate struct {
	nameWidth int
	hostWidth int
	userWidth int
	portWidth int
	authWidth int
}

func (d connectionDelegate) Height() int { return 1 }

func (d connectionDelegate) Spacing() int { return 0 }

func (d connectionDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (d connectionDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(connectionItem)
	if !ok {
		return
	}

	conn := i.connection

	// Determine Authorization Method string
	authMethod := "Agent"
	if conn.UsePassword {
		authMethod = "Password"
	} else if conn.KeyFile != "" {
		authMethod = "Key File"
	}

	// Format columns using the dynamic widths stored in the delegate
	name := conn.Name
	if conn.Pinned {
		name = "📌 " + name
	}
	name = truncate(name, d.nameWidth)
	host := truncate(conn.Host, d.hostWidth)
	user := truncate(conn.Username, d.userWidth)
	port := truncate(fmt.Sprintf("%d", conn.Port), d.portWidth)
	auth := truncate(authMethod, d.authWidth)

	// Render row
	var style lipgloss.Style
	if index == m.Index() {
		style = selectedItemStyle
	} else {
		style = itemStyle
	}

	// Build the row string using Lipgloss for alignment
	row := lipgloss.JoinHorizontal(lipgloss.Left,
		lipgloss.NewStyle().Width(d.nameWidth).Render(name),
		lipgloss.NewStyle().Width(d.hostWidth).Render(host),
		lipgloss.NewStyle().Width(d.userWidth).Render(user),
		lipgloss.NewStyle().Width(d.portWidth).Render(port),
		lipgloss.NewStyle().Width(d.authWidth).Render(auth),
	)

	fmt.Fprint(w, style.Render(row))
}

// Helper to truncate strings that are too long
func truncate(s string, max int) string {
	if max < 3 {
		return ""
	}
	if len(s) > max-1 {
		return s[:max-2] + "…"
	}
	return s
}

type ConnectionList struct {
	list              list.Model
	Connections       []config.SSHConnection
	selectedConn      *config.SSHConnection
	highlightedConn   *config.SSHConnection
	openInNewTerminal bool

	// Delete confirmation dialog
	showDeleteConfirm bool
	deleteConfirm     *DeleteConfirmation
	pendingDelete     *config.SSHConnection

	// Password modal
	showPasswordModal bool
	passwordModal     *PasswordModal

	// Rename modal
	showRenameModal bool
	renameModal     *RenameModal

	// layout stores the current column widths for header rendering
	layout connectionDelegate
}

func sortConnections(connections []config.SSHConnection) []config.SSHConnection {
	sorted := make([]config.SSHConnection, len(connections))
	copy(sorted, connections)
	slices.SortStableFunc(sorted, func(a, b config.SSHConnection) int {
		if a.Pinned && !b.Pinned {
			return -1
		}
		if !a.Pinned && b.Pinned {
			return 1
		}
		// If both have same pinned status, sort by order
		if a.Order < b.Order {
			return -1
		}
		if a.Order > b.Order {
			return 1
		}
		// If order is same, sort by name
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		return 0
	})
	return sorted
}

func NewConnectionList(connections []config.SSHConnection) *ConnectionList {
	sorted := sortConnections(connections)
	items := make([]list.Item, len(sorted))
	for i, conn := range sorted {
		items[i] = connectionItem{connection: conn}
	}

	// Initial delegate with default widths (will be resized immediately)
	defaultDelegate := connectionDelegate{
		nameWidth: 20, hostWidth: 20, userWidth: 15, portWidth: 8, authWidth: 10,
	}

	l := list.New(items, defaultDelegate, 80, 20)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)
	l.Styles.PaginationStyle = paginationStyle

	var highlighted *config.SSHConnection
	if len(sorted) > 0 {
		highlighted = &sorted[0]
	}

	cl := &ConnectionList{
		list:              l,
		Connections:       sorted,
		highlightedConn:   highlighted,
		openInNewTerminal: config.IsTmuxAvailable,
		layout:            defaultDelegate,
	}

	// Trigger an initial layout calculation
	cl.SetSize(80, 20)

	return cl
}

func (cl *ConnectionList) Init() tea.Cmd { return nil }

func (cl *ConnectionList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// If delete confirmation is showing, delegate to it
	if cl.showDeleteConfirm && cl.deleteConfirm != nil {
		var confirmModel tea.Model
		confirmModel, cmd = cl.deleteConfirm.Update(msg)
		cl.deleteConfirm = confirmModel.(*DeleteConfirmation)

		// Check if user confirmed or canceled
		if cl.deleteConfirm.IsConfirmed() {
			// User confirmed - send delete message
			cl.showDeleteConfirm = false
			if cl.pendingDelete != nil {
				deleteMsg := DeleteConnectionMsg{Connection: *cl.pendingDelete}
				cl.pendingDelete = nil
				return cl, func() tea.Msg { return deleteMsg }
			}
		} else if cl.deleteConfirm.IsCanceled() {
			// User canceled - close dialog
			cl.showDeleteConfirm = false
			cl.pendingDelete = nil
			cl.deleteConfirm = nil
		}

		return cl, cmd
	}

	// If password modal is showing, delegate to it
	if cl.showPasswordModal && cl.passwordModal != nil {
		var modalModel tea.Model
		modalModel, cmd = cl.passwordModal.Update(msg)
		cl.passwordModal = modalModel.(*PasswordModal)

		if cl.passwordModal.IsCanceled() {
			cl.showPasswordModal = false
			cl.passwordModal = nil
		}

		return cl, cmd
	}

	// If rename modal is showing, delegate to it
	if cl.showRenameModal && cl.renameModal != nil {
		var modalModel tea.Model
		modalModel, cmd = cl.renameModal.Update(msg)
		cl.renameModal = modalModel.(*RenameModal)

		if cl.renameModal.IsConfirmed() {
			cl.showRenameModal = false
			if cl.highlightedConn != nil {
				renameMsg := RenameConnectionMsg{
					Connection: *cl.highlightedConn,
					NewName:    cl.renameModal.Value(),
				}
				cl.renameModal = nil
				return cl, func() tea.Msg { return renameMsg }
			}
			cl.renameModal = nil
		} else if cl.renameModal.IsCanceled() {
			cl.showRenameModal = false
			cl.renameModal = nil
		}

		return cl, cmd
	}

	switch msg := msg.(type) {
	case ToggleOpenInNewTerminalMsg:
		return cl, nil

	case tea.WindowSizeMsg:
		cl.SetSize(msg.Width, msg.Height)
		// Also update delete confirmation if it exists
		if cl.deleteConfirm != nil {
			cl.deleteConfirm.SetSize(msg.Width, msg.Height)
		}
		// Also update password modal if it exists
		if cl.passwordModal != nil {
			cl.passwordModal.SetSize(msg.Width, msg.Height)
		}
		// Also update rename modal if it exists
		if cl.renameModal != nil {
			cl.renameModal.SetSize(msg.Width, msg.Height)
		}
		return cl, nil

	case tea.KeyMsg:
		if cl.list.FilterState() == list.Filtering {
			newList, cmd := cl.list.Update(msg)
			cl.list = newList
			return cl, cmd
		}

		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			if selectedItem := cl.list.SelectedItem(); selectedItem != nil {
				if connItem, ok := selectedItem.(connectionItem); ok {
					cl.selectedConn = &connItem.connection
					return cl, nil
				}
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("d", "D"))):
			// Show delete confirmation for highlighted connection
			if cl.highlightedConn != nil {
				cl.pendingDelete = cl.highlightedConn
				cl.deleteConfirm = NewDeleteConfirmation(cl.highlightedConn.Name)
				cl.deleteConfirm.SetSize(cl.list.Width(), cl.list.Height())
				cl.showDeleteConfirm = true
				return cl, nil
			}
		}
	}

	newList, cmd := cl.list.Update(msg)
	cl.list = newList
	if item := cl.list.SelectedItem(); item != nil {
		if connItem, ok := item.(connectionItem); ok {
			cl.highlightedConn = &connItem.connection
		}
	} else {
		cl.highlightedConn = nil
	}
	return cl, cmd
}

func (cl *ConnectionList) View() string {
	if len(cl.Connections) == 0 {
		// Simplified message since global title handles context
		return "\n\n  No connections found. Press 'a' to add a connection.\n\n"
	}

	// Note: We no longer render the Title here because the main UI View() handles it.
	// We only render the Table Headers and the List itself.

	// Construct Table Headers using the DYNAMIC layout widths
	headers := lipgloss.NewStyle().PaddingLeft(2).Render(
		lipgloss.JoinHorizontal(lipgloss.Left,
			headerStyle.Width(cl.layout.nameWidth).Render("Name"),
			headerStyle.Width(cl.layout.hostWidth).Render("Host"),
			headerStyle.Width(cl.layout.userWidth).Render("User"),
			headerStyle.Width(cl.layout.portWidth).Render("Port"),
			headerStyle.Width(cl.layout.authWidth).Render("Auth Method"),
		),
	)

	listView := lipgloss.JoinVertical(lipgloss.Left,
		headers,
		cl.list.View(),
	)

	// If delete confirmation is showing, overlay it on top
	if cl.showDeleteConfirm && cl.deleteConfirm != nil {
		// Render the list view as background, then overlay the confirmation dialog
		confirmView := cl.deleteConfirm.View()
		// Use Place to overlay the confirmation on top of the list
		return lipgloss.Place(
			cl.list.Width(),
			cl.list.Height(),
			lipgloss.Center,
			lipgloss.Center,
			confirmView,
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceForeground(lipgloss.Color("0")),
		)
	}

	// If password modal is showing, overlay it on top
	if cl.showPasswordModal && cl.passwordModal != nil {
		modalView := cl.passwordModal.View()
		return lipgloss.Place(
			cl.list.Width(),
			cl.list.Height(),
			lipgloss.Center,
			lipgloss.Center,
			modalView,
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceForeground(lipgloss.Color("0")),
		)
	}

	// If rename modal is showing, overlay it on top
	if cl.showRenameModal && cl.renameModal != nil {
		modalView := cl.renameModal.View()
		return lipgloss.Place(
			cl.list.Width(),
			cl.list.Height(),
			lipgloss.Center,
			lipgloss.Center,
			modalView,
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceForeground(lipgloss.Color("0")),
		)
	}

	return listView
}

func (cl *ConnectionList) SelectedConnection() *config.SSHConnection { return cl.selectedConn }
func (cl *ConnectionList) HighlightedConnection() *config.SSHConnection {
	return cl.highlightedConn
}
func (cl *ConnectionList) OpenInNewTerminal() bool { return cl.openInNewTerminal }

func (cl *ConnectionList) ToggleOpenInNewTerminal() {
	cl.openInNewTerminal = !cl.openInNewTerminal
}

func (cl *ConnectionList) IsShowingDeleteConfirm() bool {
	return cl.showDeleteConfirm
}

func (cl *ConnectionList) IsShowingPasswordModal() bool {
	return cl.showPasswordModal
}

func (cl *ConnectionList) IsShowingRenameModal() bool {
	return cl.showRenameModal
}

func (cl *ConnectionList) ShowPassword(conn config.SSHConnection) {
	entries := []PasswordEntry{}
	if conn.Password != "" {
		label := "Password"
		if !conn.UsePassword {
			label = "Key Passphrase"
		}
		entries = append(entries, PasswordEntry{Label: label, Password: conn.Password})
	}
	if conn.SudoPassword != "" {
		entries = append(entries, PasswordEntry{Label: "Sudo Password", Password: conn.SudoPassword})
	}

	if len(entries) == 0 {
		cl.passwordModal = NewPasswordModal("")
	} else if len(entries) == 1 {
		cl.passwordModal = NewPasswordModal(entries[0].Password)
		// Set the label correctly even for single entry
		cl.passwordModal = NewMultiPasswordModal(entries)
	} else {
		cl.passwordModal = NewMultiPasswordModal(entries)
	}

	cl.passwordModal.SetSize(cl.list.Width(), cl.list.Height())
	cl.showPasswordModal = true
}

func (cl *ConnectionList) ShowRename() {
	if cl.highlightedConn == nil {
		return
	}
	cl.renameModal = NewRenameModal(cl.highlightedConn.Name, cl.highlightedConn.Host)
	cl.renameModal.SetSize(cl.list.Width(), cl.list.Height())
	cl.showRenameModal = true
}

func (cl *ConnectionList) Rename() tea.Cmd {
	if cl.highlightedConn == nil {
		return nil
	}
	cl.ShowRename()
	return nil
}

func (cl *ConnectionList) SetConnections(connections []config.SSHConnection) {
	sorted := sortConnections(connections)
	cl.Connections = sorted
	items := make([]list.Item, len(sorted))
	for i, conn := range sorted {
		items[i] = connectionItem{connection: conn}
	}
	cl.list.SetItems(items)
}

func (cl *ConnectionList) TogglePinned() tea.Cmd {
	if cl.highlightedConn == nil {
		return nil
	}
	conn := *cl.highlightedConn
	return func() tea.Msg { return TogglePinnedMsg{Connection: conn} }
}

func (cl *ConnectionList) MoveUp() tea.Cmd {
	if cl.highlightedConn == nil {
		return nil
	}
	conn := *cl.highlightedConn
	return func() tea.Msg { return MoveConnectionUpMsg{Connection: conn} }
}

func (cl *ConnectionList) MoveDown() tea.Cmd {
	if cl.highlightedConn == nil {
		return nil
	}
	conn := *cl.highlightedConn
	return func() tea.Msg { return MoveConnectionDownMsg{Connection: conn} }
}

func (cl *ConnectionList) List() *list.Model { return &cl.list }

func (cl *ConnectionList) Reset() {
	cl.selectedConn = nil
	cl.list.Select(0)
	if len(cl.Connections) > 0 {
		cl.highlightedConn = &cl.Connections[0]
	} else {
		cl.highlightedConn = nil
	}
}

// SetSize recalculates the column widths based on available terminal space
func (cl *ConnectionList) SetSize(width, height int) {
	// Calculate height deduction:
	// 1. Global Header (managed by parent) -> 1 line
	// 2. Global Footer (managed by parent) -> 1 line
	// 3. Table Header (managed here) -> 1 line
	// Total overhead = 3 lines
	listHeight := max(height-2, 1)
	cl.list.SetHeight(listHeight)
	cl.list.SetWidth(width)

	// Calculate column widths
	cl.recalculateTableLayout(width)
}

func (cl *ConnectionList) recalculateTableLayout(totalWidth int) {
	// Subtract list padding (default is usually 2 for left padding)
	availableWidth := max(totalWidth-4, 0)

	// Define fixed or minimum widths
	const (
		minPortWidth = 12
		minAuthWidth = 16
	)

	// Strategy:
	// Port and Auth get fixed sizes if space is tight, or small slice of total
	portW := minPortWidth
	authW := minAuthWidth

	remaining := availableWidth - portW - authW

	// Distribute remaining space:
	// Name: 40%, Host: 35%, User: 25%
	nameW := int(float64(remaining) * 0.40)
	hostW := int(float64(remaining) * 0.35)
	userW := remaining - nameW - hostW // Give remainder to user to avoid rounding gaps

	// Ensure minimums
	if nameW < 10 {
		nameW = 10
	}
	if hostW < 10 {
		hostW = 10
	}
	if userW < 5 {
		userW = 5
	}

	// Create new delegate with calculated widths
	newLayout := connectionDelegate{
		nameWidth: nameW,
		hostWidth: hostW,
		userWidth: userW,
		portWidth: portW,
		authWidth: authW,
	}

	// Store layout for Header rendering
	cl.layout = newLayout

	// Update the list's delegate so rows render with new widths
	cl.list.SetDelegate(newLayout)
}
