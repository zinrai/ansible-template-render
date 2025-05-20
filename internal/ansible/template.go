package ansible

import (
	"fmt"
)

// Represents a task using the template module
type TemplateTask struct {
	Task       map[string]interface{}
	ModuleKey  string
	ModuleData map[string]interface{}
}

// Creates a new TemplateTask from a map
func NewTemplateTask(task map[string]interface{}) (*TemplateTask, bool) {
	moduleKey, moduleData := findTemplateModule(task)
	if moduleKey == "" || moduleData == nil {
		return nil, false
	}

	return &TemplateTask{
		Task:       task,
		ModuleKey:  moduleKey,
		ModuleData: moduleData,
	}, true
}

// Returns the destination path of the template
func (t *TemplateTask) GetDestPath() string {
	destPath, ok := t.ModuleData["dest"].(string)
	if !ok {
		return ""
	}
	return destPath
}

// Modifies the template task for rendering
func (t *TemplateTask) Modify() {
	// Modify destination path
	t.modifyDestinationPath()

	// Add render_config tag
	t.ensureRenderConfigTag()

	// Add delegation settings
	t.Task["delegate_to"] = "localhost"
	t.Task["run_once"] = true

	// Remove notify field - not needed for configuration rendering
	delete(t.Task, "notify")
}

// Adds template_dest_prefix to the destination path
func (t *TemplateTask) modifyDestinationPath() {
	destPath, ok := t.ModuleData["dest"].(string)
	if !ok {
		return
	}

	t.ModuleData["dest"] = fmt.Sprintf("{{ template_dest_prefix | default('') }}%s", destPath)
	t.Task[t.ModuleKey] = t.ModuleData
}

// Ensures the render_config tag is present
func (t *TemplateTask) ensureRenderConfigTag() {
	existingTags, ok := t.Task["tags"].([]interface{})
	if !ok {
		// No tags exist, create new tags array
		t.Task["tags"] = []interface{}{"render_config"}
		return
	}

	// Copy existing tags
	tags := make([]interface{}, len(existingTags))
	copy(tags, existingTags)

	// Check if render_config tag already exists
	if hasRenderConfigTag(tags) {
		t.Task["tags"] = tags
		return
	}

	// Add render_config tag
	t.Task["tags"] = append(tags, "render_config")
}

// Determines if a task uses the template module
func IsTemplateTask(task map[string]interface{}) bool {
	_, hasTemplate := task["template"]
	_, hasBuiltinTemplate := task["ansible.builtin.template"]
	return hasTemplate || hasBuiltinTemplate
}

// Modifies a template task by:
// - Adding template_dest_prefix to destination path
// - Adding render_config tag
// - Setting delegate_to: localhost and run_once: true
// - Removing notify handlers (not needed for rendering)
func ModifyTemplateTask(task map[string]interface{}) {
	templateTask, isTemplate := NewTemplateTask(task)
	if !isTemplate {
		return
	}

	templateTask.Modify()
}

// Identifies the template module and its key
func findTemplateModule(task map[string]interface{}) (string, map[string]interface{}) {
	if module, ok := task["template"].(map[string]interface{}); ok {
		return "template", module
	}

	if module, ok := task["ansible.builtin.template"].(map[string]interface{}); ok {
		return "ansible.builtin.template", module
	}

	return "", nil
}

// Checks if render_config tag exists in tags array
func hasRenderConfigTag(tags []interface{}) bool {
	for _, tag := range tags {
		tagStr, ok := tag.(string)
		if ok && tagStr == "render_config" {
			return true
		}
	}
	return false
}
