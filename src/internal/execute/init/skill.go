package execute_init

import (
	"barrakudaModKit/internal/manifest"
)

// CreateSkill scaffolds a skill-type mod: content-only, no code execution.
// specs.json's skills[] lists the doc pages; barrakuda-software renders
// each as sanitized markdown once the mod is installed.
func CreateSkill(name string) (string, error) {
	m := manifest.Manifest{
		Name:    name,
		Version: "0.0.1",
		License: "MIT",
		Type:    manifest.TypeSkill,
		Tags:    []string{"barrakuda", "skill"},
		Skills: []manifest.ModDoc{
			{Name: name, Description: name + " description", Path: "./skill.md"},
		},
	}

	manifestJSON, err := m.JSON()
	if err != nil {
		return "", err
	}

	return writeFiles(name, map[string]string{
		"specs.json": manifestJSON,
		"skill.md":      "# " + name + "\n",
	})
}
