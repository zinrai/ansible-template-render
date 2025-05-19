package ansible

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Loads an Ansible playbook file
func LoadPlaybook(path string) ([]map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading playbook file: %w", err)
	}

	var playbook []map[string]interface{}
	err = yaml.Unmarshal(data, &playbook)
	if err != nil {
		return nil, fmt.Errorf("parsing playbook file: %w", err)
	}

	return playbook, nil
}

// Extracts roles used in a playbook
func ExtractRolesFromPlaybook(playbook []map[string]interface{}) []string {
	roleMap := make(map[string]bool) // Track roles to prevent duplicates
	var roles []string

	for _, play := range playbook {
		rolesList, ok := play["roles"].([]interface{})
		if !ok {
			continue
		}

		for _, role := range rolesList {
			roleName := extractRoleName(role)
			if roleName == "" {
				continue
			}

			if roleMap[roleName] {
				continue
			}

			roleMap[roleName] = true
			roles = append(roles, roleName)
		}
	}

	return roles
}

// Extracts role name from different role specifications
func extractRoleName(role interface{}) string {
	// Direct string role name
	if roleName, ok := role.(string); ok {
		return roleName
	}

	// Role specified as a map
	roleMap, ok := role.(map[string]interface{})
	if !ok {
		return ""
	}

	// Check "role" key first
	if roleName, ok := roleMap["role"].(string); ok {
		return roleName
	}

	// Check "name" key as fallback
	if roleName, ok := roleMap["name"].(string); ok {
		return roleName
	}

	return ""
}
