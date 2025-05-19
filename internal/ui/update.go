package ui

import (
	"ssh-x-term/internal/ui/components"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// Update handles updates to the UI model
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Update spinner regardless of loading state
	if spinnerCmd := m.updateSpinner(msg); spinnerCmd != nil {
		cmds = append(cmds, spinnerCmd)
	}

	// Handle async operation messages
	switch msg := msg.(type) {
	case LoadConnectionsFinishedMsg:
		m.loading = false
		if msg.Err != nil {
			m.errorMessage = msg.Err.Error()
			return m, tea.Batch(cmds...)
		}

		m.connectionList = components.NewConnectionList(msg.Connections)
		m.connectionList.SetSize(m.width, m.listHeight())
		m.state = StateConnectionList
		return m, tea.Batch(cmds...)

	case BitwardenStatusMsg:
		m.loading = false
		if msg.Err != nil {
			m.errorMessage = msg.Err.Error()
			m.state = StateSelectStorage
			return m, tea.Batch(cmds...)
		}

		if !msg.LoggedIn {
			m.bitwardenForm = components.NewBitwardenConfigForm()
			m.state = StateBitwardenConfig
		} else if !msg.Unlocked {
			m.bitwardenUnlockForm = components.NewBitwardenUnlockForm()
			m.state = StateBitwardenUnlock
		} else {
			m.loading = true
			cmd := loadBitwardenOrganizationsCmd(m.bitwardenManager)
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	case BitwardenLoadOrganizationsMsg:
		m.loading = false
		if msg.Err != nil {
			m.errorMessage = msg.Err.Error()
			return m, tea.Batch(cmds...)
		}

		m.bitwardenOrganizationList = components.NewBitwardenOrganizationList(msg.Organizations)
		m.bitwardenOrganizationList.SetSize(m.width, m.listHeight())
		m.state = StateOrganizationSelect
		return m, tea.Batch(cmds...)

	case BitwardenLoadCollectionsMsg:
		m.loading = false
		if msg.Err != nil {
			m.errorMessage = msg.Err.Error()
			return m, tea.Batch(cmds...)
		}

		m.bitwardenCollectionList = components.NewBitwardenCollectionList(msg.Collections)
		m.bitwardenCollectionList.SetSize(m.width, m.listHeight())
		m.state = StateCollectionSelect
		return m, tea.Batch(cmds...)

	case BitwardenLoadConnectionsByCollectionMsg:
		m.loading = false
		if msg.Err != nil {
			m.errorMessage = msg.Err.Error()
			return m, tea.Batch(cmds...)
		}

		m.connectionList = components.NewConnectionList(msg.Connections)
		m.connectionList.SetSize(m.width, m.listHeight())
		m.state = StateConnectionList
		return m, tea.Batch(cmds...)

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
			return m, tea.Batch(cmds...)
		}

		m.formHasError = false
		m.storageBackend = m.bitwardenManager
		m.loading = true
		cmd := loadBitwardenOrganizationsCmd(m.bitwardenManager)
		cmds = append(cmds, cmd)
		m.bitwardenLoginForm = nil
		return m, tea.Batch(cmds...)

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
			return m, tea.Batch(cmds...)
		}

		m.formHasError = false
		m.storageBackend = m.bitwardenManager
		m.loading = true
		cmd := loadBitwardenOrganizationsCmd(m.bitwardenManager)
		cmds = append(cmds, cmd)
		m.bitwardenUnlockForm = nil
		return m, tea.Batch(cmds...)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if activeComponent := m.getActiveComponent(); activeComponent != nil {
			model, cmd := activeComponent.Update(msg)
			cmds = append(cmds, cmd)
			return m, m.handleComponentResult(model, tea.Batch(cmds...))
		}

	case tea.KeyMsg:
		if m.loading {
			// If we're loading, only allow Ctrl+C to quit
			if key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c"))) {
				return m, tea.Quit
			}
			return m, tea.Batch(cmds...)
		}

		if m.formHasError {
			if key.Matches(msg, key.NewBinding(key.WithKeys("esc"))) {
				// Handle escape key in error state
				switch m.state {
				case StateBitwardenLogin:
					m.bitwardenLoginForm = nil
					m.formHasError = false
					m.state = StateSelectStorage
					m.storageSelect = components.NewStorageSelect()
					cmds = append(cmds, m.storageSelect.Init())
					return m, tea.Batch(cmds...)

				case StateBitwardenUnlock:
					m.bitwardenUnlockForm = nil
					m.formHasError = false
					m.state = StateSelectStorage
					m.storageSelect = components.NewStorageSelect()
					cmds = append(cmds, m.storageSelect.Init())
					return m, tea.Batch(cmds...)
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
					cmds = append(cmds, cmd)
					return m, tea.Batch(cmds...)
				}
				switch {
				case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
					// Go back to collection select
					m.resetConnectionState()
					if m.state == StateSelectStorage {
						// If we're going back to storage select, properly initialize it
						m.storageSelect = components.NewStorageSelect()
						cmds = append(cmds, m.storageSelect.Init())
					}
					return m, tea.Batch(cmds...)
				case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c"))):
					return m, tea.Quit
				case msg.String() == "a":
					m.connectionForm = components.NewConnectionForm(nil)
					m.state = StateAddConnection
					cmd := m.connectionForm.Init()
					cmds = append(cmds, cmd)
					return m, tea.Batch(cmds...)
				case msg.String() == "e":
					if selectedItem := m.connectionList.HighlightedConnection(); selectedItem != nil {
						m.connectionForm = components.NewConnectionForm(selectedItem)
						m.state = StateEditConnection
						cmd := m.connectionForm.Init()
						cmds = append(cmds, cmd)
						return m, tea.Batch(cmds...)
					}
				case msg.String() == "d":
					if selectedItem := m.connectionList.HighlightedConnection(); selectedItem != nil && m.storageBackend != nil {
						m.loading = true
						if err := m.storageBackend.DeleteConnection(selectedItem.ID); err != nil {
							m.loading = false
							m.errorMessage = err.Error()
							return m, tea.Batch(cmds...)
						}

						cmd := m.ReloadConnections()
						cmds = append(cmds, cmd)
						m.connectionList.Reset()
						return m, tea.Batch(cmds...)
					}
				case msg.String() == "o":
					if m.connectionList != nil {
						m.connectionList.ToggleOpenInNewTerminal()
						return m, tea.Batch(cmds...)
					}
				}
			}

		case StateCollectionSelect:
			if m.bitwardenCollectionList != nil {
				listModel := m.bitwardenCollectionList.List()
				if listModel != nil && listModel.FilterState() == list.Filtering {
					newList, cmd := listModel.Update(msg)
					*listModel = newList
					cmds = append(cmds, cmd)
					return m, tea.Batch(cmds...)
				}
				switch {
				case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
					// Go back to organization select
					m.resetCollectionState()
					return m, tea.Batch(cmds...)
				}
			}

		case StateOrganizationSelect:
			if m.bitwardenOrganizationList != nil {
				listModel := m.bitwardenOrganizationList.List()
				if listModel != nil && listModel.FilterState() == list.Filtering {
					newList, cmd := listModel.Update(msg)
					*listModel = newList
					cmds = append(cmds, cmd)
					return m, tea.Batch(cmds...)
				}
				switch {
				case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
					// Go back to storage select
					m.resetOrganizationState()
					// Initialize storage select when going back
					cmd := m.storageSelect.Init()
					cmds = append(cmds, cmd)
					return m, tea.Batch(cmds...)
				case key.Matches(msg, key.NewBinding(key.WithKeys("o"))):
					// Toggle open personal SSH connections
					cmd := m.loadPersonalVaultConnections()
					cmds = append(cmds, cmd)
					return m, tea.Batch(cmds...)
				}
			}
		}
	}

	// Default: pass message to active component
	if activeComponent := m.getActiveComponent(); activeComponent != nil {
		model, cmd := activeComponent.Update(msg)
		cmds = append(cmds, cmd)
		return m, m.handleComponentResult(model, tea.Batch(cmds...))
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) updateSpinner(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return cmd
}
