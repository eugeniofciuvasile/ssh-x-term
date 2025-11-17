package components

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/eugeniofciuvasile/ssh-x-term/internal/config"
	"github.com/eugeniofciuvasile/ssh-x-term/internal/ssh"
)

var (
	scpHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Background(lipgloss.Color("4")).
			Foreground(lipgloss.Color("255")).
			Align(lipgloss.Center).
			Padding(0, 1)

	scpPanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(1, 2)

	scpActivePanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("205")).
				Padding(1, 2)

	scpFileStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255"))

	scpDirStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Bold(true)

	scpSelectedStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("237")).
				Foreground(lipgloss.Color("255"))

	scpErrorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("9")).
			Padding(0, 2)

	scpStatusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Background(lipgloss.Color("235")).
			Padding(0, 2)
)

// Panel represents either local or remote file panel
type Panel struct {
	Path         string
	Files        []ssh.FileInfo
	SelectedIdx  int
	ScrollOffset int
}

// SCPManagerMsg types
type (
	SCPListFilesMsg struct {
		IsLocal bool
		Files   []ssh.FileInfo
		Path    string
		Err     error
	}

	SCPOperationMsg struct {
		Operation string
		Success   bool
		Err       error
	}

	SCPConnectionMsg struct {
		Client *ssh.SFTPClient
		Err    error
	}
)

// SCPManager represents the SCP file manager component
type SCPManager struct {
	connection          config.SSHConnection
	sftpClient          *ssh.SFTPClient
	localPanel          Panel
	remotePanel         Panel
	activePanel         int // 0 = local, 1 = remote
	width               int
	height              int
	status              string
	error               string
	loading             bool
	finished            bool
	lastEscTime         time.Time
	escPressCount       int
	escTimeoutSecs      float64
	operationInProgress bool
}

// NewSCPManager creates a new SCP file manager component
func NewSCPManager(conn config.SSHConnection) *SCPManager {
	homeDir := "."
	// Try to get home directory
	if h, err := filepath.Abs("."); err == nil {
		homeDir = h
	}

	return &SCPManager{
		connection:     conn,
		localPanel:     Panel{Path: homeDir, Files: []ssh.FileInfo{}, SelectedIdx: 0},
		remotePanel:    Panel{Path: ".", Files: []ssh.FileInfo{}, SelectedIdx: 0},
		activePanel:    0, // Start with local panel active
		status:         "Connecting...",
		loading:        true,
		escTimeoutSecs: 2.0,
	}
}

// Init initializes the component
func (s *SCPManager) Init() tea.Cmd {
	return tea.Batch(
		s.connectSFTP(),
		s.listLocalFiles(),
	)
}

// Update handles component updates
func (s *SCPManager) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if s.operationInProgress {
		// Don't process input while operation is in progress
		switch msg := msg.(type) {
		case SCPOperationMsg:
			s.operationInProgress = false
			if msg.Err != nil {
				s.error = fmt.Sprintf("%s failed: %s", msg.Operation, msg.Err.Error())
			} else {
				s.status = fmt.Sprintf("%s completed successfully", msg.Operation)
				// Refresh both panels after successful operation
				return s, tea.Batch(s.listLocalFiles(), s.listRemoteFiles())
			}
			return s, nil
		case tea.WindowSizeMsg:
			s.width = msg.Width
			s.height = msg.Height
			return s, nil
		}
		return s, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.width = msg.Width
		s.height = msg.Height
		return s, nil

	case SCPConnectionMsg:
		s.loading = false
		if msg.Err != nil {
			s.error = fmt.Sprintf("Failed to connect: %s", msg.Err.Error())
			s.status = "Connection failed"
			return s, nil
		}
		s.sftpClient = msg.Client
		s.status = "Connected"
		return s, s.listRemoteFiles()

	case SCPListFilesMsg:
		if msg.Err != nil {
			s.error = fmt.Sprintf("Failed to list files: %s", msg.Err.Error())
			return s, nil
		}
		if msg.IsLocal {
			s.localPanel.Files = msg.Files
			s.localPanel.Path = msg.Path
			if s.localPanel.SelectedIdx >= len(s.localPanel.Files) {
				s.localPanel.SelectedIdx = max(0, len(s.localPanel.Files)-1)
			}
		} else {
			s.remotePanel.Files = msg.Files
			s.remotePanel.Path = msg.Path
			if s.remotePanel.SelectedIdx >= len(s.remotePanel.Files) {
				s.remotePanel.SelectedIdx = max(0, len(s.remotePanel.Files)-1)
			}
		}
		return s, nil

	case SCPOperationMsg:
		s.operationInProgress = false
		if msg.Err != nil {
			s.error = fmt.Sprintf("%s failed: %s", msg.Operation, msg.Err.Error())
		} else {
			s.status = fmt.Sprintf("%s completed successfully", msg.Operation)
			// Refresh both panels after successful operation
			return s, tea.Batch(s.listLocalFiles(), s.listRemoteFiles())
		}
		return s, nil

	case tea.KeyMsg:
		return s.handleKey(msg)
	}

	return s, nil
}

