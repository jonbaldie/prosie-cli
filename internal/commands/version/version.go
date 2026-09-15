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

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(c.Err, "error parsing flags: %v\n", err)
		return 1
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
