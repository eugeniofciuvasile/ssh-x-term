package cli

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/eugeniofciuvasile/ssh-x-term/internal/config"
)

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("170"))
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("170")).Bold(true)
	normalStyle   = lipgloss.NewStyle()
	filterStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

type SelectorModel struct {
	connections     []config.SSHConnection
	filteredIndices []int
	cursor          int
	filter          string
	choice          *config.SSHConnection
	quitting        bool
	width           int
	height          int
}

func NewSelector(connections []config.SSHConnection) *SelectorModel {
	indices := make([]int, len(connections))
	for i := range indices {
		indices[i] = i
	}

	return &SelectorModel{
		connections:     connections,
		filteredIndices: indices,
		cursor:          0,
		filter:          "",
		width:           80,
		height:          20,
	}
}

func (m *SelectorModel) Init() tea.Cmd {
	return nil
}

func (m *SelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.quitting = true
			return m, tea.Quit

		case tea.KeyEnter:
			if len(m.filteredIndices) > 0 && m.cursor < len(m.filteredIndices) {
				idx := m.filteredIndices[m.cursor]
				m.choice = &m.connections[idx]
			}
			return m, tea.Quit

		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
			}

		case tea.KeyDown:
			if m.cursor < len(m.filteredIndices)-1 {
				m.cursor++
			}

		case tea.KeyBackspace, tea.KeyDelete:
			if len(m.filter) > 0 {
				m.filter = m.filter[:len(m.filter)-1]
				m.updateFilter()
			}

		case tea.KeyRunes, tea.KeySpace:
			// Add character to filter
			if msg.Type == tea.KeySpace {
				m.filter += " "
			} else if len(msg.Runes) > 0 {
				m.filter += string(msg.Runes)
			}
			m.updateFilter()
		}
	}

	return m, nil
}

func (m *SelectorModel) updateFilter() {
	m.cursor = 0
	if m.filter == "" {
		// No filter, show all
		m.filteredIndices = make([]int, len(m.connections))
		for i := range m.filteredIndices {
			m.filteredIndices[i] = i
		}
		return
	}

	// Filter connections by name
	filterLower := strings.ToLower(m.filter)
	m.filteredIndices = []int{}
	for i, conn := range m.connections {
		if strings.Contains(strings.ToLower(conn.Name), filterLower) ||
			strings.Contains(strings.ToLower(conn.Host), filterLower) ||
			strings.Contains(strings.ToLower(conn.Username), filterLower) {
			m.filteredIndices = append(m.filteredIndices, i)
		}
	}
}

func (m *SelectorModel) View() string {
	if m.choice != nil || m.quitting {
		return ""
	}

	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("SSH Connection Selector"))
	b.WriteString("\n\n")

	// Filter input
	if m.filter != "" {
		b.WriteString(filterStyle.Render("Filter: " + m.filter))
	} else {
		b.WriteString(helpStyle.Render("Type to filter..."))
	}
	b.WriteString("\n\n")

	maxVisible := 10
	start := 0
	if m.cursor >= maxVisible {
		start = m.cursor - maxVisible + 1
	}
	end := min(start+maxVisible, len(m.filteredIndices))

	if len(m.filteredIndices) == 0 {
		b.WriteString(helpStyle.Render("No matches found"))
		b.WriteString("\n")
	} else {
		for i := start; i < end; i++ {
			idx := m.filteredIndices[i]
			conn := m.connections[idx]

			line := fmt.Sprintf("%s (%s@%s:%d)",
				conn.Name, conn.Username, conn.Host, conn.Port)

			if i == m.cursor {
				b.WriteString(selectedStyle.Render("> " + line))
			} else {
				b.WriteString(normalStyle.Render("  " + line))
			}
			b.WriteString("\n")
		}

		// Pagination info
		if len(m.filteredIndices) > maxVisible {
			b.WriteString("\n")
			b.WriteString(helpStyle.Render(fmt.Sprintf("Showing %d-%d of %d connections",
				start+1, end, len(m.filteredIndices))))
		}
	}

	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("↑/↓: navigate • Enter: select • Esc/Ctrl+C: quit"))

	return b.String()
}

func (m *SelectorModel) Choice() *config.SSHConnection {
	return m.choice
}
