package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Copies a file
func CopyFile(src, dst string) error {
	// Ensure the output directory exists
	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	// Open the source file
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening source file: %w", err)
	}
	defer srcFile.Close()

	// Create the destination file
	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("creating destination file: %w", err)
	}
	defer dstFile.Close()

	// Copy
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("copying file: %w", err)
	}

	return nil
}

// Ensures a directory exists, creating it if necessary
func EnsureDirectory(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("creating directory: %w", err)
		}
	}
	return nil
}

// Removes a directory (doesn't error if it doesn't exist)
func CleanupDirectory(path string) error {
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("removing directory: %w", err)
		}
	}
	return nil
}
