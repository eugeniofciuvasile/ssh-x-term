package components

import (
	"fmt"

	"ssh-x-term/internal/config"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
)

// connectionItem represents an item in the connection list
type connectionItem struct {
	connection config.SSHConnection
}

// FilterValue implements list.Item interface
func (i connectionItem) FilterValue() string {
	return i.connection.Name
}

// Title returns the item title for display
func (i connectionItem) Title() string {
	return i.connection.Name
}

// Description returns the item description for display
func (i connectionItem) Description() string {
	port := ""
	if i.connection.Port != 22 {
		port = fmt.Sprintf(":%d", i.connection.Port)
	}
	return fmt.Sprintf("%s@%s%s", i.connection.Username, i.connection.Host, port)
}

// ConnectionList is a component for listing SSH connections
type ConnectionList struct {
	list              list.Model
	connections       []config.SSHConnection
	selectedConn      *config.SSHConnection
	highlightedConn   *config.SSHConnection
	openInNewTerminal bool
}

// NewConnectionList creates a new connection list component
func NewConnectionList(connections []config.SSHConnection) *ConnectionList {
	// Set up list items
	items := make([]list.Item, len(connections))
	for i, conn := range connections {
		items[i] = connectionItem{connection: conn}
	}

	// Set up list
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "SSH Connections"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	// Set up keybindings
	l.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(
				key.WithKeys("a"),
				key.WithHelp("a", "add connection"),
			),
			key.NewBinding(
				key.WithKeys("e"),
				key.WithHelp("e", "edit connection"),
			),
			key.NewBinding(
				key.WithKeys("d"),
				key.WithHelp("d", "delete connection"),
			),
			key.NewBinding(
				key.WithKeys("o"),
				key.WithHelp("o", "toggle open in new terminal"),
			),
		}
	}

	var highlighted *config.SSHConnection
	if len(connections) > 0 {
		highlighted = &connections[0]
	}

	return &ConnectionList{
		list:              l,
		connections:       connections,
		highlightedConn:   highlighted,
		openInNewTerminal: false, // Default state of the checkbox
	}
}

// Init initializes the component
func (cl *ConnectionList) Init() tea.Cmd {
	return nil
}

// Update handles component updates
func (cl *ConnectionList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		cl.list.SetWidth(msg.Width)
		cl.list.SetHeight(msg.Height - 4) // Leave room for help and status
		return cl, nil

	case tea.KeyMsg:
		// Check if key message is handled by the list
		if cl.list.FilterState() == list.Filtering {
			newList, cmd := cl.list.Update(msg)
			cl.list = newList
			return cl, cmd
		}

		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			selectedItem := cl.list.SelectedItem()
			if selectedItem != nil {
				connItem, ok := selectedItem.(connectionItem)
				if ok {
					cl.selectedConn = &connItem.connection
					return cl, nil
				}
			}
		case msg.String() == "o":
			// Toggle checkbox when "o" is pressed
			cl.openInNewTerminal = !cl.openInNewTerminal

			// Update the title to reflect checkbox state
			checkboxStr := "[ ]"
			if cl.openInNewTerminal {
				checkboxStr = "[x]"
			}
			cl.list.Title = fmt.Sprintf("SSH Connections - ToggleOpenInNewTerminal %s", checkboxStr)
		}
	}

	// Handle other messages with the list component
	newList, cmd := cl.list.Update(msg)
	cl.list = newList

	// Update highlighted connection
	if item := cl.list.SelectedItem(); item != nil {
		if connItem, ok := item.(connectionItem); ok {
			cl.highlightedConn = &connItem.connection
		}
	} else {
		cl.highlightedConn = nil
	}

	return cl, cmd
}

// View renders the component
func (cl *ConnectionList) View() string {
	checkboxStr := "[ ]"
	if cl.openInNewTerminal {
		checkboxStr = "[x]"
	}
	if len(cl.connections) == 0 {
		return fmt.Sprintf("\n%s\n\n  No connections found. Press 'a' to add a connection.\n\n", titleStyle.Render("SSH Connections"))
	}
	// Display the checkbox next to the title
	header := fmt.Sprintf("%s SSH Connections", checkboxStr)

	// Return the list and the checkbox in the view
	return fmt.Sprintf("%s\n\n%s", header, cl.list.View())
}

// SelectedConnection returns the selected connection, if any
func (cl *ConnectionList) SelectedConnection() *config.SSHConnection {
	return cl.selectedConn
}

// HighlightedConnection returns the currently highlighted connection
func (cl *ConnectionList) HighlightedConnection() *config.SSHConnection {
	return cl.highlightedConn
}

// OpenInNewTerminal returns whether to open the connection in a new terminal
func (cl *ConnectionList) OpenInNewTerminal() bool {
	return cl.openInNewTerminal
}

// Set OpenInNewTerminal sets whether to open the connection in a new terminal
func (cl *ConnectionList) ToggleOpenInNewTerminal() {
	cl.openInNewTerminal = !cl.openInNewTerminal
}

// SetConnections updates the list of connections
func (cl *ConnectionList) SetConnections(connections []config.SSHConnection) {
	cl.connections = connections
	items := make([]list.Item, len(connections))
	for i, conn := range connections {
		items[i] = connectionItem{connection: conn}
	}
	cl.list.SetItems(items)
}

// List returns the underlying list model
func (cl *ConnectionList) List() *list.Model {
	return &cl.list
}

// Reset clears the selected connection
func (cl *ConnectionList) Reset() {
	cl.selectedConn = nil
	cl.list.Select(0)

	if len(cl.connections) > 0 {
		cl.highlightedConn = &cl.connections[0]
	} else {
		cl.highlightedConn = nil
	}
}
