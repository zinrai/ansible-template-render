package finder

import (
	"fmt"
	"os"
	"path/filepath"
)

// Finds a playbook file with the specified name
func FindPlaybook(name string) (string, error) {
	// Define possible file patterns
	patterns := []string{
		filepath.Join(".", fmt.Sprintf("%s.yml", name)),
		filepath.Join(".", fmt.Sprintf("%s.yaml", name)),
	}

	// Try each pattern
	for _, pattern := range patterns {
		info, err := os.Stat(pattern)
		if err != nil {
			continue
		}

		if info.IsDir() {
			continue
		}

		return pattern, nil
	}

	return "", fmt.Errorf("playbook not found: %s", name)
}
