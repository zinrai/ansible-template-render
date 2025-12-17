package processor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/zinrai/ansible-template-render/internal/logger"

	"gopkg.in/yaml.v3"
)

// ModifyInventoryForLocalExecution converts any inventory (INI, YAML, dynamic)
// to a static YAML inventory with ansible_connection=local for all hosts
func ModifyInventoryForLocalExecution(inventoryPath, destDir string) (string, error) {
	// 1. Run ansible-inventory to get YAML output
	output, err := runAnsibleInventory(inventoryPath)
	if err != nil {
		return "", fmt.Errorf("running ansible-inventory: %w", err)
	}

	// 2. Parse YAML
	var inventory map[string]interface{}
	if err := yaml.Unmarshal(output, &inventory); err != nil {
		return "", fmt.Errorf("parsing inventory yaml: %w", err)
	}

	// 3. Inject ansible_connection: local to all hosts
	injectLocalConnection(inventory)

	// 4. Write YAML file
	destPath := filepath.Join(destDir, "inventory.yaml")
	data, err := yaml.Marshal(inventory)
	if err != nil {
		return "", fmt.Errorf("marshaling inventory yaml: %w", err)
	}
	if err := os.WriteFile(destPath, data, 0644); err != nil {
		return "", fmt.Errorf("writing inventory file: %w", err)
	}

	logger.Debug("Converted inventory for local execution", "path", destPath)
	return destPath, nil
}

// runAnsibleInventory executes ansible-inventory command and returns YAML output
func runAnsibleInventory(inventoryPath string) ([]byte, error) {
	cmd := exec.Command("ansible-inventory", "-i", inventoryPath, "--list", "--yaml")
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("ansible-inventory failed: %s", string(exitErr.Stderr))
		}
		return nil, err
	}
	return output, nil
}

// injectLocalConnection recursively adds ansible_connection: local to all hosts
func injectLocalConnection(data map[string]interface{}) {
	for _, value := range data {
		group, ok := value.(map[string]interface{})
		if !ok {
			continue
		}

		// Process hosts if present
		if hosts, ok := group["hosts"].(map[string]interface{}); ok {
			for hostName, hostVars := range hosts {
				if vars, ok := hostVars.(map[string]interface{}); ok {
					vars["ansible_connection"] = "local"
				} else {
					// hostVars is nil or not a map
					hosts[hostName] = map[string]interface{}{
						"ansible_connection": "local",
					}
				}
			}
		}

		// Recursively process children
		if children, ok := group["children"].(map[string]interface{}); ok {
			injectLocalConnection(children)
		}
	}
}
