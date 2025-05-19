package ansible

import (
	"fmt"
)

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
	// Find template module and key
	moduleKey, templateModule := findTemplateModule(task)
	if moduleKey == "" || templateModule == nil {
		return
	}

	// Modify destination path
	modifyDestinationPath(task, moduleKey, templateModule)

	// Add render_config tag
	ensureRenderConfigTag(task)

	// Add delegation settings
	task["delegate_to"] = "localhost"
	task["run_once"] = true

	// Remove notify field - not needed for configuration rendering
	delete(task, "notify")
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

// Adds template_dest_prefix to the destination path
func modifyDestinationPath(task map[string]interface{}, moduleKey string, templateModule map[string]interface{}) {
	destPath, ok := templateModule["dest"].(string)
	if !ok {
		return
	}

	templateModule["dest"] = fmt.Sprintf("{{ template_dest_prefix | default('') }}%s", destPath)
	task[moduleKey] = templateModule
}

// Ensures the render_config tag is present
func ensureRenderConfigTag(task map[string]interface{}) {
	existingTags, ok := task["tags"].([]interface{})
	if !ok {
		// No tags exist, create new tags array
		task["tags"] = []interface{}{"render_config"}
		return
	}

	// Copy existing tags
	tags := make([]interface{}, len(existingTags))
	copy(tags, existingTags)

	// Check if render_config tag already exists
	if hasRenderConfigTag(tags) {
		task["tags"] = tags
		return
	}

	// Add render_config tag
	task["tags"] = append(tags, "render_config")
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
