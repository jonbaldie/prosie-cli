package generatecmd

import (
	"strings"
	"testing"

	"github.com/jonbaldie/prosie-cli/internal/command"
)

func TestFormalContentS07Rewrite(t *testing.T) {
	for _, tc := range []struct {
		selection, prompt, instruction, action string
		ok                                     bool
		want                                   string
	}{
		{"text", " ", "usable", "", false, ""},
		{"text", "", "usable", "", true, "usable"},
		{" ", "usable", "", "", false, ""},
		{"text", " ", "usable", "tighten", true, " "},
	} {
		p, err := rewriteInput(&command.Environment{In: strings.NewReader("")}, tc.selection, "", tc.prompt, tc.instruction, tc.action, false)
		if (err == nil) != tc.ok || (tc.ok && p.Instruction != tc.want) {
			t.Fatalf("case=%+v params=%+v error=%v", tc, p, err)
		}
	}
}
