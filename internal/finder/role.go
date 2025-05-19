package finder

import (
	"fmt"
	"os"
	"path/filepath"
)

// Gets the directory path for a role
func FindRolePath(roleName string) (string, error) {
	rolePath := filepath.Join("roles", roleName)

	// Check if directory exists
	info, err := os.Stat(rolePath)
	if err != nil {
		return "", fmt.Errorf("role directory not found: %s", rolePath)
	}

	if !info.IsDir() {
		return "", fmt.Errorf("role path is not a directory: %s", rolePath)
	}

	return rolePath, nil
}

// Finds the meta/main.yml file for a role
func FindRoleMetaFile(roleName string) (string, bool, error) {
	rolePath, err := FindRolePath(roleName)
	if err != nil {
		return "", false, err
	}

	// Define possible meta file paths
	metaPaths := []string{
		filepath.Join(rolePath, "meta", "main.yml"),
		filepath.Join(rolePath, "meta", "main.yaml"),
	}

	// Check each possible path
	for _, metaPath := range metaPaths {
		info, err := os.Stat(metaPath)
		if err != nil {
			continue
		}

		if info.IsDir() {
			continue
		}

		return metaPath, true, nil
	}

	// Not finding a meta file is not an error
	return "", false, nil
}
