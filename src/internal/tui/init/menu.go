package tui

import (
	"fmt"

	bubbleTea "charm.land/bubbletea/v2"
)

type MenuModel struct {
	Title   string
	Choices []string
	Cursor  int
}

func (m *MenuModel) MoveUp() {
	if m.Cursor > 0 {
		m.Cursor--
	}
}

func (m *MenuModel) MoveDown() {
	if m.Cursor < len(m.Choices)-1 {
		m.Cursor++
	}
}

func (m MenuModel) View() bubbleTea.View {
	str := m.Title + "\n\n"

	for i, choice := range m.Choices {
		cursor := " "
		if m.Cursor == i {
			cursor = ">"
		}

		str += fmt.Sprintf("%s %s\n", cursor, choice)
	}

	str += "\nPress Enter to continue.\n"

	return bubbleTea.NewView(str)
}
