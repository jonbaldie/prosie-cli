package command

import (
	"flag"
	"io"
	"reflect"
	"testing"
)

func TestFormalArgumentsValueConsumption(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	text := fs.String("text", "default", "")
	enabled := fs.Bool("bool", true, "")
	pos, err := ParseFlagsAndArgs(fs, []string{"id", "--text", "--", "--bool=false"}, Args{Max: -1})
	if err != nil || !reflect.DeepEqual(pos, []string{"id"}) || *text != "--" || *enabled {
		t.Fatalf("pos=%q text=%q bool=%v err=%v", pos, *text, *enabled, err)
	}
	visited := VisitedFlags(fs)
	if !visited["text"] || !visited["bool"] || Provided(visited, "bool", enabled) == nil {
		t.Fatalf("visited=%v", visited)
	}
}

func TestFormalArgumentsEmptyPresence(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	text := fs.String("text", "default", "")
	pos, err := ParseFlagsAndArgs(fs, []string{"--text=", "id"}, Args{Max: -1})
	if err != nil || !reflect.DeepEqual(pos, []string{"id"}) || *text != "" || Provided(VisitedFlags(fs), "text", text) == nil {
		t.Fatalf("pos=%q text=%q err=%v", pos, *text, err)
	}
}
