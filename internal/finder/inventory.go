package finder

import (
	"fmt"
	"os"
	"path/filepath"
)

// Finds an inventory file at the specified path
func FindInventory(path string) (string, error) {
	// Check if the path is absolute or relative
	var fullPath string
	if filepath.IsAbs(path) {
		fullPath = path
	} else {
		fullPath = filepath.Clean(path)
	}

	// Check if file exists
	info, err := os.Stat(fullPath)
	if err != nil {
		return "", fmt.Errorf("inventory file not found: %s", path)
	}

	if info.IsDir() {
		return "", fmt.Errorf("inventory path is a directory, not a file: %s", path)
	}

	return fullPath, nil
}
