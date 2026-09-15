package chaptercmd

import (
	"fmt"
	"github.com/jonbaldie/prosie-cli/internal/command"
)

func Execute(c *command.Environment, args []string) int {
	return command.Dispatch(c, args, map[string]command.Handler{
		"list":    executeChapterList,
		"show":    executeChapterShow,
		"create":  executeChapterCreate,
		"update":  executeChapterUpdate,
		"reorder": executeChapterReorder,
		"delete":  executeChapterDelete,
		"export":  executeChapterExport,
	}, printChapterHelp, "unknown chapter command: %s\nRun 'prosie chapter --help' for usage.\n")
}

// printChapterHelp prints help for the chapter command hierarchy.
func printChapterHelp(c *command.Environment) {
	help := `Manage chapters on Prosie.

Usage:
  prosie chapter <command> [flags]

Available Commands:
  list        List all chapters in a book
  show        Show chapter prose
  create      Create a new chapter
  update      Update chapter details or prose
  reorder     Reorder chapter sequence
  delete      Delete a chapter
  export      Export chapter prose

Flags:
  -h, --help   Show help for command

Use "prosie chapter <command> --help" for more information about a command.
`
	fmt.Fprint(c.Out, help)
}
