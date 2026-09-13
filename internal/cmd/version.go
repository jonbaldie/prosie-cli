package cmd

import (
	"flag"
	"fmt"
	"io"

	"github.com/jonbaldie/prosie-cli/internal/version"
)

func (c *RootCmd) executeVersion(args []string) int {
	fs := flag.NewFlagSet("version", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(c.Err, "error parsing flags: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = c.WriteJSON(map[string]string{
			"version": version.Version,
		})
		return 0
	}

	fmt.Fprintf(c.Out, "prosie version %s\n", version.Version)
	return 0
}
