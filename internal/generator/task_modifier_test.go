package generator

import (
	"reflect"
	"strings"
	"testing"

	"github.com/zinrai/ansible-template-render/internal/ansible"
)

func TestDirectoryTask_ToMap(t *testing.T) {
	tests := []struct {
		name            string
		destPath        string
		expectedPath    string
		expectedName    string
		expectedState   string
		expectedTags    []interface{}
		expectedRunOnce bool
	}{
		{
			name:            "simple path",
			destPath:        "/etc/app.conf",
			expectedPath:    "{{ template_dest_prefix | default('') }}/etc",
			expectedName:    "Ensure directory exists for /etc/app.conf",
			expectedState:   "directory",
			expectedTags:    []interface{}{"render_config"},
			expectedRunOnce: true,
		},
		{
			name:            "nested path",
			destPath:        "/var/lib/app/data/config.dat",
			expectedPath:    "{{ template_dest_prefix | default('') }}/var/lib/app/data",
			expectedName:    "Ensure directory exists for /var/lib/app/data/config.dat",
			expectedState:   "directory",
			expectedTags:    []interface{}{"render_config"},
			expectedRunOnce: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dirTask := NewDirectoryTask(tt.destPath)
			result := dirTask.ToMap()

			// Check task name
			name, ok := result["name"].(string)
			if !ok || name != tt.expectedName {
				t.Errorf("ToMap().name = %v, want %v", result["name"], tt.expectedName)
			}

			// Check file module
			file, ok := result["file"].(map[string]interface{})
			if !ok {
				t.Fatalf("ToMap() did not include file module")
			}

			// Check path
			path, ok := file["path"].(string)
			if !ok || path != tt.expectedPath {
				t.Errorf("ToMap().file.path = %v, want %v", file["path"], tt.expectedPath)
			}

			// Check state
			state, ok := file["state"].(string)
			if !ok || state != tt.expectedState {
				t.Errorf("ToMap().file.state = %v, want %v", file["state"], tt.expectedState)
			}

			// Check tags
			tags, ok := result["tags"].([]interface{})
			if !ok || !reflect.DeepEqual(tags, tt.expectedTags) {
				t.Errorf("ToMap().tags = %v, want %v", result["tags"], tt.expectedTags)
			}

			// Check run_once
			runOnce, ok := result["run_once"].(bool)
			if !ok || runOnce != tt.expectedRunOnce {
				t.Errorf("ToMap().run_once = %v, want %v", result["run_once"], tt.expectedRunOnce)
			}

			// Check delegate_to
			delegateTo, ok := result["delegate_to"].(string)
			if !ok || delegateTo != "localhost" {
				t.Errorf("ToMap().delegate_to = %v, want %v", result["delegate_to"], "localhost")
			}
		})
	}
}

func TestProcessTemplateTasks(t *testing.T) {
	tests := []struct {
		name                 string
		tasks                []map[string]interface{}
		expectedModified     bool
		expectedHasTemplates bool
		expectedTaskCount    int
		expectDirectoryTask  bool
	}{
		{
			name: "single template task",
			tasks: []map[string]interface{}{
				{
					"name": "Template task",
					"template": map[string]interface{}{
						"src":  "app.conf.j2",
						"dest": "/etc/app/app.conf",
					},
					"notify": []interface{}{"restart app"},
				},
			},
			expectedModified:     true,
			expectedHasTemplates: true,
			expectedTaskCount:    2, // Directory task + modified template task
			expectDirectoryTask:  true,
		},
		{
			name: "multiple template tasks with same directory",
			tasks: []map[string]interface{}{
				{
					"template": map[string]interface{}{
						"src":  "app1.conf.j2",
						"dest": "/etc/app/app1.conf",
					},
				},
				{
					"template": map[string]interface{}{
						"src":  "app2.conf.j2",
						"dest": "/etc/app/app2.conf",
					},
				},
			},
			expectedModified:     true,
			expectedHasTemplates: true,
			expectedTaskCount:    3, // One directory task + two template tasks
			expectDirectoryTask:  true,
		},
		{
			name: "no template tasks",
			tasks: []map[string]interface{}{
				{
					"name": "Copy task",
					"copy": map[string]interface{}{
						"src":  "file",
						"dest": "/etc/file",
					},
				},
			},
			expectedModified:     false,
			expectedHasTemplates: false,
			expectedTaskCount:    1, // Original task only
			expectDirectoryTask:  false,
		},
		{
			name: "mixed tasks",
			tasks: []map[string]interface{}{
				{
					"name": "Copy task",
					"copy": map[string]interface{}{
						"src":  "file",
						"dest": "/etc/file",
					},
				},
				{
					"name": "Template task",
					"template": map[string]interface{}{
						"src":  "app.conf.j2",
						"dest": "/etc/app/app.conf",
					},
				},
			},
			expectedModified:     true,
			expectedHasTemplates: true,
			expectedTaskCount:    3, // Original task + directory task + modified template task
			expectDirectoryTask:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processTemplateTasks(tt.tasks, "test_file.yml")

			if result.Modified != tt.expectedModified {
				t.Errorf("processTemplateTasks().Modified = %v, want %v", result.Modified, tt.expectedModified)
			}

			if result.HasTemplates != tt.expectedHasTemplates {
				t.Errorf("processTemplateTasks().HasTemplates = %v, want %v", result.HasTemplates, tt.expectedHasTemplates)
			}

			if len(result.Tasks) != tt.expectedTaskCount {
				t.Errorf("processTemplateTasks() returned %d tasks, want %d", len(result.Tasks), tt.expectedTaskCount)
			}

			// Check if directory task is present when expected
			if tt.expectDirectoryTask {
				foundDirTask := false
				for _, task := range result.Tasks {
					name, ok := task["name"].(string)
					if ok && strings.Contains(name, "Ensure directory exists") {
						foundDirTask = true
						break
					}
				}

				if !foundDirTask {
					t.Errorf("processTemplateTasks() did not include directory task")
				}
			}

			// If tasks contain templates, verify they are modified
			if tt.expectedHasTemplates {
				for _, task := range result.Tasks {
					if ansible.IsTemplateTask(task) {
						// Check if template dest includes template_dest_prefix
						templateTask, _ := ansible.NewTemplateTask(task)
						destPath := templateTask.GetDestPath()
						if !strings.Contains(destPath, "template_dest_prefix") {
							t.Errorf("Template task not modified: %v", destPath)
						}

						// Check notify removed
						if _, hasNotify := task["notify"]; hasNotify {
							t.Errorf("notify not removed from template task")
						}
					}
				}
			}
		})
	}
}

func TestGetTemplateDestPath(t *testing.T) {
	tests := []struct {
		name     string
		task     map[string]interface{}
		expected string
	}{
		{
			name: "standard template",
			task: map[string]interface{}{
				"template": map[string]interface{}{
					"src":  "app.conf.j2",
					"dest": "/etc/app/app.conf",
				},
			},
			expected: "/etc/app/app.conf",
		},
		{
			name: "builtin template",
			task: map[string]interface{}{
				"ansible.builtin.template": map[string]interface{}{
					"src":  "app.conf.j2",
					"dest": "/etc/app/app.conf",
				},
			},
			expected: "/etc/app/app.conf",
		},
		{
			name: "not a template",
			task: map[string]interface{}{
				"copy": map[string]interface{}{
					"src":  "file",
					"dest": "/etc/file",
				},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getTemplateDestPath(tt.task)
			if result != tt.expected {
				t.Errorf("getTemplateDestPath() = %v, want %v", result, tt.expected)
			}
		})
	}
}
