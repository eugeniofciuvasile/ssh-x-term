package components

import (
	"fmt"
	"strings"

	"github.com/eugeniofciuvasile/ssh-x-term/internal/config"

	tea "github.com/charmbracelet/bubbletea"
)

type BitwardenCollectionList struct {
	collections           []config.Collection
	selectedIndex         int
	selectedCollection    *config.Collection
	highlightedCollection *config.Collection
	width                 int
	height                int
	scrollOffset          int
}

func NewBitwardenCollectionList(collections []config.Collection) *BitwardenCollectionList {
	var highlighted *config.Collection
	if len(collections) > 0 {
		highlighted = &collections[0]
	}

	return &BitwardenCollectionList{
		collections:           collections,
		selectedIndex:         0,
		highlightedCollection: highlighted,
	}
}

func (cl *BitwardenCollectionList) Init() tea.Cmd { return nil }

func (cl *BitwardenCollectionList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			if cl.selectedIndex < len(cl.collections)-1 {
				cl.selectedIndex++
				cl.updateHighlighted()
			}
		case "enter":
			if len(cl.collections) > 0 && cl.selectedIndex < len(cl.collections) {
				cl.selectedCollection = &cl.collections[cl.selectedIndex]
				return cl, nil
			}
		}
	}

	return cl, nil
}

func (cl *BitwardenCollectionList) View() string {
	if len(cl.collections) == 0 {
		return StyleContainer.Render(
			StyleTitle.Render("Collections") + "\n\n" +
				StyleTextMuted.Render("No collections found."),
		)
	}

	var b strings.Builder

	b.WriteString(StyleTitle.Render("Select Collection"))
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
	if endIndex > len(cl.collections) {
		endIndex = len(cl.collections)
	}

	// Render list items
	for i := cl.scrollOffset; i < endIndex; i++ {
		col := cl.collections[i]
		prefix := "  "
		style := StyleNormal

		if i == cl.selectedIndex {
			style = StyleFocused
			prefix = StyleFocused.Render("❯ ")
		} else {
			prefix = StyleTextMuted.Render("  ")
		}

		b.WriteString(prefix + style.Render(col.Name) + "\n")
	}

	// Scroll indicator
	if len(cl.collections) > visibleHeight {
		scrollInfo := fmt.Sprintf("\n%s Showing %d-%d of %d",
			StyleTextMuted.Render("↕"),
			cl.scrollOffset+1,
			endIndex,
			len(cl.collections))
		b.WriteString(StyleHelp.Render(scrollInfo))
	}

	return StyleContainer.Render(b.String())
}

func (cl *BitwardenCollectionList) SelectedCollection() *config.Collection {
	return cl.selectedCollection
}

func (cl *BitwardenCollectionList) HighlightedCollection() *config.Collection {
	return cl.highlightedCollection
}

func (cl *BitwardenCollectionList) SetCollections(collections []config.Collection) {
	cl.collections = collections
	if cl.selectedIndex >= len(collections) {
		cl.selectedIndex = len(collections) - 1
	}
	if cl.selectedIndex < 0 {
		cl.selectedIndex = 0
	}
	cl.updateHighlighted()
}

func (cl *BitwardenCollectionList) List() interface{} { return nil }

func (cl *BitwardenCollectionList) Reset() {
	cl.selectedCollection = nil
	cl.selectedIndex = 0
	cl.scrollOffset = 0
	cl.updateHighlighted()
}

func (cl *BitwardenCollectionList) SetSize(width, height int) {
	if width <= 0 {
		width = 60
	}
	if height <= 0 {
		height = 20
	}
	cl.width = width
	cl.height = height
}

func (cl *BitwardenCollectionList) updateHighlighted() {
	if len(cl.collections) > 0 && cl.selectedIndex < len(cl.collections) {
		cl.highlightedCollection = &cl.collections[cl.selectedIndex]
	} else {
		cl.highlightedCollection = nil
	}
}
