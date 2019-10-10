package commands

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/miquella/sekura/operations"
)

var shell = operations.Spawn{}

// ShellCommand is the `sekura shell` subcommand
var ShellCommand = &cobra.Command{
	Use:   "shell",
	Short: "Invokes a login shell with the ECS metadata service available",
	// Args:  cobra.MaximumNArgs(1), // Disabling vault specification for now
	Args: cobra.NoArgs,

	RunE: shellRunE,
}

func init() {
	ShellCommand.Flags().StringSliceVar(&shell.Assume, "assume", nil, "Roles or role aliases to assume")
	// ShellCommand.Flags().BoolVar(&shell.Refresh, "refresh", false, "Force a refresh of the session")
	// ShellCommand.Flags().StringVar(&shell.Region, "region", "", "The region of the STS endpoint to use")
}

func shellRunE(cmd *cobra.Command, args []string) error {
	shell.Command = loginShellCommand()
	if len(args) == 1 {
		shell.VaultName = args[0]
	}
	return shell.Run()
}

func loginShellCommand() []string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}

	return []string{shell, "--login"}
}
