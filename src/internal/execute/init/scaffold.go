package execute_init

import (
	"os"
	"path/filepath"
)

// writeFiles mkdirs and writes each file relative to root, returning root on success.
func writeFiles(root string, files map[string]string) (string, error) {
	for rel, content := range files {
		full := filepath.Join(root, rel)

		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			return "", err
		}

		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			return "", err
		}
	}

	return root, nil
}
