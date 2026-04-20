package components

import (
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type PasswordModal struct {
	password    string
	copied      bool
	canceled    bool
	width       int
	height      int
	copyTimeout *time.Timer
}

type resetCopiedMsg struct{}

func NewPasswordModal(password string) *PasswordModal {
	return &PasswordModal{
		password: password,
	}
}

func (m *PasswordModal) Init() tea.Cmd {
	return nil
}

func CopyToClipboard(text string) error {
	return clipboard.WriteAll(text)
}

func (m *PasswordModal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
		return m, nil

	case resetCopiedMsg:
		m.copied = false
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "c", "C":
			err := CopyToClipboard(m.password)
			if err == nil {
				m.copied = true
				return m, tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
					return resetCopiedMsg{}
				})
			}
		case "esc", "enter", "q":
			m.canceled = true
			return m, nil
		case "ctrl+c":
			m.canceled = true
			return m, nil
		}
	}

	return m, nil
}

func (m *PasswordModal) View() string {
	if m.canceled {
		return ""
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(colorPrimary).
		Render("🔑 Connection Password")

	passDisplay := lipgloss.NewStyle().
		Foreground(lipgloss.Color("212")). // Pinkish color for password
		Bold(true).
		Render(m.password)
	
	if m.password == "" {
		passDisplay = lipgloss.NewStyle().
			Foreground(colorInactive).
			Italic(true).
			Render("(No password stored)")
	}

	copyStatus := " "
	if m.copied {
		copyStatus = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")). // Green
			Render("✓ Copied to clipboard!")
	}

	prompt := lipgloss.NewStyle().
		Foreground(colorInactive).
		Render("Press C to copy, Esc/Enter to close")

	content := lipgloss.JoinVertical(lipgloss.Center,
		title,
		"\n",
		passDisplay,
		"\n",
		copyStatus,
		"\n",
		prompt,
	)

	// Wrap in a bordered box
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPrimary).
		Padding(1, 3).
		Width(50).
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

func (m *PasswordModal) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *PasswordModal) IsCanceled() bool {
	return m.canceled
}
