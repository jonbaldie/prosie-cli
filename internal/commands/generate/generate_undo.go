package generatecmd

import (
	"context"
	"flag"
	"fmt"
	"github.com/jonbaldie/prosie-cli/internal/command"
	"io"
)

func executeGenerateUndo(c *command.Environment, args []string) int {
	fs := flag.NewFlagSet("undo", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := command.ParseFlagsAndArgs(fs, args)
	if err != nil {
		return command.FlagError(c, err, printGenerateUndoHelp)
	}

	if len(posArgs) == 0 {
		fmt.Fprintln(c.Err, "error: chapter ID is required. Usage: prosie generate undo <chapter-id> [flags]")
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

	chapter, err := cli.Generation().UndoRewrite(ctx, id)
	if err != nil {
		fmt.Fprintf(c.Err, "error undoing rewrite: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = command.WriteJSON(c, chapter)
		return 0
	}

	fmt.Fprintf(c.Out, "Undid latest rewrite for chapter %s.\n", id)
	return 0
}

// printGenerateUndoHelp prints help for the generate undo command.
func printGenerateUndoHelp(c *command.Environment) {
	help := `Undo the latest AI rewrite on a chapter.

Restores the chapter content from before the most recent rewrite.

Usage:
  prosie generate undo <chapter-id> [flags]

Flags:
  -h, --help   Show help for command
      --json   Format output as JSON
`
	fmt.Fprint(c.Out, help)
}
