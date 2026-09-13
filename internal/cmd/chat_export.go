package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
)

func (c *RootCmd) executeChatExport(args []string) int {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	output := fs.String("output", "", "Write export JSON to a file instead of stdout")
	fs.StringVar(output, "o", "", "Write export JSON to a file instead of stdout (shorthand)")
	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := parseFlagsAndArgs(fs, args)
	if err != nil {
		if err == flag.ErrHelp {
			c.PrintChatExportHelp()
			return 0
		}
		fmt.Fprintf(c.Err, "error parsing flags: %v\n", err)
		return 1
	}

	if len(posArgs) == 0 {
		fmt.Fprintln(c.Err, "error: conversation ID is required. Usage: prosie chat export <conversation-id> [flags]")
		return 1
	}

	convID := posArgs[0]

	cli, err := c.Client()
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	data, err := cli.ExportConversation(context.Background(), convID)
	if err != nil {
		fmt.Fprintf(c.Err, "error exporting conversation: %v\n", err)
		return 1
	}

	if *output != "" {
		if err := os.WriteFile(*output, data, 0644); err != nil {
			fmt.Fprintf(c.Err, "error writing to file: %v\n", err)
			return 1
		}
		if *jsonFlag {
			_ = c.WriteJSON(map[string]any{
				"id":     convID,
				"output": *output,
				"bytes":  len(data),
			})
			return 0
		}
		fmt.Fprintf(c.Out, "Exported conversation %s to %s (%d bytes).\n", convID, *output, len(data))
		return 0
	}

	if *jsonFlag {
		var parsed any
		if err := json.Unmarshal(data, &parsed); err == nil {
			_ = c.WriteJSON(parsed)
			return 0
		}
	}

	_, _ = c.Out.Write(data)
	if !bytes.HasSuffix(data, []byte("\n")) {
		fmt.Fprintln(c.Out)
	}
	return 0
}

func (c *RootCmd) PrintChatExportHelp() {
	help := `Export a chat conversation to a local JSON file or stdout.

Usage:
  prosie chat export <conversation-id> [flags]

Flags:
  -h, --help            Show help for command
      --json            Format output as JSON
  -o, --output string   Write export JSON to a file instead of stdout
`
	fmt.Fprint(c.Out, help)
}
