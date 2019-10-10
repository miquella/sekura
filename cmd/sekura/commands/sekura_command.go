package commands

import (
	"github.com/spf13/cobra"
)

// SekuraCommand is the root command for the `sekura` entrypoint
var SekuraCommand = &cobra.Command{
	Use:   "sekura",
	Short: "Sekura provides access to AWS credentials through a mock ECS metadata service",
}

func init() {
	SekuraCommand.AddCommand(ShellCommand)
}
