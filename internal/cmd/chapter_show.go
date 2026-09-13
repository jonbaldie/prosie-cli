package cmd

import (
	"context"
	"flag"
	"fmt"
	"io"
)

func (c *RootCmd) executeChapterShow(args []string) int {
	fs := flag.NewFlagSet("show", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := parseFlagsAndArgs(fs, args)
	if err != nil {
		if err == flag.ErrHelp {
			c.PrintChapterShowHelp()
			return 0
		}
		fmt.Fprintf(c.Err, "error parsing flags: %v\n", err)
		return 1
	}

	if len(posArgs) == 0 {
		fmt.Fprintln(c.Err, "error: chapter ID is required. Usage: prosie chapter show <id> [flags]")
		return 1
	}

	id := posArgs[0]

	cli, err := c.Client()
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	chapter, err := cli.GetChapter(context.Background(), id)
	if err != nil {
		fmt.Fprintf(c.Err, "error fetching chapter: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = c.WriteJSON(chapter)
		return 0
	}

	if chapter.Content == "" {
		fmt.Fprintln(c.Out, "No content.")
	} else {
		fmt.Fprintln(c.Out, chapter.Content)
	}

	return 0
}

func (c *RootCmd) PrintChapterShowHelp() {
	help := `Show chapter prose content.

Usage:
  prosie chapter show <id> [flags]

Flags:
  -h, --help   Show help for command
      --json   Format output as JSON
`
	fmt.Fprint(c.Out, help)
}
