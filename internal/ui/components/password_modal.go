package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type PasswordEntry struct {
	Label    string
	Password string
}

type PasswordModal struct {
	entries     []PasswordEntry
	index       int
	copied      bool
	canceled    bool
	width       int
	height      int
	copyTimeout *time.Timer
}

type resetCopiedMsg struct{}

func NewPasswordModal(password string) *PasswordModal {
	return &PasswordModal{
		entries: []PasswordEntry{{Label: "Password", Password: password}},
	}
}

func NewMultiPasswordModal(entries []PasswordEntry) *PasswordModal {
	return &PasswordModal{
		entries: entries,
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
		case "up", "k":
			if m.index > 0 {
				m.index--
				m.copied = false
			}
		case "down", "j":
			if m.index < len(m.entries)-1 {
				m.index++
				m.copied = false
			}
		case "c", "C":
			if len(m.entries) > 0 {
				err := CopyToClipboard(m.entries[m.index].Password)
				if err == nil {
					m.copied = true
					return m, tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
						return resetCopiedMsg{}
					})
				}
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

	titleText := "🔑 Connection Password"
	if len(m.entries) > 1 {
		titleText = "🔑 Select Password to Copy"
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(colorPrimary).
		Render(titleText)

	var content strings.Builder
	content.WriteString(title + "\n\n")

	for i, entry := range m.entries {
		style := lipgloss.NewStyle().Foreground(colorSubText)
		prefix := "  "
		if i == m.index && len(m.entries) > 1 {
			style = lipgloss.NewStyle().Foreground(colorPrimary).Bold(true)
			prefix = "> "
		}

		label := style.Render(prefix + entry.Label + ":")
		pass := entry.Password
		if pass == "" {
			pass = "(empty)"
		}

		passStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
		if i != m.index && len(m.entries) > 1 {
			passStyle = lipgloss.NewStyle().Foreground(colorInactive)
		}

		content.WriteString(fmt.Sprintf("%s %s\n", label, passStyle.Render(pass)))
	}

	copyStatus := " "
	if m.copied {
		copyStatus = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")). // Green
			Render("✓ Copied to clipboard!")
	}
	content.WriteString("\n" + copyStatus + "\n")

	promptText := "Press C to copy, Esc/Enter to close"
	if len(m.entries) > 1 {
		promptText = "↑/↓ to select, C to copy, Esc/Enter to close"
	}

	prompt := lipgloss.NewStyle().
		Foreground(colorInactive).
		Render(promptText)
	content.WriteString(prompt)

	// Wrap in a bordered box
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPrimary).
		Padding(1, 3).
		Width(55).
		Align(lipgloss.Center).
		Render(content.String())

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
