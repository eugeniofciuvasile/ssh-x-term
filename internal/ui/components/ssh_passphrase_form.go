package components

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/eugeniofciuvasile/ssh-x-term/internal/config"
)

type SSHPassphraseForm struct {
	textInput  textinput.Model
	Connection config.SSHConnection
	submitted  bool
	canceled   bool
	width      int
	height     int
}

func NewSSHPassphraseForm(conn config.SSHConnection) *SSHPassphraseForm {
	ti := textinput.New()
	ti.Placeholder = "Key Passphrase or Password"
	ti.EchoMode = textinput.EchoPassword
	ti.Focus()
	ti.Width = 50
	ti.PromptStyle = focusedStyle
	ti.TextStyle = focusedStyle

	return &SSHPassphraseForm{
		textInput:  ti,
		Connection: conn,
	}
}

func (f *SSHPassphraseForm) Init() tea.Cmd {
	return textinput.Blink
}

func (f *SSHPassphraseForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		f.SetSize(msg.Width, msg.Height)
		return f, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+c":
			f.canceled = true
			return f, nil
		case "enter":
			f.submitted = true
			return f, nil
		}
	}

	var cmd tea.Cmd
	f.textInput, cmd = f.textInput.Update(msg)
	return f, cmd
}

func (f *SSHPassphraseForm) View() string {
	var content string

	// Create a centered box
	title := fmt.Sprintf("Authentication Required for '%s'", f.Connection.Name)
	desc := fmt.Sprintf("Enter passphrase for key:\n%s", f.Connection.KeyFile)
	if f.Connection.KeyFile == "" {
		desc = fmt.Sprintf("Enter password for user '%s@%s':", f.Connection.Username, f.Connection.Host)
	}

	content = lipgloss.JoinVertical(lipgloss.Center,
		sectionTitleStyle.Render(title),
		"\n",
		lipgloss.NewStyle().Foreground(colorSubText).Render(desc),
		"\n",
		f.textInput.View(),
	)

	// Border box
	formBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPrimary).
		Padding(1, 3).
		Width(60).
		Align(lipgloss.Center).
		Render(content)

	// Center on screen
	return lipgloss.Place(
		f.width,
		max(f.height-3, 0),
		lipgloss.Center,
		lipgloss.Center,
		formBox,
	)
}

func (f *SSHPassphraseForm) SetSize(width, height int) {
	f.width = width
	f.height = height
}

func (f *SSHPassphraseForm) IsSubmitted() bool {
	return f.submitted
}

func (f *SSHPassphraseForm) IsCanceled() bool {
	return f.canceled
}

func (f *SSHPassphraseForm) Value() string {
	return f.textInput.Value()
}
