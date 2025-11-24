package ui

import (
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/eugeniofciuvasile/ssh-x-term/internal/config"
	"github.com/eugeniofciuvasile/ssh-x-term/internal/ui/components"
	"github.com/zalando/go-keyring"
)

const keyringService = "ssh-x-term" // Keyring service name

func (m *Model) handleConnectionList(model tea.Model) tea.Cmd {
	m.connectionList = model.(*components.ConnectionList)
	if conn := m.connectionList.SelectedConnection(); conn != nil {
		return m.handleSelectedConnection(conn)
	}
	return nil
}

func (m *Model) handleSelectedConnection(conn *config.SSHConnection) tea.Cmd {
	if conn.UsePassword && conn.Password == "" {
		// Retrieve the password from the keyring
		password, err := keyring.Get(keyringService, conn.ID)
		if err != nil {
			log.Printf("Failed to retrieve password from keyring for connection ID %s: %v", conn.ID, err)
			// Return message to show password prompt
			return func() tea.Msg {
				return components.SSHPasswordRequiredMsg{
					Connection: *conn,
				}
			}
		}
		conn.Password = password // Set the password from the keyring
		log.Printf("Password successfully retrieved for connection ID: %s", conn.ID)
	}

	openInNewWindow := m.connectionList.OpenInNewTerminal()
	isWindows := runtime.GOOS == "windows"
	keyPath, err := m.prepareKeyFileIfNeeded(conn)
	if err != nil {
		m.errorMessage = fmt.Sprintf("Failed to write key file: %s", err)
		return nil
	}
	
	// Update connection's KeyFile to point to the xterm_keys path if we created one
	if keyPath != "" {
		conn.KeyFile = keyPath
		// Clear Password field since it contained the key content, not a passphrase
		// If a passphrase is needed, the SSH client will trigger the passphrase form
		conn.Password = ""
	}
	
	sshArgs := m.prepareSSHArgs(conn, keyPath)
	userHost := fmt.Sprintf("%s@%s", conn.Username, conn.Host)

	if !isWindows {
		if openInNewWindow {
			m.launchTmuxWindow(conn, sshArgs)
			m.state = StateConnectionList
			m.connectionList.Reset()
			return nil
		}
		m.terminal = components.NewTerminalComponent(*conn)
		m.state = StateSSHTerminal
		m.connectionList.Reset()

		// Send initial size to terminal component so it can start the SSH session
		initCmd := m.terminal.Init()
		contentHeight := max(m.height-headerHeight-footerHeight, 12)
		sizeMsg := tea.WindowSizeMsg{
			Width:  m.width,
			Height: contentHeight,
		}
		_, sizeCmd := m.terminal.Update(sizeMsg)
		return tea.Batch(initCmd, sizeCmd)
	}
	// Windows
	if openInNewWindow {
		m.launchWindowsTerminal(conn, sshArgs, keyPath, userHost)
		m.state = StateConnectionList
		m.connectionList.Reset()
		return nil
	}
	m.terminal = components.NewTerminalComponent(*conn)
	m.state = StateSSHTerminal
	m.connectionList.Reset()

	// Send initial size to terminal component so it can start the SSH session
	initCmd := m.terminal.Init()
	contentHeight := max(m.height-headerHeight-footerHeight, 12)
	sizeMsg := tea.WindowSizeMsg{
		Width:  m.width,
		Height: contentHeight,
	}
	_, sizeCmd := m.terminal.Update(sizeMsg)
	return tea.Batch(initCmd, sizeCmd)
}

func (m *Model) prepareKeyFileIfNeeded(conn *config.SSHConnection) (string, error) {
	if !conn.UsePassword && conn.Password != "" {
		return getKeyFile(*conn)
	}
	return "", nil
}

func (m *Model) prepareSSHArgs(conn *config.SSHConnection, keyPath string) []string {
	args := []string{}
	if !conn.UsePassword && keyPath != "" {
		args = append(args, "-i", keyPath)
	}
	if conn.Port != 22 && conn.Port != 0 {
		args = append(args, "-p", strconv.Itoa(conn.Port))
	}
	userHost := fmt.Sprintf("%s@%s", conn.Username, conn.Host)
	args = append(args, userHost)
	return args
}

func (m *Model) launchTmuxWindow(conn *config.SSHConnection, sshArgs []string) {
	var sshCommand string
	usedPassh := false
	if conn.UsePassword && conn.Password != "" {
		if _, err := exec.LookPath("passh"); err == nil {
			usedPassh = true
			sshCommand = fmt.Sprintf(`passh -p %q ssh %s`, conn.Password, strings.Join(sshArgs, " "))
		} else {
			log.Print("passh not found, falling back to manual password entry or key-based login.")
			sshCommand = fmt.Sprintf("ssh %s", strings.Join(sshArgs, " "))
		}
	} else {
		sshCommand = fmt.Sprintf("ssh %s", strings.Join(sshArgs, " "))
	}

	windowName := fmt.Sprintf("%s@%s:%d - %s", conn.Username, conn.Host, conn.Port, conn.Name)
	cmd := exec.Command("tmux", "new-window", "-n", windowName, sshCommand)
	if err := cmd.Start(); err != nil {
		log.Printf("Error launching tmux window: %v", err)
	}
	if conn.UsePassword && conn.Password != "" && !usedPassh {
		log.Print("Password authentication not supported in this mode. Use manual entry or key-based login, or install passh.")
	}
}

func (m *Model) launchWindowsTerminal(conn *config.SSHConnection, sshArgs []string, keyPath, userHost string) {
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
	if err := cmd.Start(); err != nil {
		log.Printf("Error launching terminal: %v", err)
	}
	if conn.UsePassword && conn.Password != "" && !usePlink {
		log.Print("Password authentication not supported in this mode. Use manual entry, key-based login, or install plink.exe.")
	}
}
