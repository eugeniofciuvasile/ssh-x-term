package components

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type BitwardenUnlockForm struct {
	passwordInput textinput.Model
	submitted     bool
	canceled      bool
	errorMsg      string
	width         int
	height        int
}

func NewBitwardenUnlockForm() *BitwardenUnlockForm {
	ti := textinput.New()
	ti.Placeholder = "Vault Password"
	ti.EchoMode = textinput.EchoPassword
	ti.Focus()
	ti.Width = 50
	// Apply our generic input styles
	ti.PromptStyle = focusedStyle
	ti.TextStyle = focusedStyle

	return &BitwardenUnlockForm{
		passwordInput: ti,
	}
}

func (f *BitwardenUnlockForm) Init() tea.Cmd {
	return textinput.Blink
}

// Key Point 2: Handle Resize Events
func (f *BitwardenUnlockForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if f.submitted || f.canceled {
		return f, nil
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		f.SetSize(msg.Width, msg.Height)
		return f, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			f.canceled = true
			return f, nil
		case "enter":
			if f.passwordInput.Value() == "" {
				f.errorMsg = "Password required"
				return f, nil
			}
			f.submitted = true
			return f, nil
		}
	}
	var cmd tea.Cmd
	f.passwordInput, cmd = f.passwordInput.Update(msg)
	return f, cmd
}

func (f *BitwardenUnlockForm) View() string {
	if f.canceled {
		return ""
	}

	// Build the core content box
	content := lipgloss.JoinVertical(lipgloss.Center,
		"Enter your Bitwarden vault password:",
		"\n",
		f.passwordInput.View(),
	)

	if f.errorMsg != "" {
		content = lipgloss.JoinVertical(lipgloss.Center,
			content,
			"\n",
			errorStyle.Render(f.errorMsg),
		)
	}

	// Key Point 3: Calculate Layout using Place()
	// We subtract 2 from height to account for the global header/footer
	availableHeight := max(f.height-3, 0)

	// Wrap in a nice border or container style if desired
	formBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPrimary).
		Padding(1, 3).
		Width(60).            // Fixed width for the box
		Align(lipgloss.Left). // Align text inside the box to the left
		Render(content)

	return lipgloss.Place(
		f.width,
		availableHeight,
		lipgloss.Center,
		lipgloss.Center,
		formBox,
	)
}

// Key Point 4: Public SetSize method
func (f *BitwardenUnlockForm) SetSize(width, height int) {
	f.width = width
	f.height = height
}

func (f *BitwardenUnlockForm) IsSubmitted() bool {
	return f.submitted
}

func (f *BitwardenUnlockForm) IsCanceled() bool {
	return f.canceled
}

func (f *BitwardenUnlockForm) Password() string {
	return f.passwordInput.Value()
}

func (f *BitwardenUnlockForm) SetError(msg string) {
	f.errorMsg = msg
}

func (f *BitwardenUnlockForm) ResetSubmitted() {
	f.submitted = false
}
