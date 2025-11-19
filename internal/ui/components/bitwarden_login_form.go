package components

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type BitwardenLoginForm struct {
	passwordInput textinput.Model
	otpInput      textinput.Model
	stage         int // 0 = password, 1 = otp, 2 = done
	submitted     bool
	canceled      bool
	errorMsg      string
}

func NewBitwardenLoginForm() *BitwardenLoginForm {
	ti := textinput.New()
	ti.Placeholder = "Password"
	ti.EchoMode = textinput.EchoPassword
	ti.Focus()
	ti.Width = 40
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
				f.stage = 1
				f.otpInput = textinput.New()
				f.otpInput.Placeholder = "2FA Code (if enabled, else leave blank)"
				f.otpInput.CharLimit = 8
				f.otpInput.Focus()
				f.otpInput.Width = 40
				return f, nil
			case 1:
				f.submitted = true
				return f, nil
			}
		}
	}
	switch f.stage {
	case 0:
		var cmd tea.Cmd
		f.passwordInput, cmd = f.passwordInput.Update(msg)
		return f, cmd
	case 1:
		var cmd tea.Cmd
		f.otpInput, cmd = f.otpInput.Update(msg)
		return f, cmd
	}
	return f, nil
}

func (f *BitwardenLoginForm) View() string {
	if f.canceled {
		return StyleContainer.Render("Login canceled.")
	}

	var content string
	switch f.stage {
	case 0:
		content = StyleTitle.Render("Bitwarden Login") + "\n\n"
		content += StyleNormal.Render("Enter your Bitwarden password:") + "\n" + f.passwordInput.View()
		if f.errorMsg != "" {
			content += "\n" + StyleError.Render(f.errorMsg)
		}
	case 1:
		content = StyleTitle.Render("Bitwarden Login - 2FA") + "\n\n"
		content += StyleNormal.Render("Enter 2FA code if required (or leave blank):") + "\n" + f.otpInput.View()
		if f.errorMsg != "" {
			content += "\n" + StyleError.Render(f.errorMsg)
		}
	default:
		content = StyleInfo.Render("Logging in...")
	}

	return StyleContainer.Render(content)
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
