package executor

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/zinrai/ansible-template-render/internal/logger"
)

// Runs an Ansible playbook
func RunAnsible(playbookPath, outputDir string) error {
	// Ensure the output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	// Build the command line
	cmd := exec.Command(
		"ansible-playbook",
		playbookPath,
		"--tags", "render_config",
		"-e", fmt.Sprintf("template_dest_prefix=%s", outputDir),
	)

	// Use standard output and error
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	logger.Info("Executing Ansible command",
		"command", fmt.Sprintf("ansible-playbook %s --tags render_config -e \"template_dest_prefix=%s\"",
			playbookPath, outputDir))

	return cmd.Run()
}
