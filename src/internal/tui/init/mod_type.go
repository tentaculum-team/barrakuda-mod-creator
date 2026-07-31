package tui

import (
	bubbleTea "charm.land/bubbletea/v2"
)

type ModTypeModel struct {
	Menu MenuModel
}

func ModTypeInitialModel() ModTypeModel {
	return ModTypeModel{
		Menu: MenuModel{
			Title: "What type of mod do you want to create?",
			Choices: []string{
				"MCP",
				"Agent",
				"Theme",
				"Skill",
				"Provider",
			},
		},
	}
}

func (m ModTypeModel) Init() bubbleTea.Cmd {
	return nil
}

func (m ModTypeModel) Update(msg bubbleTea.Msg) (bubbleTea.Model, bubbleTea.Cmd) {
	switch msg := msg.(type) {

	case bubbleTea.KeyPressMsg:

		switch msg.String() {

		case "up":
			m.Menu.MoveUp()

		case "down":
			m.Menu.MoveDown()

		case "enter":

			switch m.Menu.Cursor {

			case 0:
				return LangViewInitialModel(), nil

			case 1:
				return AgentKindInitialModel(), nil

			case 2:
				return ThemeInitialModel(), nil

			case 3:
				return SkillInitialModel(), nil

			case 4:
				return ProviderGoInitialModel(), nil
			}
		}
	}

	return m, nil
}

func (m ModTypeModel) View() bubbleTea.View {
	return m.Menu.View()
}
