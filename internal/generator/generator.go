package generator

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/zinrai/ansible-template-render/internal/ansible"
	"github.com/zinrai/ansible-template-render/internal/config"
	"github.com/zinrai/ansible-template-render/internal/copier"
	"github.com/zinrai/ansible-template-render/internal/executor"
	"github.com/zinrai/ansible-template-render/internal/finder"
	"github.com/zinrai/ansible-template-render/internal/logger"
	"github.com/zinrai/ansible-template-render/internal/processor"
	"github.com/zinrai/ansible-template-render/internal/utils"
)

// Runs the template generation process based on the configuration
func RunTemplateGeneration(cfg *config.Config, generateOnly bool) error {
	for _, playbookConfig := range cfg.Playbooks {
		logger.Info("Processing playbook", "name", playbookConfig.Name)
		err := processPlaybook(playbookConfig, cfg, generateOnly)
		if err != nil {
			return utils.NewError(utils.ErrUnknown, fmt.Sprintf("processing playbook %s", playbookConfig.Name), err)
		}
	}
	return nil
}

func processPlaybook(playbookConfig config.PlaybookConfig, cfg *config.Config, generateOnly bool) error {
	playbookPath, err := finder.FindPlaybook(playbookConfig.Name)
	if err != nil {
		return utils.NewFileNotFoundError(playbookConfig.Name, err)
	}
	logger.Info("Found playbook", "path", playbookPath)

	inventoryPath, err := finder.FindInventory(playbookConfig.Inventory)
	if err != nil {
		return utils.NewFileNotFoundError(playbookConfig.Inventory, err)
	}
	logger.Info("Found inventory", "path", inventoryPath)

	varsDirectories := finder.FindVarsDirectories(playbookPath, inventoryPath)

	env, err := setupAndValidateEnvironment(playbookConfig.Name)
	if err != nil {
		return err
	}
	defer restoreOriginalDirectory(env)

	// Convert inventory for local execution using ansible-inventory
	tempInventoryPath, err := processor.ModifyInventoryForLocalExecution(inventoryPath, env.TempDir)
	if err != nil {
		return utils.NewError(utils.ErrUnknown, "converting inventory for local execution", err)
	}
	env.InventoryPath = tempInventoryPath
	logger.Info("Converted inventory for local execution", "path", tempInventoryPath)

	err = copier.CopyVarsDirectories(varsDirectories, env.TempDir)
	if err != nil {
		return utils.NewError(utils.ErrUnknown, "copying vars directories", err)
	}

	hasTemplates, err := processPlaybookContent(playbookPath, env, playbookConfig.Name)
	if err != nil {
		return err
	}

	if !hasTemplates {
		logger.Info("No template tasks found in playbook", "name", playbookConfig.Name)
		return nil
	}

	return executeOrGenerateInstructions(cfg, env, generateOnly)
}

func setupAndValidateEnvironment(playbookName string) (*Environment, error) {
	env, err := setupEnvironment(playbookName)
	if err != nil {
		return nil, utils.NewError(utils.ErrEnvironmentSetup, "setting up environment", err)
	}
	return env, nil
}

func restoreOriginalDirectory(env *Environment) {
	if err := os.Chdir(env.OriginalDir); err != nil {
		logger.Warn("Failed to change back to original directory", "error", err)
	}
}

func executeOrGenerateInstructions(cfg *config.Config, env *Environment, generateOnly bool) error {
	outputDir := filepath.Join(env.TempDir, "output")

	if generateOnly {
		printGenerateOnlyInstructions(env, cfg.Options.AnsibleArgs)
		return nil
	}

	if err := executeAnsible(env, cfg.Options.AnsibleArgs); err != nil {
		return err
	}

	logger.Info("Templates successfully rendered", "output", outputDir)
	return nil
}

type Environment struct {
	TempDir           string
	PlaybookPath      string
	TempPlaybookPath  string
	InventoryPath     string
	AnsibleConfigPath string
	OriginalDir       string
}

