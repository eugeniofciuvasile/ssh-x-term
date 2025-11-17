package ui

import (
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"regexp"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/eugeniofciuvasile/ssh-x-term/internal/config"
	"github.com/eugeniofciuvasile/ssh-x-term/internal/ui/components"
)

// Message types for async operations
type (
	LoadConnectionsFinishedMsg struct {
		Connections []config.SSHConnection
		Err         error
	}
	BitwardenStatusMsg struct {
		LoggedIn bool
		Unlocked bool
		Err      error
	}
	BitwardenLoadOrganizationsMsg struct {
		Organizations []config.Organization
		Err           error
	}
	BitwardenLoadCollectionsMsg struct {
		Collections []config.Collection
		Err         error
	}
	BitwardenLoadConnectionsByCollectionMsg struct {
		Connections []config.SSHConnection
		Err         error
	}
	BitwardenLoginResultMsg struct {
		Success bool
		Err     error
	}
	BitwardenUnlockResultMsg struct {
		Success bool
		Err     error
	}
	SaveConnectionResultMsg struct {
		Err error
	}
	DeleteConnectionResultMsg struct {
		Err error
	}
)

// AppState type
type AppState int

const (
	StateSelectStorage AppState = iota
	StateBitwardenConfig
	StateConnectionList
	StateAddConnection
	StateEditConnection
	StateSSHTerminal
	StateSCPFileManager
	StateBitwardenLogin
	StateBitwardenUnlock
	StateOrganizationSelect
	StateCollectionSelect

	// Layout constants for full-screen UI
	headerHeight  = 1 // Header line at top
	footerHeight  = 1 // Footer line at bottom
	minListHeight = 5
	defaultWidth  = 60
	defaultHeight = 20
)

type Model struct {
	state                     AppState
	storageSelect             *components.StorageSelect
	storageBackend            config.Storage
	configManager             *config.ConfigManager
	width                     int
	height                    int
	connectionList            *components.ConnectionList
	connectionForm            *components.ConnectionForm
	terminal                  *components.TerminalComponent
	scpManager                *components.SCPManager
	bitwardenForm             *components.BitwardenConfigForm
	errorMessage              string
	bitwardenLoginForm        *components.BitwardenLoginForm
	bitwardenManager          *config.BitwardenManager
	bitwardenUnlockForm       *components.BitwardenUnlockForm
	bitwardenOrganizationList *components.BitwardenOrganizationList
	bitwardenCollectionList   *components.BitwardenCollectionList
	spinner                   spinner.Model
	loading                   bool
	formHasError              bool
}

func NewModel() *Model {
	s := spinner.New()
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	s.Spinner = spinner.Dot
	return &Model{
		state:         StateSelectStorage,
		storageSelect: components.NewStorageSelect(),
		width:         defaultWidth,
		height:        defaultHeight,
		spinner:       s,
	}
}

func (m *Model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m *Model) listHeight() int {
	// Calculate available height for lists (total - header - footer)
	usableHeight := m.height - headerHeight - footerHeight
	if usableHeight < minListHeight {
		return minListHeight
	}
	return usableHeight
}

// === Async command creators ===

func (m *Model) loadPersonalVaultConnections() tea.Cmd {
	m.loading = true
	m.bitwardenManager.SetPersonalVault(true)
	m.storageBackend = m.bitwardenManager
	return tea.Batch(
		loadConnectionsCmd(m.bitwardenManager),
		m.spinner.Tick,
	)
}

func loadConnectionsCmd(backend config.Storage) tea.Cmd {
	return func() tea.Msg {
		switch b := backend.(type) {
		case *config.ConfigManager:
			if err := b.Load(); err != nil {
				log.Printf("LoadConnectionsFinishedMsg: error loading config manager: %v", err)
				return LoadConnectionsFinishedMsg{Err: err}
			}
			return LoadConnectionsFinishedMsg{Connections: b.ListConnections()}
		case *config.BitwardenManager:
			var err error
			if b.IsPersonalVault() {
				err = b.Load()
			} else {
				coll := b.GetSelectedCollection()
				if coll == nil {
					log.Printf("LoadConnectionsFinishedMsg: no selected collection in Bitwarden")
					return LoadConnectionsFinishedMsg{Err: fmt.Errorf("no selected collection")}
				}
				err = b.LoadConnectionsByCollectionId(coll.ID)
			}
			if err != nil {
				log.Printf("LoadConnectionsFinishedMsg: error loading bitwarden: %v", err)
				return LoadConnectionsFinishedMsg{Err: err}
			}
			return LoadConnectionsFinishedMsg{Connections: b.ListConnections()}
		default:
			log.Printf("LoadConnectionsFinishedMsg: unknown storage backend")
			return LoadConnectionsFinishedMsg{Err: fmt.Errorf("unknown storage backend")}
		}
	}
}

