package finder

import (
	"fmt"
	"os"
	"path/filepath"
)

// Finds task files for a role
func FindRoleTasks(roleName string) ([]string, error) {
	tasksDir := filepath.Join("roles", roleName, "tasks")

	// Check if tasks directory exists
	_, err := os.Stat(tasksDir)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("role tasks directory not found: %s", tasksDir)
	}

	// Find .yml files
	ymlFiles, err := filepath.Glob(filepath.Join(tasksDir, "*.yml"))
	if err != nil {
		return nil, fmt.Errorf("error searching .yml files: %w", err)
	}

	// Find .yaml files
	yamlFiles, err := filepath.Glob(filepath.Join(tasksDir, "*.yaml"))
	if err != nil {
		return nil, fmt.Errorf("error searching .yaml files: %w", err)
	}

	// Combine results
	return append(ymlFiles, yamlFiles...), nil
}

// Finds the main.yml task file for a role
func FindRoleMainTask(roleName string) (string, bool, error) {
	tasksDir := filepath.Join("roles", roleName, "tasks")

	// Check if directory exists
	_, err := os.Stat(tasksDir)
	if os.IsNotExist(err) {
		return "", false, fmt.Errorf("role tasks directory not found: %s", tasksDir)
	}

	// Define possible main task paths
	mainPaths := []string{
		filepath.Join(tasksDir, "main.yml"),
		filepath.Join(tasksDir, "main.yaml"),
	}

	// Check each possible path
	for _, mainPath := range mainPaths {
		info, err := os.Stat(mainPath)
		if err != nil {
			continue
		}

		if info.IsDir() {
			continue
		}

		return mainPath, true, nil
	}

	return "", false, fmt.Errorf("main task file not found for role: %s", roleName)
}
