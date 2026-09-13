package cmd

import (
	"fmt"
)

func (c *RootCmd) executeAuth(args []string) int {
	if len(args) == 0 {
		c.PrintAuthHelp()
		return 0
	}

	subCmd := args[0]
	subArgs := args[1:]

	switch subCmd {
	case "help", "--help", "-h":
		c.PrintAuthHelp()
		return 0
	case "login":
		return c.executeAuthLogin(subArgs)
	case "status":
		return c.executeAuthStatus(subArgs)
	case "logout":
		return c.executeAuthLogout(subArgs)
	default:
		fmt.Fprintf(c.Err, "unknown auth command: %s\nRun 'prosie auth --help' for usage.\n", subCmd)
		return 1
	}
}

// PrintAuthHelp prints help for the auth command hierarchy.
func (c *RootCmd) PrintAuthHelp() {
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
