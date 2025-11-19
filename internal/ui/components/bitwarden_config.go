package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type BitwardenConfigForm struct {
	inputs     []textinput.Model
	focusIndex int
	submitted  bool
	canceled   bool
	ErrorMsg   string
}

// NewBitwardenConfigForm creates a new Bitwarden config form
func NewBitwardenConfigForm() *BitwardenConfigForm {
	inputs := make([]textinput.Model, 2)

	// Server URL
	inputs[0] = textinput.New()
	inputs[0].Placeholder = "Bitwarden Server URL"
	inputs[0].Focus()
	inputs[0].Width = 32
	inputs[0].Prompt = RenderPrompt(true)
	inputs[0].PromptStyle = StyleFocused
	inputs[0].TextStyle = StyleFocused

	// Email
	inputs[1] = textinput.New()
	inputs[1].Placeholder = "Email"
	inputs[1].Width = 32
	inputs[1].Prompt = RenderPrompt(false)
	inputs[1].PromptStyle = StyleBlurred
	inputs[1].TextStyle = StyleBlurred

	return &BitwardenConfigForm{
		inputs:     inputs,
		focusIndex: 0,
	}
}

func (f *BitwardenConfigForm) Init() tea.Cmd {
	return textinput.Blink
}

func (f *BitwardenConfigForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			f.canceled = true
			return f, nil
		case "tab", "down":
			f.focusIndex++
			if f.focusIndex > len(f.inputs) {
				f.focusIndex = 0
			}
		case "shift+tab", "up":
			f.focusIndex--
			if f.focusIndex < 0 {
				f.focusIndex = len(f.inputs)
			}
		case "enter":
			if f.focusIndex == len(f.inputs) {
				if valid, err := f.validateForm(); valid {
					f.submitted = true
					return f, nil
				} else {
					f.ErrorMsg = err
				}
			}
		}
	}

	// Set focus/blurring on inputs
	for i := 0; i < len(f.inputs); i++ {
		if i == f.focusIndex {
			f.inputs[i].Focus()
			f.inputs[i].PromptStyle = StyleFocused
			f.inputs[i].TextStyle = StyleFocused
			f.inputs[i].Prompt = RenderPrompt(true)
		} else {
			f.inputs[i].Blur()
			f.inputs[i].PromptStyle = StyleBlurred
			f.inputs[i].TextStyle = StyleBlurred
			f.inputs[i].Prompt = RenderPrompt(false)
		}
	}

	if f.focusIndex < len(f.inputs) {
		newInput, cmd := f.inputs[f.focusIndex].Update(msg)
		f.inputs[f.focusIndex] = newInput
		cmds = append(cmds, cmd)
	}

	return f, tea.Batch(cmds...)
}

func (f *BitwardenConfigForm) View() string {
	var b strings.Builder

	b.WriteString(StyleTitle.Render("Bitwarden Storage Setup"))
	b.WriteString("\n\n")
	b.WriteString(StyleNormal.Render("Server URL:"))
	b.WriteString("\n")
	b.WriteString(f.inputs[0].View())
	b.WriteString("\n\n")
	b.WriteString(StyleNormal.Render("Email:"))
	b.WriteString("\n")
	b.WriteString(f.inputs[1].View())
	b.WriteString("\n\n")

	// Render submit button
	button := RenderButton("Submit", f.focusIndex == len(f.inputs))
	b.WriteString("\n")
	b.WriteString(button)
	b.WriteString("\n")

	// Show error message if any
	if f.ErrorMsg != "" {
		fmt.Fprintf(&b, "\n%s\n", StyleError.Render(f.ErrorMsg))
	}

	return StyleContainer.Render(b.String())
}

func (f *BitwardenConfigForm) IsSubmitted() bool {
	return f.submitted
}

func (f *BitwardenConfigForm) IsCanceled() bool {
	return f.canceled
}

func (f *BitwardenConfigForm) Config() (serverURL, email string) {
	return f.inputs[0].Value(), f.inputs[1].Value()
}

func (f *BitwardenConfigForm) validateForm() (bool, string) {
	if strings.TrimSpace(f.inputs[1].Value()) == "" {
		return false, "Email is required."
	}
	return true, ""
}
