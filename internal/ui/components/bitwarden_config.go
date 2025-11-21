package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type BitwardenConfigForm struct {
	inputs     []textinput.Model
	focusIndex int
	submitted  bool
	canceled   bool
	ErrorMsg   string
	width      int
	height     int
}

// NewBitwardenConfigForm creates a new Bitwarden config form
func NewBitwardenConfigForm() *BitwardenConfigForm {
	inputs := make([]textinput.Model, 2)

	// Server URL
	inputs[0] = textinput.New()
	inputs[0].Placeholder = "https://bitwarden.com"
	inputs[0].Focus()
	inputs[0].Width = 50
	inputs[0].Prompt = "" // Clean look
	inputs[0].PromptStyle = focusedStyle
	inputs[0].TextStyle = focusedStyle

	// Email
	inputs[1] = textinput.New()
	inputs[1].Placeholder = "user@example.com"
	inputs[1].Width = 50
	inputs[1].Prompt = ""
	inputs[1].PromptStyle = blurredStyle
	inputs[1].TextStyle = blurredStyle

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
	case tea.WindowSizeMsg:
		f.SetSize(msg.Width, msg.Height)
		return f, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			f.canceled = true
			return f, nil
		case "tab", "down", "enter":
			// Special handling for Enter on the Submit button
			if msg.String() == "enter" && f.focusIndex == len(f.inputs) {
				if valid, err := f.validateForm(); valid {
					f.submitted = true
					return f, nil
				} else {
					f.ErrorMsg = err
					return f, nil
				}
			}

			// Cycle focus
			f.focusIndex++
			if f.focusIndex > len(f.inputs) {
				f.focusIndex = 0
			}
		case "shift+tab", "up":
			f.focusIndex--
			if f.focusIndex < 0 {
				f.focusIndex = len(f.inputs)
			}
		}
	}

	// Set focus/blurring on inputs
	for i := 0; i < len(f.inputs); i++ {
		if i == f.focusIndex {
			f.inputs[i].Focus()
			f.inputs[i].PromptStyle = focusedStyle
			f.inputs[i].TextStyle = focusedStyle
		} else {
			f.inputs[i].Blur()
			f.inputs[i].PromptStyle = blurredStyle
			f.inputs[i].TextStyle = blurredStyle
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
	// 1. Build the form content with Left Alignment
	var b strings.Builder

	// Labels
	labelStyle := lipgloss.NewStyle().Foreground(colorSubText).MarginBottom(0)

	b.WriteString(sectionTitleStyle.Render("Bitwarden Setup"))
	b.WriteString("\n\n")

	b.WriteString(labelStyle.Render("Server URL"))
	b.WriteString("\n")
	b.WriteString(f.inputs[0].View())
	b.WriteString("\n\n")

	b.WriteString(labelStyle.Render("Email"))
	b.WriteString("\n")
	b.WriteString(f.inputs[1].View())
	b.WriteString("\n\n")

	// Render submit button
	button := blurredButton
	if f.focusIndex == len(f.inputs) {
		button = focusedButton
	}
	b.WriteString(button)

	if f.ErrorMsg != "" {
		b.WriteString("\n\n")
		b.WriteString(errorStyle.Render(f.ErrorMsg))
	}

	// 2. Wrap content in a Border Box
	formBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPrimary).
		Padding(1, 3).
		Width(60).            // Fixed width for the box
		Align(lipgloss.Left). // Align text inside the box to the left
		Render(b.String())

	// 3. Center the Box on the Screen
	availableHeight := max(f.height-3, 0)

	return lipgloss.Place(
		f.width,
		availableHeight,
		lipgloss.Center, // Horizontal Center
		lipgloss.Center, // Vertical Center
		formBox,
	)
}

func (f *BitwardenConfigForm) SetSize(width, height int) {
	f.width = width
	f.height = height
}

func (f *BitwardenConfigForm) IsSubmitted() bool {
	return f.submitted
}

func (f *BitwardenConfigForm) IsCanceled() bool {
	return f.canceled
}

func (f *BitwardenConfigForm) Config() (serverURL, email string) {
	url := f.inputs[0].Value()
	if url == "" {
		url = "https://bitwarden.com"
	}
	return url, f.inputs[1].Value()
}

func (f *BitwardenConfigForm) validateForm() (bool, string) {
	if strings.TrimSpace(f.inputs[1].Value()) == "" {
		return false, "Email is required."
	}
	return true, ""
}
