package executor

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/zinrai/ansible-template-render/internal/logger"
)

// Runs an Ansible playbook
func RunAnsible(playbookPath, outputDir string) error {
	// Ensure the output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	// Define command arguments
	templatePrefix := fmt.Sprintf("template_dest_prefix=%s", outputDir)
	args := []string{
		playbookPath,
		"--tags", "render_config",
		"-e", templatePrefix,
	}

	// Build the command
	cmd := exec.Command("ansible-playbook", args...)

	// Use standard output and error
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Build the command string for logging
	// Escape the template_dest_prefix value for proper logging
	loggedTemplatePrefix := fmt.Sprintf("\"template_dest_prefix=%s\"", outputDir)
	logArgs := []string{
		playbookPath,
		"--tags", "render_config",
		"-e", loggedTemplatePrefix,
	}
	cmdString := "ansible-playbook " + strings.Join(logArgs, " ")

	logger.Info("Executing Ansible command", "command", cmdString)

	return cmd.Run()
}