func setupEnvironment(playbookName string) (*Environment, error) {
	tempDir := fmt.Sprintf("tmp-%s", playbookName)

	if err := os.RemoveAll(tempDir); err != nil {
		return nil, utils.NewError(utils.ErrUnknown, "cleaning existing directory", err)
	}

	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, utils.NewError(utils.ErrUnknown, "creating temp directory", err)
	}

	err := createRequiredDirectories(tempDir)
	if err != nil {
		return nil, err
	}

	currentDir, err := os.Getwd()
	if err != nil {
		return nil, utils.NewError(utils.ErrUnknown, "getting current directory", err)
	}

	return &Environment{
		TempDir:     tempDir,
		OriginalDir: currentDir,
	}, nil
}

func createRequiredDirectories(tempDir string) error {
	rolesDir := filepath.Join(tempDir, "roles")
	if err := os.MkdirAll(rolesDir, 0755); err != nil {
		return utils.NewError(utils.ErrUnknown, "creating roles directory", err)
	}

	outputDir := filepath.Join(tempDir, "output")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return utils.NewError(utils.ErrUnknown, "creating output directory", err)
	}

	return nil
}

func processPlaybookContent(playbookPath string, env *Environment, playbookName string) (bool, error) {
	playbook, err := ansible.LoadPlaybook(playbookPath)
	if err != nil {
		return false, utils.NewError(utils.ErrUnknown, "loading playbook", err)
	}

	playbookCopier := &copier.PlaybookCopier{}
	tempPlaybookPath, err := playbookCopier.CopyPlaybook(playbookPath, env.TempDir)
	if err != nil {
		return false, utils.NewError(utils.ErrUnknown, "copying playbook", err)
	}
	env.PlaybookPath = playbookPath
	env.TempPlaybookPath = tempPlaybookPath

	if err := createAnsibleConfig(env); err != nil {
		return false, err
	}

	directRoles := ansible.ExtractRolesFromPlaybook(playbook)
	logger.Info("Found direct roles", "roles", directRoles)

	resolvedRoles := make(map[string]bool)
	allRoles, err := gatherAllRoles(directRoles, resolvedRoles)
	if err != nil {
		return false, err
	}

	uniqueRoles := removeDuplicates(allRoles)
	logger.Info("All roles (including dependencies)", "roles", uniqueRoles)

	if err := copier.CopyAllRoles(uniqueRoles, env.TempDir); err != nil {
		return false, utils.NewError(utils.ErrUnknown, "copying roles", err)
	}

	hasTemplates, err := processor.ProcessAllRoles(uniqueRoles, env.TempDir, playbookName)
	if err != nil {
		return false, utils.NewError(utils.ErrUnknown, "processing role tasks", err)
	}

	return hasTemplates, nil
}

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

func createAnsibleConfig(env *Environment) error {
	ansibleCfgPath := filepath.Join(env.TempDir, "ansible.cfg")

	ansibleCfgContent := `[defaults]
retry_files_enabled = False
local_tmp = ansible-tmp
`

	if err := os.WriteFile(ansibleCfgPath, []byte(ansibleCfgContent), 0644); err != nil {
		return utils.NewError(utils.ErrUnknown, "writing ansible.cfg file", err)
	}

	env.AnsibleConfigPath = ansibleCfgPath
	return nil
}

func printGenerateOnlyInstructions(env *Environment, ansibleArgs string) {
	tempPlaybookBasename := filepath.Base(env.TempPlaybookPath)
	absAnsibleCfgPath, _ := filepath.Abs(env.AnsibleConfigPath)
	inventoryBasename := filepath.Base(env.InventoryPath)

	logger.Info("Generated Ansible files in generate-only mode",
		"playbook", tempPlaybookBasename,
		"inventory", inventoryBasename,
		"dir", env.TempDir)

	cmd := fmt.Sprintf("cd %s && ANSIBLE_CONFIG=%s ansible-playbook %s --tags render_config -i %s",
		env.TempDir, absAnsibleCfgPath, tempPlaybookBasename, inventoryBasename)
	if ansibleArgs != "" {
		cmd += fmt.Sprintf(" %s", ansibleArgs)
	}
	logger.Info("To execute manually:", "command", cmd)
}

func executeAnsible(env *Environment, ansibleArgs string) error {
	execEnv := executor.ExecutionEnvironment{
		WorkingDir:        env.TempDir,
		PlaybookPath:      filepath.Base(env.TempPlaybookPath),
		InventoryPath:     filepath.Base(env.InventoryPath),
		AnsibleConfigPath: env.AnsibleConfigPath,
		AnsibleArgs:       ansibleArgs,
	}

	return executor.RunAnsible(execEnv)
}

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
