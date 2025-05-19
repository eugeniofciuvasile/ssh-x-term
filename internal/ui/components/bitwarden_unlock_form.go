package components

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type BitwardenUnlockForm struct {
	passwordInput textinput.Model
	submitted     bool
	canceled      bool
	errorMsg      string
}

func NewBitwardenUnlockForm() *BitwardenUnlockForm {
	ti := textinput.New()
	ti.Placeholder = "Vault Password"
	ti.EchoMode = textinput.EchoPassword
	ti.Focus()
	ti.Width = 40
	return &BitwardenUnlockForm{
		passwordInput: ti,
	}
}

func (f *BitwardenUnlockForm) Init() tea.Cmd {
	return textinput.Blink
}

func (f *BitwardenUnlockForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		return "Unlock canceled."
	}
	return "Enter your Bitwarden vault password to unlock:\n" + f.passwordInput.View() + "\n" + f.errorMsg
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
