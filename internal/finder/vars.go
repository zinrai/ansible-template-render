package finder

import (
	"os"
	"path/filepath"
)

// Holds paths to group_vars and host_vars directories
type VarsDirectories struct {
	GroupVars string
	HostVars  string
}

// Searches for group_vars and host_vars directories
func FindVarsDirectories(playbookPath, inventoryPath string) VarsDirectories {
	var result VarsDirectories

	// Check in current directory root level
	if dirExists("group_vars") {
		result.GroupVars = "group_vars"
	}

	if dirExists("host_vars") {
		result.HostVars = "host_vars"
	}

	// Check in the same directory as the playbook
	playbookDir := filepath.Dir(playbookPath)
	if playbookDir != "." {
		checkDirInLocation(playbookDir, "group_vars", &result.GroupVars)
		checkDirInLocation(playbookDir, "host_vars", &result.HostVars)
	}

	// Check in the same directory as the inventory
	inventoryDir := filepath.Dir(inventoryPath)
	if inventoryDir != "." && inventoryDir != playbookDir {
		checkDirInLocation(inventoryDir, "group_vars", &result.GroupVars)
		checkDirInLocation(inventoryDir, "host_vars", &result.HostVars)
	}

	return result
}

// Checks if a directory exists
func dirExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

// Checks for a directory in a specific location
// and updates the result variable if found and not already set
func checkDirInLocation(location, dirName string, result *string) {
	if *result != "" {
		return // Skip if already found
	}

	path := filepath.Join(location, dirName)
	if dirExists(path) {
		*result = path
	}
}
