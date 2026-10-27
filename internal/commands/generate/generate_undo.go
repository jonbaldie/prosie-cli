package generatecmd

import (
	"flag"
	"fmt"
	"github.com/jonbaldie/prosie-cli/internal/command"
	"io"
)

func executeGenerateUndo(c *command.Environment, args []string) int {
	fs := flag.NewFlagSet("undo", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := command.ParseFlagsAndArgs(fs, args, command.Args{
		Min: 1, Max: 1, Usage: "prosie generate undo <chapter-id> [flags]", Missing: "chapter ID is required",
	})
	if err != nil {
		return command.FlagError(c, err, printGenerateUndoHelp)
	}

	id := posArgs[0]

	cli, err := command.Client(c)
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	ctx := c.Ctx()

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
