package components

import (
	"fmt"
	"ssh-x-term/internal/config"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// organizationItem represents an organization in the list
type organizationItem struct {
	organization config.Organization
}

func (i organizationItem) FilterValue() string {
	return i.organization.Name
}

func (i organizationItem) Title() string {
	return i.organization.Name
}

// BitwardenOrganizationList is a Bubble Tea component for listing Bitwarden organizations
type BitwardenOrganizationList struct {
	list           list.Model
	organizations  []config.Organization
	selectedOrg    *config.Organization
	highlightedOrg *config.Organization
}

// NewBitwardenOrganizationList creates a new organization list component.
// width and height should be set to the current terminal size.
func NewBitwardenOrganizationList(organizations []config.Organization, width, height int) *BitwardenOrganizationList {
	items := make([]list.Item, len(organizations))
	for i, org := range organizations {
		items[i] = organizationItem{organization: org}
	}

	if width <= 0 {
		width = 60
	}
	if height <= 0 {
		height = 20
	}

	l := list.New(items, list.NewDefaultDelegate(), width, height)
	l.Title = "Organizations"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	var highlighted *config.Organization
	if len(organizations) > 0 {
		highlighted = &organizations[0]
	}

	return &BitwardenOrganizationList{
		list:           l,
		organizations:  organizations,
		highlightedOrg: highlighted,
	}
}

func (cl *BitwardenOrganizationList) Init() tea.Cmd {
	return nil
}

func (cl *BitwardenOrganizationList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		cl.list.SetWidth(msg.Width)
		cl.list.SetHeight(msg.Height - 4) // Leave room for help and status
		return cl, nil

	case tea.KeyMsg:
		if cl.list.FilterState() == list.Filtering {
			newList, cmd := cl.list.Update(msg)
			cl.list = newList
			return cl, cmd
		}
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			selectedItem := cl.list.SelectedItem()
			if selectedItem != nil {
				orgItem, ok := selectedItem.(organizationItem)
				if ok {
					cl.selectedOrg = &orgItem.organization
					return cl, nil
				}
			}
		}
	}

	newList, cmd := cl.list.Update(msg)
	cl.list = newList

	if item := cl.list.SelectedItem(); item != nil {
		if orgItem, ok := item.(organizationItem); ok {
			cl.highlightedOrg = &orgItem.organization
		}
	} else {
		cl.highlightedOrg = nil
	}

	return cl, cmd
}

func (cl *BitwardenOrganizationList) View() string {
	if len(cl.organizations) == 0 {
		return fmt.Sprintf("\n%s\n\n  No organizations found.\n\n", titleStyle.Render("Organizations"))
	}
	return fmt.Sprintf("%s", cl.list.View())
}

func (cl *BitwardenOrganizationList) SelectedOrganization() *config.Organization {
	return cl.selectedOrg
}

func (cl *BitwardenOrganizationList) HighlightedOrganization() *config.Organization {
	return cl.highlightedOrg
}

func (cl *BitwardenOrganizationList) SetOrganizations(organizations []config.Organization) {
	cl.organizations = organizations
	items := make([]list.Item, len(organizations))
	for i, org := range organizations {
		items[i] = organizationItem{organization: org}
	}
	cl.list.SetItems(items)
}

func (cl *BitwardenOrganizationList) List() *list.Model {
	return &cl.list
}

func (cl *BitwardenOrganizationList) Reset() {
	cl.selectedOrg = nil
	cl.list.Select(0)
	if len(cl.organizations) > 0 {
		cl.highlightedOrg = &cl.organizations[0]
	} else {
		cl.highlightedOrg = nil
	}
}

func (cl *BitwardenOrganizationList) SetWidth(width int) {
	cl.list.SetWidth(width)
}

func (cl *BitwardenOrganizationList) SetHeight(height int) {
	cl.list.SetHeight(height)
}
