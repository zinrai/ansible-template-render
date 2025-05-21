package copier

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/zinrai/ansible-template-render/internal/finder"
	"github.com/zinrai/ansible-template-render/internal/logger"
	"github.com/zinrai/ansible-template-render/internal/utils"
)

// Copies vars directories to the temporary directory
func CopyVarsDirectories(varsDirectories finder.VarsDirectories, destDir string) error {
	if varsDirectories.GroupVars != "" {
		destGroupVars := filepath.Join(destDir, "group_vars")
		if err := copyDir(varsDirectories.GroupVars, destGroupVars); err != nil {
			return fmt.Errorf("copying group_vars directory: %w", err)
		}
		logger.Info("Copied group_vars directory", "from", varsDirectories.GroupVars, "to", destGroupVars)
	}

	if varsDirectories.HostVars != "" {
		destHostVars := filepath.Join(destDir, "host_vars")
		if err := copyDir(varsDirectories.HostVars, destHostVars); err != nil {
			return fmt.Errorf("copying host_vars directory: %w", err)
		}
		logger.Info("Copied host_vars directory", "from", varsDirectories.HostVars, "to", destHostVars)
	}

	return nil
}

// Recursively copies a directory
func copyDir(src, dst string) error {
	// Create destination directory
	if err := os.MkdirAll(dst, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dst, err)
	}

	// Read entries from source directory
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("reading directory %s: %w", src, err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			// Recursively copy subdirectories
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			// Copy files
			if err := utils.CopyFile(srcPath, dstPath); err != nil {
				return fmt.Errorf("copying file %s: %w", srcPath, err)
			}
		}
	}

	return nil
}
