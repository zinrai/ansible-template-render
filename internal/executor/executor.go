package executor

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/zinrai/ansible-template-render/internal/logger"
)

// Represents the environment for Ansible execution
type ExecutionEnvironment struct {
	WorkingDir        string // Working directory
	PlaybookPath      string // Path to the playbook (relative path)
	InventoryPath     string // Path to the inventory (relative path)
	AnsibleConfigPath string // Path to the ansible.cfg file (absolute path recommended)
}

// Executes an Ansible playbook
func RunAnsible(env ExecutionEnvironment) error {
	// Save the original directory
	originalDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting current directory: %w", err)
	}

	// Change to the working directory (restore when done)
	if err := os.Chdir(env.WorkingDir); err != nil {
		return fmt.Errorf("changing to working directory: %w", err)
	}
	defer os.Chdir(originalDir)

	// Set the ANSIBLE_CONFIG environment variable
	if env.AnsibleConfigPath != "" {
		os.Setenv("ANSIBLE_CONFIG", env.AnsibleConfigPath)
	}

	// Build command line arguments
	args := []string{
		env.PlaybookPath,
		"--tags", "render_config",
	}

	// Add inventory path if specified
	if env.InventoryPath != "" {
		args = append(args, "-i", env.InventoryPath)
	}

	// Create command
	cmd := exec.Command("ansible-playbook", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Command string for logging
	cmdString := fmt.Sprintf("ansible-playbook %s --tags render_config", env.PlaybookPath)
	if env.InventoryPath != "" {
		cmdString += fmt.Sprintf(" -i %s", env.InventoryPath)
	}
	logger.Info("Executing Ansible command", "command", cmdString)

	return cmd.Run()
}