func loadBitwardenStatusCmd(bw *config.BitwardenManager) tea.Cmd {
	return func() tea.Msg {
		loggedIn, unlocked, err := bw.Status()
		if err != nil {
			log.Printf("BitwardenStatusMsg: error getting status: %v", err)
		}
		return BitwardenStatusMsg{
			LoggedIn: loggedIn,
			Unlocked: unlocked,
			Err:      err,
		}
	}
}

func loadBitwardenOrganizationsCmd(bw *config.BitwardenManager) tea.Cmd {
	return func() tea.Msg {
		if err := bw.LoadOrganizations(); err != nil {
			log.Printf("BitwardenLoadOrganizationsMsg: error loading organizations: %v", err)
			return BitwardenLoadOrganizationsMsg{Err: err}
		}
		return BitwardenLoadOrganizationsMsg{Organizations: bw.ListOrganizations()}
	}
}

func loadBitwardenCollectionsCmd(bw *config.BitwardenManager, orgID string) tea.Cmd {
	return func() tea.Msg {
		if err := bw.LoadCollectionsByOrganizationId(orgID); err != nil {
			log.Printf("BitwardenLoadCollectionsMsg: error loading collections: %v", err)
			return BitwardenLoadCollectionsMsg{Err: err}
		}
		return BitwardenLoadCollectionsMsg{Collections: bw.ListCollections()}
	}
}

func loadBitwardenConnectionsByCollectionCmd(bw *config.BitwardenManager, collectionID string) tea.Cmd {
	return func() tea.Msg {
		if err := bw.LoadConnectionsByCollectionId(collectionID); err != nil {
			log.Printf("BitwardenLoadConnectionsByCollectionMsg: error loading connections: %v", err)
			return BitwardenLoadConnectionsByCollectionMsg{Err: err}
		}
		return BitwardenLoadConnectionsByCollectionMsg{Connections: bw.ListConnections()}
	}
}

func loginBitwardenCmd(bw *config.BitwardenManager, password, otp string) tea.Cmd {
	return func() tea.Msg {
		if err := bw.Login(password, otp); err != nil {
			log.Printf("BitwardenLoginResultMsg: error logging in: %v", err)
			return BitwardenLoginResultMsg{Success: false, Err: err}
		}
		return BitwardenLoginResultMsg{Success: true}
	}
}

func unlockBitwardenCmd(bw *config.BitwardenManager, password string) tea.Cmd {
	return func() tea.Msg {
		if err := bw.Unlock(password); err != nil {
			log.Printf("BitwardenUnlockResultMsg: error unlocking: %v", err)
			return BitwardenUnlockResultMsg{Success: false, Err: err}
		}
		return BitwardenUnlockResultMsg{Success: true}
	}
}

func saveConnectionCmd(
	backend config.Storage,
	bitwardenManager *config.BitwardenManager,
	storageSelect *components.StorageSelect,
	bitwardenCollectionList *components.BitwardenCollectionList,
	bitwardenOrganizationList *components.BitwardenOrganizationList,
	conn config.SSHConnection,
	isEdit bool,
) tea.Cmd {
	return func() tea.Msg {
		var err error
		if isEdit {
			err = backend.EditConnection(conn)
		} else if storageSelect.IsChosen() {
			switch storageSelect.SelectedBackend() {
			case components.StorageLocal:
				err = backend.AddConnection(conn)
			case components.StorageBitwarden:
				if bitwardenManager.IsPersonalVault() {
					err = backend.AddConnection(conn)
				} else {
					var collectionID, organizationID string
					if selCol, selOrg := bitwardenCollectionList.SelectedCollection(), bitwardenOrganizationList.SelectedOrganization(); selCol != nil && selOrg != nil {
						collectionID = selCol.ID
						organizationID = selOrg.ID
					}
					err = bitwardenManager.AddConnectionInCollectionAndOrganization(conn, organizationID, collectionID)
				}
			}
		}
		return SaveConnectionResultMsg{Err: err}
	}
}

