package components

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type BitwardenLoginForm struct {
	passwordInput textinput.Model
	otpInput      textinput.Model
	stage         int // 0 = password, 1 = otp
	submitted     bool
	canceled      bool
	errorMsg      string
	width         int
	height        int
}

func NewBitwardenLoginForm() *BitwardenLoginForm {
	ti := textinput.New()
	ti.Placeholder = "Master Password"
	ti.EchoMode = textinput.EchoPassword
	ti.Focus()
	ti.Width = 50

	// Style styling
	ti.PromptStyle = focusedStyle
	ti.TextStyle = focusedStyle

	return &BitwardenLoginForm{
		passwordInput: ti,
		stage:         0,
	}
}

func (f *BitwardenLoginForm) Init() tea.Cmd {
	return textinput.Blink
}

func (f *BitwardenLoginForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			switch f.stage {
			case 0:
				if f.passwordInput.Value() == "" {
					f.errorMsg = "Password required"
					return f, nil
				}
				// Move to Stage 2: OTP
				f.stage = 1
				f.errorMsg = "" // Clear errors

				f.otpInput = textinput.New()
				f.otpInput.Placeholder = "Leave blank if disabled"
				f.otpInput.CharLimit = 6 // Standard OTP length usually
				f.otpInput.Focus()
				f.otpInput.Width = 40
				f.otpInput.PromptStyle = focusedStyle
				f.otpInput.TextStyle = focusedStyle

				return f, nil
			case 1:
				f.submitted = true
				return f, nil
			}
		}
	}

	var cmd tea.Cmd
	switch f.stage {
	case 0:
		f.passwordInput, cmd = f.passwordInput.Update(msg)
	case 1:
		f.otpInput, cmd = f.otpInput.Update(msg)
	}
	return f, cmd
}

func (f *BitwardenLoginForm) View() string {
	if f.canceled {
		return ""
	}

	var content string
	var title string
	var promptLabel string
	var activeInput string

	switch f.stage {
	case 0:
		title = "Bitwarden Login"
		promptLabel = "Enter your master password:"
		activeInput = f.passwordInput.View()
	case 1:
		title = "Two-Factor Authentication"
		promptLabel = "Enter 2FA code (or press Enter to skip):"
		activeInput = f.otpInput.View()
	default:
		content = "Logging in..."
	}

	// Build layout if not in default state
	if content == "" {
		content = lipgloss.JoinVertical(lipgloss.Center,
			sectionTitleStyle.Render(title),
			promptLabel,
			"\n",
			activeInput,
		)

		if f.errorMsg != "" {
			content = lipgloss.JoinVertical(lipgloss.Center,
				content,
				"\n",
				errorStyle.Render(f.errorMsg),
			)
		}
	}

	// Create the Bordered Box
	formBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPrimary).
		Padding(1, 3).
		Width(60).
		Align(lipgloss.Left). // Center text inside box
		Render(content)

	// Center Box on Screen
	availableHeight := max(f.height-3, 0)
	return lipgloss.Place(
		f.width,
		availableHeight,
		lipgloss.Center,
		lipgloss.Center,
		formBox,
	)
}

func (f *BitwardenLoginForm) SetSize(width, height int) {
	f.width = width
	f.height = height
}

func (f *BitwardenLoginForm) IsSubmitted() bool {
	return f.submitted
}

func (f *BitwardenLoginForm) IsCanceled() bool {
	return f.canceled
}

func (f *BitwardenLoginForm) Password() string {
	return f.passwordInput.Value()
}

func (f *BitwardenLoginForm) OTP() string {
	return f.otpInput.Value()
}

func (f *BitwardenLoginForm) SetError(msg string) {
	f.errorMsg = msg
}

func (f *BitwardenLoginForm) ResetSubmitted() {
	f.submitted = false
}
