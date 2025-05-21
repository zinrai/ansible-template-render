package generator

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/zinrai/ansible-template-render/internal/ansible"
	"github.com/zinrai/ansible-template-render/internal/config"
	"github.com/zinrai/ansible-template-render/internal/copier"
	"github.com/zinrai/ansible-template-render/internal/finder"
	"github.com/zinrai/ansible-template-render/internal/logger"
	"github.com/zinrai/ansible-template-render/internal/processor"
	"github.com/zinrai/ansible-template-render/internal/utils"
)

// Runs the template generation process based on the configuration
func RunTemplateGeneration(cfg *config.Config) error {
	// Process each playbook
	for _, playbookConfig := range cfg.Playbooks {
		logger.Info("Processing playbook", "name", playbookConfig.Name)

		err := processPlaybook(playbookConfig, cfg)
		if err != nil {
			return utils.NewError(utils.ErrUnknown, fmt.Sprintf("processing playbook %s", playbookConfig.Name), err)
		}
	}

	return nil
}

// Processes a single playbook
func processPlaybook(playbookConfig config.PlaybookConfig, cfg *config.Config) error {
	// Find the playbook file
	playbookPath, err := finder.FindPlaybook(playbookConfig.Name)
	if err != nil {
		return utils.NewFileNotFoundError(playbookConfig.Name, err)
	}
	logger.Info("Found playbook", "path", playbookPath)

	// Find the inventory file
	inventoryPath, err := finder.FindInventory(playbookConfig.Inventory)
	if err != nil {
		return utils.NewFileNotFoundError(playbookConfig.Inventory, err)
	}
	logger.Info("Found inventory", "path", inventoryPath)

	// Find vars directories (group_vars and host_vars)
	varsDirectories := finder.FindVarsDirectories(playbookPath, inventoryPath)

	// Create and setup the environment
	env, err := setupAndValidateEnvironment(playbookConfig.Name, cfg.Options)
	if err != nil {
		return err
	}

	// Always restore original directory when done
	defer restoreOriginalDirectory(env)

	// Copy inventory file to temp directory
	tempInventoryPath, err := copier.CopyInventory(inventoryPath, env.TempDir)
	if err != nil {
		return utils.NewError(utils.ErrUnknown, "copying inventory", err)
	}
	env.InventoryPath = tempInventoryPath
	logger.Info("Copied inventory file", "path", tempInventoryPath)

	// Copy vars directories to temp directory
	err = copier.CopyVarsDirectories(varsDirectories, env.TempDir)
	if err != nil {
		return utils.NewError(utils.ErrUnknown, "copying vars directories", err)
	}

	// Process the playbook and roles
	hasTemplates, err := processPlaybookContent(playbookPath, env, playbookConfig.Name)
	if err != nil {
		return err
	}

	if !hasTemplates {
		logger.Info("No template tasks found in playbook", "name", playbookConfig.Name)
		return nil
	}

	// Execute or generate instructions
	return executeOrGenerateInstructions(playbookConfig, cfg, env, hasTemplates)
}

// Setup and validate the environment
func setupAndValidateEnvironment(playbookName string, opts config.Options) (*Environment, error) {
	env, err := setupEnvironment(playbookName, opts)
	if err != nil {
		return nil, utils.NewError(utils.ErrEnvironmentSetup, "setting up environment", err)
	}

	return env, nil
}

// Restore the original directory
func restoreOriginalDirectory(env *Environment) {
	if err := os.Chdir(env.OriginalDir); err != nil {
		logger.Warn("Failed to change back to original directory", "error", err)
	}
}

// Execute or generate instructions
func executeOrGenerateInstructions(playbookConfig config.PlaybookConfig, cfg *config.Config, env *Environment, hasTemplates bool) error {
	outputDir := filepath.Join(env.TempDir, "output")

	// Generate-only mode: display instructions and exit
	if cfg.Options.GenerateOnly {
		printGenerateOnlyInstructions(env)
		return nil
	}

	// Execute Ansible
	if err := executeAnsible(env); err != nil {
		return err
	}

	logger.Info("Templates successfully rendered", "output", outputDir)
	return nil
}

// Holds the processing environment details
type Environment struct {
	TempDir           string
	PlaybookPath      string
	TempPlaybookPath  string
	InventoryPath     string
	AnsibleConfigPath string
	OriginalDir       string
}

// Creates and prepares the processing environment
func setupEnvironment(playbookName string, opts config.Options) (*Environment, error) {
	// Create a temporary directory
	tempDir := fmt.Sprintf("tmp-%s", playbookName)

	// Remove existing directory if it exists
	if err := os.RemoveAll(tempDir); err != nil {
		return nil, utils.NewError(utils.ErrUnknown, "cleaning existing directory", err)
	}

	// Create the directory
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, utils.NewError(utils.ErrUnknown, "creating temp directory", err)
	}

	// Create the subdirectories
	err := createRequiredDirectories(tempDir)
	if err != nil {
		return nil, err
	}

	// Save the current directory
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, utils.NewError(utils.ErrUnknown, "getting current directory", err)
	}

	return &Environment{
		TempDir:     tempDir,
		OriginalDir: currentDir,
	}, nil
}

// Create required directories
func createRequiredDirectories(tempDir string) error {
	// Create the roles directory
	rolesDir := filepath.Join(tempDir, "roles")
	if err := os.MkdirAll(rolesDir, 0755); err != nil {
		return utils.NewError(utils.ErrUnknown, "creating roles directory", err)
	}

	// Create the output directory
	outputDir := filepath.Join(tempDir, "output")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return utils.NewError(utils.ErrUnknown, "creating output directory", err)
	}

	return nil
}

