package cmd

import (
	"fmt"
)

func (c *RootCmd) executeChapter(args []string) int {
	if len(args) == 0 {
		c.PrintChapterHelp()
		return 0
	}

	subCmd := args[0]
	subArgs := args[1:]

	switch subCmd {
	case "help", "--help", "-h":
		c.PrintChapterHelp()
		return 0
	case "list":
		return c.executeChapterList(subArgs)
	case "show":
		return c.executeChapterShow(subArgs)
	case "create":
		return c.executeChapterCreate(subArgs)
	case "update":
		return c.executeChapterUpdate(subArgs)
	case "reorder":
		return c.executeChapterReorder(subArgs)
	case "delete":
		return c.executeChapterDelete(subArgs)
	case "export":
		return c.executeChapterExport(subArgs)
	default:
		fmt.Fprintf(c.Err, "unknown chapter command: %s\nRun 'prosie chapter --help' for usage.\n", subCmd)
		return 1
	}
}

// PrintChapterHelp prints help for the chapter command hierarchy.
func (c *RootCmd) PrintChapterHelp() {
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
