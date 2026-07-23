package tui

import (
	execute "barrakudaModKit/internal/execute/init"
)

func GoMcpInitialModel() ScaffoldModel {
	return ScaffoldModel{Title: "MCP (Go) — project name", Create: execute.CreateGoMcp}
}

func TypescriptMcpInitialModel() ScaffoldModel {
	return ScaffoldModel{Title: "MCP (Typescript) — project name", Create: execute.CreateTypescriptMcp}
}

func PythonMcpInitialModel() ScaffoldModel {
	return ScaffoldModel{Title: "MCP (Python) — project name", Create: execute.CreatePythonMcp}
}

func AgentInitialModel() ScaffoldModel {
	return ScaffoldModel{Title: "Agent (Docker) — project name", Create: execute.CreateAgent}
}

func ThemeInitialModel() ScaffoldModel {
	return ScaffoldModel{Title: "Theme — name", Create: execute.CreateTheme}
}

func SkillInitialModel() ScaffoldModel {
	return ScaffoldModel{Title: "Skill — name", Create: execute.CreateSkill}
}

func ProviderGoInitialModel() ScaffoldModel {
	return ScaffoldModel{Title: "Provider (Go) — project name", Create: execute.CreateGoProvider}
}
