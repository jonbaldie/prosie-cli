package generatecmd

import (
	"fmt"
	"github.com/jonbaldie/prosie-cli/internal/command"
)

func Execute(c *command.Environment, args []string) int {
	if len(args) == 0 {
		printGenerateHelp(c)
		return 0
	}

	subCmd := args[0]
	subArgs := args[1:]

	switch subCmd {
	case "help", "--help", "-h":
		printGenerateHelp(c)
		return 0
	case "continue":
		return executeGenerateContinue(c, subArgs)
	case "reject":
		return executeGenerateReject(c, subArgs)
	case "rewrite":
		return executeGenerateRewrite(c, subArgs)
	case "summarize":
		return executeGenerateSummarize(c, subArgs)
	default:
		fmt.Fprintf(c.Err, "unknown generate command: %s\nRun 'prosie generate --help' for usage.\n", subCmd)
		return 1
	}
}

// printGenerateHelp prints help for the generate command hierarchy.
func printGenerateHelp(c *command.Environment) {
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
