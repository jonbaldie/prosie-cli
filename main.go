package main

import (
	"os"

	"github.com/jonbaldie/prosie-cli/internal/cmd"
)

func main() {
	rootCmd := cmd.NewRootCmd()
	os.Exit(rootCmd.Execute(os.Args[1:]))
}
