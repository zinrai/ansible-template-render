package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/zinrai/ansible-template-render/internal/config"
	"github.com/zinrai/ansible-template-render/internal/generator"
	"github.com/zinrai/ansible-template-render/internal/logger"
)

var (
	version = "dev"
)

func main() {
	configFile := flag.String("config", "", "Config file path")
	outputDir := flag.String("output-dir", "", "Override output directory")
	showVersion := flag.Bool("version", false, "Show version")
	keepTempFiles := flag.Bool("keep-temp", false, "Keep temporary files")
	generateOnly := flag.Bool("generate-only", false, "Generate modified Ansible files without executing")
	logLevel := flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	flag.Parse()

	// Set up logging
	setupLogging(*logLevel)

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

	// Override output directory if specified
	if *outputDir != "" {
		cfg.OutputBaseDir = *outputDir
	}

	// Override options if specified
	cfg.Options.KeepTempFiles = *keepTempFiles || cfg.Options.KeepTempFiles
	cfg.Options.GenerateOnly = *generateOnly || cfg.Options.GenerateOnly

	// Run template generation
	err = generator.RunTemplateGeneration(cfg)
	if err != nil {
		logger.Error("Error occurred", "error", err)
		os.Exit(1)
	}

	if cfg.Options.GenerateOnly {
		logger.Info("Modified Ansible files generated successfully. Use --keep-temp to see the files.")
	} else {
		logger.Info("Successfully generated template files")
	}
}

// Configures the logger based on the specified level
func setupLogging(level string) {
	var logLevel slog.Level

	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	logger.Initialize(logLevel, nil)
}
