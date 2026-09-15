package generatecmd

import (
	"context"
	"flag"
	"fmt"
	"github.com/jonbaldie/prosie-cli/internal/command"
	"io"
)

func executeGenerateReject(c *command.Environment, args []string) int {
	fs := flag.NewFlagSet("reject", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := command.ParseFlagsAndArgs(fs, args)
	if err != nil {
		return command.FlagError(c, err, printGenerateRejectHelp)
	}

	if len(posArgs) == 0 {
		fmt.Fprintln(c.Err, "error: chapter ID is required. Usage: prosie generate reject <chapter-id> [flags]")
		return 1
	}

	id := posArgs[0]

	cli, err := command.Client(c)
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	ctx := c.Context
	if ctx == nil {
		ctx = context.Background()
	}

	chapter, err := cli.Generation().RejectContinuation(ctx, id)
	if err != nil {
		fmt.Fprintf(c.Err, "error rejecting continuation: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = command.WriteJSON(c, chapter)
		return 0
	}

	fmt.Fprintf(c.Out, "Reverted latest continuation for chapter %s.\n", id)
	return 0
}

// printGenerateRejectHelp prints help for the generate reject command.
func printGenerateRejectHelp(c *command.Environment) {
	help := `Revert the latest AI continuation on a chapter.

Usage:
  prosie generate reject <chapter-id> [flags]

Flags:
  -h, --help   Show help for command
      --json   Format output as JSON
`
	fmt.Fprint(c.Out, help)
}
