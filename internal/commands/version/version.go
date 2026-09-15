package versioncmd

import (
	"flag"
	"fmt"
	"github.com/jonbaldie/prosie-cli/internal/command"
	"io"

	"github.com/jonbaldie/prosie-cli/internal/version"
)

func Execute(c *command.Environment, args []string) int {
	fs := flag.NewFlagSet("version", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	if _, err := command.ParseFlagsAndArgs(fs, args); err != nil {
		return command.FlagError(c, err, printVersionHelp)
	}

	if *jsonFlag {
		_ = command.WriteJSON(c, map[string]string{
			"version": version.Version,
		})
		return 0
	}

	fmt.Fprintf(c.Out, "prosie version %s\n", version.Version)
	return 0
}

// printVersionHelp prints help for the version command.
func printVersionHelp(c *command.Environment) {
	help := `Display the CLI version.

Usage:
  prosie version [flags]

Flags:
  -h, --help   Show help for command
      --json   Output in JSON format
`
	fmt.Fprint(c.Out, help)
}
