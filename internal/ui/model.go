package ui

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
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
	headerLines = 4
	footerLines = 4
)

type Model struct {
	state               AppState
	storageSelect       *components.StorageSelect
	storageBackend      config.Storage
	configManager       *config.ConfigManager
	width               int
	height              int
	connectionList      *components.ConnectionList
	connectionForm      *components.ConnectionForm
	terminal            *components.TerminalComponent
	bitwardenForm       *components.BitwardenConfigForm
	errorMessage        string
	bitwardenLoginForm  *components.BitwardenLoginForm
	bitwardenManager    *config.BitwardenManager
	bitwardenUnlockForm *components.BitwardenUnlockForm
}

func NewModel() *Model {
	return &Model{
		state:         StateSelectStorage,
		storageSelect: components.NewStorageSelect(),
	}
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) LoadConnections() {
	if m.storageBackend != nil {
		if err := m.storageBackend.Load(); err != nil {
			m.errorMessage = fmt.Sprintf("Failed to reload connections: %v", err)
		}
		width, height := m.width, m.height
		if width <= 0 {
			width = 60
		}
		if height <= 0 {
			height = 20
		}
		visibleListHeight := height - headerLines - footerLines
		if visibleListHeight < 5 {
			visibleListHeight = 5
		}
		m.connectionList = components.NewConnectionList(m.storageBackend.ListConnections(), width, visibleListHeight)
	}
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
	keyPath := filepath.Join(dir, fmt.Sprintf("id_%s", conn.Name))
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
	case StateConnectionList:
		if m.connectionList == nil {
			return nil
		}
		return m.connectionList
	case StateAddConnection, StateEditConnection:
		if m.connectionForm == nil {
			return nil
		}
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
				width, height := m.width, m.height
				if width <= 0 {
					width = 60
				}
				if height <= 0 {
					height = 20
				}
				visibleListHeight := height - headerLines - footerLines
				if visibleListHeight < 5 {
					visibleListHeight = 5
				}
				m.connectionList = components.NewConnectionList(cm.ListConnections(), width, visibleListHeight)
				m.state = StateConnectionList
				// Reset the storage select so it's clean if user comes back
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
					// Already logged in and unlocked
					m.storageBackend = m.bitwardenManager
					width, height := m.width, m.height
					if width <= 0 {
						width = 60
					}
					if height <= 0 {
						height = 20
					}
					visibleListHeight := height - headerLines - footerLines
					if visibleListHeight < 5 {
						visibleListHeight = 5
					}
					m.connectionList = components.NewConnectionList(m.storageBackend.ListConnections(), width, visibleListHeight)
					m.state = StateConnectionList
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
			// Attempt login via CLI
			password := m.bitwardenLoginForm.Password()
			otp := m.bitwardenLoginForm.OTP()
			if err := m.bitwardenManager.Login(password, otp); err != nil {
				m.bitwardenLoginForm.SetError(fmt.Sprintf("Login failed: %v", err))
				m.bitwardenLoginForm = nil
				m.state = StateSelectStorage
				m.storageSelect = components.NewStorageSelect()
				fmt.Printf("Error logging in to Bitwarden: %s\n")
				return nil
			}
			m.storageBackend = m.bitwardenManager
			width, height := m.width, m.height
			if width <= 0 {
				width = 60
			}
			if height <= 0 {
				height = 20
			}
			if err := m.bitwardenManager.Load(); err != nil {
				m.errorMessage = fmt.Sprintf("Failed to load Bitwarden connections: %v", err)
			}
			m.connectionList = components.NewConnectionList(m.bitwardenManager.ListConnections(), width, height)
			m.state = StateConnectionList
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
			width, height := m.width, m.height
			if width <= 0 {
				width = 60
			}
			if height <= 0 {
				height = 20
			}
			visibleListHeight := height - headerLines - footerLines
			if visibleListHeight < 5 {
				visibleListHeight = 5
			}
			if err := m.bitwardenManager.Load(); err != nil {
				m.errorMessage = fmt.Sprintf("Failed to load Bitwarden connections: %v", err)
			}
			m.connectionList = components.NewConnectionList(m.storageBackend.ListConnections(), width, visibleListHeight)
			m.state = StateConnectionList
			m.bitwardenUnlockForm = nil
			return nil
		}
		return nil

	case StateConnectionList:
		m.connectionList = model.(*components.ConnectionList)
		if conn := m.connectionList.SelectedConnection(); conn != nil {
			openInNewWindow := m.connectionList.OpenInNewTerminal()
			isWindows := runtime.GOOS == "windows"
			var keyPath string

			// If using key-based authentication, persist private key to a temp file for ssh -i
			if !conn.UsePassword && conn.Password != "" {
				var err error
				keyPath, err = getTempKeyFile(*conn)
				if err != nil {
					m.errorMessage = fmt.Sprintf("Failed to write key file: %s", err)
					return nil
				}
			}

			// Construct ssh arguments for both inline and new window
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
				// ----------- Unix-like platforms -----------
				if openInNewWindow {
					// Use tmux new-window for launching external ssh session
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
					// Inline SSH session in TUI
					m.terminal = components.NewTerminalComponent(*conn)
					m.state = StateSSHTerminal
					m.connectionList.Reset()
					return m.terminal.Init()
				}
			} else {
				// ----------- Windows platforms -----------
				if openInNewWindow {
					// Prefer plink.exe for password automation if available
					usePlink := false
					if conn.UsePassword && conn.Password != "" {
						if _, err := exec.LookPath("plink.exe"); err == nil {
							usePlink = true
						}
					}

					var cmd *exec.Cmd
					if usePlink {
						// Build plink command
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
						// Fallback to OpenSSH on Windows
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
					// Inline SSH session in TUI
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
				err = m.storageBackend.AddConnection(conn)
			}
			if err != nil {
				m.errorMessage = fmt.Sprintf("Failed to save connection: %s", err)
			} else {
				m.connectionForm = nil
				m.state = StateConnectionList
				if err := m.storageBackend.Load(); err != nil {
					m.errorMessage = fmt.Sprintf("Failed to reload connections: %s", err)
				}
				width, height := m.width, m.height
				if width <= 0 {
					width = 60
				}
				if height <= 0 {
					height = 20
				}
				visibleListHeight := height - headerLines - footerLines
				if visibleListHeight < 5 {
					visibleListHeight = 5
				}
				m.connectionList = components.NewConnectionList(m.storageBackend.ListConnections(), width, visibleListHeight)
			}
			return nil
		}

	case StateSSHTerminal:
		m.terminal = model.(*components.TerminalComponent)
		if m.terminal.IsFinished() {
			m.terminal = nil // reset the terminal component
			m.state = StateConnectionList
			m.connectionList.Reset()
			return nil
		}
	}
	return cmd
}
