package ansible

import (
	"reflect"
	"testing"
)

func TestPlaybookRoleExtractor_Extract(t *testing.T) {
	tests := []struct {
		name     string
		playbook []map[string]interface{}
		expected []string
	}{
		{
			name: "simple roles",
			playbook: []map[string]interface{}{
				{
					"hosts": "all",
					"roles": []interface{}{"role1", "role2"},
				},
			},
			expected: []string{"role1", "role2"},
		},
		{
			name: "mixed role formats",
			playbook: []map[string]interface{}{
				{
					"hosts": "all",
					"roles": []interface{}{
						"role1",
						map[string]interface{}{
							"role": "role2",
							"vars": map[string]interface{}{
								"var1": "value1",
							},
						},
						map[string]interface{}{
							"name": "role3",
							"tags": []interface{}{"tag1"},
						},
					},
				},
			},
			expected: []string{"role1", "role2", "role3"},
		},
		{
			name: "multiple plays",
			playbook: []map[string]interface{}{
				{
					"hosts": "web",
					"roles": []interface{}{"web_role"},
				},
				{
					"hosts": "db",
					"roles": []interface{}{"db_role"},
				},
			},
			expected: []string{"web_role", "db_role"},
		},
		{
			name: "duplicate roles",
			playbook: []map[string]interface{}{
				{
					"hosts": "all",
					"roles": []interface{}{"role1", "role1", "role2"},
				},
			},
			expected: []string{"role1", "role2"},
		},
		{
			name: "no roles",
			playbook: []map[string]interface{}{
				{
					"hosts": "all",
					"tasks": []interface{}{
						map[string]interface{}{
							"name":    "Task 1",
							"command": "echo hello",
						},
					},
				},
			},
			expected: []string{},
		},
		{
			name:     "empty playbook",
			playbook: []map[string]interface{}{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := PlaybookRoleExtractor{}
			result := extractor.Extract(tt.playbook)

			// Check if results have the same elements regardless of order
			if len(result) != len(tt.expected) {
				t.Errorf("Extract() returned %d roles, want %d", len(result), len(tt.expected))
			}

			// Convert results to map for easier comparison
			resultMap := make(map[string]bool)
			for _, r := range result {
				resultMap[r] = true
			}

			// Check all expected roles are in result
			for _, e := range tt.expected {
				if !resultMap[e] {
					t.Errorf("Extract() missing expected role %s", e)
				}
			}
		})
	}
}

func TestPlaybookRoleExtractor_extractRoleName(t *testing.T) {
	tests := []struct {
		name     string
		role     interface{}
		expected string
	}{
		{
			name:     "string role",
			role:     "role1",
			expected: "role1",
		},
		{
			name: "role key",
			role: map[string]interface{}{
				"role": "role2",
				"vars": map[string]interface{}{},
			},
			expected: "role2",
		},
		{
			name: "name key",
			role: map[string]interface{}{
				"name": "role3",
				"tags": []interface{}{},
			},
			expected: "role3",
		},
		{
			name: "no valid key",
			role: map[string]interface{}{
				"vars": map[string]interface{}{},
			},
			expected: "",
		},
		{
			name:     "non-string and non-map",
			role:     123,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := PlaybookRoleExtractor{}
			result := extractor.extractRoleName(tt.role)
			if result != tt.expected {
				t.Errorf("extractRoleName() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestExtractRolesFromPlaybook(t *testing.T) {
	playbook := []map[string]interface{}{
		{
			"hosts": "all",
			"roles": []interface{}{"role1", "role2"},
		},
	}

	expected := []string{"role1", "role2"}
	result := ExtractRolesFromPlaybook(playbook)

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("ExtractRolesFromPlaybook() = %v, want %v", result, expected)
	}
}
