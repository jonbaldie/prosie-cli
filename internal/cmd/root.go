package cmd

import (
	"context"
	"fmt"
	"github.com/jonbaldie/prosie-cli/internal/command"
	authcmd "github.com/jonbaldie/prosie-cli/internal/commands/auth"
	bookcmd "github.com/jonbaldie/prosie-cli/internal/commands/book"
	chaptercmd "github.com/jonbaldie/prosie-cli/internal/commands/chapter"
	chatcmd "github.com/jonbaldie/prosie-cli/internal/commands/chat"
	codexcmd "github.com/jonbaldie/prosie-cli/internal/commands/codex"
	generatecmd "github.com/jonbaldie/prosie-cli/internal/commands/generate"
	seriescmd "github.com/jonbaldie/prosie-cli/internal/commands/series"
	versioncmd "github.com/jonbaldie/prosie-cli/internal/commands/version"
	"io"
	"net/http"
	"os"

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
	Context       context.Context
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
	env := &command.Environment{In: c.In, Out: c.Out, Err: c.Err, ConfigPath: c.ConfigPath, HTTPClient: c.HTTPClient, BrowserOpener: c.BrowserOpener, Context: c.Context}
	return command.Dispatch(env, args, map[string]command.Handler{
		"auth":      authcmd.Execute,
		"book":      bookcmd.Execute,
		"chapter":   chaptercmd.Execute,
		"chat":      chatcmd.Execute,
		"codex":     codexcmd.Execute,
		"generate":  generatecmd.Execute,
		"series":    seriescmd.Execute,
		"version":   versioncmd.Execute,
		"--version": versioncmd.Execute, "-v": versioncmd.Execute,
		"--json": func(env *command.Environment, _ []string) int { printHelp(env); return 0 },
	}, printHelp, "unknown command: %s\nRun 'prosie --help' for usage.\n")
}

func printHelp(c *command.Environment) {
	help := `prosie - Command-line interface for the Prosie novel writing platform

Usage:
  prosie <command> [flags]

Available Commands:
  auth        Manage authentication (login, status, logout)
  book        Manage books (list, show, create, update, delete, duplicate, export, import)
  chapter     Manage chapters (list, show, create, update, delete, reorder, export)
  chat        Manage novel chat assistant (list, show, send, stream, export, import, delete)
  codex       Manage narrative facts and lore notes (list, show, create, update, delete)
  generate    Generate AI prose (continue, reject, rewrite, summarize)
  series      Manage book collections and shared lore (list, show, create, update, delete, attach, detach)
  version     Display the CLI version

Flags:
  -h, --help      Show help for command
  -v, --version   Show CLI version
      --json      Format output as JSON

Use "prosie <command> --help" for more information about a command.
`
	fmt.Fprint(c.Out, help)
}
