package ui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/eugeniofciuvasile/ssh-x-term/internal/ui/components"
)

// Update handles updates to the UI model
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		// Only update spinner if loading
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case LoadConnectionsFinishedMsg:
		m.loading = false
		if msg.Err != nil {
			m.errorMessage = msg.Err.Error()
			return m, nil
		}
		m.connectionList = components.NewConnectionList(msg.Connections)
		m.connectionList.SetSize(m.width, m.listHeight())
		m.state = StateConnectionList
		return m, nil

	case BitwardenStatusMsg:
		m.loading = false
		if msg.Err != nil {
			m.errorMessage = msg.Err.Error()
			m.state = StateSelectStorage
			return m, nil
		}
		if !msg.LoggedIn {
			m.bitwardenForm = components.NewBitwardenConfigForm()
			m.state = StateBitwardenConfig
		} else if !msg.Unlocked {
			m.bitwardenUnlockForm = components.NewBitwardenUnlockForm()
			m.state = StateBitwardenUnlock
		} else {
			m.loading = true
			return m, tea.Batch(
				loadBitwardenOrganizationsCmd(m.bitwardenManager),
				m.spinner.Tick,
			)
		}
		return m, nil

	case BitwardenLoadOrganizationsMsg:
		m.loading = false
		if msg.Err != nil {
			m.errorMessage = msg.Err.Error()
			return m, nil
		}
		m.bitwardenOrganizationList = components.NewBitwardenOrganizationList(msg.Organizations)
		m.bitwardenOrganizationList.SetSize(m.width, m.listHeight())
		m.state = StateOrganizationSelect
		return m, nil

	case BitwardenLoadCollectionsMsg:
		m.loading = false
		if msg.Err != nil {
			m.errorMessage = msg.Err.Error()
			return m, nil
		}
		m.bitwardenCollectionList = components.NewBitwardenCollectionList(msg.Collections)
		m.bitwardenCollectionList.SetSize(m.width, m.listHeight())
		m.state = StateCollectionSelect
		return m, nil

	case BitwardenLoadConnectionsByCollectionMsg:
		m.loading = false
		if msg.Err != nil {
			m.errorMessage = msg.Err.Error()
			return m, nil
		}
		m.connectionList = components.NewConnectionList(msg.Connections)
		m.connectionList.SetSize(m.width, m.listHeight())
		m.state = StateConnectionList
		return m, nil

	case BitwardenLoginResultMsg:
		m.loading = false
		if !msg.Success || msg.Err != nil {
			if msg.Err != nil {
				m.bitwardenLoginForm.SetError(msg.Err.Error())
			} else {
				m.bitwardenLoginForm.SetError("Login failed")
			}
			m.formHasError = true
			m.bitwardenLoginForm.ResetSubmitted()
			m.bitwardenLoginForm = components.NewBitwardenLoginForm()
			return m, nil
		}
		m.formHasError = false
		m.storageBackend = m.bitwardenManager
		m.loading = true
		m.bitwardenLoginForm = nil
		return m, tea.Batch(
			loadBitwardenOrganizationsCmd(m.bitwardenManager),
			m.spinner.Tick,
		)

	case BitwardenUnlockResultMsg:
		m.loading = false
		if !msg.Success || msg.Err != nil {
			if msg.Err != nil {
				m.bitwardenUnlockForm.SetError(msg.Err.Error())
			} else {
				m.bitwardenUnlockForm.SetError("Unlock failed")
			}
			m.formHasError = true
			m.bitwardenUnlockForm.ResetSubmitted()
			m.bitwardenUnlockForm = components.NewBitwardenUnlockForm()
			return m, nil
		}
		m.formHasError = false
		m.storageBackend = m.bitwardenManager
		m.loading = true
		m.bitwardenUnlockForm = nil
		return m, tea.Batch(
			loadBitwardenOrganizationsCmd(m.bitwardenManager),
			m.spinner.Tick,
		)

	case components.ToggleOpenInNewTerminalMsg:
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if activeComponent := m.getActiveComponent(); activeComponent != nil {
			model, cmd := activeComponent.Update(msg)
			return m, m.handleComponentResult(model, cmd)
		}

	case tea.KeyMsg:
		if m.loading {
			if key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c"))) {
				return m, tea.Quit
			}
			return m, nil
		}
		if m.formHasError {
			if key.Matches(msg, key.NewBinding(key.WithKeys("esc"))) {
				switch m.state {
				case StateBitwardenLogin:
					m.bitwardenLoginForm = nil
					m.formHasError = false
					m.state = StateSelectStorage
					m.storageSelect = components.NewStorageSelect()
					return m, m.storageSelect.Init()
				case StateBitwardenUnlock:
					m.bitwardenUnlockForm = nil
					m.formHasError = false
					m.state = StateSelectStorage
					m.storageSelect = components.NewStorageSelect()
					return m, m.storageSelect.Init()
				}
			}
		}
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
					m.resetConnectionState()
					if m.state == StateSelectStorage {
						m.storageSelect = components.NewStorageSelect()
						return m, m.storageSelect.Init()
					}
					return m, nil
				case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c"))):
					return m, tea.Quit
				case msg.String() == "a":
					m.connectionForm = components.NewConnectionForm(nil)
					m.state = StateAddConnection
					return m, m.connectionForm.Init()
				case msg.String() == "e":
					if selectedItem := m.connectionList.HighlightedConnection(); selectedItem != nil {
						m.connectionForm = components.NewConnectionForm(selectedItem)
						m.state = StateEditConnection
						return m, m.connectionForm.Init()
					}
				case msg.String() == "d":
					if selectedItem := m.connectionList.HighlightedConnection(); selectedItem != nil && m.storageBackend != nil {
						m.loading = true
						if err := m.storageBackend.DeleteConnection(selectedItem.ID); err != nil {
							m.loading = false
							m.errorMessage = err.Error()
							return m, nil
						}
						cmd := m.ReloadConnections()
						m.connectionList.Reset()
						return m, cmd
					}
				case msg.String() == "o":
					if m.connectionList != nil {
						m.connectionList.ToggleOpenInNewTerminal()
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
					m.resetCollectionState()
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
					m.resetOrganizationState()
					return m, m.storageSelect.Init()
				case key.Matches(msg, key.NewBinding(key.WithKeys("o"))):
					return m, m.loadPersonalVaultConnections()
				}
			}
		}
	}

	// Default: pass message to active component
	if activeComponent := m.getActiveComponent(); activeComponent != nil {
		model, cmd := activeComponent.Update(msg)
		return m, m.handleComponentResult(model, cmd)
	}
	return m, nil
}
