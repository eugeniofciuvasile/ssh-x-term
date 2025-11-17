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

// InputMode represents the current input mode
type InputMode int

const (
	ModeNormal InputMode = iota
	ModeSearch
	ModeCreateFile
	ModeRename
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
	inputMode           InputMode
	inputBuffer         string
	searchMatches       []int          // Indices of matching files in current panel
	searchSelectedIdx   int            // Current position in search results
	recursiveResults    []ssh.FileInfo // Files found in recursive search
	originalFiles       []ssh.FileInfo // Original file list before search
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
		inputMode:      ModeNormal,
		inputBuffer:    "",
		searchMatches:  []int{},
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

	// Build status/footer with input prompt if in input mode
	var statusText string
	if s.error != "" {
		statusText = scpErrorStyle.Render(s.error)
		s.error = "" // Clear error after displaying
	} else if s.inputMode != ModeNormal {
		// Show input prompt
		prompt := s.status + s.inputBuffer
		if s.inputMode == ModeSearch && len(s.searchMatches) > 0 {
			prompt += fmt.Sprintf(" [%d/%d matches]", s.searchSelectedIdx+1, len(s.searchMatches))
		}
		statusText = scpStatusStyle.Width(s.width).Render(prompt)
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

		icon := s.getFileIcon(file)

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

// getFileIcon returns an appropriate icon for the file based on its type
func (s *SCPManager) getFileIcon(file ssh.FileInfo) string {
	// Hidden files (starting with .)
	if strings.HasPrefix(file.Name, ".") && file.Name != ".." {
		if file.IsDir {
			return "📂" // Hidden directory
		}
		return "🔒" // Hidden file
	}

	// Directories
	if file.IsDir {
		return "📁"
	}

	// Check if executable (has execute permission)
	if strings.Contains(file.Mode, "x") {
		return "⚙️"
	}

	// File extensions
	ext := strings.ToLower(filepath.Ext(file.Name))
	switch ext {
	// Programming languages
	case ".go":
		return "🐹"
	case ".py":
		return "🐍"
	case ".js", ".ts", ".jsx", ".tsx":
		return "📜"
	case ".java":
		return "☕"
	case ".c", ".cpp", ".cc", ".h", ".hpp":
		return "🔧"
	case ".rs":
		return "🦀"
	case ".rb":
		return "💎"
	case ".php":
		return "🐘"
	case ".sh", ".bash", ".zsh":
		return "🐚"

	// Web files
	case ".html", ".htm":
		return "🌐"
	case ".css", ".scss", ".sass":
		return "🎨"
	case ".json", ".xml", ".yaml", ".yml", ".toml":
		return "⚙️"

	// Documents
	case ".md", ".markdown":
		return "📝"
	case ".txt", ".log":
		return "📄"
	case ".pdf":
		return "📕"
	case ".doc", ".docx":
		return "📘"

	// Images
	case ".png", ".jpg", ".jpeg", ".gif", ".bmp", ".svg", ".ico":
		return "🖼️"

	// Archives
	case ".zip", ".tar", ".gz", ".bz2", ".xz", ".7z", ".rar":
		return "📦"

	// Media
	case ".mp3", ".wav", ".flac", ".ogg":
		return "🎵"
	case ".mp4", ".avi", ".mkv", ".mov":
		return "🎬"

	// Data/Database
	case ".db", ".sqlite", ".sql":
		return "💾"

	// Config files
	case ".conf", ".config", ".ini", ".env":
		return "⚙️"

	// Default
	default:
		return "📄"
	}
}

// handleKey handles keyboard input
func (s *SCPManager) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle input modes first
	switch s.inputMode {
	case ModeSearch, ModeCreateFile, ModeRename:
		return s.handleInputMode(msg)
	}

	// Normal mode key handling
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
		// Create new file
		s.inputMode = ModeCreateFile
		s.inputBuffer = ""
		s.status = "Create file (use / for directories): "
		return s, nil

	case "r":
		// Rename file
		panel := s.getActivePanel()
		if panel.SelectedIdx >= 0 && panel.SelectedIdx < len(panel.Files) {
			s.inputMode = ModeRename
			s.inputBuffer = panel.Files[panel.SelectedIdx].Name
			s.status = "Rename to: "
		}
		return s, nil

	case "d", "x":
		// Delete file
		return s, s.deleteFile()

	case "/":
		// Search mode - now recursive
		s.inputMode = ModeSearch
		s.inputBuffer = ""
		s.searchMatches = []int{}
		s.searchSelectedIdx = 0
		s.status = "Recursive search: "
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

// handleInputMode handles key input when in search, create, or rename mode
func (s *SCPManager) handleInputMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel input mode and restore original files if in search mode
		if s.inputMode == ModeSearch && len(s.originalFiles) > 0 {
			panel := s.getActivePanel()
			panel.Files = s.originalFiles
			s.originalFiles = nil
			panel.SelectedIdx = 0
			panel.ScrollOffset = 0
		}
		s.inputMode = ModeNormal
		s.inputBuffer = ""
		s.searchMatches = []int{}
		s.recursiveResults = []ssh.FileInfo{}
		s.status = "Cancelled"
		return s, nil

	case "enter":
		// Execute the action based on mode
		switch s.inputMode {
		case ModeSearch:
			return s.executeSearch()
		case ModeCreateFile:
			return s.executeCreateFile()
		case ModeRename:
			return s.executeRename()
		}
		return s, nil

	case "backspace":
		// Remove last character
		if len(s.inputBuffer) > 0 {
			s.inputBuffer = s.inputBuffer[:len(s.inputBuffer)-1]
		}
		// Update search results in real-time
		if s.inputMode == ModeSearch {
			s.updateSearchResults()
		}
		return s, nil

	case "ctrl+u":
		// Clear input buffer
		s.inputBuffer = ""
		if s.inputMode == ModeSearch {
			s.searchMatches = []int{}
		}
		return s, nil

	case "up", "ctrl+p":
		// Navigate search results
		if s.inputMode == ModeSearch && len(s.searchMatches) > 0 {
			if s.searchSelectedIdx > 0 {
				s.searchSelectedIdx--
			}
		}
		return s, nil

	case "down", "ctrl+n":
		// Navigate search results
		if s.inputMode == ModeSearch && len(s.searchMatches) > 0 {
			if s.searchSelectedIdx < len(s.searchMatches)-1 {
				s.searchSelectedIdx++
			}
		}
		return s, nil

	default:
		// Add character to input buffer if it's a printable character
		if len(msg.String()) == 1 {
			s.inputBuffer += msg.String()
			// Update search results in real-time
			if s.inputMode == ModeSearch {
				s.updateSearchResults()
			}
		}
		return s, nil
	}
}

