package authcmd

import (
	"fmt"
	"github.com/jonbaldie/prosie-cli/internal/command"
)

func Execute(c *command.Environment, args []string) int {
	return command.Dispatch(c, args, map[string]command.Handler{
		"login":  executeAuthLogin,
		"status": executeAuthStatus,
		"logout": executeAuthLogout,
	}, printAuthHelp, "unknown auth command: %s\nRun 'prosie auth --help' for usage.\n")
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
