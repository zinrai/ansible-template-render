package generator

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/zinrai/ansible-template-render/internal/ansible"
	"github.com/zinrai/ansible-template-render/internal/finder"
	"github.com/zinrai/ansible-template-render/internal/logger"
	"github.com/zinrai/ansible-template-render/internal/utils"
)

// Processes the tasks of a role
func ProcessRoleTasks(roleName, tempDir string) (bool, error) {
	// Find the role's task files
	taskFiles, err := finder.FindRoleTasks(roleName)
	if err != nil {
		return false, fmt.Errorf("finding role tasks: %w", err)
	}

	hasTemplates := false

	// Process each task file
	for _, taskFile := range taskFiles {
		fileHasTemplates, err := processTaskFile(taskFile, tempDir)
		if err != nil {
			return false, err
		}

		if fileHasTemplates {
			hasTemplates = true
		}
	}

	return hasTemplates, nil
}

// Processes a single task file
func processTaskFile(taskFile, tempDir string) (bool, error) {
	// Create the corresponding path in the temporary directory
	relPath, err := filepath.Rel(".", taskFile)
	if err != nil {
		return false, fmt.Errorf("getting relative path: %w", err)
	}

	tempTaskFile := filepath.Join(tempDir, relPath)

	// Load the task file
	tasks, err := ansible.LoadTaskFile(taskFile)
	if err != nil {
		return false, fmt.Errorf("loading task file %s: %w", taskFile, err)
	}

	// Check template tasks and insert directory creation tasks
	tasks, modified, hasTemplates := processTemplateTasks(tasks, taskFile)

	// No modifications needed, just copy the file
	if !modified {
		err := utils.CopyFile(taskFile, tempTaskFile)
		if err != nil {
			return false, fmt.Errorf("copying task file: %w", err)
		}
		return hasTemplates, nil
	}

	// Save the modified tasks
	tempTaskDir := filepath.Dir(tempTaskFile)
	if err := os.MkdirAll(tempTaskDir, 0755); err != nil {
		return false, fmt.Errorf("creating temp task directory: %w", err)
	}

	if err := ansible.SaveTaskFile(tasks, tempTaskFile); err != nil {
		return false, fmt.Errorf("saving modified task file: %w", err)
	}

	return hasTemplates, nil
}

// Processes template tasks, inserting directory creation tasks and modifying templates
func processTemplateTasks(tasks []map[string]interface{}, taskFile string) ([]map[string]interface{}, bool, bool) {
	var result []map[string]interface{}
	modified := false
	hasTemplates := false
	processedDirs := make(map[string]bool) // Track processed directories to avoid duplicates

	for _, task := range tasks {
		if ansible.IsTemplateTask(task) {
			// Extract destination path
			destPath := getTemplateDestPath(task)

			if destPath != "" {
				// Create directory task for this template
				dirPath := filepath.Dir(destPath)
				if !processedDirs[dirPath] {
					dirTask := createDirectoryTask(destPath)
					result = append(result, dirTask)
					processedDirs[dirPath] = true
					modified = true
				}
			}

			// Deep copy and modify the template task
			taskCopy, err := utils.DeepCopy(task)
			if err != nil {
				logger.Warn("Error copying task", "error", err)
				result = append(result, task) // Use original if copying fails
				continue
			}

			// Modify the template task
			ansible.ModifyTemplateTask(taskCopy.(map[string]interface{}))
			result = append(result, taskCopy.(map[string]interface{}))

			modified = true
			hasTemplates = true
			logger.Info("Modified template task", "file", taskFile)
		} else {
			// Non-template task, add as is
			result = append(result, task)
		}
	}

	return result, modified, hasTemplates
}

// Creates a directory creation task for a template destination
func createDirectoryTask(destPath string) map[string]interface{} {
	dirPath := filepath.Dir(destPath)

	return map[string]interface{}{
		"name": fmt.Sprintf("Ensure directory exists for %s", destPath),
		"file": map[string]interface{}{
			"path":  fmt.Sprintf("{{ template_dest_prefix | default('') }}%s", dirPath),
			"state": "directory",
			"mode":  "0755",
		},
		"delegate_to": "localhost",
		"run_once":    true,
		"tags":        []interface{}{"render_config"},
	}
}

// Gets the destination path from a template task
func getTemplateDestPath(task map[string]interface{}) string {
	if template, ok := task["template"].(map[string]interface{}); ok {
		if dest, ok := template["dest"].(string); ok {
			return dest
		}
	}

	if template, ok := task["ansible.builtin.template"].(map[string]interface{}); ok {
		if dest, ok := template["dest"].(string); ok {
			return dest
		}
	}

	return ""
}

// Modifies template tasks and reports if modifications were made
func modifyTemplateTasks(tasks []map[string]interface{}, taskFile string) (bool, bool) {
	modified := false
	hasTemplates := false

	for i, task := range tasks {
		if !ansible.IsTemplateTask(task) {
			continue
		}

		// Deep copy the task
		taskCopy, err := utils.DeepCopy(task)
		if err != nil {
			logger.Warn("Error copying task", "error", err)
			continue
		}

		// Modify the copied task
		ansible.ModifyTemplateTask(taskCopy.(map[string]interface{}))
		tasks[i] = taskCopy.(map[string]interface{})

		modified = true
		hasTemplates = true

		logger.Info("Modified template task", "file", taskFile)
	}

	return modified, hasTemplates
}
