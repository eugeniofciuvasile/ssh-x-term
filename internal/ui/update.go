package ui

import (
	"ssh-x-term/internal/ui/components"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// Update handles updates to the UI model
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Pass window size to active component
		if activeComponent := m.getActiveComponent(); activeComponent != nil {
			model, cmd := activeComponent.Update(msg)
			return m, m.handleComponentResult(model, cmd)
		}

	case tea.KeyMsg:
		listModel := m.connectionList.List()

		if listModel.FilterState() == list.Filtering {
			newList, cmd := listModel.Update(msg)
			*listModel = newList // update the pointer target
			return m, cmd
		}

		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c"))):
			return m, tea.Quit

		case m.state == StateConnectionList && msg.String() == "a":
			// Add new connection
			m.connectionForm = components.NewConnectionForm(nil)
			m.state = StateAddConnection
			return m, m.connectionForm.Init()

		case m.state == StateConnectionList && msg.String() == "e":
			// Edit selected connection
			selectedItem := m.connectionList.HighlightedConnection()
			if selectedItem != nil {
				m.connectionForm = components.NewConnectionForm(selectedItem)
				m.state = StateEditConnection
				return m, m.connectionForm.Init()
			}

		case m.state == StateConnectionList && msg.String() == "d":
			// Delete selected connection
			selectedItem := m.connectionList.HighlightedConnection()
			if selectedItem != nil {
				err := m.configManager.DeleteConnection(selectedItem.ID)
				if err != nil {
					m.errorMessage = err.Error()
				} else {
					m.LoadConnections()
					m.connectionList.Reset()
				}
				return m, nil
			}
			//
			// case m.state == StateConnectionList && msg.String() == "o":
			// 	// Mark next selected connection as open in a new terminal
			// 	m.connectionList.ToggleOpenInNewTerminal()
			// 	c.openCheckbox.Checked = c.openInNewTerminal
			// 	return m, nil
		}
	}

	// Pass message to active component
	if activeComponent := m.getActiveComponent(); activeComponent != nil {
		model, cmd := activeComponent.Update(msg)
		return m, m.handleComponentResult(model, cmd)
	}

	return m, cmd
}
