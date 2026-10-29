package chaptercmd

import (
	"context"
	"flag"
	"fmt"
	"github.com/jonbaldie/prosie-cli/internal/client"
	"github.com/jonbaldie/prosie-cli/internal/command"
	"io"
)

func executeChapterDelete(c *command.Environment, args []string) int {
	fs := flag.NewFlagSet("delete", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	yesFlag := fs.Bool("yes", false, "Skip confirmation prompt")
	fs.BoolVar(yesFlag, "y", false, "Skip confirmation prompt (shorthand)")
	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := command.ParseFlagsAndArgs(fs, args, command.Args{
		Min: 1, Max: 1, Usage: "prosie chapter delete <id> [flags]", Missing: "chapter ID is required",
	})
	if err != nil {
		return command.FlagError(c, err, printChapterDeleteHelp)
	}

	id := posArgs[0]

	return command.ExecuteDelete(c, command.DeleteOptions{Resource: "chapter", ID: id, Yes: *yesFlag, JSON: *jsonFlag},
		func(ctx context.Context, cli *client.Client) error {
			return cli.Chapters().DeleteChapter(ctx, id)
		})
}

func printChapterDeleteHelp(c *command.Environment) {
	help := `Delete a chapter.

Usage:
  prosie chapter delete <id> [flags]

Flags:
  -h, --help   Show help for command
      --json   Format output as JSON
  -y, --yes    Skip confirmation prompt
`
	fmt.Fprint(c.Out, help)
}
