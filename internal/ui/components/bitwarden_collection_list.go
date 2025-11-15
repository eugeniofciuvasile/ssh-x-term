package components

import (
	"fmt"
	"github.com/eugeniofciuvasile/ssh-x-term/internal/config"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type collectionItem struct {
	collection config.Collection
}

func (i collectionItem) FilterValue() string { return i.collection.Name }
func (i collectionItem) Title() string       { return i.collection.Name }
func (i collectionItem) Description() string { return "" }

type BitwardenCollectionList struct {
	list                  list.Model
	collections           []config.Collection
	selectedCollection    *config.Collection
	highlightedCollection *config.Collection
}

func NewBitwardenCollectionList(collections []config.Collection) *BitwardenCollectionList {
	items := make([]list.Item, len(collections))
	for i, col := range collections {
		items[i] = collectionItem{collection: col}
	}
	l := list.New(items, list.NewDefaultDelegate(), 60, 20)
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

func (cl *BitwardenCollectionList) Init() tea.Cmd { return nil }

func (cl *BitwardenCollectionList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
				if colItem, ok := selectedItem.(collectionItem); ok {
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
	return cl.list.View()
}

func (cl *BitwardenCollectionList) SelectedCollection() *config.Collection {
	return cl.selectedCollection
}

func (cl *BitwardenCollectionList) HighlightedCollection() *config.Collection {
	return cl.highlightedCollection
}

func (cl *BitwardenCollectionList) SetCollections(collections []config.Collection) {
	cl.collections = collections
	items := make([]list.Item, len(collections))
	for i, collection := range collections {
		items[i] = collectionItem{collection: collection}
	}
	cl.list.SetItems(items)
}

func (cl *BitwardenCollectionList) List() *list.Model { return &cl.list }

func (cl *BitwardenCollectionList) Reset() {
	cl.selectedCollection = nil
	cl.list.Select(0)
	if len(cl.collections) > 0 {
		cl.highlightedCollection = &cl.collections[0]
	} else {
		cl.highlightedCollection = nil
	}
}

func (cl *BitwardenCollectionList) SetSize(width, height int) {
	if width <= 0 {
		width = 60
	}
	if height <= 0 {
		height = 20
	}
	cl.list.SetWidth(width)
	// Use full available height for the list
	cl.list.SetHeight(height)
}
