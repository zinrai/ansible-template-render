package processor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAddLocalConnectionToHosts(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "basic inventory",
			input: `[webservers]
web1 ansible_host=192.168.1.10
web2 ansible_host=192.168.1.11
`,
			expected: `[webservers]
web1 ansible_host=192.168.1.10 ansible_connection=local
web2 ansible_host=192.168.1.11 ansible_connection=local
`,
		},
		{
			name: "with existing connection",
			input: `[webservers]
web1 ansible_host=192.168.1.10 ansible_connection=ssh
web2 ansible_host=192.168.1.11
`,
			expected: `[webservers]
web1 ansible_host=192.168.1.10 ansible_connection=local
web2 ansible_host=192.168.1.11 ansible_connection=local
`,
		},
		{
			name: "with vars section",
			input: `[webservers]
web1 ansible_host=192.168.1.10
web2 ansible_host=192.168.1.11

[webservers:vars]
ansible_user=admin
ansible_connection=ssh
`,
			expected: `[webservers]
web1 ansible_host=192.168.1.10 ansible_connection=local
web2 ansible_host=192.168.1.11 ansible_connection=local

[webservers:vars]
ansible_user=admin
ansible_connection=ssh
`,
		},
		{
			name: "with children section",
			input: `[production]
web1 ansible_host=192.168.1.10
db1 ansible_host=192.168.1.20

[all:children]
production
staging
`,
			expected: `[production]
web1 ansible_host=192.168.1.10 ansible_connection=local
db1 ansible_host=192.168.1.20 ansible_connection=local

[all:children]
production
staging
`,
		},
		{
			name: "with comments and empty lines",
			input: `# Webservers
[webservers]
# Main web server
web1 ansible_host=192.168.1.10

# Database servers
[dbservers]
db1 ansible_host=192.168.1.20
`,
			expected: `# Webservers
[webservers]
# Main web server
web1 ansible_host=192.168.1.10 ansible_connection=local

# Database servers
[dbservers]
db1 ansible_host=192.168.1.20 ansible_connection=local
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := addLocalConnectionToHosts(tt.input)
			if result != tt.expected {
				t.Errorf("addLocalConnectionToHosts() result does not match expected output\nGot:\n%s\nExpected:\n%s", result, tt.expected)
			}
		})
	}
}

func TestModifyInventoryForLocalExecution(t *testing.T) {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "inventory-test-")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test inventory file
	inventoryContent := `[webservers]
web1 ansible_host=192.168.1.10
web2 ansible_host=192.168.1.11 ansible_connection=ssh
`
	inventoryPath := filepath.Join(tempDir, "inventory")
	err = os.WriteFile(inventoryPath, []byte(inventoryContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test inventory file: %v", err)
	}

	// Call the function to modify the inventory
	err = ModifyInventoryForLocalExecution(inventoryPath)
	if err != nil {
		t.Fatalf("ModifyInventoryForLocalExecution() error = %v", err)
	}

	// Read the modified file
	modifiedContent, err := os.ReadFile(inventoryPath)
	if err != nil {
		t.Fatalf("Failed to read modified inventory file: %v", err)
	}

	// Check if modifications were made correctly
	expected := `[webservers]
web1 ansible_host=192.168.1.10 ansible_connection=local
web2 ansible_host=192.168.1.11 ansible_connection=local
`
	if string(modifiedContent) != expected {
		t.Errorf("ModifyInventoryForLocalExecution() did not modify the file correctly\nGot:\n%s\nExpected:\n%s",
			string(modifiedContent), expected)
	}
}

func TestModifyInventoryForLocalExecution_FileError(t *testing.T) {
	// Test with non-existent file
	err := ModifyInventoryForLocalExecution("/path/to/nonexistent/inventory")
	if err == nil {
		t.Errorf("ModifyInventoryForLocalExecution() expected error for non-existent file, got nil")
	}
	if !strings.Contains(err.Error(), "reading inventory file") {
		t.Errorf("ModifyInventoryForLocalExecution() error message does not match expected pattern: %v", err)
	}
}
