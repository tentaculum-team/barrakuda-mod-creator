package tui

import (
	bubbleTea "charm.land/bubbletea/v2"
)

type LangViewModel struct {
	Menu  MenuModel
	Error string
}

func LangViewInitialModel() LangViewModel {
	return LangViewModel{
		Menu: MenuModel{
			Title: "What programming language do you want to use?",
			Choices: []string{
				"Golang",
				"Typescript",
				"Python",
				"Rust - coming soon",
				"Java (Spring Boot) - coming soon",
				"C++ - coming soon",
				"C# - coming soon",
				"Ruby - coming soon",
				"PHP - coming soon",
				"Elixir - coming soon",
				"Lua - coming soon",
			},
		},
	}
}

func (m LangViewModel) Init() bubbleTea.Cmd {
	return nil
}

func (view LangViewModel) Update(msg bubbleTea.Msg) (bubbleTea.Model, bubbleTea.Cmd) {

	switch msg := msg.(type) {

	case bubbleTea.KeyPressMsg:

		switch msg.String() {

		case "up":
			view.Menu.MoveUp()

		case "down":
			view.Menu.MoveDown()

		case "enter":

			switch view.Menu.Cursor {

			case 0:
				return GoMcpInitialModel(), nil

			case 1:
				return TypescriptMcpInitialModel(), nil

			case 2:
				return PythonMcpInitialModel(), nil

			default:
				view.Error = "This language is not yet supported."
			}
		}
	}

	return view, nil
}

func (m LangViewModel) View() bubbleTea.View {

	str := m.Menu.View().Content

	if m.Error != "" {
		str += "\n" + m.Error + "\n"
	}

	return bubbleTea.NewView(str)
}
