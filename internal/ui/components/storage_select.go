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

type StorageSelect struct {
	options       []string
	selectedIndex int
	chosen        bool
	canceled      bool
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
	out := "Choose storage backend:\n\n"
	for i, opt := range s.options {
		style := lipgloss.NewStyle()
		prefix := "  "
		if i == s.selectedIndex {
			style = style.Bold(true).Foreground(lipgloss.Color("205"))
			prefix = "> "
		}
		out += style.Render(prefix+opt) + "\n"
	}
	out += "\n(Use ↑/↓ and Enter)"
	return out
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
