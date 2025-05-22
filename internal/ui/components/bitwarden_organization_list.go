package components

import (
	"fmt"
	"github.com/eugeniofciuvasile/ssh-x-term/internal/config"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type organizationItem struct {
	organization config.Organization
}

func (i organizationItem) FilterValue() string { return i.organization.Name }
func (i organizationItem) Title() string       { return i.organization.Name }
func (i organizationItem) Description() string { return "" }

type BitwardenOrganizationList struct {
	list           list.Model
	organizations  []config.Organization
	selectedOrg    *config.Organization
	highlightedOrg *config.Organization
}

func NewBitwardenOrganizationList(organizations []config.Organization) *BitwardenOrganizationList {
	items := make([]list.Item, len(organizations))
	for i, org := range organizations {
		items[i] = organizationItem{organization: org}
	}
	l := list.New(items, list.NewDefaultDelegate(), 60, 20)
	l.Title = "Organizations"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	// Use the same styles as connection list
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle
	l.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "open personal vault")),
		}
	}

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

func (cl *BitwardenOrganizationList) Init() tea.Cmd { return nil }

func (cl *BitwardenOrganizationList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		cl.SetSize(msg.Width, msg.Height)
		return cl, nil
	case tea.KeyMsg:
		if cl.list.FilterState() == list.Filtering {
			newList, cmd := cl.list.Update(msg)
			cl.list = newList
			return cl, cmd
		}
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			if selectedItem := cl.list.SelectedItem(); selectedItem != nil {
				if orgItem, ok := selectedItem.(organizationItem); ok {
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
	return cl.list.View()
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

func (cl *BitwardenOrganizationList) List() *list.Model { return &cl.list }

func (cl *BitwardenOrganizationList) Reset() {
	cl.selectedOrg = nil
	cl.list.Select(0)
	if len(cl.organizations) > 0 {
		cl.highlightedOrg = &cl.organizations[0]
	} else {
		cl.highlightedOrg = nil
	}
}

func (cl *BitwardenOrganizationList) SetSize(width, height int) {
	if width <= 0 {
		width = 60
	}
	if height <= 0 {
		height = 20
	}
	cl.list.SetWidth(width)
	cl.list.SetHeight(height - 4)
}
