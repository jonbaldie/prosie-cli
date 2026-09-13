package cmd

import (
	"context"
	"flag"
	"fmt"
	"io"
)

func (c *RootCmd) executeGenerateReject(args []string) int {
	fs := flag.NewFlagSet("reject", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := parseFlagsAndArgs(fs, args)
	if err != nil {
		if err == flag.ErrHelp {
			c.PrintGenerateRejectHelp()
			return 0
		}
		fmt.Fprintf(c.Err, "error parsing flags: %v\n", err)
		return 1
	}

	if len(posArgs) == 0 {
		fmt.Fprintln(c.Err, "error: chapter ID is required. Usage: prosie generate reject <chapter-id> [flags]")
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

	chapter, err := cli.RejectContinuation(ctx, id)
	if err != nil {
		fmt.Fprintf(c.Err, "error rejecting continuation: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = c.WriteJSON(chapter)
		return 0
	}

	fmt.Fprintf(c.Out, "Reverted latest continuation for chapter %s.\n", id)
	return 0
}

// PrintGenerateRejectHelp prints help for the generate reject command.
func (c *RootCmd) PrintGenerateRejectHelp() {
	help := `Revert the latest AI continuation on a chapter.

Usage:
  prosie generate reject <chapter-id> [flags]

Flags:
  -h, --help   Show help for command
      --json   Format output as JSON
`
	fmt.Fprint(c.Out, help)
}
