package generatecmd

import (
	"context"
	"flag"
	"fmt"
	"github.com/jonbaldie/prosie-cli/internal/client"
	"github.com/jonbaldie/prosie-cli/internal/command"
	"io"
)

func executeGenerateSummarize(c *command.Environment, args []string) int {
	fs := flag.NewFlagSet("summarize", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	jsonFlag := fs.Bool("json", false, "Output in JSON format")
	persistFlag := fs.Bool("persist", true, "Save generated summary to chapter")
	noPersistFlag := fs.Bool("no-persist", false, "Do not save changes to chapter")

	posArgs, err := command.ParseFlagsAndArgs(fs, args)
	if err != nil {
		return command.FlagError(c, err, printGenerateSummarizeHelp)
	}

	if len(posArgs) == 0 {
		fmt.Fprintln(c.Err, "error: chapter ID is required. Usage: prosie generate summarize <chapter-id> [flags]")
		return 1
	}

	id := posArgs[0]

	cli, err := command.Client(c)
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	ctx := c.Context
	if ctx == nil {
		ctx = context.Background()
	}

	params := client.SummarizeParams{Persist: *persistFlag && !*noPersistFlag}
	res, err := cli.Generation().Summarize(ctx, id, params)
	if err != nil {
		fmt.Fprintf(c.Err, "error generating summary: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = command.WriteJSON(c, res)
		return 0
	}

	fmt.Fprintln(c.Out, res.Summary)
	return 0
}

// printGenerateSummarizeHelp prints help for the generate summarize command.
func printGenerateSummarizeHelp(c *command.Environment) {
	help := `Generate or update a one-line chapter summary.

Usage:
  prosie generate summarize <chapter-id> [flags]

Flags:
  -h, --help         Show help for command
      --json         Format output as JSON
      --no-persist   Do not save generated summary to chapter
      --persist      Save generated summary to chapter (default true)
`
	fmt.Fprint(c.Out, help)
}
