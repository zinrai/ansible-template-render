package utils

import (
	"gopkg.in/yaml.v3"
)

// Performs a deep copy of a map[string]interface{}
func DeepCopyMap(original map[string]interface{}) map[string]interface{} {
	if original == nil {
		return nil
	}

	// Marshal to YAML
	data, err := yaml.Marshal(original)
	if err != nil {
		// Return an empty map on error
		return make(map[string]interface{})
	}

	// Unmarshal to create a copy
	var copy map[string]interface{}
	err = yaml.Unmarshal(data, &copy)
	if err != nil {
		// Return an empty map on error
		return make(map[string]interface{})
	}

	return copy
}

// Performs a deep copy of a task list
func DeepCopyTaskList(original []map[string]interface{}) []map[string]interface{} {
	if original == nil {
		return nil
	}

	copy := make([]map[string]interface{}, len(original))
	for i, task := range original {
		copy[i] = DeepCopyMap(task)
	}

	return copy
}
