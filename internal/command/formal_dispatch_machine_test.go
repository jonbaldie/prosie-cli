package command

import (
	"bytes"
	"flag"
	"fmt"
	"reflect"
	"testing"
)

func TestFormalDispatchCopyIsolation(t *testing.T) {
	var out, errOut bytes.Buffer
	parent := &Environment{Out: &out, Err: &errOut}
	var modes []bool
	var positions [][]string
	leaf := func(c *Environment, args []string) int {
		fs := flag.NewFlagSet("leaf", flag.ContinueOnError)
		fs.SetOutput(c.Err)
		jsonMode := fs.Bool("json", c.JSON, "")
		pos, err := ParseFlagsAndArgs(fs, args, Args{Max: 1})
		if err != nil {
			return 1
		}
		modes = append(modes, *jsonMode)
		positions = append(positions, pos)
		return 0
	}
	help := func(c *Environment) { fmt.Fprint(c.Out, "help") }
	group := func(c *Environment, args []string) int {
		code := Dispatch(c, args, map[string]Handler{"leaf": leaf}, help, "unknown %s")
		if c.JSON {
			t.Error("nested dispatch changed its caller")
		}
		return code
	}
	commands := map[string]Handler{"group": group, "leaf": leaf}
	if code := Dispatch(parent, []string{"group", "--json", "leaf", "x", "--json=false"}, commands, help, "unknown %s"); code != 0 {
		t.Fatalf("first exit %d: %s", code, errOut.String())
	}
	if parent.JSON {
		t.Error("dispatch changed root caller")
	}
	if code := Dispatch(parent, []string{"leaf", "--", "--json"}, commands, help, "unknown %s"); code != 0 {
		t.Fatalf("sibling exit %d", code)
	}
	if !reflect.DeepEqual(modes, []bool{false, false}) || !reflect.DeepEqual(positions, [][]string{{"x"}, {"--json"}}) {
		t.Fatalf("modes %v positions %v", modes, positions)
	}
	if code := Dispatch(parent, []string{"--json", "leaf"}, commands, help, "unknown %s"); code != 0 || !modes[2] || parent.JSON {
		t.Fatalf("inherited mode: code=%d modes=%v parent=%v", code, modes, parent.JSON)
	}
	if code := Dispatch(parent, []string{"group", "--help"}, commands, help, "unknown %s"); code != 0 || out.String() != "help" {
		t.Fatalf("help code=%d output=%q", code, out.String())
	}
	if code := Dispatch(parent, []string{"group", "missing"}, commands, help, "unknown %s"); code != 1 || errOut.String() != "unknown missing" {
		t.Fatalf("unknown code=%d stderr=%q", code, errOut.String())
	}
}
