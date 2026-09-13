package cmd

import (
	"fmt"
)

func (c *RootCmd) executeBook(args []string) int {
	if len(args) == 0 {
		c.PrintBookHelp()
		return 0
	}

	subCmd := args[0]
	subArgs := args[1:]

	switch subCmd {
	case "help", "--help", "-h":
		c.PrintBookHelp()
		return 0
	case "list":
		return c.executeBookList(subArgs)
	case "show":
		return c.executeBookShow(subArgs)
	case "create":
		return c.executeBookCreate(subArgs)
	case "update":
		return c.executeBookUpdate(subArgs)
	case "delete":
		return c.executeBookDelete(subArgs)
	case "duplicate":
		return c.executeBookDuplicate(subArgs)
	case "export":
		return c.executeBookExport(subArgs)
	case "import":
		return c.executeBookImport(subArgs)
	default:
		fmt.Fprintf(c.Err, "unknown book command: %s\nRun 'prosie book --help' for usage.\n", subCmd)
		return 1
	}
}

// PrintBookHelp prints help for the book command hierarchy.
func (c *RootCmd) PrintBookHelp() {
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
