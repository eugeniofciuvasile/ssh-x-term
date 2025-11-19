package components

import (
	"github.com/charmbracelet/lipgloss"
)

// Unified color palette for the entire application
var (
	// Primary colors
	ColorPrimary   = lipgloss.Color("#00D9FF") // Cyan - primary accent
	ColorSecondary = lipgloss.Color("#7C3AED") // Purple - secondary accent
	ColorSuccess   = lipgloss.Color("#10B981") // Green
	ColorWarning   = lipgloss.Color("#F59E0B") // Amber
	ColorError     = lipgloss.Color("#EF4444") // Red
	ColorInfo      = lipgloss.Color("#3B82F6") // Blue

	// Neutral colors
	ColorText         = lipgloss.Color("#E5E7EB") // Light gray text
	ColorTextMuted    = lipgloss.Color("#9CA3AF") // Muted gray
	ColorTextDimmed   = lipgloss.Color("#6B7280") // Dimmed gray
	ColorBackground   = lipgloss.Color("#1F2937") // Dark background
	ColorBackgroundAlt = lipgloss.Color("#111827") // Darker background

	// Border colors
	ColorBorder       = lipgloss.Color("#374151") // Border gray
	ColorBorderFocus  = ColorPrimary              // Focused border
)

// Global styles used across all components
var (
	// Header style - shown at top of screen
	StyleHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			Background(ColorBackgroundAlt).
			Padding(0, 2)

	// Footer style - shown at bottom of screen
	StyleFooter = lipgloss.NewStyle().
			Foreground(ColorTextMuted).
			Background(ColorBackgroundAlt).
			Padding(0, 2)

	// Title style for component headers
	StyleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			MarginBottom(1)

	// Subtitle style
	StyleSubtitle = lipgloss.NewStyle().
			Foreground(ColorTextMuted).
			MarginBottom(1)

	// Focused input/item style
	StyleFocused = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	// Blurred/inactive style
	StyleBlurred = lipgloss.NewStyle().
			Foreground(ColorTextDimmed)

	// Normal text style
	StyleNormal = lipgloss.NewStyle().
			Foreground(ColorText)

	// Error message style
	StyleError = lipgloss.NewStyle().
			Foreground(ColorError).
			Bold(true)

	// Success message style
	StyleSuccess = lipgloss.NewStyle().
			Foreground(ColorSuccess).
			Bold(true)

	// Warning message style
	StyleWarning = lipgloss.NewStyle().
			Foreground(ColorWarning).
			Bold(true)

	// Info message style
	StyleInfo = lipgloss.NewStyle().
			Foreground(ColorInfo)

	// Button styles
	StyleButtonFocused = lipgloss.NewStyle().
				Foreground(ColorBackground).
				Background(ColorPrimary).
				Bold(true).
				Padding(0, 2)

	StyleButtonBlurred = lipgloss.NewStyle().
				Foreground(ColorTextMuted).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorBorder).
				Padding(0, 1)

	// Container styles
	StyleContainer = lipgloss.NewStyle().
			Padding(1, 2)

	// List styles
	StyleListTitle = lipgloss.NewStyle().
			MarginLeft(2).
			Foreground(ColorPrimary).
			Bold(true)

	StyleListItem = lipgloss.NewStyle().
			Foreground(ColorText)

	StyleListItemSelected = lipgloss.NewStyle().
				Foreground(ColorPrimary).
				Bold(true)

	// Loading/spinner style
	StyleSpinner = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	// Help text style
	StyleHelp = lipgloss.NewStyle().
			Foreground(ColorTextMuted)

	// Additional text color styles for convenience
	StyleTextMuted = lipgloss.NewStyle().
			Foreground(ColorTextMuted)
)

// Helper functions for rendering common UI elements

// RenderButton renders a button with the given label and focused state
func RenderButton(label string, focused bool) string {
	if focused {
		return StyleButtonFocused.Render(" " + label + " ")
	}
	return StyleButtonBlurred.Render(" " + label + " ")
}

// RenderPrompt renders an input prompt with focus state
func RenderPrompt(focused bool) string {
	if focused {
		return StyleFocused.Render("> ")
	}
	return StyleBlurred.Render("> ")
}

// CenterText centers text horizontally within the given width
func CenterText(text string, width int) string {
	return lipgloss.Place(width, 1,
		lipgloss.Center, lipgloss.Center,
		text)
}

// CenterVertically centers content vertically within the given height
func CenterVertically(content string, height int) string {
	return lipgloss.Place(0, height,
		lipgloss.Center, lipgloss.Center,
		content)
}

// CenterContent centers content both horizontally and vertically
func CenterContent(content string, width, height int) string {
	return lipgloss.Place(width, height,
		lipgloss.Center, lipgloss.Center,
		content)
}
