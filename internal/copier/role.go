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
	logger.Debug("Walking directory", "path", srcPath)

	// Check if directory exists (follow symlinks)
	if err := c.validateSourceDirectory(srcPath); err != nil {
		return err
	}

	// Get directory entries
	entries, err := os.ReadDir(srcPath)
	if err != nil {
		return fmt.Errorf("reading directory %s: %w", srcPath, err)
	}

	logger.Debug("Directory entries", "path", srcPath, "count", len(entries))

	// Process directories first
	if err := c.copyDirectories(srcPath, destPath, entries); err != nil {
		return err
	}

	// Then process files
	if err := c.copyFiles(srcPath, destPath, entries); err != nil {
		return err
	}

	return nil
}

// Validates that the source path is a directory
func (c *RoleCopier) validateSourceDirectory(srcPath string) error {
	srcInfo, err := os.Stat(srcPath) // Stat follows symlinks
	if err != nil {
		return fmt.Errorf("checking source path %s: %w", srcPath, err)
	}

	if !srcInfo.IsDir() {
		return fmt.Errorf("source path is not a directory: %s", srcPath)
	}

	return nil
}

// Copies directories recursively
func (c *RoleCopier) copyDirectories(srcPath, destPath string, entries []os.DirEntry) error {
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		entryPath := filepath.Join(srcPath, entry.Name())
		destEntryPath := filepath.Join(destPath, entry.Name())

		// Create the directory
		if err := os.MkdirAll(destEntryPath, 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", destEntryPath, err)
		}

		// Recursively copy subdirectory
		if err := c.copyRoleContents(entryPath, destEntryPath); err != nil {
			return err
		}
	}

	return nil
}

// Copies files
func (c *RoleCopier) copyFiles(srcPath, destPath string, entries []os.DirEntry) error {
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		entryPath := filepath.Join(srcPath, entry.Name())
		destEntryPath := filepath.Join(destPath, entry.Name())

		// Handle regular files and symlinks
		if err := c.copyFileWithLogging(entryPath, destEntryPath); err != nil {
			// Log but continue with other files
			logger.Warn("Error copying file", "src", entryPath, "dest", destEntryPath, "error", err)
		}
	}

	return nil
}

// Copies a file with proper error handling and logging
func (c *RoleCopier) copyFileWithLogging(srcPath, destPath string) error {
	// Get file info (following symlinks)
	fileInfo, err := os.Stat(srcPath)
	if err != nil {
		return fmt.Errorf("getting file info: %w", err)
	}

	if fileInfo.IsDir() {
		// This shouldn't happen but check anyway
		return fmt.Errorf("expected file but found directory: %s", srcPath)
	}

	logger.Debug("Copying file", "src", srcPath, "dest", destPath)
	if err := utils.CopyFile(srcPath, destPath); err != nil {
		return fmt.Errorf("copying file: %w", err)
	}

	return nil
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
