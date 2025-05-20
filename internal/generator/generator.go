package generator

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/zinrai/ansible-template-render/internal/ansible"
	"github.com/zinrai/ansible-template-render/internal/config"
	"github.com/zinrai/ansible-template-render/internal/executor"
	"github.com/zinrai/ansible-template-render/internal/finder"
	"github.com/zinrai/ansible-template-render/internal/logger"
	"github.com/zinrai/ansible-template-render/internal/utils"
)

// Runs the template generation process based on the configuration
func RunTemplateGeneration(cfg *config.Config) error {
	// Ensure the output base directory exists
	if err := os.MkdirAll(cfg.OutputBaseDir, 0755); err != nil {
		return utils.NewError(utils.ErrUnknown, "failed to create output base directory", err)
	}

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

	// Create and setup the environment
	env, err := setupAndValidateEnvironment(playbookConfig.Name, cfg.Options)
	if err != nil {
		return err
	}

	// Clean up the environment when done, unless the user wants to keep it
	if !cfg.Options.KeepTempFiles && !cfg.Options.GenerateOnly {
		defer cleanupEnvironment(env)
	}

	// Process roles and templates
	hasTemplates, err := processPlaybookRolesAndTemplates(playbookPath, env)
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

// Process playbook roles and templates
func processPlaybookRolesAndTemplates(playbookPath string, env *Environment) (bool, error) {
	// Process roles and templates
	hasTemplates, err := processPlaybookContent(playbookPath, env)
	if err != nil {
		return false, err
	}

	return hasTemplates, nil
}

// Execute or generate instructions
func executeOrGenerateInstructions(playbookConfig config.PlaybookConfig, cfg *config.Config, env *Environment, hasTemplates bool) error {
	// Determine output directory
	outputDir := determineOutputDirectory(cfg.OutputBaseDir, playbookConfig.Name, env.TempDir)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return utils.NewError(utils.ErrUnknown, "creating output directory", err)
	}

	// Generate-only mode: display instructions and exit
	if cfg.Options.GenerateOnly {
		printGenerateOnlyInstructions(env, outputDir)
		return nil
	}

	// Execute Ansible
	if err := executeAnsible(env, outputDir); err != nil {
		return err
	}

	logger.Info("Templates successfully rendered", "output", outputDir)

	// If keeping temp files, inform the user
	if cfg.Options.KeepTempFiles {
		relTempDir, _ := filepath.Rel(".", env.TempDir)
		logger.Info("Temporary files kept", "path", relTempDir)
	}

	return nil
}

// Holds the processing environment details
type Environment struct {
	TempDir           string
	PlaybookPath      string
	TempPlaybookPath  string
	AnsibleConfigPath string
	OriginalDir       string
}

