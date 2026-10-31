package chatcmd

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/jonbaldie/prosie-cli/internal/command"
	"io"
)

func executeChatExport(c *command.Environment, args []string) int {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	output := fs.String("output", "", "Write export JSON to a file instead of stdout")
	fs.StringVar(output, "o", "", "Write export JSON to a file instead of stdout (shorthand)")
	jsonFlag := fs.Bool("json", c.JSON, "Output in JSON format")

	posArgs, err := command.ParseFlagsAndArgs(fs, args, command.Args{
		Min: 1, Max: 1, Usage: "prosie chat export <conversation-id> [flags]", Missing: "conversation ID is required",
	})
	if err != nil {
		return command.FlagError(c, err, printChatExportHelp)
	}

	convID := posArgs[0]

	cli, err := command.Client(c)
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	data, err := cli.Conversations().ExportConversation(c.Ctx(), convID)
	if err != nil {
		fmt.Fprintf(c.Err, "error exporting conversation: %v\n", err)
		return 1
	}

	if *output != "" {
		return command.WriteExportFile(c, "conversation "+convID, convID, *output, data, *jsonFlag)
	}

	if *jsonFlag {
		var parsed any
		if err := json.Unmarshal(data, &parsed); err != nil {
			fmt.Fprintf(c.Err, "error decoding conversation export: %v\n", err)
			return 1
		}
		_ = command.WriteJSON(c, parsed)
		return 0
	}

	_, _ = c.Out.Write(data)
	if !bytes.HasSuffix(data, []byte("\n")) {
		fmt.Fprintln(c.Out)
	}
	return 0
}

func printChatExportHelp(c *command.Environment) {
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
