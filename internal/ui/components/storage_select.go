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
	descriptions  []string
	selectedIndex int
	chosen        bool
	canceled      bool
	width         int
	height        int
}

func NewStorageSelect() *StorageSelect {
	return &StorageSelect{
		options: []string{"Local Storage", "Bitwarden"},
		descriptions: []string{
			"Local JSON file (no password)",
			"Sync with Bitwarden vault",
		},
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
		// Support Left/Right navigation for the side-by-side layout
		case "left", "h", "up", "k":
			if s.selectedIndex > 0 {
				s.selectedIndex--
			}
		case "right", "l", "down", "j":
			if s.selectedIndex < len(s.options)-1 {
				s.selectedIndex++
			}
		case "enter":
			s.chosen = true
			return s, nil
		case "ctrl+c", "esc":
			s.canceled = true
			return s, nil
		}
	}
	return s, nil
}

func (s *StorageSelect) View() string {
	// --- Styles specific to this component ---

	// increasing width helps text wrapping, fixing height ensures alignment
	const (
		cardWidth  = 34
		cardHeight = 11
	)

	// Base style for the card content (inner)
	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Width(cardWidth).
		Height(cardHeight).
		Padding(1, 2).
		Align(lipgloss.Center, lipgloss.Center)

	// Active State (Purple)
	activeCardStyle := cardStyle.
		BorderForeground(colorPrimary).
		Foreground(colorText)

	// Inactive State (Gray)
	inactiveCardStyle := cardStyle.
		BorderForeground(colorInactive).
		Foreground(colorSubText)

	// Text Styles
	titleStyle := lipgloss.NewStyle().Bold(true).MarginTop(1).MarginBottom(1)
	descStyle := lipgloss.NewStyle().Foreground(colorSubText).Align(lipgloss.Center)

	// --- Render Cards ---

	var cards []string

	for i, option := range s.options {
		var currentCardStyle lipgloss.Style
		var currentTitleStyle lipgloss.Style
		var symbolColor lipgloss.Color
		var symbol string

		if i == s.selectedIndex {
			currentCardStyle = activeCardStyle
			currentTitleStyle = titleStyle.Foreground(colorPrimary)
			symbolColor = colorSecondary // Blue for active icon
		} else {
			currentCardStyle = inactiveCardStyle
			currentTitleStyle = titleStyle.Foreground(colorInactive)
			symbolColor = colorInactive
		}

		if i == 0 { // Local (Box)
			symbol =
				`   _______
  /      /|
 /______/ |
 |      | /
 |______|/`
		} else { // Secure (Key)
			symbol =
				`   .--.
  /.-. '----------.
  \'-' .--"--""-"-'
   '--'
      `
		}

		// Apply color to symbol
		renderedSymbol := lipgloss.NewStyle().Foreground(symbolColor).Render(symbol)

		content := lipgloss.JoinVertical(lipgloss.Center,
			renderedSymbol,
			currentTitleStyle.Render(option),
			descStyle.Render(s.descriptions[i]),
		)

		cards = append(cards, currentCardStyle.Render(content))
	}

	// --- Layout ---

	// Join cards horizontally with a larger gap
	ui := lipgloss.JoinHorizontal(lipgloss.Center,
		cards[0],
		"      ", // 6 spaces gap
		cards[1],
	)

	// Center vertically and horizontally in the full available space
	return lipgloss.Place(
		s.width,
		s.height-3, // Reserve space for global footer
		lipgloss.Center,
		lipgloss.Center,
		ui,
	)
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

// SetSize allows the parent model to manually set the dimensions
// This is crucial when returning to this view without a window resize event
func (s *StorageSelect) SetSize(width, height int) {
	s.width = width
	s.height = height
}
