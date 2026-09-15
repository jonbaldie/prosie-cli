package authcmd

import (
	"fmt"
	"github.com/jonbaldie/prosie-cli/internal/command"
)

func Execute(c *command.Environment, args []string) int {
	if len(args) == 0 {
		printAuthHelp(c)
		return 0
	}

	subCmd := args[0]
	subArgs := args[1:]

	switch subCmd {
	case "help", "--help", "-h":
		printAuthHelp(c)
		return 0
	case "login":
		return executeAuthLogin(c, subArgs)
	case "status":
		return executeAuthStatus(c, subArgs)
	case "logout":
		return executeAuthLogout(c, subArgs)
	default:
		fmt.Fprintf(c.Err, "unknown auth command: %s\nRun 'prosie auth --help' for usage.\n", subCmd)
		return 1
	}
}

// printAuthHelp prints help for the auth command hierarchy.
func printAuthHelp(c *command.Environment) {
	help := `Manage authentication with Prosie.

Usage:
  prosie auth <command> [flags]

Available Commands:
  login       Log in through OAuth device flow or personal access token
  status      Inspect authentication status and credentials
  logout      Clear saved credentials

Flags:
  -h, --help   Show help for command

Use "prosie auth <command> --help" for more information about a command.
`
	fmt.Fprint(c.Out, help)
}
