package components

import (
	"fmt"
	"strings"

	"github.com/eugeniofciuvasile/ssh-x-term/internal/config"

	tea "github.com/charmbracelet/bubbletea"
)

type BitwardenOrganizationList struct {
	organizations  []config.Organization
	selectedIndex  int
	selectedOrg    *config.Organization
	highlightedOrg *config.Organization
	width          int
	height         int
	scrollOffset   int
}

func NewBitwardenOrganizationList(organizations []config.Organization) *BitwardenOrganizationList {
	var highlighted *config.Organization
	if len(organizations) > 0 {
		highlighted = &organizations[0]
	}

	return &BitwardenOrganizationList{
		organizations:  organizations,
		selectedIndex:  0,
		highlightedOrg: highlighted,
	}
}

func (cl *BitwardenOrganizationList) Init() tea.Cmd { return nil }

func (cl *BitwardenOrganizationList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		cl.SetSize(msg.Width, msg.Height)
		return cl, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if cl.selectedIndex > 0 {
				cl.selectedIndex--
				cl.updateHighlighted()
			}
		case "down", "j":
			if cl.selectedIndex < len(cl.organizations)-1 {
				cl.selectedIndex++
				cl.updateHighlighted()
			}
		case "enter":
			if len(cl.organizations) > 0 && cl.selectedIndex < len(cl.organizations) {
				cl.selectedOrg = &cl.organizations[cl.selectedIndex]
				return cl, nil
			}
		}
	}

	return cl, nil
}

func (cl *BitwardenOrganizationList) View() string {
	if len(cl.organizations) == 0 {
		return StyleContainer.Render(
			StyleTitle.Render("Organizations") + "\n\n" +
				StyleTextMuted.Render("No organizations found."),
		)
	}

	var b strings.Builder

	b.WriteString(StyleTitle.Render("Select Organization"))
	b.WriteString("\n\n")

	// Calculate visible area
	visibleHeight := cl.height - 8
	if visibleHeight < 5 {
		visibleHeight = 5
	}

	// Adjust scroll offset
	if cl.selectedIndex < cl.scrollOffset {
		cl.scrollOffset = cl.selectedIndex
	}
	if cl.selectedIndex >= cl.scrollOffset+visibleHeight {
		cl.scrollOffset = cl.selectedIndex - visibleHeight + 1
	}

	endIndex := cl.scrollOffset + visibleHeight
	if endIndex > len(cl.organizations) {
		endIndex = len(cl.organizations)
	}

	// Render list items
	for i := cl.scrollOffset; i < endIndex; i++ {
		org := cl.organizations[i]
		prefix := "  "
		style := StyleNormal

		if i == cl.selectedIndex {
			style = StyleFocused
			prefix = StyleFocused.Render("❯ ")
		} else {
			prefix = StyleTextMuted.Render("  ")
		}

		b.WriteString(prefix + style.Render(org.Name) + "\n")
	}

	// Scroll indicator
	if len(cl.organizations) > visibleHeight {
		scrollInfo := fmt.Sprintf("\n%s Showing %d-%d of %d",
			StyleTextMuted.Render("↕"),
			cl.scrollOffset+1,
			endIndex,
			len(cl.organizations))
		b.WriteString(StyleHelp.Render(scrollInfo))
	}

	return StyleContainer.Render(b.String())
}

func (cl *BitwardenOrganizationList) SelectedOrganization() *config.Organization {
	return cl.selectedOrg
}

func (cl *BitwardenOrganizationList) HighlightedOrganization() *config.Organization {
	return cl.highlightedOrg
}

func (cl *BitwardenOrganizationList) SetOrganizations(organizations []config.Organization) {
	cl.organizations = organizations
	if cl.selectedIndex >= len(organizations) {
		cl.selectedIndex = len(organizations) - 1
	}
	if cl.selectedIndex < 0 {
		cl.selectedIndex = 0
	}
	cl.updateHighlighted()
}

func (cl *BitwardenOrganizationList) List() interface{} { return nil }

func (cl *BitwardenOrganizationList) Reset() {
	cl.selectedOrg = nil
	cl.selectedIndex = 0
	cl.scrollOffset = 0
	cl.updateHighlighted()
}

func (cl *BitwardenOrganizationList) SetSize(width, height int) {
	if width <= 0 {
		width = 60
	}
	if height <= 0 {
		height = 20
	}
	cl.width = width
	cl.height = height
}

func (cl *BitwardenOrganizationList) updateHighlighted() {
	if len(cl.organizations) > 0 && cl.selectedIndex < len(cl.organizations) {
		cl.highlightedOrg = &cl.organizations[cl.selectedIndex]
	} else {
		cl.highlightedOrg = nil
	}
}
