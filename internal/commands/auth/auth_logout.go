package authcmd

import (
	"flag"
	"fmt"
	"github.com/jonbaldie/prosie-cli/internal/command"
	"io"
	"os"

	"github.com/jonbaldie/prosie-cli/internal/auth"
)

func executeAuthLogout(c *command.Environment, args []string) int {
	fs := flag.NewFlagSet("logout", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	if err := fs.Parse(args); err != nil {
		return command.FlagError(c, err, printAuthLogoutHelp)
	}

	res, err := auth.Logout(c.ConfigPath)
	if err != nil {
		fmt.Fprintf(c.Err, "logout error: %v\n", err)
		return 1
	}

	envTokenPresent := os.Getenv("PROSIE_API_TOKEN") != ""

	if *jsonFlag {
		output := map[string]any{
			"status":             res.Status,
			"message":            res.Message,
			"env_token_override": envTokenPresent,
		}
		_ = command.WriteJSON(c, output)
		return 0
	}

	fmt.Fprintln(c.Out, "Logged out from Prosie.")
	if envTokenPresent {
		fmt.Fprintln(c.Out, "Note: PROSIE_API_TOKEN is set in your environment and will continue to authenticate requests.")
	}
	return 0
}

func printAuthLogoutHelp(c *command.Environment) {
	fmt.Fprint(c.Out, `Clear saved credentials. PROSIE_API_TOKEN remains active if set.

Usage:
  prosie auth logout [flags]

Flags:
  -h, --help            Show help for command
      --json            Format output as JSON
`)
}