// View renders the component
func (s *SCPManager) View() string {
	if s.finished {
		return ""
	}

	// Build header
	headerText := fmt.Sprintf(
		"SCP File Manager: %s@%s:%d - %s",
		s.connection.Username, s.connection.Host, s.connection.Port, s.connection.Name,
	)
	header := scpHeaderStyle.Width(s.width).Render(headerText)

	// Build content with split panels
	content := s.renderPanels()

	// Build status/footer
	var statusText string
	if s.error != "" {
		statusText = scpErrorStyle.Render(s.error)
		s.error = "" // Clear error after displaying
	} else {
		statusText = scpStatusStyle.Width(s.width).Render(s.status)
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, content, statusText)
}

// renderPanels renders the split panel view
func (s *SCPManager) renderPanels() string {
	if s.loading {
		return "\n  Connecting to remote server...\n"
	}

	// Calculate panel dimensions
	panelWidth := (s.width / 2) - 4
	panelHeight := s.height - 6 // Reserve space for header, footer, borders

	// Render local panel
	localTitle := "Local: " + s.localPanel.Path
	localContent := s.renderPanelContent(&s.localPanel, panelHeight)

	var localPanel string
	if s.activePanel == 0 {
		localPanel = scpActivePanelStyle.Width(panelWidth).Render(
			lipgloss.JoinVertical(lipgloss.Left, localTitle, "", localContent),
		)
	} else {
		localPanel = scpPanelStyle.Width(panelWidth).Render(
			lipgloss.JoinVertical(lipgloss.Left, localTitle, "", localContent),
		)
	}

	// Render remote panel
	remoteTitle := "Remote: " + s.remotePanel.Path
	remoteContent := s.renderPanelContent(&s.remotePanel, panelHeight)

	var remotePanel string
	if s.activePanel == 1 {
		remotePanel = scpActivePanelStyle.Width(panelWidth).Render(
			lipgloss.JoinVertical(lipgloss.Left, remoteTitle, "", remoteContent),
		)
	} else {
		remotePanel = scpPanelStyle.Width(panelWidth).Render(
			lipgloss.JoinVertical(lipgloss.Left, remoteTitle, "", remoteContent),
		)
	}

	// Join panels horizontally
	return lipgloss.JoinHorizontal(lipgloss.Top, localPanel, remotePanel)
}

// renderPanelContent renders the file list for a panel
func (s *SCPManager) renderPanelContent(panel *Panel, maxHeight int) string {
	if len(panel.Files) == 0 {
		return "  (empty directory)"
	}

	var lines []string
	visibleStart := panel.ScrollOffset
	visibleEnd := min(visibleStart+maxHeight, len(panel.Files))

	for i := visibleStart; i < visibleEnd; i++ {
		file := panel.Files[i]
		var line string

		icon := "  "
		if file.IsDir {
			icon = "📁"
		} else {
			icon = "📄"
		}

		fileName := file.Name
		if len(fileName) > 30 {
			fileName = fileName[:27] + "..."
		}

		if file.IsDir {
			line = fmt.Sprintf("%s %s", icon, scpDirStyle.Render(fileName))
		} else {
			sizeStr := formatSize(file.Size)
			line = fmt.Sprintf("%s %-30s %10s", icon, fileName, sizeStr)
		}

		if i == panel.SelectedIdx {
			line = scpSelectedStyle.Render(line)
		}

		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// handleKey handles keyboard input
func (s *SCPManager) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Double ESC to exit
		now := time.Now()
		timeSinceLastEsc := now.Sub(s.lastEscTime).Seconds()

		if s.escPressCount > 0 && timeSinceLastEsc <= s.escTimeoutSecs {
			s.finished = true
			if s.sftpClient != nil {
				s.sftpClient.Close()
			}
			s.escPressCount = 0
			s.lastEscTime = time.Time{}
			return s, nil
		}

		s.escPressCount = 1
		s.lastEscTime = now
		return s, nil

	case "tab":
		// Switch between panels
		s.activePanel = 1 - s.activePanel
		return s, nil

	case "up", "k":
		panel := s.getActivePanel()
		if panel.SelectedIdx > 0 {
			panel.SelectedIdx--
			// Auto-scroll
			if panel.SelectedIdx < panel.ScrollOffset {
				panel.ScrollOffset = panel.SelectedIdx
			}
		}
		return s, nil

	case "down", "j":
		panel := s.getActivePanel()
		if panel.SelectedIdx < len(panel.Files)-1 {
			panel.SelectedIdx++
			// Auto-scroll
			maxVisible := s.height - 8
			if panel.SelectedIdx >= panel.ScrollOffset+maxVisible {
				panel.ScrollOffset = panel.SelectedIdx - maxVisible + 1
			}
		}
		return s, nil

	case "enter":
		// Enter directory
		return s, s.enterDirectory()

	case "backspace", "h":
		// Go up one directory
		return s, s.goUpDirectory()

	case "g":
		// Get file (download from remote to local)
		if s.activePanel == 1 {
			return s, s.downloadFile()
		}
		return s, nil

	case "u":
		// Upload file (from local to remote)
		if s.activePanel == 0 {
			return s, s.uploadFile()
		}
		return s, nil

	case "n":
		// Create new file (not implemented yet)
		s.status = "Create file: not yet implemented"
		return s, nil

	case "r":
		// Rename file (not implemented yet)
		s.status = "Rename file: not yet implemented"
		return s, nil

	case "ctrl+l":
		// Refresh current panel
		if s.activePanel == 0 {
			return s, s.listLocalFiles()
		}
		return s, s.listRemoteFiles()
	}

	// Reset ESC tracking on any other key
	if s.escPressCount > 0 {
		s.escPressCount = 0
		s.lastEscTime = time.Time{}
	}

	return s, nil
}

