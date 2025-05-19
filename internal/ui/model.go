package ui

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"ssh-x-term/internal/config"
	"ssh-x-term/internal/ui/components"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Message types for async operations
type LoadConnectionsFinishedMsg struct {
	Connections []config.SSHConnection
	Err         error
}

type BitwardenStatusMsg struct {
	LoggedIn bool
	Unlocked bool
	Err      error
}

type BitwardenLoadOrganizationsMsg struct {
	Organizations []config.Organization
	Err           error
}

type BitwardenLoadCollectionsMsg struct {
	Collections []config.Collection
	Err         error
}

type BitwardenLoadConnectionsByCollectionMsg struct {
	Connections []config.SSHConnection
	Err         error
}

type BitwardenLoginResultMsg struct {
	Success bool
	Err     error
}

type BitwardenUnlockResultMsg struct {
	Success bool
	Err     error
}

type AppState int

const (
	StateSelectStorage AppState = iota
	StateBitwardenConfig
	StateConnectionList
	StateAddConnection
	StateEditConnection
	StateSSHTerminal
	StateBitwardenLogin
	StateBitwardenUnlock
	StateOrganizationSelect
	StateCollectionSelect
	headerLines   = 4
	footerLines   = 4
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
		loading:       false,
	}
}

func (m *Model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m *Model) listHeight() int {
	usableHeight := m.height - headerLines - footerLines
	if usableHeight < minListHeight {
		return minListHeight
	}
	return usableHeight
}

// Async commands
func loadConnectionsCmd(backend config.Storage) tea.Cmd {
	return func() tea.Msg {
		var err error
		if cm, ok := backend.(*config.ConfigManager); ok {
			err = cm.Load()
			if err != nil {
				return LoadConnectionsFinishedMsg{Err: err}
			}
			return LoadConnectionsFinishedMsg{Connections: cm.ListConnections()}
		} else if bw, ok := backend.(*config.BitwardenManager); ok {
			if bw.IsPersonalVault() {
				err = bw.Load()
			} else {
				err = bw.LoadConnectionsByCollectionId(bw.GetSelectedCollection().ID)
			}
			if err != nil {
				return LoadConnectionsFinishedMsg{Err: err}
			}
			return LoadConnectionsFinishedMsg{Connections: bw.ListConnections()}
		}
		return LoadConnectionsFinishedMsg{Err: fmt.Errorf("unknown storage backend")}
	}
}

func loadBitwardenStatusCmd(bw *config.BitwardenManager) tea.Cmd {
	return func() tea.Msg {
		loggedIn, unlocked, err := bw.Status()
		return BitwardenStatusMsg{
			LoggedIn: loggedIn,
			Unlocked: unlocked,
			Err:      err,
		}
	}
}

func loadBitwardenOrganizationsCmd(bw *config.BitwardenManager) tea.Cmd {
	return func() tea.Msg {
		err := bw.LoadOrganizations()
		if err != nil {
			return BitwardenLoadOrganizationsMsg{Err: err}
		}
		return BitwardenLoadOrganizationsMsg{Organizations: bw.ListOrganizations()}
	}
}

func loadBitwardenCollectionsCmd(bw *config.BitwardenManager, orgID string) tea.Cmd {
	return func() tea.Msg {
		err := bw.LoadCollectionsByOrganizationId(orgID)
		if err != nil {
			return BitwardenLoadCollectionsMsg{Err: err}
		}
		return BitwardenLoadCollectionsMsg{Collections: bw.ListCollections()}
	}
}

func loadBitwardenConnectionsByCollectionCmd(bw *config.BitwardenManager, collectionID string) tea.Cmd {
	return func() tea.Msg {
		err := bw.LoadConnectionsByCollectionId(collectionID)
		if err != nil {
			return BitwardenLoadConnectionsByCollectionMsg{Err: err}
		}
		return BitwardenLoadConnectionsByCollectionMsg{Connections: bw.ListConnections()}
	}
}

func loginBitwardenCmd(bw *config.BitwardenManager, password, otp string) tea.Cmd {
	return func() tea.Msg {
		err := bw.Login(password, otp)
		if err != nil {
			return BitwardenLoginResultMsg{Success: false, Err: err}
		}
		return BitwardenLoginResultMsg{Success: true}
	}
}

func unlockBitwardenCmd(bw *config.BitwardenManager, password string) tea.Cmd {
	return func() tea.Msg {
		err := bw.Unlock(password)
		if err != nil {
			return BitwardenUnlockResultMsg{Success: false, Err: err}
		}
		return BitwardenUnlockResultMsg{Success: true}
	}
}

