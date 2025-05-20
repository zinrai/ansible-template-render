package copier

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/zinrai/ansible-template-render/internal/utils"
)

// Handles copying playbook files
type PlaybookCopier struct{}

// Copies a playbook file to the destination directory
func (c *PlaybookCopier) CopyPlaybook(playbookPath, destDir string) (string, error) {
	destPlaybookPath := filepath.Join(destDir, filepath.Base(playbookPath))

	// Create destination directory if needed
	destDirPath := filepath.Dir(destPlaybookPath)
	if err := os.MkdirAll(destDirPath, 0755); err != nil {
		return "", fmt.Errorf("creating destination directory: %w", err)
	}

	// Copy the playbook file
	if err := utils.CopyFile(playbookPath, destPlaybookPath); err != nil {
		return "", fmt.Errorf("copying playbook file: %w", err)
	}

	return destPlaybookPath, nil
}
