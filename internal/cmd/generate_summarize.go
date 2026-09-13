package cmd

import (
	"context"
	"flag"
	"fmt"
	"io"
)

func (c *RootCmd) executeGenerateSummarize(args []string) int {
	fs := flag.NewFlagSet("summarize", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := parseFlagsAndArgs(fs, args)
	if err != nil {
		if err == flag.ErrHelp {
			c.PrintGenerateSummarizeHelp()
			return 0
		}
		fmt.Fprintf(c.Err, "error parsing flags: %v\n", err)
		return 1
	}

	if len(posArgs) == 0 {
		fmt.Fprintln(c.Err, "error: chapter ID is required. Usage: prosie generate summarize <chapter-id> [flags]")
		return 1
	}

	id := posArgs[0]

	cli, err := c.Client()
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	ctx := c.Context
	if ctx == nil {
		ctx = context.Background()
	}

	res, err := cli.Summarize(ctx, id)
	if err != nil {
		fmt.Fprintf(c.Err, "error generating summary: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = c.WriteJSON(res)
		return 0
	}

	fmt.Fprintln(c.Out, res.Summary)
	return 0
}

// PrintGenerateSummarizeHelp prints help for the generate summarize command.
func (c *RootCmd) PrintGenerateSummarizeHelp() {
	help := `Generate or update a one-line chapter summary.

Usage:
  prosie generate summarize <chapter-id> [flags]

Flags:
  -h, --help   Show help for command
      --json   Format output as JSON
`
	fmt.Fprint(c.Out, help)
}
