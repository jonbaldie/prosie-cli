package chaptercmd

import (
	"flag"

	"github.com/jonbaldie/prosie-cli/internal/client"
	"github.com/jonbaldie/prosie-cli/internal/command"
)

type chapterInput struct {
	title, content, summary, filePath *string
	pov, character                    *string
	order                             *int
}

func newChapterInput(fs *flag.FlagSet, contentHelp string) chapterInput {
	return chapterInput{
		title:     fs.String("title", "", "Chapter title (sent as name)"),
		content:   fs.String("content", "", contentHelp),
		summary:   fs.String("summary", "", "Chapter summary"),
		filePath:  fs.String("file", "", "Path to file containing chapter prose ('-' for standard input)"),
		pov:       fs.String("pov", "", "Chapter point of view"),
		character: fs.String("character", "", "Chapter character"),
		order:     fs.Int("order", 0, "Chapter order"),
	}
}

func (input chapterInput) params(c *command.Environment, fs *flag.FlagSet) (client.UpdateChapterParams, error) {
	params, err := command.ChapterInput(c, fs, input.title, input.content, input.summary, input.filePath)
	if err != nil {
		return params, err
	}
	visited := command.VisitedFlags(fs)
	params.CustomPOV = command.Provided(visited, "pov", input.pov)
	params.CustomCharacter = command.Provided(visited, "character", input.character)
	params.Order = command.Provided(visited, "order", input.order)
	return params, nil
}
