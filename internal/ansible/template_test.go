package ansible

import (
	"reflect"
	"strings"
	"testing"
)

func TestIsTemplateTask(t *testing.T) {
	tests := []struct {
		name     string
		task     map[string]interface{}
		expected bool
	}{
		{
			name: "standard template module",
			task: map[string]interface{}{
				"template": map[string]interface{}{
					"src":  "source.j2",
					"dest": "/etc/config",
				},
			},
			expected: true,
		},
		{
			name: "builtin template module",
			task: map[string]interface{}{
				"ansible.builtin.template": map[string]interface{}{
					"src":  "source.j2",
					"dest": "/etc/config",
				},
			},
			expected: true,
		},
		{
			name: "not a template task",
			task: map[string]interface{}{
				"copy": map[string]interface{}{
					"src":  "source",
					"dest": "/etc/config",
				},
			},
			expected: false,
		},
		{
			name:     "empty task",
			task:     map[string]interface{}{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsTemplateTask(tt.task)
			if result != tt.expected {
				t.Errorf("IsTemplateTask() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNewTemplateTask(t *testing.T) {
	// Test with standard template module
	standardTask := map[string]interface{}{
		"template": map[string]interface{}{
			"src":  "source.j2",
			"dest": "/etc/config",
		},
	}

	task, ok := NewTemplateTask(standardTask)
	if !ok {
		t.Fatalf("NewTemplateTask() failed to recognize template task")
	}

	if task.ModuleKey != "template" {
		t.Errorf("ModuleKey = %v, want %v", task.ModuleKey, "template")
	}

	dest, ok := task.ModuleData["dest"].(string)
	if !ok || dest != "/etc/config" {
		t.Errorf("ModuleData[dest] = %v, want %v", task.ModuleData["dest"], "/etc/config")
	}

	// Test with non-template task
	nonTemplateTask := map[string]interface{}{
		"command": "echo test",
	}

	_, ok = NewTemplateTask(nonTemplateTask)
	if ok {
		t.Errorf("NewTemplateTask() should return false for non-template task")
	}
}

func TestTemplateTask_GetDestPath(t *testing.T) {
	tests := []struct {
		name     string
		task     map[string]interface{}
		expected string
	}{
		{
			name: "standard path",
			task: map[string]interface{}{
				"template": map[string]interface{}{
					"dest": "/etc/config",
				},
			},
			expected: "/etc/config",
		},
		{
			name: "builtin module path",
			task: map[string]interface{}{
				"ansible.builtin.template": map[string]interface{}{
					"dest": "/var/lib/app",
				},
			},
			expected: "/var/lib/app",
		},
		{
			name: "missing dest",
			task: map[string]interface{}{
				"template": map[string]interface{}{
					"src": "source.j2",
				},
			},
			expected: "",
		},
		{
			name: "dest not a string",
			task: map[string]interface{}{
				"template": map[string]interface{}{
					"dest": 123,
				},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			templateTask, ok := NewTemplateTask(tt.task)
			if !ok {
				if tt.expected != "" {
					t.Fatalf("NewTemplateTask() failed to recognize template task")
				}
				return
			}

			result := templateTask.GetDestPath()
			if result != tt.expected {
				t.Errorf("GetDestPath() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTemplateTask_Modify(t *testing.T) {
	// Test task with all features to modify
	fullTask := map[string]interface{}{
		"name": "Configure app",
		"template": map[string]interface{}{
			"src":  "app.conf.j2",
			"dest": "/etc/app/config.conf",
		},
		"notify": []interface{}{"restart app"},
	}

	templateTask, ok := NewTemplateTask(fullTask)
	if !ok {
		t.Fatalf("NewTemplateTask() failed to recognize template task")
	}

	// Modify the task - Added "test-playbook" as playbookName argument
	templateTask.Modify("test-playbook")

	// Verify destination path has the prefix
	module := fullTask["template"].(map[string]interface{})
	expectedDest := "output/etc/app/config.conf"
	if module["dest"] != expectedDest {
		t.Errorf("Destination not modified correctly: got %v, want %v", module["dest"], expectedDest)
	}

	// Verify tags added
	tags, ok := fullTask["tags"].([]interface{})
	if !ok {
		t.Errorf("Tags were not added")
	} else if len(tags) != 1 || tags[0] != "render_config" {
		t.Errorf("Tags not added correctly: %v", tags)
	}

	// Verify delegation settings
	delegateTo, ok := fullTask["delegate_to"].(string)
	if !ok || delegateTo != "localhost" {
		t.Errorf("delegate_to not set correctly: %v", fullTask["delegate_to"])
	}

	runOnce, ok := fullTask["run_once"].(bool)
	if !ok || !runOnce {
		t.Errorf("run_once not set correctly: %v", fullTask["run_once"])
	}

	// Verify notify removed
	if _, hasNotify := fullTask["notify"]; hasNotify {
		t.Errorf("notify was not removed")
	}
}

func TestModifyTemplateTask(t *testing.T) {
	// Create a copy of the task to verify it's not modified when not a template
	nonTemplateTask := map[string]interface{}{
		"copy": map[string]interface{}{"src": "file", "dest": "/etc/file"},
	}
	nonTemplateCopy := make(map[string]interface{})
	for k, v := range nonTemplateTask {
		nonTemplateCopy[k] = v
	}

	// Test with non-template task - Added "test-playbook" as playbookName argument
	ModifyTemplateTask(nonTemplateTask, "test-playbook")
	if !reflect.DeepEqual(nonTemplateTask, nonTemplateCopy) {
		t.Errorf("ModifyTemplateTask() should not modify non-template tasks")
	}

	// Test with template task
	templateTask := map[string]interface{}{
		"template": map[string]interface{}{
			"src":  "app.conf.j2",
			"dest": "/etc/app/config.conf",
		},
		"notify": []interface{}{"restart app"},
	}

	// Modify the task - Added "test-playbook" as playbookName argument
	ModifyTemplateTask(templateTask, "test-playbook")

	// Verify task was modified
	module := templateTask["template"].(map[string]interface{})
	destPath := module["dest"].(string)
	if !strings.HasPrefix(destPath, "output/") {
		t.Errorf("ModifyTemplateTask() did not modify the destination path correctly: %s", destPath)
	}

	if _, hasNotify := templateTask["notify"]; hasNotify {
		t.Errorf("ModifyTemplateTask() did not remove notify field")
	}
}

func TestHasRenderConfigTag(t *testing.T) {
	tests := []struct {
		name     string
		tags     []interface{}
		expected bool
	}{
		{
			name:     "empty tags",
			tags:     []interface{}{},
			expected: false,
		},
		{
			name:     "has render_config",
			tags:     []interface{}{"setup", "render_config", "config"},
			expected: true,
		},
		{
			name:     "no render_config",
			tags:     []interface{}{"setup", "config"},
			expected: false,
		},
		{
			name:     "non-string tags",
			tags:     []interface{}{123, true, "config"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasRenderConfigTag(tt.tags)
			if result != tt.expected {
				t.Errorf("hasRenderConfigTag() = %v, want %v", result, tt.expected)
			}
		})
	}
}
