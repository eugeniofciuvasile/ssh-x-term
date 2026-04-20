package components

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type RenameModal struct {
	textInput   textinput.Model
	currentName string
	host        string
	confirmed   bool
	canceled    bool
	width       int
	height      int
}

func NewRenameModal(currentName, host string) *RenameModal {
	ti := textinput.New()
	ti.SetValue(currentName)
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 40
	ti.Prompt = "New Name: "
	ti.PromptStyle = focusedStyle
	ti.TextStyle = focusedStyle

	return &RenameModal{
		textInput:   ti,
		currentName: currentName,
		host:        host,
	}
}

func (m *RenameModal) Init() tea.Cmd {
	return textinput.Blink
}

func (m *RenameModal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if m.textInput.Value() != "" {
				m.confirmed = true
				return m, nil
			}
		case "esc":
			m.canceled = true
			return m, nil
		case "ctrl+c":
			m.canceled = true
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m *RenameModal) View() string {
	if m.canceled {
		return ""
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(colorPrimary).
		Render("✏ Rename Connection")

	details := lipgloss.NewStyle().
		Foreground(colorSubText).
		Render(fmt.Sprintf("Current: %s (%s)", m.currentName, m.host))

	prompt := lipgloss.NewStyle().
		Foreground(colorInactive).
		Render("Press Enter to confirm, Esc to cancel")

	content := lipgloss.JoinVertical(lipgloss.Center,
		title,
		"\n",
		details,
		"\n",
		m.textInput.View(),
		"\n\n",
		prompt,
	)

	// Wrap in a bordered box
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPrimary).
		Padding(1, 3).
		Width(60).
		Align(lipgloss.Center).
		Render(content)

	// Center on screen
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		box,
	)
}

func (m *RenameModal) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *RenameModal) IsConfirmed() bool {
	return m.confirmed
}

func (m *RenameModal) IsCanceled() bool {
	return m.canceled
}

func (m *RenameModal) Value() string {
	return m.textInput.Value()
}
