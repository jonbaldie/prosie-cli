package chaptercmd

import (
	"context"
	"flag"
	"fmt"
	"github.com/jonbaldie/prosie-cli/internal/command"
	"io"
	"strconv"
)

func executeChapterDelete(c *command.Environment, args []string) int {
	fs := flag.NewFlagSet("delete", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	yesFlag := fs.Bool("yes", false, "Skip confirmation prompt")
	fs.BoolVar(yesFlag, "y", false, "Skip confirmation prompt (shorthand)")
	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := command.ParseFlagsAndArgs(fs, args)
	if err != nil {
		return command.FlagError(c, err, printChapterDeleteHelp)
	}

	if len(posArgs) == 0 {
		fmt.Fprintln(c.Err, "error: chapter ID is required. Usage: prosie chapter delete <id> [flags]")
		return 1
	}

	id := posArgs[0]

	if !command.ConfirmDeletion(c, *yesFlag, "chapter", id) {
		return 0
	}

	cli, err := command.Client(c)
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	if err := cli.Chapters().DeleteChapter(context.Background(), id); err != nil {
		fmt.Fprintf(c.Err, "error deleting chapter: %v\n", err)
		return 1
	}

	if *jsonFlag {
		var idVal any = id
		if intVal, err := strconv.Atoi(id); err == nil {
			idVal = intVal
		}
		_ = command.WriteJSON(c, map[string]any{
			"id":      idVal,
			"deleted": true,
		})
		return 0
	}

	fmt.Fprintf(c.Out, "Deleted chapter %s.\n", id)
	return 0
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
