package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/zinrai/ansible-template-render/internal/generator"
	"github.com/zinrai/ansible-template-render/internal/logger"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

const usage = `Usage:
  ansible-template-render run      [-i INV] PLAYBOOK [-- ANSIBLE_ARGS...]
  ansible-template-render generate [-i INV] PLAYBOOK [-- ANSIBLE_ARGS...]
  ansible-template-render version
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		runSubcommand(os.Args[2:], false)
	case "generate":
		runSubcommand(os.Args[2:], true)
	case "version":
		fmt.Printf("ansible-template-render %s (commit %s, built %s)\n", version, commit, date)
	case "-h", "--help", "help":
		fmt.Fprint(os.Stdout, usage)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}
}

func runSubcommand(args []string, generateOnly bool) {
	name := "run"
	if generateOnly {
		name = "generate"
	}

	beforeDash, afterDash := splitAtDoubleDash(args)

	fs := flag.NewFlagSet(name, flag.ExitOnError)
	inventory := fs.String("i", "", "Path to the inventory file (required)")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: ansible-template-render %s [-i INV] PLAYBOOK [-- ANSIBLE_ARGS...]\n", name)
		fs.PrintDefaults()
	}
	if err := fs.Parse(beforeDash); err != nil {
		os.Exit(2)
	}

	positional := fs.Args()
	if len(positional) != 1 {
		fs.Usage()
		os.Exit(2)
	}
	playbook := positional[0]

	if *inventory == "" {
		logger.Error("-i is required")
		fs.Usage()
		os.Exit(2)
	}

	ansibleArgs := strings.Join(afterDash, " ")

	if err := generator.RunTemplateGeneration(playbook, *inventory, ansibleArgs, generateOnly); err != nil {
		logger.Error("Error occurred", "error", err)
		os.Exit(1)
	}

	if generateOnly {
		logger.Info("Modified Ansible files generated successfully.")
	} else {
		logger.Info("Successfully generated template files")
	}
}

func splitAtDoubleDash(args []string) ([]string, []string) {
	for i, a := range args {
		if a == "--" {
			return args[:i], args[i+1:]
		}
	}
	return args, nil
}
