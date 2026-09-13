package cmd

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
)

func (c *RootCmd) executeChapterExport(args []string) int {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	output := fs.String("output", "", "Write output to a file instead of stdout")
	fs.StringVar(output, "o", "", "Write output to a file instead of stdout (shorthand)")
	format := fs.String("format", "markdown", "Export format (markdown, docx)")
	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := parseFlagsAndArgs(fs, args)
	if err != nil {
		if err == flag.ErrHelp {
			c.PrintChapterExportHelp()
			return 0
		}
		fmt.Fprintf(c.Err, "error parsing flags: %v\n", err)
		return 1
	}

	if len(posArgs) == 0 {
		fmt.Fprintln(c.Err, "error: chapter ID is required. Usage: prosie chapter export <id> [flags]")
		return 1
	}

	id := posArgs[0]

	cli, err := c.Client()
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	data, err := cli.ExportChapter(context.Background(), id, *format)
	if err != nil {
		fmt.Fprintf(c.Err, "error exporting chapter: %v\n", err)
		return 1
	}

	var idVal any = id
	if intVal, err := strconv.Atoi(id); err == nil {
		idVal = intVal
	}

	if *output != "" {
		if err := os.WriteFile(*output, data, 0644); err != nil {
			fmt.Fprintf(c.Err, "error writing to file: %v\n", err)
			return 1
		}
		if *jsonFlag {
			_ = c.WriteJSON(map[string]any{
				"id":     idVal,
				"output": *output,
				"bytes":  len(data),
			})
			return 0
		}
		fmt.Fprintf(c.Out, "Exported chapter %s to %s (%d bytes).\n", id, *output, len(data))
		return 0
	}

	if *jsonFlag {
		_ = c.WriteJSON(map[string]any{
			"id":      idVal,
			"content": string(data),
			"bytes":   len(data),
		})
		return 0
	}

	_, _ = c.Out.Write(data)
	return 0
}

func (c *RootCmd) PrintChapterExportHelp() {
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
