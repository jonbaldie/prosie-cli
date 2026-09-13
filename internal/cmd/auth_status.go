package cmd

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/jonbaldie/prosie-cli/internal/auth"
)

func (c *RootCmd) executeAuthStatus(args []string) int {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(c.Err, "error parsing flags: %v\n", err)
		return 1
	}

	status, err := auth.InspectStatus(context.Background(), c.ConfigPath, c.HTTPClient)
	if err != nil {
		if *jsonFlag {
			_ = c.WriteJSON(status)
			return 1
		}
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	if !status.Authenticated {
		if *jsonFlag {
			_ = c.WriteJSON(status)
			return 1
		}
		fmt.Fprintln(c.Err, "You are not logged in. Run 'prosie auth login' or set PROSIE_API_TOKEN.")
		return 1
	}

	if *jsonFlag {
		_ = c.WriteJSON(status)
		return 0
	}

	fmt.Fprintf(c.Out, "Logged in to %s as %s (%s)\n", status.ApiURL, status.User.Name, status.User.Email)
	fmt.Fprintf(c.Out, "Token source: %s\n", status.TokenSource)
	if len(status.Scopes) > 0 {
		fmt.Fprintf(c.Out, "Token scopes: %s\n", strings.Join(status.Scopes, ", "))
	}
	return 0
}
