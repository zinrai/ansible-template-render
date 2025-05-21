package processor

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/zinrai/ansible-template-render/internal/logger"
)

// Modifies the inventory file to force local connection
// for all hosts by adding or replacing ansible_connection=local
func ModifyInventoryForLocalExecution(inventoryPath string) error {
	// Read the original file
	content, err := os.ReadFile(inventoryPath)
	if err != nil {
		return fmt.Errorf("reading inventory file: %w", err)
	}

	// Process the content
	modifiedContent := addLocalConnectionToHosts(string(content))

	// Create a temporary file in the same directory
	dir := filepath.Dir(inventoryPath)
	tempFile, err := os.CreateTemp(dir, "inventory-*.tmp")
	if err != nil {
		return fmt.Errorf("creating temporary file: %w", err)
	}
	tempFilePath := tempFile.Name()

	// Ensure the temp file is removed on function exit if an error occurs
	defer func() {
		if err != nil {
			os.Remove(tempFilePath)
		}
	}()

	// Write modified content to the temporary file
	_, err = tempFile.WriteString(modifiedContent)
	if err != nil {
		tempFile.Close()
		return fmt.Errorf("writing to temporary file: %w", err)
	}

	// Close the file before renaming
	if err = tempFile.Close(); err != nil {
		return fmt.Errorf("closing temporary file: %w", err)
	}

	// Replace the original file with the modified one
	if err = os.Rename(tempFilePath, inventoryPath); err != nil {
		return fmt.Errorf("replacing original inventory file: %w", err)
	}

	logger.Debug("Inventory file modified for local execution", "path", inventoryPath)
	return nil
}

// Regular expressions used for inventory processing
var (
	sectionRegex           = regexp.MustCompile(`^\s*\[(.*?)\]\s*$`)
	commentRegex           = regexp.MustCompile(`^\s*#.*$`)
	ansibleConnectionRegex = regexp.MustCompile(`(ansible_connection\s*=\s*)[^\s]+`)
)

// Processes the inventory content line by line
// and adds or replaces ansible_connection=local for all hosts in regular group sections
func addLocalConnectionToHosts(content string) string {
	scanner := bufio.NewScanner(strings.NewReader(content))
	var result strings.Builder
	var currentSection string

	for scanner.Scan() {
		line := scanner.Text()
		processedLine := processInventoryLine(line, &currentSection)
		result.WriteString(processedLine + "\n")
	}

	return result.String()
}

// Processes a single line from the inventory file
// and returns the processed line
func processInventoryLine(line string, currentSection *string) string {
	// Check if this is a section header
	if sectionMatch := sectionRegex.FindStringSubmatch(line); len(sectionMatch) > 1 {
		// Update current section tracking
		*currentSection = updateSectionTracking(sectionMatch[1])
		return line
	}

	// Skip empty lines and comments
	if isEmptyOrComment(line) {
		return line
	}

	// If not in a host group section, return unchanged
	if *currentSection == "" {
		return line
	}

	// We're in a host group section, so this should be a host line
	// Process it to ensure ansible_connection=local is present
	return modifyHostLine(line)
}

// Updates the current section tracking based on section name
// Returns the section name if it's a regular group, or empty string for special sections
func updateSectionTracking(sectionName string) string {
	if strings.Contains(sectionName, ":") {
		// This is a special section like [group:vars] or [group:children], not a regular host group
		return ""
	}
	return sectionName
}

// Checks if a line is empty or a comment
func isEmptyOrComment(line string) bool {
	return len(strings.TrimSpace(line)) == 0 || commentRegex.MatchString(line)
}

// Adds or replaces ansible_connection=local in a host line
func modifyHostLine(line string) string {
	if ansibleConnectionRegex.MatchString(line) {
		// Replace existing ansible_connection value with 'local'
		modifiedLine := ansibleConnectionRegex.ReplaceAllString(line, "${1}local")
		logger.Debug("Replaced ansible_connection in host", "line", line, "result", modifiedLine)
		return modifiedLine
	}

	// Add ansible_connection=local to the end
	modifiedLine := line + " ansible_connection=local"
	logger.Debug("Added ansible_connection=local to host", "line", line, "result", modifiedLine)
	return modifiedLine
}
