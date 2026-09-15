package bookcmd

import (
	"fmt"
	"github.com/jonbaldie/prosie-cli/internal/command"
)

func Execute(c *command.Environment, args []string) int {
	return command.Dispatch(c, args, map[string]command.Handler{
		"list":      executeBookList,
		"show":      executeBookShow,
		"create":    executeBookCreate,
		"update":    executeBookUpdate,
		"delete":    executeBookDelete,
		"duplicate": executeBookDuplicate,
		"export":    executeBookExport,
		"import":    executeBookImport,
	}, printBookHelp, "unknown book command: %s\nRun 'prosie book --help' for usage.\n")
}

// printBookHelp prints help for the book command hierarchy.
func printBookHelp(c *command.Environment) {
	help := `Manage books on Prosie.

Usage:
  prosie book <command> [flags]

Available Commands:
  list        List all books
  show        Show book details and chapters
  create      Create a new book
  update      Update book details
  delete      Delete a book
  duplicate   Duplicate a book
  export      Export full book prose
  import      Import a DOCX manuscript file

Flags:
  -h, --help   Show help for command

Use "prosie book <command> --help" for more information about a command.
`
	fmt.Fprint(c.Out, help)
}
