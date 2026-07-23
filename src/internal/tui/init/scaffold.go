package tui

import (
	bubbleTea "charm.land/bubbletea/v2"
)

// ScaffoldModel drives all 6 "create a mod" screens: name input -> write skeleton ->
// SuccessModel. The 6 screens differ only in title and which Create func they call.
type ScaffoldModel struct {
	Title  string
	Create func(name string) (string, error)
	Input  TextInputModel
	Error  string
}

func (m ScaffoldModel) Init() bubbleTea.Cmd {
	return nil
}

func (m ScaffoldModel) Update(msg bubbleTea.Msg) (bubbleTea.Model, bubbleTea.Cmd) {
	key, ok := msg.(bubbleTea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	switch key.String() {
	case "esc":
		return ModTypeInitialModel(), nil

	case "enter":
		if m.Input.Value == "" {
			m.Error = "Name can't be empty."
			return m, nil
		}

		path, err := m.Create(m.Input.Value)
		if err != nil {
			m.Error = err.Error()
			return m, nil
		}

		return SuccessModel{Path: path}, nil

	default:
		m.Input.Handle(key)
		m.Error = ""
	}

	return m, nil
}

func (m ScaffoldModel) View() bubbleTea.View {
	str := m.Title + "\n\n" + m.Input.View("Name")

	if m.Error != "" {
		str += "\n" + m.Error + "\n"
	}

	return bubbleTea.NewView(str)
}
