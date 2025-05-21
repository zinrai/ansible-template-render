package copier

import (
	"fmt"
	"path/filepath"

	"github.com/zinrai/ansible-template-render/internal/utils"
)

// Copies an inventory file to the destination directory
func CopyInventory(inventoryPath, destDir string) (string, error) {
	destInventoryPath := filepath.Join(destDir, filepath.Base(inventoryPath))

	// Copy the inventory file
	if err := utils.CopyFile(inventoryPath, destInventoryPath); err != nil {
		return "", fmt.Errorf("copying inventory file: %w", err)
	}

	return destInventoryPath, nil
}
