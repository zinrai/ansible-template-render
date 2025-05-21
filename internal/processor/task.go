package processor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zinrai/ansible-template-render/internal/ansible"
	"github.com/zinrai/ansible-template-render/internal/finder"
	"github.com/zinrai/ansible-template-render/internal/logger"
	"github.com/zinrai/ansible-template-render/internal/utils"
)

// Processes role tasks
type TaskProcessor struct{}

// Represents a directory creation task
type DirectoryTask struct {
	DestPath     string
	PlaybookName string
}

// Creates a new directory task
func NewDirectoryTask(destPath string, playbookName string) *DirectoryTask {
	return &DirectoryTask{
		DestPath:     destPath,
		PlaybookName: playbookName,
	}
}

// Converts the directory task to a map representation
func (d *DirectoryTask) ToMap() map[string]interface{} {
	// Create the full output path
	outputPath := filepath.Join("output", d.DestPath)
	dirPath := filepath.Dir(outputPath)

	return map[string]interface{}{
		"name": fmt.Sprintf("Ensure directory exists for %s", outputPath),
		"file": map[string]interface{}{
			"path":  dirPath,
			"state": "directory",
			"mode":  "0755",
		},
		"delegate_to": "localhost",
		"run_once":    true,
		"tags":        []interface{}{"render_config"},
	}
}

// Represents the result of processing tasks
type ProcessResult struct {
	Tasks        []map[string]interface{}
	Modified     bool
	HasTemplates bool
}

// Processes template tasks, inserting directory creation tasks and modifying templates
func ProcessTemplateTasks(tasks []map[string]interface{}, taskFile string, playbookName string) ProcessResult {
	var result []map[string]interface{}
	modified := false
	hasTemplates := false
	processedDirs := make(map[string]bool) // Track processed directories to avoid duplicates

	for _, task := range tasks {
		if ansible.IsTemplateTask(task) {
			// Handle template task
			taskResult, dirModified := handleTemplateTask(task, processedDirs, taskFile, playbookName)
			result = append(result, taskResult...)

			modified = true
			hasTemplates = true
			if dirModified {
				modified = true
			}
		} else {
			// Non-template task, add as is
			result = append(result, task)
		}
	}

	return ProcessResult{
		Tasks:        result,
		Modified:     modified,
		HasTemplates: hasTemplates,
	}
}

// Processes a single template task
func handleTemplateTask(task map[string]interface{}, processedDirs map[string]bool, taskFile string, playbookName string) ([]map[string]interface{}, bool) {
	var result []map[string]interface{}
	dirModified := false

	// Convert to TemplateTask object
	templateTask, _ := ansible.NewTemplateTask(task)

	// Add directory task if needed
	dirTask := createDirectoryTaskIfNeeded(templateTask, processedDirs, playbookName)
	if dirTask != nil {
		result = append(result, dirTask)
		dirModified = true
	}

	// Copy and modify template task
	modifiedTask := copyAndModifyTemplateTask(task, playbookName)
	result = append(result, modifiedTask)

	logger.Info("Modified template task", "file", taskFile)

	return result, dirModified
}

// Creates a directory task if needed
func createDirectoryTaskIfNeeded(templateTask *ansible.TemplateTask, processedDirs map[string]bool, playbookName string) map[string]interface{} {
	destPath := templateTask.GetDestPath()
	if destPath == "" {
		return nil
	}

	// Create the full output path
	outputPath := filepath.Join("output", destPath)
	dirPath := filepath.Dir(outputPath)

	if processedDirs[dirPath] {
		return nil
	}

	processedDirs[dirPath] = true
	return NewDirectoryTask(destPath, playbookName).ToMap()
}

// Creates a modified copy of a template task
func copyAndModifyTemplateTask(task map[string]interface{}, playbookName string) map[string]interface{} {
	taskCopy, err := utils.DeepCopy(task)
	if err != nil {
		logger.Warn("Error copying task", "error", err)
		return task // Use original if copying fails
	}

	// Modify the template task with playbook name
	ansible.ModifyTemplateTask(taskCopy.(map[string]interface{}), playbookName)
	return taskCopy.(map[string]interface{})
}

// Gets the destination path from a template task
func getTemplateDestPath(task map[string]interface{}) string {
	templateTask, isTemplate := ansible.NewTemplateTask(task)
	if !isTemplate {
		return ""
	}

	return templateTask.GetDestPath()
}

// Processes all tasks in a role, looking for and modifying templates
func (p *TaskProcessor) ProcessRoleTasks(roleName, tempDir string, playbookName string) (bool, error) {
	// Find task files
	taskFiles, err := finder.FindRoleTasks(roleName)
	if err != nil {
		// If tasks directory doesn't exist, it's not an error - just no templates
		if os.IsNotExist(err) || strings.Contains(err.Error(), "not found") {
			return false, nil
		}
		return false, err
	}

	// If no task files found, return without error
	if len(taskFiles) == 0 {
		return false, nil
	}

	hasTemplates := false

	// Process each task file
	for _, taskFile := range taskFiles {
		fileHasTemplates, err := p.ProcessTaskFile(taskFile, tempDir, playbookName)
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
func (p *TaskProcessor) ProcessTaskFile(taskFile, tempDir string, playbookName string) (bool, error) {
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

	// Process the tasks
	result := ProcessTemplateTasks(tasks, taskFile, playbookName)

	// No modifications needed, just copy the file
	if !result.Modified {
		err := utils.CopyFile(taskFile, tempTaskFile)
		if err != nil {
			return false, fmt.Errorf("copying task file: %w", err)
		}
		return result.HasTemplates, nil
	}

	// Save the modified tasks
	tempTaskDir := filepath.Dir(tempTaskFile)
	if err := os.MkdirAll(tempTaskDir, 0755); err != nil {
		return false, fmt.Errorf("creating temp task directory: %w", err)
	}

	if err := ansible.SaveTaskFile(result.Tasks, tempTaskFile); err != nil {
		return false, fmt.Errorf("saving modified task file: %w", err)
	}

	return result.HasTemplates, nil
}

// Processes tasks for all roles
func ProcessAllRoles(roles []string, tempDir string, playbookName string) (bool, error) {
	processor := &TaskProcessor{}
	hasTemplates := false

	for _, roleName := range roles {
		logger.Info("Processing role tasks", "name", roleName)

		roleHasTemplates, err := processor.ProcessRoleTasks(roleName, tempDir, playbookName)
		if err != nil {
			logger.Warn("Error processing role tasks", "role", roleName, "error", err)
			continue // Skip to next role
		}

		if roleHasTemplates {
			hasTemplates = true
		}
	}

	return hasTemplates, nil
}
