package cmd

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/jonbaldie/prosie-cli/internal/auth"
)

func (c *RootCmd) executeAuthLogout(args []string) int {
	fs := flag.NewFlagSet("logout", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(c.Err, "error parsing flags: %v\n", err)
		return 1
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
		_ = c.WriteJSON(output)
		return 0
	}

	fmt.Fprintln(c.Out, "Logged out from Prosie.")
	if envTokenPresent {
		fmt.Fprintln(c.Out, "Note: PROSIE_API_TOKEN is set in your environment and will continue to authenticate requests.")
	}
	return 0
}
