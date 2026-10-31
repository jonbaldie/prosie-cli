package bookcmd

import (
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
	format := fs.String("format", "markdown", "Export format (markdown, docx)")
	jsonFlag := fs.Bool("json", c.JSON, "Output in JSON format")

	posArgs, err := command.ParseFlagsAndArgs(fs, args, command.Args{
		Min: 1, Max: 1, Usage: "prosie book export <id> [flags]", Missing: "book ID is required",
	})
	if err != nil {
		return command.FlagError(c, err, printBookExportHelp)
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

	data, err := cli.Books().ExportStory(c.Ctx(), id, *format)
	if err != nil {
		fmt.Fprintf(c.Err, "error exporting book: %v\n", err)
		return 1
	}

	return command.ExportProse(c, fmt.Sprintf("book %d", id), id, *output, *format, data, *jsonFlag)
}

func printBookExportHelp(c *command.Environment) {
	help := `Export full book prose to stdout or a file.

Usage:
  prosie book export <id> [flags]

Flags:
      --format string   Export format (markdown, docx) (default "markdown")
  -h, --help            Show help for command
      --json            Format output as JSON (DOCX content is base64, with "encoding": "base64")
  -o, --output string   Write output to a file instead of stdout
`
	fmt.Fprint(c.Out, help)
}
