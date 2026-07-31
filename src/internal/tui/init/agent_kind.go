package tui

import (
	bubbleTea "charm.land/bubbletea/v2"
)

// AgentKindModel picks between a plain Docker sidecar (no chat integration)
// and a chat-selectable agent provider (declares agent_provider, appears in
// barrakuda-software's chat AI picker) — mirrors LangViewModel's pattern of
// a sub-menu under one ModTypeModel choice.
type AgentKindModel struct {
	Menu MenuModel
}

func AgentKindInitialModel() AgentKindModel {
	return AgentKindModel{
		Menu: MenuModel{
			Title: "What kind of agent mod?",
			Choices: []string{
				"Generic Docker sidecar",
				"Chat agent provider (appears in the AI picker)",
			},
		},
	}
}

func (m AgentKindModel) Init() bubbleTea.Cmd {
	return nil
}

func (m AgentKindModel) Update(msg bubbleTea.Msg) (bubbleTea.Model, bubbleTea.Cmd) {
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
				return AgentInitialModel(), nil

			case 1:
				return AgentProviderInitialModel(), nil
			}
		}
	}

	return m, nil
}

func (m AgentKindModel) View() bubbleTea.View {
	return m.Menu.View()
}
