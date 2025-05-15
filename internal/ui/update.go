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
		if m.connectionList != nil {
			m.connectionList.SetWidth(m.width)
			m.connectionList.SetHeight(m.height - 4)
		}
		if activeComponent := m.getActiveComponent(); activeComponent != nil {
			model, cmd := activeComponent.Update(msg)
			return m, m.handleComponentResult(model, cmd)
		}

	case tea.KeyMsg:
		if m.state == StateConnectionList && m.connectionList != nil {
			listModel := m.connectionList.List()
			if listModel != nil && listModel.FilterState() == list.Filtering {
				newList, cmd := listModel.Update(msg)
				*listModel = newList
				return m, cmd
			}

			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
				m.state = StateSelectStorage
				return m, m.storageSelect.Init()

			case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c"))):
				return m, tea.Quit

			case msg.String() == "a":
				m.connectionForm = components.NewConnectionForm(nil)
				m.state = StateAddConnection
				return m, m.connectionForm.Init()

			case msg.String() == "e":
				selectedItem := m.connectionList.HighlightedConnection()
				if selectedItem != nil {
					m.connectionForm = components.NewConnectionForm(selectedItem)
					m.state = StateEditConnection
					return m, m.connectionForm.Init()
				}

			case msg.String() == "d":
				selectedItem := m.connectionList.HighlightedConnection()
				if selectedItem != nil && m.configManager != nil {
					err := m.configManager.DeleteConnection(selectedItem.ID)
					if err != nil {
						m.errorMessage = err.Error()
					} else {
						m.LoadConnections()
						m.connectionList.Reset()
					}
					return m, nil
				}
			}
		}
	}

	// Default: pass message to active component
	if activeComponent := m.getActiveComponent(); activeComponent != nil {
		model, cmd := activeComponent.Update(msg)
		return m, m.handleComponentResult(model, cmd)
	}

	return m, cmd
}
