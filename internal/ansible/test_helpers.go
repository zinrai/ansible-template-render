package ansible

import (
	"os"
	"path/filepath"
	"testing"
)

// Provides utilities for testing with Ansible playbooks
type TestPlaybookHelper struct {
	TempDir string
}

// Creates a new test helper with a temporary directory
func NewTestPlaybookHelper(t *testing.T) *TestPlaybookHelper {
	tempDir, err := os.MkdirTemp("", "ansible-test-")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	return &TestPlaybookHelper{
		TempDir: tempDir,
	}
}

// Removes the temporary directory
func (h *TestPlaybookHelper) Cleanup() {
	os.RemoveAll(h.TempDir)
}

// Creates a role directory structure
func (h *TestPlaybookHelper) CreateRoleDirectory(t *testing.T, roleName string) string {
	rolePath := filepath.Join(h.TempDir, "roles", roleName)
	dirs := []string{
		filepath.Join(rolePath, "tasks"),
		filepath.Join(rolePath, "templates"),
		filepath.Join(rolePath, "meta"),
	}

	for _, dir := range dirs {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			t.Fatalf("Failed to create role directory %s: %v", dir, err)
		}
	}

	return rolePath
}

// Creates a meta file for a role
func (h *TestPlaybookHelper) CreateMetaFile(t *testing.T, roleName, content string) string {
	metaPath := filepath.Join(h.TempDir, "roles", roleName, "meta", "main.yml")
	err := os.WriteFile(metaPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create meta file: %v", err)
	}
	return metaPath
}
