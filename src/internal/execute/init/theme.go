package execute_init

import (
	"barrakudaModKit/internal/manifest"
)

// CreateTheme scaffolds a theme-type mod: a single specs.json carrying
// colors/fonts/icons inline (no separate theme.json — barrakuda-software
// reads theme tokens straight off extensions.manifest).
func CreateTheme(name string) (string, error) {
	m := manifest.Manifest{
		Name:    name,
		Version: "0.0.1",
		License: "MIT",
		Type:    manifest.TypeTheme,
		Tags:    []string{"barrakuda", "theme"},
	}

	manifestJSON, err := m.JSON()
	if err != nil {
		return "", err
	}

	return writeFiles(name, map[string]string{
		"specs.json": manifestJSON,
	})
}
