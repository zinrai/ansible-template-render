package ansible

import (
	"fmt"
	"os"

	"github.com/zinrai/ansible-template-render/internal/finder"
	"github.com/zinrai/ansible-template-render/internal/logger"

	"gopkg.in/yaml.v3"
)

// Represents a role dependency
type RoleDependency struct {
	Name string
}

// Represents a role's meta information
type RoleMeta struct {
	Dependencies []interface{} `yaml:"dependencies"`
}

// Resolves role dependencies
type RoleDependencyResolver struct{}

// Gets the dependencies of a role
func (r *RoleDependencyResolver) GetDependencies(roleName string) ([]string, error) {
	metaPath, exists, err := finder.FindRoleMetaFile(roleName)
	if err != nil {
		return nil, err
	}

	// No meta file means no dependencies
	if !exists {
		return []string{}, nil
	}

	// Read and parse meta file
	meta, err := r.loadRoleMeta(metaPath)
	if err != nil {
		return nil, err
	}

	// Extract dependencies
	return r.extractDependencies(meta.Dependencies), nil
}

// Loads and parses a role's meta file
func (r *RoleDependencyResolver) loadRoleMeta(metaPath string) (RoleMeta, error) {
	var meta RoleMeta

	data, err := os.ReadFile(metaPath)
	if err != nil {
		return meta, fmt.Errorf("reading meta file: %w", err)
	}

	err = yaml.Unmarshal(data, &meta)
	if err != nil {
		return meta, fmt.Errorf("parsing meta file: %w", err)
	}

	// Ensure Dependencies is never nil
	if meta.Dependencies == nil {
		meta.Dependencies = make([]interface{}, 0)
	}

	return meta, nil
}

// Extracts dependency role names
func (r *RoleDependencyResolver) extractDependencies(dependencies []interface{}) []string {
	result := make([]string, 0)

	if dependencies == nil {
		return result
	}

	for _, dep := range dependencies {
		// Handle string dependencies
		if roleName, ok := dep.(string); ok {
			result = append(result, roleName)
			continue
		}

		// Handle map dependencies
		depMap, ok := dep.(map[string]interface{})
		if !ok {
			continue
		}

		// Check "role" key first
		if roleName, ok := depMap["role"].(string); ok {
			result = append(result, roleName)
			continue
		}

		// Check "name" key as fallback
		if roleName, ok := depMap["name"].(string); ok {
			result = append(result, roleName)
		}
	}

	return result
}

// Gets the dependencies of a role
func GetRoleDependencies(roleName string) ([]string, error) {
	resolver := RoleDependencyResolver{}
	return resolver.GetDependencies(roleName)
}

// Recursively resolves role dependencies
func ResolveRoleDependencies(roleName string, resolved map[string]bool) ([]string, error) {
	// Skip already resolved roles
	if resolved[roleName] {
		return nil, nil
	}

	// Mark this role as resolved
	resolved[roleName] = true

	// Get role dependencies
	dependencies, err := GetRoleDependencies(roleName)
	if err != nil {
		logger.Warn("Error getting dependencies", "role", roleName, "error", err)
		dependencies = []string{} // Continue with no dependencies on error
	}

	// Resolve dependencies recursively
	var allRoles []string
	for _, dep := range dependencies {
		depRoles, err := ResolveRoleDependencies(dep, resolved)
		if err != nil {
			logger.Warn("Error resolving dependencies", "role", dep, "error", err)
			continue // Skip this dependency on error
		}
		allRoles = append(allRoles, depRoles...)
	}

	// Add this role (after its dependencies)
	allRoles = append(allRoles, roleName)

	return allRoles, nil
}
