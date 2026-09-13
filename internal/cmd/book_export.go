package cmd

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
)

func (c *RootCmd) executeBookExport(args []string) int {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	output := fs.String("output", "", "Write output to a file instead of stdout")
	fs.StringVar(output, "o", "", "Write output to a file instead of stdout (shorthand)")
	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := parseFlagsAndArgs(fs, args)
	if err != nil {
		if err == flag.ErrHelp {
			c.PrintBookExportHelp()
			return 0
		}
		fmt.Fprintf(c.Err, "error parsing flags: %v\n", err)
		return 1
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

	cli, err := c.Client()
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	data, err := cli.ExportStory(context.Background(), id, "markdown")
	if err != nil {
		fmt.Fprintf(c.Err, "error exporting book: %v\n", err)
		return 1
	}

	if *output != "" {
		if err := os.WriteFile(*output, data, 0644); err != nil {
			fmt.Fprintf(c.Err, "error writing to file: %v\n", err)
			return 1
		}
		if *jsonFlag {
			_ = c.WriteJSON(map[string]any{
				"id":     id,
				"output": *output,
				"bytes":  len(data),
			})
			return 0
		}
		fmt.Fprintf(c.Out, "Exported book %d to %s (%d bytes).\n", id, *output, len(data))
		return 0
	}

	if *jsonFlag {
		_ = c.WriteJSON(map[string]any{
			"id":      id,
			"content": string(data),
			"bytes":   len(data),
		})
		return 0
	}

	_, _ = c.Out.Write(data)
	return 0
}

func (c *RootCmd) PrintBookExportHelp() {
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
