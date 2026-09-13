package cmd

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/jonbaldie/prosie-cli/internal/client"
)

func (c *RootCmd) executeChapterCreate(args []string) int {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	title := fs.String("title", "", "Chapter title")
	content := fs.String("content", "", "Initial chapter prose content")
	summary := fs.String("summary", "", "Chapter summary")
	filePath := fs.String("file", "", "Path to file containing chapter prose")
	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := parseFlagsAndArgs(fs, args)
	if err != nil {
		if err == flag.ErrHelp {
			c.PrintChapterCreateHelp()
			return 0
		}
		fmt.Fprintf(c.Err, "error parsing flags: %v\n", err)
		return 1
	}

	if len(posArgs) == 0 {
		fmt.Fprintln(c.Err, "error: book ID is required. Usage: prosie chapter create <book-id> [flags]")
		return 1
	}

	bookID := posArgs[0]

	visited := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) {
		visited[f.Name] = true
	})

	var contentVal string
	hasContent := false

	if visited["file"] && *filePath != "" {
		data, err := os.ReadFile(*filePath)
		if err != nil {
			fmt.Fprintf(c.Err, "error reading file: %v\n", err)
			return 1
		}
		contentVal = string(data)
		hasContent = true
	} else if visited["content"] {
		contentVal = *content
		hasContent = true
	}

	params := client.CreateChapterParams{}
	if visited["title"] {
		params.Title = title
	}
	if hasContent {
		params.Content = &contentVal
	}
	if visited["summary"] {
		params.Summary = summary
	}

	cli, err := c.Client()
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	chapter, err := cli.CreateChapter(context.Background(), bookID, params)
	if err != nil {
		fmt.Fprintf(c.Err, "error creating chapter: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = c.WriteJSON(chapter)
		return 0
	}

	fmt.Fprintf(c.Out, "Created chapter %q (ID: %d).\n", chapter.DisplayTitle(), chapter.ID)
	return 0
}

func (c *RootCmd) PrintChapterCreateHelp() {
	help := `Create a new chapter in a book.

Usage:
  prosie chapter create <book-id> [flags]

Flags:
      --content string   Initial chapter prose content
      --file string      Path to file containing chapter prose
  -h, --help             Show help for command
      --json             Format output as JSON
      --summary string   Chapter summary
      --title string     Chapter title
`
	fmt.Fprint(c.Out, help)
}
