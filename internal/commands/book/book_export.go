package bookcmd

import (
	"context"
	"flag"
	"fmt"
	"github.com/jonbaldie/prosie-cli/internal/command"
	"io"
	"strconv"
)

func executeBookExport(c *command.Environment, args []string) int {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	output := fs.String("output", "", "Write output to a file instead of stdout")
	fs.StringVar(output, "o", "", "Write output to a file instead of stdout (shorthand)")
	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := command.ParseFlagsAndArgs(fs, args)
	if err != nil {
		return command.FlagError(c, err, printBookExportHelp)
	}

	if len(posArgs) == 0 {
		fmt.Fprintln(c.Err, "error: book ID is required. Usage: prosie book export <id> [flags]")
		return 1
	}

	id, err := strconv.Atoi(posArgs[0])
	if err != nil {
		fmt.Fprintf(c.Err, "error: invalid book ID %q: must be an integer\n", posArgs[0])
		return 1
	}

	cli, err := command.Client(c)
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	data, err := cli.Books().ExportStory(context.Background(), id, "markdown")
	if err != nil {
		fmt.Fprintf(c.Err, "error exporting book: %v\n", err)
		return 1
	}

	return command.ExportProse(c, fmt.Sprintf("book %d", id), id, *output, data, *jsonFlag)
}

func printBookExportHelp(c *command.Environment) {
	help := `Export full book prose to stdout or a file.

Usage:
  prosie book export <id> [flags]

Flags:
  -h, --help            Show help for command
      --json            Format output as JSON
  -o, --output string   Write output to a file instead of stdout
`
	fmt.Fprint(c.Out, help)
}
