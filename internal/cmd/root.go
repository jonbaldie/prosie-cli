package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/jonbaldie/prosie-cli/internal/auth"
)

// RootCmd orchestrates CLI execution and dependency injection.
type RootCmd struct {
	In            io.Reader
	Out           io.Writer
	Err           io.Writer
	ConfigPath    string
	HTTPClient    *http.Client
	BrowserOpener auth.BrowserOpener
}

// NewRootCmd initializes a RootCmd with standard operating system streams and defaults.
func NewRootCmd() *RootCmd {
	return &RootCmd{
		In:            os.Stdin,
		Out:           os.Stdout,
		Err:           os.Stderr,
		ConfigPath:    "",
		HTTPClient:    nil,
		BrowserOpener: auth.DefaultBrowserOpener,
	}
}

// Execute routes command-line arguments and returns the process exit code.
func (c *RootCmd) Execute(args []string) int {
	if len(args) == 0 {
		c.PrintHelp()
		return 0
	}

	cmdName := args[0]
	subArgs := args[1:]

	switch cmdName {
	case "help", "--help", "-h":
		c.PrintHelp()
		return 0
	case "version", "--version", "-v":
		return c.executeVersion(subArgs)
	case "auth":
		return c.executeAuth(subArgs)
	default:
		// Check for global flags like --json without command
		if cmdName == "--json" {
			c.PrintHelp()
			return 0
		}
		fmt.Fprintf(c.Err, "unknown command: %s\nRun 'prosie --help' for usage.\n", cmdName)
		return 1
	}
}

// PrintHelp prints root usage information.
func (c *RootCmd) PrintHelp() {
	help := `prosie - Command-line interface for the Prosie novel writing platform

Usage:
  prosie <command> [flags]

Available Commands:
  auth        Manage authentication (login, status, logout)
  version     Display the CLI version

Flags:
  -h, --help      Show help for command
  -v, --version   Show CLI version
      --json      Format output as JSON

Use "prosie <command> --help" for more information about a command.
`
	fmt.Fprint(c.Out, help)
}

// WriteJSON encodes a value as formatted JSON to the configured output writer.
func (c *RootCmd) WriteJSON(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = c.Out.Write(data)
	return err
}

// hasJSONFlag checks if --json is present in the arguments list.
func hasJSONFlag(args []string) bool {
	for _, arg := range args {
		if arg == "--json" || strings.HasPrefix(arg, "--json=") {
			return true
		}
	}
	return false
}