// Creates and prepares the processing environment
func setupEnvironment(playbookName string, opts config.Options) (*Environment, error) {
	// Create a temporary directory
	tempDir := fmt.Sprintf("tmp-%s", playbookName)
	if err := os.RemoveAll(tempDir); err != nil {
		return nil, utils.NewError(utils.ErrUnknown, "cleaning temp directory", err)
	}
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

// Cleans up the processing environment
func cleanupEnvironment(env *Environment) {
	if err := os.Chdir(env.OriginalDir); err != nil {
		logger.Warn("Failed to change back to original directory", "error", err)
	}

	if err := os.RemoveAll(env.TempDir); err != nil {
		logger.Warn("Failed to remove temp directory", "error", err)
	}
}

// Processes the content of a playbook
func processPlaybookContent(playbookPath string, env *Environment) (bool, error) {
	// Load the playbook
	playbook, err := ansible.LoadPlaybook(playbookPath)
	if err != nil {
		return false, utils.NewError(utils.ErrUnknown, "loading playbook", err)
	}

	// Extract and process roles
	hasTemplates, err := extractAndProcessRoles(playbook, env)
	if err != nil {
		return false, err
	}

	// Copy the playbook and create config
	if err := preparePlaybookFiles(playbookPath, env); err != nil {
		return false, err
	}

	return hasTemplates, nil
}

// Extract and process roles from the playbook
func extractAndProcessRoles(playbook []map[string]interface{}, env *Environment) (bool, error) {
	// Extract roles from the playbook
	directRoles := ansible.ExtractRolesFromPlaybook(playbook)
	logger.Info("Found direct roles", "roles", directRoles)

	// Process all roles and their dependencies
	_, hasTemplates, err := resolveAndProcessRoles(directRoles, env)
	if err != nil {
		return false, err
	}

	return hasTemplates, nil
}

// Prepare playbook files (copy and create config)
func preparePlaybookFiles(playbookPath string, env *Environment) error {
	// Copy the playbook file to the temporary directory
	if err := copyPlaybookToTemp(playbookPath, env); err != nil {
		return err
	}

	// Create Ansible configuration
	if err := createAnsibleConfig(env); err != nil {
		return err
	}

	return nil
}

// Resolves role dependencies and processes all roles
func resolveAndProcessRoles(directRoles []string, env *Environment) ([]string, bool, error) {
	// Resolve role dependencies
	resolvedRoles := make(map[string]bool)
	allRoles, err := gatherAllRoles(directRoles, resolvedRoles)
	if err != nil {
		// Error already logged in gatherAllRoles
		return nil, false, nil
	}

	// Remove duplicates
	uniqueRoles := removeDuplicates(allRoles)
	logger.Info("All roles (including dependencies)", "roles", uniqueRoles)

	// Process each role
	hasTemplates, err := processRoles(uniqueRoles, env)
	if err != nil {
		return nil, false, err
	}

	return uniqueRoles, hasTemplates, nil
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

// Process all roles
func processRoles(roles []string, env *Environment) (bool, error) {
	hasTemplates := false

	for _, role := range roles {
		logger.Info("Processing role", "name", role)

		// Process the role's tasks
		roleHasTemplates, err := ProcessRoleTasks(role, env.TempDir)
		if err != nil {
			logger.Warn("Error processing role", "role", role, "error", err)
			continue
		}

		if roleHasTemplates {
			hasTemplates = true
		}
	}

	return hasTemplates, nil
}

// Copies the playbook to the temporary directory
func copyPlaybookToTemp(playbookPath string, env *Environment) error {
	tempPlaybookPath := filepath.Join(env.TempDir, filepath.Base(playbookPath))

	if err := os.MkdirAll(filepath.Dir(tempPlaybookPath), 0755); err != nil {
		return utils.NewError(utils.ErrUnknown, "creating temp playbook directory", err)
	}

	if err := utils.CopyFile(playbookPath, tempPlaybookPath); err != nil {
		return utils.NewError(utils.ErrUnknown, "copying playbook file", err)
	}

	env.PlaybookPath = playbookPath
	env.TempPlaybookPath = tempPlaybookPath

	return nil
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

// Determines the output directory for a playbook
func determineOutputDirectory(configOutputDir, playbookName, tempDir string) string {
	// If output directory is explicitly specified in config, use it
	if configOutputDir != "" {
		return filepath.Join(configOutputDir, playbookName)
	}

	// Otherwise, use tmp-{playbookName}/output
	return filepath.Join(tempDir, "output")
}

// Prints instructions for generate-only mode
func printGenerateOnlyInstructions(env *Environment, outputDir string) {
	relTempDir, err := filepath.Rel(".", env.TempDir)
	if err != nil {
		relTempDir = env.TempDir
	}

	absAnsibleCfgPath, _ := filepath.Abs(env.AnsibleConfigPath)
	tempPlaybookBasename := filepath.Base(env.TempPlaybookPath)

	logger.Info("Generated Ansible files in generate-only mode",
		"playbook", tempPlaybookBasename,
		"dir", relTempDir)

	logger.Info("To execute manually:",
		"command", fmt.Sprintf("cd %s && ANSIBLE_CONFIG=%s ansible-playbook %s --tags render_config -e \"template_dest_prefix=%s\"",
			relTempDir, absAnsibleCfgPath, tempPlaybookBasename, outputDir))
}

// Executes Ansible in the temporary environment
func executeAnsible(env *Environment, outputDir string) error {
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

	// Execute Ansible
	tempPlaybookBasename := filepath.Base(env.TempPlaybookPath)
	err = executor.RunAnsible(tempPlaybookBasename, outputDir)
	if err != nil {
		return utils.NewAnsibleExecutionError("executing ansible", err)
	}

	return nil
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