// Helper functions

func (s *SCPManager) getActivePanel() *Panel {
	if s.activePanel == 0 {
		return &s.localPanel
	}
	return &s.remotePanel
}

func (s *SCPManager) connectSFTP() tea.Cmd {
	return func() tea.Msg {
		client, err := ssh.NewSFTPClient(s.connection)
		if err != nil {
			return SCPConnectionMsg{nil, err}
		}
		return SCPConnectionMsg{client, nil}
	}
}

func (s *SCPManager) listLocalFiles() tea.Cmd {
	return func() tea.Msg {
		files, err := ssh.ListLocalFiles(s.localPanel.Path)
		if err != nil {
			return SCPListFilesMsg{IsLocal: true, Err: err}
		}
		return SCPListFilesMsg{IsLocal: true, Files: files, Path: s.localPanel.Path}
	}
}

func (s *SCPManager) listRemoteFiles() tea.Cmd {
	return func() tea.Msg {
		if s.sftpClient == nil {
			return SCPListFilesMsg{IsLocal: false, Err: fmt.Errorf("not connected")}
		}
		files, err := s.sftpClient.ListFiles(s.remotePanel.Path)
		if err != nil {
			return SCPListFilesMsg{IsLocal: false, Err: err}
		}
		return SCPListFilesMsg{IsLocal: false, Files: files, Path: s.remotePanel.Path}
	}
}

func (s *SCPManager) enterDirectory() tea.Cmd {
	panel := s.getActivePanel()
	if panel.SelectedIdx >= len(panel.Files) {
		return nil
	}

	file := panel.Files[panel.SelectedIdx]
	if !file.IsDir {
		return nil // Not a directory
	}

	newPath := filepath.Join(panel.Path, file.Name)
	panel.Path = newPath
	panel.SelectedIdx = 0
	panel.ScrollOffset = 0

	if s.activePanel == 0 {
		return s.listLocalFiles()
	}
	return s.listRemoteFiles()
}

func (s *SCPManager) goUpDirectory() tea.Cmd {
	panel := s.getActivePanel()
	parent := filepath.Dir(panel.Path)
	if parent == panel.Path {
		return nil // Already at root
	}

	panel.Path = parent
	panel.SelectedIdx = 0
	panel.ScrollOffset = 0

	if s.activePanel == 0 {
		return s.listLocalFiles()
	}
	return s.listRemoteFiles()
}

func (s *SCPManager) downloadFile() tea.Cmd {
	if s.sftpClient == nil {
		return nil
	}

	if s.remotePanel.SelectedIdx >= len(s.remotePanel.Files) {
		return nil
	}

	file := s.remotePanel.Files[s.remotePanel.SelectedIdx]
	if file.IsDir {
		s.status = "Cannot download directories (yet)"
		return nil
	}

	s.operationInProgress = true
	s.status = "Downloading " + file.Name + "..."

	remotePath := filepath.Join(s.remotePanel.Path, file.Name)
	localPath := filepath.Join(s.localPanel.Path, file.Name)

	return func() tea.Msg {
		err := s.sftpClient.DownloadFile(remotePath, localPath)
		if err != nil {
			return SCPOperationMsg{Operation: "Download", Success: false, Err: err}
		}
		return SCPOperationMsg{Operation: "Download", Success: true}
	}
}

func (s *SCPManager) uploadFile() tea.Cmd {
	if s.sftpClient == nil {
		return nil
	}

	if s.localPanel.SelectedIdx >= len(s.localPanel.Files) {
		return nil
	}

	file := s.localPanel.Files[s.localPanel.SelectedIdx]
	if file.IsDir {
		s.status = "Cannot upload directories (yet)"
		return nil
	}

	s.operationInProgress = true
	s.status = "Uploading " + file.Name + "..."

	localPath := filepath.Join(s.localPanel.Path, file.Name)
	remotePath := filepath.Join(s.remotePanel.Path, file.Name)

	return func() tea.Msg {
		err := s.sftpClient.UploadFile(localPath, remotePath)
		if err != nil {
			return SCPOperationMsg{Operation: "Upload", Success: false, Err: err}
		}
		return SCPOperationMsg{Operation: "Upload", Success: true}
	}
}

// IsFinished returns whether the component is finished
func (s *SCPManager) IsFinished() bool {
	return s.finished
}

// Utility functions

func formatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