func deleteConnectionCmd(backend config.Storage, id string) tea.Cmd {
	return func() tea.Msg {
		err := backend.DeleteConnection(id)
		return DeleteConnectionResultMsg{Err: err}
	}
}

// Helpers
func sanitizeFileName(name string) string {
	return regexp.MustCompile(`[^a-zA-Z0-9]`).ReplaceAllString(name, "_")
}

func getKeyFile(conn config.SSHConnection) (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(usr.HomeDir, ".ssh", "xterm_keys")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	safeName := sanitizeFileName(conn.Name)
	keyPath := filepath.Join(dir, fmt.Sprintf("id_%s", safeName))
	if err := os.WriteFile(keyPath, []byte(conn.Password), 0600); err != nil {
		return "", err
	}
	if conn.PublicKey != "" {
		_ = os.WriteFile(keyPath+".pub", []byte(conn.PublicKey), 0644)
	}
	return keyPath, nil
}

func (m *Model) getActiveComponent() tea.Model {
	switch m.state {
	case StateSelectStorage:
		return m.storageSelect
	case StateBitwardenConfig:
		return m.bitwardenForm
	case StateBitwardenLogin:
		return m.bitwardenLoginForm
	case StateBitwardenUnlock:
		return m.bitwardenUnlockForm
	case StateOrganizationSelect:
		return m.bitwardenOrganizationList
	case StateCollectionSelect:
		return m.bitwardenCollectionList
	case StateConnectionList:
		return m.connectionList
	case StateAddConnection, StateEditConnection:
		return m.connectionForm
	case StateSSHTerminal:
		return m.terminal
	case StateSCPFileManager:
		return m.scpManager
	default:
		return nil
	}
}

