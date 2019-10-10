package main

import (
	"os"

	"github.com/miquella/sekura/cmd/sekura/commands"
)

func main() {
	if err := commands.SekuraCommand.Execute(); err != nil {
		os.Exit(1)
	}
}