// Processes the content of a playbook
func processPlaybookContent(playbookPath string, env *Environment, playbookName string) (bool, error) {
	// Load the playbook
	playbook, err := ansible.LoadPlaybook(playbookPath)
	if err != nil {
		return false, utils.NewError(utils.ErrUnknown, "loading playbook", err)
	}

	// 1. Copy the playbook to temp directory
	playbookCopier := &copier.PlaybookCopier{}
	tempPlaybookPath, err := playbookCopier.CopyPlaybook(playbookPath, env.TempDir)
	if err != nil {
		return false, utils.NewError(utils.ErrUnknown, "copying playbook", err)
	}
	env.PlaybookPath = playbookPath
	env.TempPlaybookPath = tempPlaybookPath

	// 2. Create Ansible configuration
	if err := createAnsibleConfig(env); err != nil {
		return false, err
	}

	// 3. Extract roles from the playbook and resolve dependencies
	directRoles := ansible.ExtractRolesFromPlaybook(playbook)
	logger.Info("Found direct roles", "roles", directRoles)

	resolvedRoles := make(map[string]bool)
	allRoles, err := gatherAllRoles(directRoles, resolvedRoles)
	if err != nil {
		return false, err
	}

	// 4. Remove duplicates
	uniqueRoles := removeDuplicates(allRoles)
	logger.Info("All roles (including dependencies)", "roles", uniqueRoles)

	// 5. Copy all roles to temp directory
	if err := copier.CopyAllRoles(uniqueRoles, env.TempDir); err != nil {
		return false, utils.NewError(utils.ErrUnknown, "copying roles", err)
	}

	// 6. Process tasks in each role
	hasTemplates, err := processor.ProcessAllRoles(uniqueRoles, env.TempDir, playbookName)
	if err != nil {
		return false, utils.NewError(utils.ErrUnknown, "processing role tasks", err)
	}

	return hasTemplates, nil
}

// Gather all roles including dependencies
func gatherAllRoles(directRoles []string, resolvedRoles map[string]bool) ([]string, error) {
	var allRoles []string

	for _, role := range directRoles {
		roleList, err := ansible.ResolveRoleDependencies(role, resolvedRoles)
		if err != nil {
			logger.Warn("Error resolving dependencies", "role", role, "error", err)
			continue
		}
		allRoles = append(allRoles, roleList...)
	}

	return allRoles, nil
}

// Creates the Ansible configuration file
func createAnsibleConfig(env *Environment) error {
	ansibleCfgPath := filepath.Join(env.TempDir, "ansible.cfg")

	ansibleCfgContent := fmt.Sprintf(`[defaults]
roles_path = %s
host_key_checking = False
retry_files_enabled = False
local_tmp = %s/ansible-tmp

[ssh_connection]
pipelining = True
`, filepath.Join(env.TempDir, "roles"), env.TempDir)

	if err := os.WriteFile(ansibleCfgPath, []byte(ansibleCfgContent), 0644); err != nil {
		return utils.NewError(utils.ErrUnknown, "writing ansible.cfg file", err)
	}

	env.AnsibleConfigPath = ansibleCfgPath

	return nil
}

// Prints instructions for generate-only mode
func printGenerateOnlyInstructions(env *Environment) {
	tempPlaybookBasename := filepath.Base(env.TempPlaybookPath)
	absAnsibleCfgPath, _ := filepath.Abs(env.AnsibleConfigPath)

	logger.Info("Generated Ansible files in generate-only mode",
		"playbook", tempPlaybookBasename,
		"inventory", env.InventoryPath,
		"dir", env.TempDir)

	logger.Info("To execute manually:",
		"command", fmt.Sprintf("cd %s && ANSIBLE_CONFIG=%s ansible-playbook %s --tags render_config -i %s",
			env.TempDir, absAnsibleCfgPath, tempPlaybookBasename, env.InventoryPath))
}

// Executes Ansible in the temporary environment
func executeAnsible(env *Environment) error {
	// Change to the temporary directory
	if err := os.Chdir(env.TempDir); err != nil {
		return utils.NewError(utils.ErrUnknown, "changing to temp directory", err)
	}

	// Set the ANSIBLE_CONFIG environment variable
	absAnsibleCfgPath, err := filepath.Abs(env.AnsibleConfigPath)
	if err != nil {
		return utils.NewError(utils.ErrUnknown, "getting absolute path for ansible.cfg", err)
	}
	os.Setenv("ANSIBLE_CONFIG", absAnsibleCfgPath)

	// Execute Ansible with inventory
	tempPlaybookBasename := filepath.Base(env.TempPlaybookPath)
	args := []string{
		tempPlaybookBasename,
		"--tags", "render_config",
		"-i", env.InventoryPath,
	}

	cmd := exec.Command("ansible-playbook", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Command string for logging
	cmdString := fmt.Sprintf("ansible-playbook %s --tags render_config -i %s",
		tempPlaybookBasename, env.InventoryPath)
	logger.Info("Executing Ansible command", "command", cmdString)

	return cmd.Run()
}

// Removes duplicate strings from a slice
func removeDuplicates(elements []string) []string {
	encountered := map[string]bool{}
	result := []string{}

	for _, element := range elements {
		if !encountered[element] {
			encountered[element] = true
			result = append(result, element)
		}
	}

	return result
}
