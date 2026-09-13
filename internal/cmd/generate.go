package cmd

import (
	"fmt"
)

func (c *RootCmd) executeGenerate(args []string) int {
	if len(args) == 0 {
		c.PrintGenerateHelp()
		return 0
	}

	subCmd := args[0]
	subArgs := args[1:]

	switch subCmd {
	case "help", "--help", "-h":
		c.PrintGenerateHelp()
		return 0
	case "continue":
		return c.executeGenerateContinue(subArgs)
	case "reject":
		return c.executeGenerateReject(subArgs)
	case "rewrite":
		return c.executeGenerateRewrite(subArgs)
	case "summarize":
		return c.executeGenerateSummarize(subArgs)
	default:
		fmt.Fprintf(c.Err, "unknown generate command: %s\nRun 'prosie generate --help' for usage.\n", subCmd)
		return 1
	}
}

// PrintGenerateHelp prints help for the generate command hierarchy.
func (c *RootCmd) PrintGenerateHelp() {
	help := `Manage AI prose generation on Prosie.

Usage:
  prosie generate <command> [flags]

Available Commands:
  continue    Generate AI prose continuation for a chapter
  reject      Revert latest AI continuation on a chapter
  rewrite     Rewrite selected chapter text
  summarize   Generate or update one-line chapter summary

Flags:
  -h, --help   Show help for command

Use "prosie generate <command> --help" for more information about a command.
`
	fmt.Fprint(c.Out, help)
}
