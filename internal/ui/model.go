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

	tea "github.com/charmbracelet/bubbletea"
)

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
}

func NewModel() *Model {
	return &Model{
		state:         StateSelectStorage,
		storageSelect: components.NewStorageSelect(),
		width:         defaultWidth,
		height:        defaultHeight,
	}
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) listHeight() int {
	usableHeight := m.height - headerLines - footerLines
	if usableHeight < minListHeight {
		return minListHeight
	}
	return usableHeight
}

func (m *Model) ReloadConnections() {
	switch backend := m.storageBackend.(type) {
	case *config.ConfigManager:
		if err := backend.Load(); err != nil {
			m.errorMessage = "Failed to reload connections: " + err.Error()
			return
		}
		m.connectionList = components.NewConnectionList(backend.ListConnections())
		m.connectionList.SetSize(m.width, m.listHeight())
	case *config.BitwardenManager:
		if m.bitwardenManager.IsPersonalVault() {
			if err := backend.Load(); err != nil {
				m.errorMessage = "Failed to reload Bitwarden connections: " + err.Error()
				return
			}
			m.connectionList = components.NewConnectionList(backend.ListConnections())
			m.connectionList.SetSize(m.width, m.listHeight())
		} else {
			if m.bitwardenCollectionList != nil && m.bitwardenCollectionList.HighlightedCollection() != nil {
				collection := m.bitwardenCollectionList.HighlightedCollection()
				if err := backend.LoadConnectionsByCollectionId(collection.ID); err != nil {
					m.errorMessage = "Failed to reload Bitwarden connections: " + err.Error()
					return
				}
				m.connectionList = components.NewConnectionList(backend.ListConnections())
				m.connectionList.SetSize(m.width, m.listHeight())
			}
		}
	}
}

