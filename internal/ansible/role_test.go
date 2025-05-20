package ansible

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRoleDependencyResolver_extractDependencies(t *testing.T) {
	tests := []struct {
		name         string
		dependencies []interface{}
		expected     []string
	}{
		{
			name:         "string dependencies",
			dependencies: []interface{}{"role1", "role2"},
			expected:     []string{"role1", "role2"},
		},
		{
			name: "map dependencies with role key",
			dependencies: []interface{}{
				map[string]interface{}{
					"role": "role3",
					"vars": map[string]interface{}{},
				},
				map[string]interface{}{
					"role": "role4",
				},
			},
			expected: []string{"role3", "role4"},
		},
		{
			name: "map dependencies with name key",
			dependencies: []interface{}{
				map[string]interface{}{
					"name": "role5",
				},
			},
			expected: []string{"role5"},
		},
		{
			name: "mixed dependency types",
			dependencies: []interface{}{
				"role6",
				map[string]interface{}{
					"role": "role7",
				},
				map[string]interface{}{
					"name": "role8",
				},
				123, // Invalid type - should be ignored
			},
			expected: []string{"role6", "role7", "role8"},
		},
		{
			name:         "empty dependencies",
			dependencies: []interface{}{},
			expected:     []string{}, // Empty slice, not nil
		},
		{
			name:         "nil dependencies",
			dependencies: nil,
			expected:     []string{}, // Empty slice, not nil
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := RoleDependencyResolver{}
			result := resolver.extractDependencies(tt.dependencies)

			// Length Comparison
			if len(result) != len(tt.expected) {
				t.Errorf("extractDependencies() returned %d items, want %d items",
					len(result), len(tt.expected))
				return
			}

			// Comparison of contents considering order
			for i, v := range tt.expected {
				if i < len(result) && result[i] != v {
					t.Errorf("extractDependencies()[%d] = %v, want %v",
						i, result[i], v)
				}
			}
		})
	}
}

func TestRoleDependencyResolver_loadRoleMeta(t *testing.T) {
	// Create a temporary meta file
	tempDir, err := os.MkdirTemp("", "ansible-test-")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	metaContent := `---
dependencies:
  - role1
  - role: role2
    vars:
      var1: value1
`

	metaPath := filepath.Join(tempDir, "main.yml")
	err = os.WriteFile(metaPath, []byte(metaContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write meta file: %v", err)
	}

	resolver := RoleDependencyResolver{}
	meta, err := resolver.loadRoleMeta(metaPath)
	if err != nil {
		t.Fatalf("loadRoleMeta() error = %v", err)
	}

	if len(meta.Dependencies) != 2 {
		t.Errorf("loadRoleMeta() returned meta with %d dependencies, want 2", len(meta.Dependencies))
	}

	// Non-existent file should return error
	_, err = resolver.loadRoleMeta(filepath.Join(tempDir, "non_existent.yml"))
	if err == nil {
		t.Errorf("loadRoleMeta() should return error for non-existent file")
	}
}

// This test uses an integration test helper since it requires more complex setup
func TestResolveRoleDependencies(t *testing.T) {
	helper := NewTestPlaybookHelper(t)
	defer helper.Cleanup()

	// Create several roles with dependencies
	helper.CreateRoleDirectory(t, "role1")
	helper.CreateRoleDirectory(t, "role2")
	helper.CreateRoleDirectory(t, "role3")
	helper.CreateRoleDirectory(t, "role4")

	// role1 depends on role2 and role3
	helper.CreateMetaFile(t, "role1", `---
dependencies:
  - role2
  - role3
`)

	// role3 depends on role4
	helper.CreateMetaFile(t, "role3", `---
dependencies:
  - role4
`)

	// role2 and role4 have no dependencies
	helper.CreateMetaFile(t, "role2", `---
dependencies: []
`)

	// Save current directory and change to test directory
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(currentDir)

	os.Chdir(helper.TempDir)

	// Test resolving role1's dependencies
	resolved := make(map[string]bool)
	roles, err := ResolveRoleDependencies("role1", resolved)
	if err != nil {
		t.Fatalf("ResolveRoleDependencies() error = %v", err)
	}

	// Expected order: role4, role3, role2, role1 (dependencies before the role)
	// But the exact order isn't guaranteed, just that dependencies come before the role
	expectedRoles := []string{"role2", "role4", "role3", "role1"}

	// Check if all expected roles are present
	if len(roles) != len(expectedRoles) {
		t.Errorf("ResolveRoleDependencies() returned %d roles, want %d", len(roles), len(expectedRoles))
	}

	// Convert to map for easier presence checking
	rolesMap := make(map[string]bool)
	for _, role := range roles {
		rolesMap[role] = true
	}

	for _, expected := range expectedRoles {
		if !rolesMap[expected] {
			t.Errorf("ResolveRoleDependencies() missing expected role %s", expected)
		}
	}

	// Roles should appear after their dependencies
	role1Index := -1
	role3Index := -1
	role4Index := -1

	for i, role := range roles {
		switch role {
		case "role1":
			role1Index = i
		case "role3":
			role3Index = i
		case "role4":
			role4Index = i
		}
	}

	// role4 should come before role3
	if role4Index >= 0 && role3Index >= 0 && role4Index > role3Index {
		t.Errorf("role4 (%d) should come before role3 (%d)", role4Index, role3Index)
	}

	// role3 should come before role1
	if role3Index >= 0 && role1Index >= 0 && role3Index > role1Index {
		t.Errorf("role3 (%d) should come before role1 (%d)", role3Index, role1Index)
	}
}
