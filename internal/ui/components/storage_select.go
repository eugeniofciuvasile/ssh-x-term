package components

import (
	tea "github.com/charmbracelet/bubbletea"
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
	content += StyleTitle.Render("Choose storage backend:") + "\n\n"

	// Options
	for i, opt := range s.options {
		prefix := "  "
		style := StyleNormal
		if i == s.selectedIndex {
			style = StyleFocused
			prefix = StyleFocused.Render("❯ ")
		} else {
			prefix = StyleTextMuted.Render("  ")
		}
		content += prefix + style.Render(opt) + "\n"
	}

	content += "\n"

	return StyleContainer.Render(content)
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
