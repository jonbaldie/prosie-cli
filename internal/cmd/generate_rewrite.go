package cmd

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jonbaldie/prosie-cli/internal/client"
)

func (c *RootCmd) executeGenerateRewrite(args []string) int {
	fs := flag.NewFlagSet("rewrite", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	selectionFlag := fs.String("selection", "", "Text selection to rewrite")
	selectionFileFlag := fs.String("selection-file", "", "File containing text selection to rewrite")
	promptFlag := fs.String("prompt", "", "Custom rewrite instructions")
	instructionFlag := fs.String("instruction", "", "Custom rewrite instructions (alias for --prompt)")
	actionFlag := fs.String("action", "", "Preset rewrite action key (e.g. show-not-tell, tighten)")
	persistFlag := fs.Bool("persist", false, "Save rewritten prose to chapter")
	streamFlag := fs.Bool("stream", false, "Stream rewrite tokens in real time")
	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := parseFlagsAndArgs(fs, args)
	if err != nil {
		if err == flag.ErrHelp {
			c.PrintGenerateRewriteHelp()
			return 0
		}
		fmt.Fprintf(c.Err, "error parsing flags: %v\n", err)
		return 1
	}

	if len(posArgs) == 0 {
		fmt.Fprintln(c.Err, "error: chapter ID is required. Usage: prosie generate rewrite <chapter-id> [flags]")
		return 1
	}

	id := posArgs[0]

	selection := *selectionFlag
	if selection == "" && *selectionFileFlag != "" {
		data, err := os.ReadFile(*selectionFileFlag)
		if err != nil {
			fmt.Fprintf(c.Err, "error reading selection file: %v\n", err)
			return 1
		}
		selection = string(data)
	}

	if strings.TrimSpace(selection) == "" {
		fmt.Fprintln(c.Err, "error: --selection or --selection-file is required")
		return 1
	}

	prompt := *promptFlag
	if prompt == "" {
		prompt = *instructionFlag
	}

	if strings.TrimSpace(prompt) == "" && strings.TrimSpace(*actionFlag) == "" {
		fmt.Fprintln(c.Err, "error: --prompt or --action is required")
		return 1
	}

	cli, err := c.Client()
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	ctx := c.Context
	if ctx == nil {
		ctx = context.Background()
	}

	params := client.RewriteParams{
		Selection:   selection,
		Instruction: prompt,
		Action:      *actionFlag,
		Persist:     *persistFlag,
	}

	if *streamFlag && !*jsonFlag {
		var lastToken string
		res, err := cli.StreamRewrite(ctx, id, params, func(token string) {
			fmt.Fprint(c.Out, token)
			lastToken = token
		})
		if err != nil {
			fmt.Fprintf(c.Err, "error rewriting text: %v\n", err)
			return 1
		}

		if res != nil && !strings.HasSuffix(res.Prose, "\n") && lastToken != "" && !strings.HasSuffix(lastToken, "\n") {
			fmt.Fprintln(c.Out)
		}
		return 0
	}

	res, err := cli.Rewrite(ctx, id, params)
	if err != nil {
		fmt.Fprintf(c.Err, "error rewriting text: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = c.WriteJSON(res)
		return 0
	}

	fmt.Fprintln(c.Out, res.Prose)
	return 0
}

// PrintGenerateRewriteHelp prints help for the generate rewrite command.
func (c *RootCmd) PrintGenerateRewriteHelp() {
	help := `Rewrite selected chapter prose using a prompt or preset action.

Usage:
  prosie generate rewrite <chapter-id> [flags]

Flags:
  -h, --help             Show help for command
      --selection        Text selection to rewrite
      --selection-file   File containing text selection to rewrite
      --prompt           Custom rewrite instructions
      --action           Preset action key (e.g. show-not-tell, tighten)
      --persist          Save rewritten prose to chapter
      --stream           Stream rewrite tokens in real time
      --json             Output in JSON format
`
	fmt.Fprint(c.Out, help)
}