func (m *Model) SetSize(width, height int) {
	if width <= 0 {
		width = defaultWidth
	}
	if height <= 0 {
		height = defaultHeight
	}
	m.width = width
	m.height = height

	if m.connectionList != nil {
		m.connectionList.SetSize(width, m.listHeight())
	}
	if m.bitwardenOrganizationList != nil {
		m.bitwardenOrganizationList.SetSize(width, m.listHeight())
	}
	if m.bitwardenCollectionList != nil {
		m.bitwardenCollectionList.SetSize(width, m.listHeight())
	}
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
				cm, err := config.NewConfigManager()
				if err != nil {
					m.errorMessage = fmt.Sprintf("Error initializing local config: %s", err)
					return nil
				}
				if err := cm.Load(); err != nil {
					m.errorMessage = fmt.Sprintf("Error loading config: %s", err)
					return nil
				}
				m.storageBackend = cm
				m.configManager = cm
				m.connectionList = components.NewConnectionList(cm.ListConnections())
				m.connectionList.SetSize(m.width, m.listHeight())
				m.state = StateConnectionList
				m.storageSelect = components.NewStorageSelect()
			case components.StorageBitwarden:
				bwm, err := config.NewBitwardenManager(&config.BitwardenConfig{})
				if err != nil {
					m.errorMessage = fmt.Sprintf("Error initializing Bitwarden: %s", err)
					return nil
				}
				m.bitwardenManager = bwm
				loggedIn, unlocked, err := bwm.Status()
				if err != nil {
					m.errorMessage = fmt.Sprintf("Error checking Bitwarden status: %s", err)
					return nil
				}
				if !loggedIn {
					m.bitwardenForm = components.NewBitwardenConfigForm()
					m.state = StateBitwardenConfig
				} else if !unlocked {
					m.bitwardenUnlockForm = components.NewBitwardenUnlockForm()
					m.state = StateBitwardenUnlock
				} else {
					if err := m.bitwardenManager.LoadOrganizations(); err != nil {
						m.errorMessage = fmt.Sprintf("Failed to load Bitwarden organizations: %v", err)
						m.state = StateBitwardenUnlock
						return nil
					}
					m.bitwardenOrganizationList = components.NewBitwardenOrganizationList(m.bitwardenManager.ListOrganizations())
					m.bitwardenOrganizationList.SetSize(m.width, m.listHeight())
					m.state = StateOrganizationSelect
				}
				return nil
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
			password := m.bitwardenLoginForm.Password()
			otp := m.bitwardenLoginForm.OTP()
			if err := m.bitwardenManager.Login(password, otp); err != nil {
				m.bitwardenLoginForm.SetError(fmt.Sprintf("Login failed: %v", err))
				m.bitwardenLoginForm = nil
				m.state = StateSelectStorage
				m.storageSelect = components.NewStorageSelect()
				return nil
			}
			m.storageBackend = m.bitwardenManager
			if err := m.bitwardenManager.LoadOrganizations(); err != nil {
				m.errorMessage = fmt.Sprintf("Failed to load Bitwarden organizations: %v", err)
			}
			m.bitwardenOrganizationList = components.NewBitwardenOrganizationList(m.bitwardenManager.ListOrganizations())
			m.bitwardenOrganizationList.SetSize(m.width, m.listHeight())
			m.state = StateOrganizationSelect
			m.bitwardenLoginForm = nil
			return nil
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
			password := m.bitwardenUnlockForm.Password()
			if err := m.bitwardenManager.Unlock(password); err != nil {
				m.bitwardenUnlockForm.SetError(fmt.Sprintf("Unlock failed: %v", err))
				m.state = StateSelectStorage
				m.bitwardenForm = nil
				m.bitwardenUnlockForm = nil
				m.storageSelect = components.NewStorageSelect()
				return nil
			}
			m.storageBackend = m.bitwardenManager
			if err := m.bitwardenManager.LoadOrganizations(); err != nil {
				m.errorMessage = fmt.Sprintf("Failed to load Bitwarden organizations: %v", err)
			}
			m.bitwardenOrganizationList = components.NewBitwardenOrganizationList(m.bitwardenManager.ListOrganizations())
			m.bitwardenOrganizationList.SetSize(m.width, m.listHeight())
			m.state = StateOrganizationSelect
			m.bitwardenUnlockForm = nil
			return nil
		}
		return nil

	case StateOrganizationSelect:
		m.bitwardenOrganizationList = model.(*components.BitwardenOrganizationList)
		if org := m.bitwardenOrganizationList.SelectedOrganization(); org != nil {
			m.storageBackend = m.bitwardenManager
			if err := m.bitwardenManager.LoadCollectionsByOrganizationId(org.ID); err != nil {
				m.errorMessage = fmt.Sprintf("Failed to load Bitwarden collections: %v", err)
			}
			m.bitwardenCollectionList = components.NewBitwardenCollectionList(m.bitwardenManager.ListCollections())
			m.bitwardenCollectionList.SetSize(m.width, m.listHeight())
			m.state = StateCollectionSelect
			return nil
		}

	case StateCollectionSelect:
		m.bitwardenCollectionList = model.(*components.BitwardenCollectionList)
		if collection := m.bitwardenCollectionList.SelectedCollection(); collection != nil {
			m.storageBackend = m.bitwardenManager
			if err := m.bitwardenManager.LoadConnectionsByCollectionId(collection.ID); err != nil {
				m.errorMessage = fmt.Sprintf("Failed to load Bitwarden connections: %v", err)
			}
			m.connectionList = components.NewConnectionList(m.bitwardenManager.ListConnections())
			m.connectionList.SetSize(m.width, m.listHeight())
			m.state = StateConnectionList
			return nil
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
			// Handle errors: Show error message and return to connection list
			if err != nil {
				m.errorMessage = fmt.Sprintf("Failed to save connection: %s", err)
				m.connectionForm = nil        // Reset the connection form
				m.state = StateConnectionList // Return to the connection list state
				m.ReloadConnections()         // Refresh the connection list
				return nil
			} else {
				m.connectionForm = nil
				m.state = StateConnectionList
				m.ReloadConnections()
			}
			return nil
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
