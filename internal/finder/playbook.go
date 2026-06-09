package finder

import (
	"fmt"
	"os"
)

// Verifies a playbook file at the specified path
func FindPlaybook(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("playbook not found: %s", path)
	}

	if info.IsDir() {
		return "", fmt.Errorf("playbook path is a directory, not a file: %s", path)
	}

	return path, nil
}
