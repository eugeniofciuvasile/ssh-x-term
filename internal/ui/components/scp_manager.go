package components

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/eugeniofciuvasile/ssh-x-term/internal/config"
	"github.com/eugeniofciuvasile/ssh-x-term/internal/ssh"
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
		Client     *ssh.SFTPClient
		WorkingDir string
		Err        error
	}
)

// InputMode represents the current input mode
type InputMode int

const (
	ModeNormal InputMode = iota
	ModeSearch
	ModeCreateFile
	ModeRename
	ModeChangeDir
	ModeConfirmDelete
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
	deleteTarget        *ssh.FileInfo  // File to delete (pending confirmation)
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
		s.remotePanel.Path = msg.WorkingDir

		return s, s.listRemoteFiles()

	case SSHPassphraseRequiredMsg:
		return s, func() tea.Msg {
			return msg
		}

	case SSHPasswordRequiredMsg:
		return s, func() tea.Msg {
			return msg
		}

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
		"%s@%s:%d - %s",
		s.connection.Username, s.connection.Host, s.connection.Port, s.connection.Name,
	)
	header := scpHeaderStyle.Width(s.width).Render(headerText)

	// Build status/footer with input prompt if in input mode
	var statusText string

	// Styles for status messages
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true) // Green
	errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)      // Red
	normalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))            // Grey/Normal

	// Container style ensures it is centered and full width
	containerStyle := scpStatusStyle.Width(s.width).Align(lipgloss.Center)

	if s.error != "" {
		// Render Error (Red)
		msg := errStyle.Render(s.error)
		statusText = containerStyle.Render(msg)
		s.error = "" // Clear error after displaying
	} else if s.inputMode != ModeNormal {
		// Show input prompt
		prompt := s.status + s.inputBuffer
		if s.inputMode == ModeSearch && len(s.searchMatches) > 0 {
			prompt += fmt.Sprintf(" [%d/%d matches]", s.searchSelectedIdx+1, len(s.searchMatches))
		}
		statusText = containerStyle.Render(prompt)
	} else {
		// Show Status
		if strings.Contains(strings.ToLower(s.status), "successfully") {
			// Render Success (Green)
			msg := successStyle.Render(s.status)
			statusText = containerStyle.Render(msg)
		} else {
			// Render Normal
			msg := normalStyle.Render(s.status)
			statusText = containerStyle.Render(msg)
		}
	}

	// Calculate remaining height for content panels
	// 2 lines for header + 1 line filter + 2 line for footer = 5 lines reserved
	contentHeight := max(s.height-5, 0)

	// Build content with split panels
	content := s.renderPanels(contentHeight)

	return lipgloss.JoinVertical(lipgloss.Left, header, content, statusText)
}

// renderPanels renders the split panel view
func (s *SCPManager) renderPanels(availableHeight int) string {
	if s.loading {
		return lipgloss.NewStyle().
			Height(availableHeight).
			Width(s.width).
			Align(lipgloss.Center, lipgloss.Center).
			Render("Connecting to remote server...")
	}

	// Calculate panel dimensions
	// Width: Split in half, minus border spacing (approx 2 chars for borders/padding)
	panelWidth := max((s.width/2)-2, 10)

	// Height: available height minus borders (5 lines for top/bottom borders)
	panelHeight := max(availableHeight-5, 0)

	// Render local panel
	localTitle := "Local: " + s.localPanel.Path
	localContent := s.renderPanelContent(&s.localPanel, panelHeight, panelWidth)

	var localPanel string
	localStyle := scpPanelStyle
	if s.activePanel == 0 {
		localStyle = scpActivePanelStyle
	}

	localPanel = localStyle.
		Width(panelWidth).
		Height(availableHeight - 2).
		Render(lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Width(panelWidth).Render(localTitle),
			"",
			localContent,
		))

	// Render remote panel
	remoteTitle := "Remote: " + s.remotePanel.Path
	remoteContent := s.renderPanelContent(&s.remotePanel, panelHeight, panelWidth)

	var remotePanel string
	remoteStyle := scpPanelStyle
	if s.activePanel == 1 {
		remoteStyle = scpActivePanelStyle
	}

	remotePanel = remoteStyle.
		Width(panelWidth).
		Height(availableHeight - 2).
		Render(lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Width(panelWidth).Render(remoteTitle),
			"",
			remoteContent,
		))

	// Join panels horizontally
	return lipgloss.JoinHorizontal(lipgloss.Top, localPanel, remotePanel)
}

