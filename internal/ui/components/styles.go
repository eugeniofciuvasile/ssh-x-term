package components

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

// --- Palette based on the Gopher Bubble Tea Image ---
var (
	// The vibrant purple from the window header and straw
	colorPrimary = lipgloss.Color("#974FD7")
	// The cyan/blue from the Gopher's skin and "ssh" text
	colorSecondary = lipgloss.Color("#00ADD8")
	// The cream/beige from the tea drink (used for headers and highlights)
	colorAccent = lipgloss.Color("#F0D8B2")
	// Standard text colors
	colorText     = lipgloss.Color("#FAFAFA")
	colorSubText  = lipgloss.Color("#7D7D7D")
	colorError    = lipgloss.Color("#FF5555")
	colorInactive = lipgloss.Color("#4D4D4D")
)

var (
	// --- General Layout Styles ---

	// Main View Titles
	titleStyle = lipgloss.NewStyle().
			MarginLeft(2).
			MarginTop(1).
			Foreground(colorPrimary). // Purple
			Bold(true)

	// Standard bold header for sections
	sectionTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorPrimary).
				MarginBottom(1)

	// Generic container padding
	containerStyle = lipgloss.NewStyle().
			Padding(1, 2)

	// --- List & Table Styles ---

	paginationStyle = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle       = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)

	// Table column headers - Now uses the Cream Accent color
	headerStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	// Standard row item
	itemStyle = lipgloss.NewStyle().PaddingLeft(2)

	// Selected row item
	selectedItemStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), false, false, false, true).
				BorderForeground(colorPrimary). // Purple border
				Foreground(colorPrimary).       // Purple text
				PaddingLeft(1)

	// --- Form & Input Styles (Consolidated) ---

	// Focused input fields
	focusedStyle = lipgloss.NewStyle().Foreground(colorPrimary)
	// Blurred/Inactive input fields
	blurredStyle = lipgloss.NewStyle().Foreground(colorInactive)
	noStyle      = lipgloss.NewStyle()

	// Buttons
	focusedButton = focusedStyle.Render("[ Submit ]")
	blurredButton = fmt.Sprintf("[ %s ]", blurredStyle.Render("Submit"))

	// Error messages
	errorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorError).
			Padding(0, 2)

	// --- SCP / File Manager Styles ---

	// Header for file panels
	scpHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Background(colorPrimary). // Purple background
			Foreground(colorText).
			Align(lipgloss.Center).
			Padding(0, 1)

	// Inactive file panel
	scpPanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorInactive).
			Padding(1, 2)

	// Active file panel
	scpActivePanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorSecondary). // Blue border for active focus
				Padding(1, 2)

	// Directory names
	scpDirStyle = lipgloss.NewStyle().
			Foreground(colorSecondary). // Blue text
			Bold(true)

	// Selected file in list - Uses Cream Accent text on dark background
	scpSelectedStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("237")).
				Foreground(colorAccent).
				Bold(true)

	scpStatusStyle = lipgloss.NewStyle().
			Foreground(colorSubText).
			Background(lipgloss.Color("235")).
			Padding(0, 2)

	// --- Terminal Styles ---

	terminalHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Background(colorSecondary). // Blue background
				Foreground(colorText).
				Align(lipgloss.Center).
				Padding(0, 1)

	terminalErrorStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorError).
				Align(lipgloss.Center).
				Padding(1, 0)
)

// --- Re-export aliases for backward compatibility ---

var (
	// Bitwarden specific aliases mapped to generic styles
	bwTitleStyle       = sectionTitleStyle
	bwConfigTitleStyle = sectionTitleStyle
	bwUnlockTitleStyle = sectionTitleStyle

	bwFormStyle       = containerStyle
	bwConfigFormStyle = containerStyle
	bwUnlockFormStyle = containerStyle

	bwFocusedStyle  = focusedStyle
	bwBlurredStyle  = blurredStyle
	bwFocusedButton = focusedButton
	bwBlurredButton = blurredButton

	// Storage selection aliases
	storageSelectStyle      = containerStyle
	storageSelectTitleStyle = sectionTitleStyle

	// Generic Form aliases
	formStyle      = containerStyle
	formTitleStyle = sectionTitleStyle
	scpErrorStyle  = errorStyle
)
