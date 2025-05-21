package executor

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/zinrai/ansible-template-render/internal/logger"
)

// Runs an Ansible playbook
func RunAnsible(playbookPath, outputDir string, inventoryPath string) error {
	// Ensure the output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	// Define command arguments
	args := []string{
		playbookPath,
		"--tags", "render_config",
		"-i", inventoryPath,
	}

	// Build the command
	cmd := exec.Command("ansible-playbook", args...)

	// Use standard output and error
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Build the command string for logging
	logArgs := []string{
		playbookPath,
		"--tags", "render_config",
		"-i", inventoryPath,
	}
	cmdString := "ansible-playbook " + strings.Join(logArgs, " ")

	logger.Info("Executing Ansible command", "command", cmdString)

	return cmd.Run()
}
