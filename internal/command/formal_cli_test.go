package command

import (
	"bytes"
	"context"
	"flag"
	"io"
	"strings"
	"testing"

	"github.com/jonbaldie/prosie-cli/internal/client"
)

func TestFormalCliDeletionApproval(t *testing.T) {
	t.Setenv("PROSIE_API_TOKEN", "test-token")
	for _, tc := range []struct {
		answer    string
		yes, want bool
	}{
		{"", false, false}, {"no\n", false, false}, {" YES \n", false, true}, {"y\n", false, true}, {"no\n", true, true},
	} {
		var out, stderr bytes.Buffer
		env := &Environment{In: strings.NewReader(tc.answer), Out: &out, Err: &stderr, ConfigPath: t.TempDir() + "/missing"}
		called := false
		code := ExecuteDelete(env, DeleteOptions{Resource: "book", ID: 1, Yes: tc.yes}, func(context.Context, *client.Client) error { called = true; return nil })
		if code != 0 || called != tc.want {
			t.Errorf("answer %q yes=%v: code=%d called=%v, want %v", tc.answer, tc.yes, code, called, tc.want)
		}
	}
}

func TestFormalCliDispatchAndInput(t *testing.T) {
	var errOut bytes.Buffer
	env := &Environment{In: strings.NewReader("input"), Out: io.Discard, Err: &errOut}
	handler := func(c *Environment, args []string) int {
		if !c.JSON || len(args) != 1 || args[0] != "--json=false" {
			t.Errorf("dispatch JSON=%v args=%v", c.JSON, args)
		}
		return 0
	}
	if code := Dispatch(env, []string{"--json", "book", "--json=false"}, map[string]Handler{"book": handler}, func(*Environment) {}, "%s"); code != 0 || env.JSON {
		t.Fatalf("dispatch code=%d leaked=%v", code, env.JSON)
	}
	fs := flag.NewFlagSet("formal", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	value := fs.String("name", "", "")
	args, err := ParseFlagsAndArgs(fs, []string{"id", "--name", "--"}, Args{Min: 1, Max: 1})
	if err != nil || *value != "--" || len(args) != 1 || args[0] != "id" {
		t.Fatalf("parse %v %q %v", args, *value, err)
	}
	text, err := ReadContent(env, "", "-", "content")
	if err != nil || text != "input" {
		t.Fatalf("read %q %v", text, err)
	}
	if _, err = ReadContent(env, "inline", "-", "content"); err == nil {
		t.Fatal("expected conflict")
	}
}
