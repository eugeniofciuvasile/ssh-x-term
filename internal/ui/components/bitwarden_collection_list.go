package components

import (
	"fmt"
	"ssh-x-term/internal/config"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// collectionItem represents a collection in the list
type collectionItem struct {
	collection config.Collection
}

func (i collectionItem) FilterValue() string {
	return i.collection.Name
}

func (i collectionItem) Title() string {
	return i.collection.Name
}

// BitwardenCollectionList is a Bubble Tea component for listing Bitwarden collections
type BitwardenCollectionList struct {
	list                  list.Model
	collections           []config.Collection
	selectedCollection    *config.Collection
	highlightedCollection *config.Collection
}

// NewBitwardenCollectionList creates a new collection list component.
// width and height should be set to the current terminal size.
func NewBitwardenCollectionList(collections []config.Collection, width, height int) *BitwardenCollectionList {
	items := make([]list.Item, len(collections))
	for i, col := range collections {
		items[i] = collectionItem{collection: col}
	}

	if width <= 0 {
		width = 60
	}
	if height <= 0 {
		height = 20
	}

	l := list.New(items, list.NewDefaultDelegate(), width, height)
	l.Title = "Collections"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	var highlighted *config.Collection
	if len(collections) > 0 {
		highlighted = &collections[0]
	}

	return &BitwardenCollectionList{
		list:                  l,
		collections:           collections,
		highlightedCollection: highlighted,
	}
}

func (cl *BitwardenCollectionList) Init() tea.Cmd {
	return nil
}

func (cl *BitwardenCollectionList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
				colItem, ok := selectedItem.(collectionItem)
				if ok {
					cl.selectedCollection = &colItem.collection
					return cl, nil
				}
			}
		}
	}

	newList, cmd := cl.list.Update(msg)
	cl.list = newList

	if item := cl.list.SelectedItem(); item != nil {
		if colItem, ok := item.(collectionItem); ok {
			cl.highlightedCollection = &colItem.collection
		}
	} else {
		cl.highlightedCollection = nil
	}

	return cl, cmd
}

func (cl *BitwardenCollectionList) View() string {
	if len(cl.collections) == 0 {
		return fmt.Sprintf("\n%s\n\n  No collections found.\n\n", titleStyle.Render("Collections"))
	}
	return fmt.Sprintf("%s", cl.list.View())
}

func (cl *BitwardenCollectionList) SelectedOrganization() *config.Collection {
	return cl.selectedCollection
}

func (cl *BitwardenCollectionList) HighlightedOrganization() *config.Collection {
	return cl.highlightedCollection
}

func (cl *BitwardenCollectionList) SetOrganizations(collections []config.Collection) {
	cl.collections = collections
	items := make([]list.Item, len(collections))
	for i, collection := range collections {
		items[i] = collectionItem{collection: collection}
	}
	cl.list.SetItems(items)
}

func (cl *BitwardenCollectionList) List() *list.Model {
	return &cl.list
}

func (cl *BitwardenCollectionList) Reset() {
	cl.selectedCollection = nil
	cl.list.Select(0)
	if len(cl.collections) > 0 {
		cl.highlightedCollection = &cl.collections[0]
	} else {
		cl.highlightedCollection = nil
	}
}

func (cl *BitwardenCollectionList) SetWidth(width int) {
	cl.list.SetWidth(width)
}

func (cl *BitwardenCollectionList) SetHeight(height int) {
	cl.list.SetHeight(height)
}
