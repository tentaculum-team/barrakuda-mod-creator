package tui

import (
	bubbleTea "charm.land/bubbletea/v2"
)

// SuccessModel is the shared "skeleton written" screen every scaffold flow ends on.
type SuccessModel struct {
	Path string
}

func (m SuccessModel) Init() bubbleTea.Cmd {
	return nil
}

func (m SuccessModel) Update(msg bubbleTea.Msg) (bubbleTea.Model, bubbleTea.Cmd) {
	if key, ok := msg.(bubbleTea.KeyPressMsg); ok {
		switch key.String() {
		case "esc", "enter":
			return ModTypeInitialModel(), nil
		}
	}

	return m, nil
}

func (m SuccessModel) View() bubbleTea.View {
	return bubbleTea.NewView("Skeleton written to " + m.Path + "\n\nPress enter/esc to go back.\n")
}
