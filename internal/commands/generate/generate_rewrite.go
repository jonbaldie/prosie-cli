package generatecmd

import (
	"context"
	"flag"
	"fmt"
	"github.com/jonbaldie/prosie-cli/internal/command"
	"io"
	"os"
	"strings"

	"github.com/jonbaldie/prosie-cli/internal/client"
)

func executeGenerateRewrite(c *command.Environment, args []string) int {
	fs := flag.NewFlagSet("rewrite", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	selectionFlag := fs.String("selection", "", "Text selection to rewrite")
	selectionFileFlag := fs.String("selection-file", "", "File containing text selection to rewrite")
	promptFlag := fs.String("prompt", "", "Custom rewrite instructions")
	instructionFlag := fs.String("instruction", "", "Custom rewrite instructions (alias for --prompt)")
	actionFlag := fs.String("action", "", "Preset rewrite action key: show, tighten, voice, or user-<id>")
	persistFlag := fs.Bool("persist", false, "Save rewritten prose to chapter")
	streamFlag := fs.Bool("stream", false, "Stream rewrite tokens in real time")
	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := command.ParseFlagsAndArgs(fs, args)
	if err != nil {
		return command.FlagError(c, err, printGenerateRewriteHelp)
	}

	if len(posArgs) == 0 {
		fmt.Fprintln(c.Err, "error: chapter ID is required. Usage: prosie generate rewrite <chapter-id> [flags]")
		return 1
	}

	id := posArgs[0]

	params, err := rewriteInput(*selectionFlag, *selectionFileFlag, *promptFlag, *instructionFlag, *actionFlag, *persistFlag)
	if err != nil {
		fmt.Fprintln(c.Err, err)
		return 1
	}

	cli, err := command.Client(c)
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	ctx := generationContext(c)

	if *streamFlag && !*jsonFlag {
		return streamRewrite(c, cli.Generation(), ctx, id, params)
	}

	res, err := cli.Generation().Rewrite(ctx, id, params)
	if err != nil {
		fmt.Fprintf(c.Err, "error rewriting text: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = command.WriteJSON(c, res)
		return 0
	}

	fmt.Fprintln(c.Out, res.Prose)
	return 0
}

// printGenerateRewriteHelp prints help for the generate rewrite command.
func printGenerateRewriteHelp(c *command.Environment) {
	help := `Rewrite selected chapter prose using a prompt or preset action.

Usage:
  prosie generate rewrite <chapter-id> [flags]

Flags:
  -h, --help             Show help for command
      --selection        Text selection to rewrite
      --selection-file   File containing text selection to rewrite
      --prompt           Custom rewrite instructions
      --action           Preset action key: show, tighten, voice, or user-<id>
      --persist          Save rewritten prose to chapter
      --stream           Stream rewrite tokens in real time
      --json             Output in JSON format
`
	fmt.Fprint(c.Out, help)
}

func rewriteInput(selection, filePath, prompt, instruction, action string, persist bool) (client.RewriteParams, error) {
	params := client.RewriteParams{Action: action, Persist: persist}
	var err error
	params.Selection, err = rewriteSelection(selection, filePath)
	if err != nil {
		return params, err
	}
	if prompt == "" {
		prompt = instruction
	}
	if strings.TrimSpace(prompt) == "" && strings.TrimSpace(action) == "" {
		return params, fmt.Errorf("error: --prompt or --action is required")
	}
	params.Instruction = prompt
	return params, nil
}

func rewriteSelection(selection, filePath string) (string, error) {
	if selection == "" && filePath != "" {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("error reading selection file: %w", err)
		}
		selection = string(data)
	}
	if strings.TrimSpace(selection) == "" {
		return "", fmt.Errorf("error: --selection or --selection-file is required")
	}
	return selection, nil
}

func streamRewrite(c *command.Environment, generation *client.Generation, ctx context.Context, id string, params client.RewriteParams) int {
	output := newTokenOutput(c.Out)
	res, err := generation.StreamRewrite(ctx, id, params, output.write)
	if err != nil {
		fmt.Fprintf(c.Err, "error rewriting text: %v\n", err)
		return 1
	}
	if res != nil {
		output.finish(res.Prose)
	}
	return 0
}
