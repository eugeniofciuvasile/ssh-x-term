package ui

import (
	"fmt"
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
		if activeComponent := m.getActiveComponent(); activeComponent != nil {
			model, cmd := activeComponent.Update(msg)
			return m, m.handleComponentResult(model, cmd)
		}

	case tea.KeyMsg:
		switch m.state {
		case StateConnectionList:
			if m.connectionList != nil {
				listModel := m.connectionList.List()
				if listModel != nil && listModel.FilterState() == list.Filtering {
					newList, cmd := listModel.Update(msg)
					*listModel = newList
					return m, cmd
				}
				switch {
				case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
					// Go back to collection select
					if m.connectionList != nil {
						m.connectionList.Reset()
					}
					if m.bitwardenCollectionList != nil {
						m.bitwardenCollectionList.Reset()
					}
					if m.bitwardenCollectionList == nil {
						m.state = StateSelectStorage
					} else {
						m.state = StateCollectionSelect
					}
					return m, nil
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
					if selectedItem != nil && m.storageBackend != nil {
						err := m.storageBackend.DeleteConnection(selectedItem.ID)
						if err != nil {
							m.errorMessage = err.Error()
						} else {
							m.ReloadConnections()
							m.connectionList.Reset()
						}
						return m, nil
					}
				}
			}

		case StateCollectionSelect:
			if m.bitwardenCollectionList != nil {
				listModel := m.bitwardenCollectionList.List()
				if listModel != nil && listModel.FilterState() == list.Filtering {
					newList, cmd := listModel.Update(msg)
					*listModel = newList
					return m, cmd
				}
				switch {
				case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
					// Go back to organization select
					if m.bitwardenCollectionList != nil {
						m.bitwardenCollectionList.Reset()
					}
					if m.bitwardenOrganizationList != nil {
						m.bitwardenOrganizationList.Reset()
					}
					m.state = StateOrganizationSelect
					return m, nil
				}
			}

		case StateOrganizationSelect:
			if m.bitwardenOrganizationList != nil {
				listModel := m.bitwardenOrganizationList.List()
				if listModel != nil && listModel.FilterState() == list.Filtering {
					newList, cmd := listModel.Update(msg)
					*listModel = newList
					return m, cmd
				}
				switch {
				case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
					// Go back to storage select
					if m.bitwardenOrganizationList != nil {
						m.bitwardenOrganizationList.Reset()
					}
					m.state = StateSelectStorage
					return m, m.storageSelect.Init()
				case key.Matches(msg, key.NewBinding(key.WithKeys("o"))):
					// Toggle open personal SSH connections
					m.bitwardenManager.SetPersonalVault(true)
					m.storageBackend = m.bitwardenManager
					if err := m.bitwardenManager.Load(); err != nil {
						m.errorMessage = fmt.Sprintf("Failed to load Bitwarden personal connections: %v", err)
					}
					m.connectionList = components.NewConnectionList(m.bitwardenManager.ListConnections())
					m.connectionList.SetSize(m.width, m.listHeight())
					m.state = StateConnectionList
					return m, m.connectionList.Init()
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
