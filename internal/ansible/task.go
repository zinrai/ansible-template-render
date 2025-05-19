package ansible

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Loads a task file
func LoadTaskFile(path string) ([]map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading task file: %w", err)
	}

	// Try to parse as a task list first
	var tasks []map[string]interface{}
	err = yaml.Unmarshal(data, &tasks)
	if err == nil {
		return tasks, nil
	}

	// If list parsing fails, try parsing as a single task or task container
	return parseSingleTaskOrContainer(data)
}

// Parses a file with a single task or task container
func parseSingleTaskOrContainer(data []byte) ([]map[string]interface{}, error) {
	var singleTask map[string]interface{}
	err := yaml.Unmarshal(data, &singleTask)
	if err != nil {
		return nil, fmt.Errorf("parsing task file: %w", err)
	}

	// Empty file - return empty task list
	if len(singleTask) == 0 {
		return []map[string]interface{}{}, nil
	}

	// Check if it's a tasks container
	tasksList, ok := singleTask["tasks"].([]interface{})
	if !ok {
		// Not a tasks container, treat as a single task
		return []map[string]interface{}{singleTask}, nil
	}

	// Convert tasks from interface{} to map[string]interface{}
	return convertTasksList(tasksList)
}

// Converts a list of interface{} to task maps
func convertTasksList(tasksList []interface{}) ([]map[string]interface{}, error) {
	tasks := make([]map[string]interface{}, 0, len(tasksList))

	for _, taskItem := range tasksList {
		taskMap, ok := taskItem.(map[string]interface{})
		if !ok {
			continue // Skip non-map tasks
		}
		tasks = append(tasks, taskMap)
	}

	return tasks, nil
}

// Saves a task file
func SaveTaskFile(tasks []map[string]interface{}, path string) error {
	// Ensure the output directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	data, err := yaml.Marshal(tasks)
	if err != nil {
		return fmt.Errorf("marshaling tasks: %w", err)
	}

	err = os.WriteFile(path, data, 0644)
	if err != nil {
		return fmt.Errorf("writing task file: %w", err)
	}

	return nil
}