func (m *Model) handleComponentResult(model tea.Model, cmd tea.Cmd) tea.Cmd {
	switch m.state {
	case StateSelectStorage:
		m.storageSelect = model.(*components.StorageSelect)
		if m.storageSelect.IsCanceled() {
			return tea.Quit
		}
		if m.storageSelect.IsChosen() {
			switch m.storageSelect.SelectedBackend() {
			case components.StorageLocal:
				m.loading = true
				cm, err := config.NewConfigManager()
				if err != nil {
					m.errorMessage = fmt.Sprintf("Error initializing local config: %s", err)
					m.loading = false
					return nil
				}
				m.storageBackend = cm
				m.configManager = cm
				return tea.Batch(
					loadConnectionsCmd(cm),
					m.spinner.Tick,
				)
			case components.StorageBitwarden:
				m.loading = true
				bwm, err := config.NewBitwardenManager(&config.BitwardenConfig{})
				if err != nil {
					m.errorMessage = fmt.Sprintf("Error initializing Bitwarden: %s", err)
					m.loading = false
					return nil
				}
				m.bitwardenManager = bwm
				return tea.Batch(
					loadBitwardenStatusCmd(bwm),
					m.spinner.Tick,
				)
			}
		}
		return nil

	case StateBitwardenConfig:
		m.bitwardenForm = model.(*components.BitwardenConfigForm)
		if m.bitwardenForm.IsCanceled() {
			m.bitwardenForm = nil
			m.state = StateSelectStorage
			m.storageSelect = components.NewStorageSelect()
			return nil
		}
		if m.bitwardenForm.IsSubmitted() {
			serverURL, email := m.bitwardenForm.Config()
			cfg := &config.BitwardenConfig{ServerURL: serverURL, Email: email}
			bwm, err := config.NewBitwardenManager(cfg)
			if err != nil {
				m.bitwardenForm.ErrorMsg = err.Error()
				return nil
			}
			m.bitwardenManager = bwm
			m.bitwardenLoginForm = components.NewBitwardenLoginForm()
			m.state = StateBitwardenLogin
			return nil
		}
		return nil

	case StateBitwardenLogin:
		m.bitwardenLoginForm = model.(*components.BitwardenLoginForm)
		if m.bitwardenLoginForm.IsCanceled() {
			m.bitwardenLoginForm = nil
			m.state = StateSelectStorage
			m.storageSelect = components.NewStorageSelect()
			return nil
		}
		if m.bitwardenLoginForm.IsSubmitted() {
			m.loading = true
			return tea.Batch(
				loginBitwardenCmd(m.bitwardenManager, m.bitwardenLoginForm.Password(), m.bitwardenLoginForm.OTP()),
				m.spinner.Tick,
			)
		}
		return nil

	case StateBitwardenUnlock:
		m.bitwardenUnlockForm = model.(*components.BitwardenUnlockForm)
		if m.bitwardenUnlockForm.IsCanceled() {
			m.bitwardenUnlockForm = nil
			m.state = StateSelectStorage
			m.storageSelect = components.NewStorageSelect()
			return nil
		}
		if m.bitwardenUnlockForm.IsSubmitted() {
			m.loading = true
			return tea.Batch(
				unlockBitwardenCmd(m.bitwardenManager, m.bitwardenUnlockForm.Password()),
				m.spinner.Tick,
			)
		}
		return nil

	case StateOrganizationSelect:
		m.bitwardenOrganizationList = model.(*components.BitwardenOrganizationList)
		if org := m.bitwardenOrganizationList.SelectedOrganization(); org != nil {
			m.loading = true
			m.storageBackend = m.bitwardenManager
			return tea.Batch(
				loadBitwardenCollectionsCmd(m.bitwardenManager, org.ID),
				m.spinner.Tick,
			)
		}

	case StateCollectionSelect:
		m.bitwardenCollectionList = model.(*components.BitwardenCollectionList)
		if collection := m.bitwardenCollectionList.SelectedCollection(); collection != nil {
			m.loading = true
			m.bitwardenManager.SetSelectedCollection(collection)
			m.storageBackend = m.bitwardenManager
			return tea.Batch(
				loadBitwardenConnectionsByCollectionCmd(m.bitwardenManager, collection.ID),
				m.spinner.Tick,
			)
		}

	case StateConnectionList:
		return m.handleConnectionList(model)

	case StateAddConnection, StateEditConnection:
		m.connectionForm = model.(*components.ConnectionForm)
		if m.connectionForm.IsCanceled() {
			m.connectionForm = nil
			m.state = StateConnectionList
			m.connectionList.Reset()
			return nil
		}
		if m.connectionForm.IsSubmitted() {
			m.loading = true
			conn := m.connectionForm.Connection()
			return tea.Batch(
				saveConnectionCmd(
					m.storageBackend,
					m.bitwardenManager,
					m.storageSelect,
					m.bitwardenCollectionList,
					m.bitwardenOrganizationList,
					conn,
					m.state == StateEditConnection,
				),
				m.spinner.Tick,
			)
		}
	case StateSSHTerminal:
		m.terminal = model.(*components.TerminalComponent)
		if m.terminal.IsFinished() {
			m.terminal = nil
			m.state = StateConnectionList
			m.connectionList.Reset()
			return nil
		}
	case StateSCPFileManager:
		m.scpManager = model.(*components.SCPManager)
		if m.scpManager.IsFinished() {
			m.scpManager = nil
			m.state = StateConnectionList
			m.connectionList.Reset()
			return nil
		}
	}
	return cmd
}

// State reset helpers
func (m *Model) resetConnectionState() {
	if m.connectionList != nil {
		m.connectionList.Reset()
	}
	switch m.storageSelect.SelectedBackend() {
	case components.StorageLocal:
		m.state = StateSelectStorage
		m.storageSelect = components.NewStorageSelect()
	case components.StorageBitwarden:
		if m.bitwardenCollectionList != nil {
			m.bitwardenCollectionList.Reset()
		}
		if m.bitwardenCollectionList == nil {
			if m.bitwardenOrganizationList != nil {
				m.bitwardenOrganizationList.Reset()
			}
			m.bitwardenManager.SetPersonalVault(false)
			m.state = StateOrganizationSelect
		} else if m.bitwardenManager.IsPersonalVault() {
			if m.bitwardenOrganizationList != nil {
				m.bitwardenOrganizationList.Reset()
			}
			m.bitwardenManager.SetPersonalVault(false)
			m.state = StateOrganizationSelect
		} else {
			m.state = StateCollectionSelect
		}
	}
}

func (m *Model) resetCollectionState() {
	if m.bitwardenCollectionList != nil {
		m.bitwardenCollectionList.Reset()
	}
	if m.bitwardenOrganizationList != nil {
		m.bitwardenOrganizationList.Reset()
	}
	m.state = StateOrganizationSelect
}

func (m *Model) resetOrganizationState() {
	if m.bitwardenOrganizationList != nil {
		m.bitwardenOrganizationList.Reset()
	}
	m.state = StateSelectStorage
	m.storageSelect = components.NewStorageSelect()
}
