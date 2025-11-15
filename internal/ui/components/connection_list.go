package components

import (
	"fmt"
	"github.com/eugeniofciuvasile/ssh-x-term/internal/config"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle      = lipgloss.NewStyle().MarginLeft(2)
	paginationStyle = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle       = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
)

type ToggleOpenInNewTerminalMsg struct{}

type connectionItem struct {
	connection config.SSHConnection
}

func (i connectionItem) FilterValue() string { return i.connection.Name }

func (i connectionItem) Title() string { return i.connection.Name }

func (i connectionItem) Description() string {
	port := ""
	if i.connection.Port != 22 && i.connection.Port != 0 {
		port = fmt.Sprintf(":%d", i.connection.Port)
	}
	return fmt.Sprintf("%s@%s%s", i.connection.Username, i.connection.Host, port)
}

type ConnectionList struct {
	list              list.Model
	connections       []config.SSHConnection
	selectedConn      *config.SSHConnection
	highlightedConn   *config.SSHConnection
	openInNewTerminal bool
}

func NewConnectionList(connections []config.SSHConnection) *ConnectionList {
	items := make([]list.Item, len(connections))
	for i, conn := range connections {
		items[i] = connectionItem{connection: conn}
	}
	l := list.New(items, list.NewDefaultDelegate(), 60, 20)
	if config.IsTmuxAvailable {
		l.Title = "SSH Connections - Toggle open in new terminal [x]"
	} else {
		l.Title = "SSH Connections - Toggle open in new terminal [ ]"
	}
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	l.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add connection")),
			key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit connection")),
			key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete connection")),
			key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "toggle open in new terminal")),
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
		openInNewTerminal: config.IsTmuxAvailable,
	}
}

func sendRefresh() tea.Cmd {
	return tea.Tick(time.Millisecond, func(t time.Time) tea.Msg {
		return ToggleOpenInNewTerminalMsg{}
	})
}

func (cl *ConnectionList) Init() tea.Cmd { return nil }

func (cl *ConnectionList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case ToggleOpenInNewTerminalMsg:
		return cl, nil

	case tea.WindowSizeMsg:
		cl.SetSize(msg.Width, msg.Height)
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
		case msg.String() == "o":
			cl.openInNewTerminal = !cl.openInNewTerminal

			checkboxStr := "[ ]"
			if cl.openInNewTerminal {
				checkboxStr = "[x]"
			}
			cl.list.Title = fmt.Sprintf("SSH Connections - Toggle open in new terminal %s", checkboxStr)

			return cl, sendRefresh()
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
	if len(cl.connections) == 0 {
		return fmt.Sprintf("\n%s\n\n  No connections found. Press 'a' to add a connection.\n\n", titleStyle.Render("SSH Connections"))
	}
	return cl.list.View()
}

func (cl *ConnectionList) SelectedConnection() *config.SSHConnection { return cl.selectedConn }

func (cl *ConnectionList) HighlightedConnection() *config.SSHConnection {
	return cl.highlightedConn
}

func (cl *ConnectionList) OpenInNewTerminal() bool { return cl.openInNewTerminal }

func (cl *ConnectionList) ToggleOpenInNewTerminal() {
	cl.openInNewTerminal = !cl.openInNewTerminal
	checkboxStr := "[ ]"
	if cl.openInNewTerminal {
		checkboxStr = "[x]"
	}
	cl.list.Title = fmt.Sprintf("SSH Connections - Toggle open in new terminal %s", checkboxStr)
}

func (cl *ConnectionList) SetConnections(connections []config.SSHConnection) {
	cl.connections = connections
	items := make([]list.Item, len(connections))
	for i, conn := range connections {
		items[i] = connectionItem{connection: conn}
	}
	cl.list.SetItems(items)
}

func (cl *ConnectionList) List() *list.Model { return &cl.list }

func (cl *ConnectionList) Reset() {
	cl.selectedConn = nil
	cl.list.Select(0)
	if len(cl.connections) > 0 {
		cl.highlightedConn = &cl.connections[0]
	} else {
		cl.highlightedConn = nil
	}
}

func (cl *ConnectionList) SetSize(width, height int) {
	if width <= 0 {
		width = 60
	}
	if height <= 0 {
		height = 20
	}
	cl.list.SetWidth(width)
	cl.list.SetHeight(height - 4)
}