// updateSearchResults performs recursive fuzzy search from current directory
func (s *SCPManager) updateSearchResults() {
	panel := s.getActivePanel()
	s.recursiveResults = []ssh.FileInfo{}
	s.searchMatches = []int{}
	s.searchSelectedIdx = 0

	if s.inputBuffer == "" {
		// Restore original files if search is cleared
		if len(s.originalFiles) > 0 {
			panel.Files = s.originalFiles
			s.originalFiles = nil
		}
		return
	}

	// Store original files if not already stored
	if len(s.originalFiles) == 0 {
		s.originalFiles = make([]ssh.FileInfo, len(panel.Files))
		copy(s.originalFiles, panel.Files)
	}

	query := strings.ToLower(s.inputBuffer)

	// Start recursive search
	s.recursiveSearchDir(panel.Path, query, "")

	// Update panel with search results
	if len(s.recursiveResults) > 0 {
		panel.Files = s.recursiveResults
		panel.SelectedIdx = 0
		panel.ScrollOffset = 0
		// All results are matches, so searchMatches contains all indices
		for i := range s.recursiveResults {
			s.searchMatches = append(s.searchMatches, i)
		}
	} else {
		// No results, show empty
		panel.Files = []ssh.FileInfo{}
	}
}

// recursiveSearchDir recursively searches for files matching the query
func (s *SCPManager) recursiveSearchDir(basePath, query, relativePath string) {
	var files []ssh.FileInfo
	var err error

	currentPath := filepath.Join(basePath, relativePath)

	if s.activePanel == 0 {
		// Local search
		files, err = ssh.ListLocalFiles(currentPath)
	} else {
		// Remote search
		if s.sftpClient == nil {
			return
		}
		files, err = s.sftpClient.ListFiles(currentPath)
	}

	if err != nil {
		return
	}

	for _, file := range files {
		// Skip . and ..
		if file.Name == "." || file.Name == ".." {
			continue
		}

		fullRelPath := filepath.Join(relativePath, file.Name)

		// Check if filename matches
		if s.fuzzyMatch(query, strings.ToLower(file.Name)) {
			// Create a new FileInfo with relative path in name for display
			displayFile := ssh.FileInfo{
				Name:  fullRelPath,
				Size:  file.Size,
				IsDir: file.IsDir,
				Mode:  file.Mode,
			}
			s.recursiveResults = append(s.recursiveResults, displayFile)
		}

		// Recurse into directories (limit depth to prevent infinite loops)
		if file.IsDir && len(strings.Split(fullRelPath, string(filepath.Separator))) < 10 {
			s.recursiveSearchDir(basePath, query, fullRelPath)
		}
	}
}

// fuzzyMatch performs a simple fuzzy match
func (s *SCPManager) fuzzyMatch(query, target string) bool {
	if query == "" {
		return true
	}

	qi := 0
	for _, c := range target {
		if qi < len(query) && rune(query[qi]) == c {
			qi++
		}
	}
	return qi == len(query)
}

// executeSearch finalizes the search and keeps filtered results
func (s *SCPManager) executeSearch() (tea.Model, tea.Cmd) {
	panel := s.getActivePanel()

	if len(s.recursiveResults) > 0 {
		// Keep the search results displayed
		s.status = fmt.Sprintf("Found %d matches (ESC to restore)", len(s.recursiveResults))
	} else {
		s.status = "No matches found"
		// Restore original files
		if len(s.originalFiles) > 0 {
			panel.Files = s.originalFiles
			s.originalFiles = nil
		}
	}

	s.inputMode = ModeNormal
	s.inputBuffer = ""
	return s, nil
}

