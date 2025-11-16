package components

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type StorageBackend int

const (
	StorageLocal StorageBackend = iota
	StorageBitwarden
)

var (
	storageSelectStyle = lipgloss.NewStyle().
				Padding(1, 2)

	storageSelectTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("205")).
				MarginBottom(1)
)

type StorageSelect struct {
	options       []string
	selectedIndex int
	chosen        bool
	canceled      bool
	width         int
	height        int
}

func NewStorageSelect() *StorageSelect {
	return &StorageSelect{
		options: []string{"Local (file)", "Bitwarden"},
	}
}

func (s *StorageSelect) Init() tea.Cmd {
	return nil
}

func (s *StorageSelect) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.width = msg.Width
		s.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if s.selectedIndex > 0 {
				s.selectedIndex--
			}
		case "down", "j":
			if s.selectedIndex < len(s.options)-1 {
				s.selectedIndex++
			}
		case "enter":
			s.chosen = true
			return s, nil
		case "esc":
			s.canceled = true
			return s, nil
		}
	}
	return s, nil
}

func (s *StorageSelect) View() string {
	var content string

	// Title
	content += storageSelectTitleStyle.Render("Choose storage backend:") + "\n\n"

	// Options
	for i, opt := range s.options {
		style := lipgloss.NewStyle()
		prefix := "  "
		if i == s.selectedIndex {
			style = style.Bold(true).Foreground(lipgloss.Color("205"))
			prefix = "> "
		}
		content += style.Render(prefix+opt) + "\n"
	}

	content += "\n"

	return storageSelectStyle.Render(content)
}

func (s *StorageSelect) SelectedBackend() StorageBackend {
	return StorageBackend(s.selectedIndex)
}

func (s *StorageSelect) IsChosen() bool {
	return s.chosen
}

func (s *StorageSelect) IsCanceled() bool {
	return s.canceled
}
