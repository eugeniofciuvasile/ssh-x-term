package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	bwFocusedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	bwBlurredStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	bwFocusedButton = bwFocusedStyle.Render("[ Submit ]")
	bwBlurredButton = fmt.Sprintf("[ %s ]", bwBlurredStyle.Render("Submit"))

	bwConfigFormStyle = lipgloss.NewStyle().
				Padding(1, 2)

	bwConfigTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("205")).
				MarginBottom(1)
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
	inputs[0].Prompt = "> "
	inputs[0].PromptStyle = bwFocusedStyle
	inputs[0].TextStyle = bwFocusedStyle

	// Email
	inputs[1] = textinput.New()
	inputs[1].Placeholder = "Email"
	inputs[1].Width = 32
	inputs[1].Prompt = "> "
	inputs[1].PromptStyle = bwBlurredStyle
	inputs[1].TextStyle = bwBlurredStyle

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
			f.inputs[i].PromptStyle = bwFocusedStyle
			f.inputs[i].TextStyle = bwFocusedStyle
		} else {
			f.inputs[i].Blur()
			f.inputs[i].PromptStyle = bwBlurredStyle
			f.inputs[i].TextStyle = bwBlurredStyle
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

	b.WriteString(bwConfigTitleStyle.Render("Bitwarden Storage Setup"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("%s\n%s\n\n", "Server URL:", f.inputs[0].View()))
	b.WriteString(fmt.Sprintf("%s\n%s\n\n", "Email:", f.inputs[1].View()))

	// Render submit button
	button := bwBlurredButton
	if f.focusIndex == len(f.inputs) {
		button = bwFocusedButton
	}
	fmt.Fprintf(&b, "\n%s\n", button)

	// Show error message if any
	if f.ErrorMsg != "" {
		fmt.Fprintf(&b, "\n%s\n", lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(f.ErrorMsg))
	}

	return bwConfigFormStyle.Render(b.String())
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