// renderPanelContent renders the file list for a panel
func (s *SCPManager) renderPanelContent(panel *Panel, maxHeight int, maxWidth int) string {
	if len(panel.Files) == 0 {
		return "  (empty directory)"
	}

	if panel.SelectedIdx < panel.ScrollOffset {
		panel.ScrollOffset = panel.SelectedIdx
	}
	if panel.SelectedIdx >= panel.ScrollOffset+maxHeight {
		panel.ScrollOffset = panel.SelectedIdx - maxHeight + 1
	}

	var lines []string
	visibleStart := panel.ScrollOffset
	visibleEnd := min(visibleStart+maxHeight, len(panel.Files))

	// Define styles for columns
	nameStyle := lipgloss.NewStyle()
	sizeStyle := lipgloss.NewStyle().Width(8).Align(lipgloss.Right).Foreground(colorSubText)
	dateStyle := lipgloss.NewStyle().Width(12).Align(lipgloss.Right).Foreground(colorInactive)
	permStyle := lipgloss.NewStyle().Width(10).Align(lipgloss.Right).Foreground(colorInactive)
	metaStyle := lipgloss.NewStyle().Width(10).Align(lipgloss.Right).Foreground(colorInactive)

	// Calculate available width for name
	// Structure: [Icon 2][Sp 1][Name ?][Sp 2][Size 8][Sp 1][Date 12][Sp 1][Perm 10][Sp 1][Meta 10]
	// Strict Sum: 2 + 1 + 2 + 8 + 1 + 12 + 1 + 10 + 1 + 10 = 48
	// We use 52 to provide a small buffer against edge-wrapping
	fixedWidths := 52
	nameWidth := max(maxWidth-fixedWidths, 10)

	for i := visibleStart; i < visibleEnd; i++ {
		file := panel.Files[i]

		icon := s.getFileIcon(file)

		// Format Name with strict truncation
		name := file.Name
		if len(name) > nameWidth {
			name = name[:nameWidth-1] + "…"
		}

		var nameRendered string
		if file.IsDir {
			nameRendered = scpDirStyle.Width(nameWidth).Render(name)
		} else {
			nameRendered = nameStyle.Width(nameWidth).Render(name)
		}

		// Format Metadata
		sizeStr := formatSize(file.Size)
		dateStr := file.ModTime.Format("Jan 02 15:04")
		permStr := file.Perm
		ownerStr := fmt.Sprintf("%s:%s", file.Owner, file.Group)
		if len(ownerStr) > 10 {
			ownerStr = ownerStr[:9] + "…"
		}

		// Build Row with spaces
		line := lipgloss.JoinHorizontal(lipgloss.Bottom,
			icon, " ",
			nameRendered,
			"  ", // 2 spaces gap after name
			sizeStyle.Render(sizeStr), " ",
			dateStyle.Render(dateStr), " ",
			permStyle.Render(permStr), " ",
			metaStyle.Render(ownerStr),
		)

		if i == panel.SelectedIdx {
			line = scpSelectedStyle.Render(line)
		}

		lines = append(lines, line)
	}

	remainingLines := maxHeight - len(lines)
	if remainingLines > 0 {
		lines = append(lines, strings.Repeat("\n", remainingLines))
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
	// Handle input modes first - route ALL input modes to handleInputMode
	switch s.inputMode {
	case ModeSearch, ModeCreateFile, ModeRename, ModeChangeDir, ModeConfirmDelete:
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
			// Auto-scroll logic handled in view
		}
		return s, nil

	case "down", "j":
		panel := s.getActivePanel()
		if panel.SelectedIdx < len(panel.Files)-1 {
			panel.SelectedIdx++
			// Auto-scroll logic handled in view
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
		// Delete file - show confirmation first
		panel := s.getActivePanel()
		if panel.SelectedIdx >= 0 && panel.SelectedIdx < len(panel.Files) {
			file := panel.Files[panel.SelectedIdx]
			s.deleteTarget = &file
			s.inputMode = ModeConfirmDelete
			typeStr := "file"
			if file.IsDir {
				typeStr = "directory"
			}
			s.status = fmt.Sprintf("Delete %s '%s'? (y/n): ", typeStr, file.Name)
		}
		return s, nil

	case "c":
		// Change directory - cd command
		s.inputMode = ModeChangeDir
		s.inputBuffer = ""
		s.status = "Change directory (absolute path): "
		return s, nil

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
	// Special handling for delete confirmation
	if s.inputMode == ModeConfirmDelete {
		return s.handleDeleteConfirmation(msg)
	}

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
		s.deleteTarget = nil
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
		case ModeChangeDir:
			return s.executeChangeDir()
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
		// Silently skip directories that cannot be accessed during search
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
				Name:    fullRelPath,
				Size:    file.Size,
				IsDir:   file.IsDir,
				Mode:    file.Mode,
				ModTime: file.ModTime,
				Perm:    file.Perm,
				Owner:   file.Owner,
				Group:   file.Group,
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
		s.error = "File name cannot be empty"
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
		s.error = "New name cannot be empty"
		s.inputMode = ModeNormal
		return s, nil
	}

	panel := s.getActivePanel()
	if panel.SelectedIdx < 0 || panel.SelectedIdx >= len(panel.Files) {
		s.error = "No file selected"
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

// handleDeleteConfirmation handles the delete confirmation prompt
func (s *SCPManager) handleDeleteConfirmation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		// Confirm delete
		if s.deleteTarget == nil {
			s.inputMode = ModeNormal
			s.status = "No file to delete"
			return s, nil
		}

		file := *s.deleteTarget
		panel := s.getActivePanel()
		filePath := filepath.Join(panel.Path, file.Name)

		s.operationInProgress = true
		s.inputMode = ModeNormal
		s.deleteTarget = nil
		s.status = fmt.Sprintf("Deleting %s...", file.Name)

		return s, func() tea.Msg {
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

	case "n", "N", "esc":
		// Cancel delete
		s.inputMode = ModeNormal
		s.deleteTarget = nil
		s.status = "Delete cancelled"
		return s, nil

	default:
		// Ignore other keys
		return s, nil
	}
}

// executeChangeDir changes to the specified directory
func (s *SCPManager) executeChangeDir() (tea.Model, tea.Cmd) {
	if s.inputBuffer == "" {
		s.error = "Path cannot be empty"
		s.inputMode = ModeNormal
		return s, nil
	}

	panel := s.getActivePanel()
	newPath := s.inputBuffer

	// Clean up the path
	if !filepath.IsAbs(newPath) {
		// If relative path, make it absolute from current directory
		newPath = filepath.Join(panel.Path, newPath)
	}
	newPath = filepath.Clean(newPath)

	s.inputMode = ModeNormal
	s.inputBuffer = ""
	s.operationInProgress = true
	s.status = "Changing directory..."

	// Validate directory exists before changing
	return s, func() tea.Msg {
		var files []ssh.FileInfo
		var err error

		if s.activePanel == 0 {
			files, err = ssh.ListLocalFiles(newPath)
		} else {
			if s.sftpClient == nil {
				return SCPListFilesMsg{IsLocal: s.activePanel == 0, Err: fmt.Errorf("not connected")}
			}
			files, err = s.sftpClient.ListFiles(newPath)
		}

		if err != nil {
			s.operationInProgress = false
			return SCPOperationMsg{Operation: "Change directory", Success: false, Err: fmt.Errorf("directory does not exist or is inaccessible")}
		}

		// Directory is valid, update panel
		panel.Path = newPath
		panel.SelectedIdx = 0
		panel.ScrollOffset = 0
		s.operationInProgress = false

		return SCPListFilesMsg{IsLocal: s.activePanel == 0, Files: files, Path: newPath}
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
			var passphraseErr *ssh.PassphraseRequiredError
			if errors.As(err, &passphraseErr) {
				return SSHPassphraseRequiredMsg{
					Connection: s.connection,
					KeyFile:    passphraseErr.KeyFile,
				}
			}
			var passwordErr *ssh.PasswordRequiredError
			if errors.As(err, &passwordErr) {
				return SSHPasswordRequiredMsg{
					Connection: s.connection,
				}
			}
			return SCPConnectionMsg{nil, "", err}
		}

		// Fetch remote WD
		wd, err := client.GetWorkingDir()
		if err != nil {
			wd = "." // Fallback
		}

		return SCPConnectionMsg{client, wd, nil}
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

	s.operationInProgress = true
	s.status = "Entering directory..."

	return func() tea.Msg {
		var files []ssh.FileInfo
		var err error

		if s.activePanel == 0 {
			files, err = ssh.ListLocalFiles(newPath)
		} else {
			if s.sftpClient == nil {
				s.operationInProgress = false
				return SCPOperationMsg{Operation: "Enter directory", Success: false, Err: fmt.Errorf("not connected")}
			}
			files, err = s.sftpClient.ListFiles(newPath)
		}

		if err != nil {
			s.operationInProgress = false
			return SCPOperationMsg{Operation: "Enter directory", Success: false, Err: fmt.Errorf("cannot access directory: %w", err)}
		}

		// Update panel path
		panel.Path = newPath
		panel.SelectedIdx = 0
		panel.ScrollOffset = 0
		s.operationInProgress = false

		return SCPListFilesMsg{IsLocal: s.activePanel == 0, Files: files, Path: newPath}
	}
}

func (s *SCPManager) goUpDirectory() tea.Cmd {
	panel := s.getActivePanel()
	parent := filepath.Dir(panel.Path)
	if parent == panel.Path {
		s.error = "Already at root directory"
		return nil
	}

	s.operationInProgress = true
	s.status = "Going up directory..."

	return func() tea.Msg {
		var files []ssh.FileInfo
		var err error

		if s.activePanel == 0 {
			files, err = ssh.ListLocalFiles(parent)
		} else {
			if s.sftpClient == nil {
				s.operationInProgress = false
				return SCPOperationMsg{Operation: "Go up directory", Success: false, Err: fmt.Errorf("not connected")}
			}
			files, err = s.sftpClient.ListFiles(parent)
		}

		if err != nil {
			s.operationInProgress = false
			return SCPOperationMsg{Operation: "Go up directory", Success: false, Err: fmt.Errorf("cannot access parent directory: %w", err)}
		}

		// Update panel path
		panel.Path = parent
		panel.SelectedIdx = 0
		panel.ScrollOffset = 0
		s.operationInProgress = false

		return SCPListFilesMsg{IsLocal: s.activePanel == 0, Files: files, Path: parent}
	}
}

func (s *SCPManager) downloadFile() tea.Cmd {
	if s.sftpClient == nil {
		s.error = "Not connected to remote server"
		return nil
	}

	if s.remotePanel.SelectedIdx >= len(s.remotePanel.Files) {
		s.error = "No file selected"
		return nil
	}

	file := s.remotePanel.Files[s.remotePanel.SelectedIdx]

	s.operationInProgress = true
	if file.IsDir {
		s.status = "Downloading directory " + file.Name + " (recursive)..."
	} else {
		s.status = "Downloading " + file.Name + "..."
	}

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
		s.error = "Not connected to remote server"
		return nil
	}

	if s.localPanel.SelectedIdx >= len(s.localPanel.Files) {
		s.error = "No file selected"
		return nil
	}

	file := s.localPanel.Files[s.localPanel.SelectedIdx]

	s.operationInProgress = true
	if file.IsDir {
		s.status = "Uploading directory " + file.Name + " (recursive)..."
	} else {
		s.status = "Uploading " + file.Name + "..."
	}

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
