package copier

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/zinrai/ansible-template-render/internal/logger"
	"github.com/zinrai/ansible-template-render/internal/utils"
)

// Handles copying role structures
type RoleCopier struct{}

// Copies a role's directory structure to the destination directory
func (c *RoleCopier) CopyRole(roleName string, destDir string) error {
	srcRolePath := filepath.Join("roles", roleName)
	destRolePath := filepath.Join(destDir, "roles", roleName)

	// Check if source role directory exists
	_, err := os.Stat(srcRolePath)
	if os.IsNotExist(err) {
		return fmt.Errorf("role directory not found: %s", srcRolePath)
	}

	// Create destination role directory
	if err := os.MkdirAll(destRolePath, 0755); err != nil {
		return fmt.Errorf("creating role directory: %w", err)
	}

	// Copy role directory contents recursively
	return c.copyRoleContents(srcRolePath, destRolePath)
}

// Recursively copies the role contents
func (c *RoleCopier) copyRoleContents(srcPath, destPath string) error {
	return filepath.Walk(srcPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip if same as source directory
		if path == srcPath {
			return nil
		}

		// Calculate relative path
		relPath, err := filepath.Rel(srcPath, path)
		if err != nil {
			return fmt.Errorf("calculating relative path: %w", err)
		}

		destItemPath := filepath.Join(destPath, relPath)

		if info.IsDir() {
			// Create directory
			if err := os.MkdirAll(destItemPath, 0755); err != nil {
				return fmt.Errorf("creating directory %s: %w", destItemPath, err)
			}
			return nil
		}

		// Copy file
		return utils.CopyFile(path, destItemPath)
	})
}

// Copies all specified roles to the destination directory
func CopyAllRoles(roles []string, destDir string) error {
	copier := &RoleCopier{}

	for _, roleName := range roles {
		logger.Info("Copying role", "name", roleName)

		err := copier.CopyRole(roleName, destDir)
		if err != nil {
			logger.Warn("Error copying role", "role", roleName, "error", err)
			// Continue despite errors, so we copy as many roles as possible
		}
	}

	return nil
}
