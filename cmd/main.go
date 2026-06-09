package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/zinrai/ansible-template-render/internal/config"
	"github.com/zinrai/ansible-template-render/internal/generator"
	"github.com/zinrai/ansible-template-render/internal/logger"
)

var (
	version = "0.3.0"
)

func main() {
	configFile := flag.String("config", "", "Config file path")
	showVersion := flag.Bool("version", false, "Show version")
	generateOnly := flag.Bool("generate-only", false, "Generate modified Ansible files without executing")
	flag.Parse()

	if *showVersion {
		fmt.Printf("ansible-template-render version %s\n", version)
		os.Exit(0)
	}

	if *configFile == "" {
		logger.Error("Config file is required")
		fmt.Println("Usage: ansible-template-render --config [config file]")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Load configuration
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		logger.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	// Run template generation
	err = generator.RunTemplateGeneration(cfg, *generateOnly)
	if err != nil {
		logger.Error("Error occurred", "error", err)
		os.Exit(1)
	}

	if *generateOnly {
		logger.Info("Modified Ansible files generated successfully.")
	} else {
		logger.Info("Successfully generated template files")
	}
}