func (m *Model) ReloadConnections() tea.Cmd {
	m.loading = true
	return loadConnectionsCmd(m.storageBackend)
}

// Helper to load personal vault connections
func (m *Model) loadPersonalVaultConnections() tea.Cmd {
	m.loading = true
	m.bitwardenManager.SetPersonalVault(true)
	m.storageBackend = m.bitwardenManager
	return loadConnectionsCmd(m.bitwardenManager)
}

// sanitizeFileName replaces all non-alphanumeric characters with underscores
func sanitizeFileName(name string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9]`)
	return re.ReplaceAllString(name, "_")
}

func getTempKeyFile(conn config.SSHConnection) (string, error) {
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
				return loadConnectionsCmd(cm)
			case components.StorageBitwarden:
				m.loading = true
				bwm, err := config.NewBitwardenManager(&config.BitwardenConfig{})
				if err != nil {
					m.errorMessage = fmt.Sprintf("Error initializing Bitwarden: %s", err)
					m.loading = false
					return nil
				}
				m.bitwardenManager = bwm
				return loadBitwardenStatusCmd(bwm)
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
			password := m.bitwardenLoginForm.Password()
			otp := m.bitwardenLoginForm.OTP()
			return loginBitwardenCmd(m.bitwardenManager, password, otp)
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
			password := m.bitwardenUnlockForm.Password()
			return unlockBitwardenCmd(m.bitwardenManager, password)
		}
		return nil

	case StateOrganizationSelect:
		m.bitwardenOrganizationList = model.(*components.BitwardenOrganizationList)
		if org := m.bitwardenOrganizationList.SelectedOrganization(); org != nil {
			m.loading = true
			m.storageBackend = m.bitwardenManager
			return loadBitwardenCollectionsCmd(m.bitwardenManager, org.ID)
		}

	case StateCollectionSelect:
		m.bitwardenCollectionList = model.(*components.BitwardenCollectionList)
		if collection := m.bitwardenCollectionList.SelectedCollection(); collection != nil {
			m.loading = true
			m.bitwardenManager.SetSelectedCollection(collection)
			m.storageBackend = m.bitwardenManager
			return loadBitwardenConnectionsByCollectionCmd(m.bitwardenManager, collection.ID)
		}

	case StateConnectionList:
		m.connectionList = model.(*components.ConnectionList)
		if conn := m.connectionList.SelectedConnection(); conn != nil {
			openInNewWindow := m.connectionList.OpenInNewTerminal()
			isWindows := runtime.GOOS == "windows"
			var keyPath string

			if !conn.UsePassword && conn.Password != "" {
				var err error
				keyPath, err = getTempKeyFile(*conn)
				if err != nil {
					m.errorMessage = fmt.Sprintf("Failed to write key file: %s", err)
					return nil
				}
			}

			sshArgs := []string{}
			if !conn.UsePassword && keyPath != "" {
				sshArgs = append(sshArgs, "-i", keyPath)
			}
			if conn.Port != 22 && conn.Port != 0 {
				sshArgs = append(sshArgs, "-p", strconv.Itoa(conn.Port))
			}
			userHost := fmt.Sprintf("%s@%s", conn.Username, conn.Host)
			sshArgs = append(sshArgs, userHost)

			if !isWindows {
				if openInNewWindow {
					var sshCommand string
					useSshpass := false

					if conn.UsePassword && conn.Password != "" {
						if _, err := exec.LookPath("sshpass"); err == nil {
							useSshpass = true
							sshCommand = fmt.Sprintf(
								"sshpass -p %q ssh %s",
								conn.Password,
								strings.Join(sshArgs, " "),
							)
						} else {
							tea.Println("sshpass not found, falling back to manual password entry or key-based login.")
							sshCommand = fmt.Sprintf("ssh %s", strings.Join(sshArgs, " "))
						}
					} else {
						sshCommand = fmt.Sprintf("ssh %s", strings.Join(sshArgs, " "))
					}

					windowName := fmt.Sprintf("%s@%s:%d - %s", conn.Username, conn.Host, conn.Port, conn.Name)
					cmd := exec.Command("tmux", "new-window", "-n", windowName, sshCommand)
					err := cmd.Start()
					if err != nil {
						fmt.Fprintf(os.Stderr, "Error launching tmux window: %v\n", err)
					}
					if conn.UsePassword && conn.Password != "" && !useSshpass {
						tea.Println("Password authentication not supported in this mode. Use manual entry or key-based login.")
					}
					m.state = StateConnectionList
					m.connectionList.Reset()
					return nil
				} else {
					m.terminal = components.NewTerminalComponent(*conn)
					m.state = StateSSHTerminal
					m.connectionList.Reset()
					return m.terminal.Init()
				}
			} else {
				if openInNewWindow {
					usePlink := false
					if conn.UsePassword && conn.Password != "" {
						if _, err := exec.LookPath("plink.exe"); err == nil {
							usePlink = true
						}
					}

					var cmd *exec.Cmd
					if usePlink {
						plinkArgs := []string{"-ssh", userHost, "-pw", conn.Password}
						if conn.Port != 22 && conn.Port != 0 {
							plinkArgs = append(plinkArgs, "-P", strconv.Itoa(conn.Port))
						}
						if !conn.UsePassword && keyPath != "" {
							plinkArgs = append(plinkArgs, "-i", keyPath)
						}
						cmd = exec.Command("cmd", "/C", "start", "", "plink.exe")
						cmd.Args = append(cmd.Args, plinkArgs...)
					} else {
						cmd = exec.Command("cmd", "/C", "start", "", "ssh")
						cmd.Args = append(cmd.Args, sshArgs...)
					}

					err := cmd.Start()
					if err != nil {
						fmt.Fprintf(os.Stderr, "Error launching terminal: %v\n", err)
					}
					if conn.UsePassword && conn.Password != "" && !usePlink {
						tea.Println("Password authentication not supported in this mode. Use manual entry, key-based login, or install plink.exe.")
					}
					m.state = StateConnectionList
					m.connectionList.Reset()
					return nil
				} else {
					m.terminal = components.NewTerminalComponent(*conn)
					m.state = StateSSHTerminal
					m.connectionList.Reset()
					return m.terminal.Init()
				}
			}
		}

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
			var err error
			if m.state == StateEditConnection {
				err = m.storageBackend.EditConnection(conn)
			} else {
				if m.storageSelect.IsChosen() {
					switch m.storageSelect.SelectedBackend() {
					case components.StorageLocal:
						err = m.storageBackend.AddConnection(conn)
					case components.StorageBitwarden:
						if m.bitwardenManager.IsPersonalVault() {
							err = m.storageBackend.AddConnection(conn)
						} else {
							selectedCollection := m.bitwardenCollectionList.SelectedCollection()
							selectedOrg := m.bitwardenOrganizationList.SelectedOrganization()
							var collectionID, organizationID string
							if selectedCollection != nil && selectedOrg != nil {
								collectionID = selectedCollection.ID
								organizationID = selectedOrg.ID
							}
							err = m.bitwardenManager.AddConnectionInCollectionAndOrganization(conn, organizationID, collectionID)
						}
					}
				}
			}

			if err != nil {
				m.loading = false
				m.errorMessage = fmt.Sprintf("Failed to save connection: %s", err)
				m.connectionForm = nil
				m.state = StateConnectionList
				return loadConnectionsCmd(m.storageBackend)
			}

			m.connectionForm = nil
			m.state = StateConnectionList
			return loadConnectionsCmd(m.storageBackend)
		}
	case StateSSHTerminal:
		m.terminal = model.(*components.TerminalComponent)
		if m.terminal.IsFinished() {
			m.terminal = nil
			m.state = StateConnectionList
			m.connectionList.Reset()
			return nil
		}
	}
	return cmd
}

// Helper to reset connection state
func (m *Model) resetConnectionState() {
	if m.connectionList != nil {
		m.connectionList.Reset()
	}
	switch m.storageSelect.SelectedBackend() {
	case components.StorageLocal:
		m.state = StateSelectStorage
		// Create a new storage select when returning to this state
		m.storageSelect = components.NewStorageSelect()
	case components.StorageBitwarden:
		if m.bitwardenCollectionList != nil {
			m.bitwardenCollectionList.Reset()
		}
		if m.bitwardenCollectionList == nil {
			m.state = StateSelectStorage
			// Create a new storage select when returning to this state
			m.storageSelect = components.NewStorageSelect()
		} else {
			m.state = StateCollectionSelect
		}
	}
}

// Helper to reset collection state
func (m *Model) resetCollectionState() {
	if m.bitwardenCollectionList != nil {
		m.bitwardenCollectionList.Reset()
	}
	if m.bitwardenOrganizationList != nil {
		m.bitwardenOrganizationList.Reset()
	}
	m.state = StateOrganizationSelect
}

// Helper to reset organization state
func (m *Model) resetOrganizationState() {
	if m.bitwardenOrganizationList != nil {
		m.bitwardenOrganizationList.Reset()
	}
	m.state = StateSelectStorage
	// Create a new storage select when returning to this state
	m.storageSelect = components.NewStorageSelect()
}