// executeCreateFile creates a new file or directory structure
func (s *SCPManager) executeCreateFile() (tea.Model, tea.Cmd) {
	if s.inputBuffer == "" {
		s.status = "File name cannot be empty"
		s.inputMode = ModeNormal
		return s, nil
	}

	panel := s.getActivePanel()
	fullPath := filepath.Join(panel.Path, s.inputBuffer)

	// Check if path contains directory separators
	if strings.Contains(s.inputBuffer, "/") {
		// Create directory structure
		dir := filepath.Dir(fullPath)

		s.operationInProgress = true
		s.inputMode = ModeNormal
		s.status = "Creating file and directories..."

		return s, func() tea.Msg {
			var err error

			if s.activePanel == 0 {
				// Local file system
				// Create directories if they don't exist
				err = s.createLocalDirAndFile(dir, fullPath)
			} else {
				// Remote file system
				err = s.createRemoteDirAndFile(dir, fullPath)
			}

			if err != nil {
				return SCPOperationMsg{Operation: "Create file", Success: false, Err: err}
			}
			return SCPOperationMsg{Operation: "Create file", Success: true}
		}
	} else {
		// Simple file creation
		s.operationInProgress = true
		s.inputMode = ModeNormal
		s.status = "Creating file..."

		return s, func() tea.Msg {
			var err error

			if s.activePanel == 0 {
				// Local file
				err = ssh.CreateLocalFile(fullPath)
			} else {
				// Remote file
				if s.sftpClient == nil {
					return SCPOperationMsg{Operation: "Create file", Success: false, Err: fmt.Errorf("not connected")}
				}
				err = s.sftpClient.CreateFile(fullPath)
			}

			if err != nil {
				return SCPOperationMsg{Operation: "Create file", Success: false, Err: err}
			}
			return SCPOperationMsg{Operation: "Create file", Success: true}
		}
	}
}

// executeRename renames the selected file or directory
func (s *SCPManager) executeRename() (tea.Model, tea.Cmd) {
	if s.inputBuffer == "" {
		s.status = "New name cannot be empty"
		s.inputMode = ModeNormal
		return s, nil
	}

	panel := s.getActivePanel()
	if panel.SelectedIdx < 0 || panel.SelectedIdx >= len(panel.Files) {
		s.status = "No file selected"
		s.inputMode = ModeNormal
		return s, nil
	}

	oldFile := panel.Files[panel.SelectedIdx]
	oldPath := filepath.Join(panel.Path, oldFile.Name)
	newPath := filepath.Join(panel.Path, s.inputBuffer)

	s.operationInProgress = true
	s.inputMode = ModeNormal
	s.status = fmt.Sprintf("Renaming %s to %s...", oldFile.Name, s.inputBuffer)

	return s, func() tea.Msg {
		var err error

		if s.activePanel == 0 {
			// Local file
			err = ssh.RenameLocalFile(oldPath, newPath)
		} else {
			// Remote file
			if s.sftpClient == nil {
				return SCPOperationMsg{Operation: "Rename", Success: false, Err: fmt.Errorf("not connected")}
			}
			err = s.sftpClient.RenameFile(oldPath, newPath)
		}

		if err != nil {
			return SCPOperationMsg{Operation: "Rename", Success: false, Err: err}
		}
		return SCPOperationMsg{Operation: "Rename", Success: true}
	}
}

// createLocalDirAndFile creates directory structure and file locally
func (s *SCPManager) createLocalDirAndFile(dir, filePath string) error {
	return ssh.CreateLocalDirAndFile(dir, filePath)
}

// createRemoteDirAndFile creates directory structure and file remotely
func (s *SCPManager) createRemoteDirAndFile(dir, filePath string) error {
	if s.sftpClient == nil {
		return fmt.Errorf("not connected")
	}
	return s.sftpClient.CreateDirAndFile(dir, filePath)
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

func (s *SCPManager) deleteFile() tea.Cmd {
	panel := s.getActivePanel()
	if panel.SelectedIdx < 0 || panel.SelectedIdx >= len(panel.Files) {
		return nil
	}

	file := panel.Files[panel.SelectedIdx]
	filePath := filepath.Join(panel.Path, file.Name)

	s.operationInProgress = true
	s.status = fmt.Sprintf("Deleting %s...", file.Name)

	return func() tea.Msg {
		var err error

		if s.activePanel == 0 {
			// Local file
			err = ssh.DeleteLocalFile(filePath, file.IsDir)
		} else {
			// Remote file
			if s.sftpClient == nil {
				return SCPOperationMsg{Operation: "Delete", Success: false, Err: fmt.Errorf("not connected")}
			}
			err = s.sftpClient.DeleteFile(filePath, file.IsDir)
		}

		if err != nil {
			return SCPOperationMsg{Operation: "Delete", Success: false, Err: err}
		}
		return SCPOperationMsg{Operation: "Delete", Success: true}
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
