package chaptercmd

import (
	"context"
	"flag"
	"fmt"
	"github.com/jonbaldie/prosie-cli/internal/command"
	"io"
	"strconv"
)

func executeChapterExport(c *command.Environment, args []string) int {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	output := fs.String("output", "", "Write output to a file instead of stdout")
	fs.StringVar(output, "o", "", "Write output to a file instead of stdout (shorthand)")
	format := fs.String("format", "markdown", "Export format (markdown, docx)")
	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := command.ParseFlagsAndArgs(fs, args)
	if err != nil {
		return command.FlagError(c, err, printChapterExportHelp)
	}

	if len(posArgs) == 0 {
		fmt.Fprintln(c.Err, "error: chapter ID is required. Usage: prosie chapter export <id> [flags]")
		return 1
	}

	id := posArgs[0]

	cli, err := command.Client(c)
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	data, err := cli.Chapters().ExportChapter(context.Background(), id, *format)
	if err != nil {
		fmt.Fprintf(c.Err, "error exporting chapter: %v\n", err)
		return 1
	}

	var idVal any = id
	if intVal, err := strconv.Atoi(id); err == nil {
		idVal = intVal
	}

	return command.ExportProse(c, fmt.Sprintf("chapter %s", id), idVal, *output, data, *jsonFlag)
}

func printChapterExportHelp(c *command.Environment) {
	help := `Export chapter prose to stdout or a file.

Usage:
  prosie chapter export <id> [flags]

Flags:
      --format string   Export format (markdown, docx) (default "markdown")
  -h, --help            Show help for command
      --json            Format output as JSON
  -o, --output string   Write output to a file instead of stdout
`
	fmt.Fprint(c.Out, help)
}
