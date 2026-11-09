package command

import (
	"context"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jonbaldie/prosie-cli/internal/client"
)

func TestDeletionConfirmationBounded(t *testing.T) {
	t.Setenv("PROSIE_API_TOKEN", "test-token")
	configPath := filepath.Join(t.TempDir(), "missing.json")
	check := func(input string, yes bool) {
		t.Helper()
		line, _, _ := strings.Cut(input, "\n")
		line = strings.Trim(line, " \t\n\v\f\r")
		confirmed := yes || strings.EqualFold(line, "y") || strings.EqualFold(line, "yes")
		calls := 0
		env := &Environment{In: strings.NewReader(input), Out: io.Discard, Err: io.Discard, ConfigPath: configPath}
		code := ExecuteDelete(env, DeleteOptions{Resource: "book", ID: 1, Yes: yes}, func(context.Context, *client.Client) error {
			calls++
			return nil
		})
		want := 0
		if confirmed {
			want = 1
		}
		if code != 0 || calls != want {
			t.Fatalf("input %q, yes %v: exit %d, delete calls %d, want %d", input, yes, code, calls, want)
		}
	}
	for _, yes := range []bool{false, true} {
		check("", yes)
		for a := 0; a < 128; a++ {
			check(string([]byte{byte(a)}), yes)
			for b := 0; b < 128; b++ {
				check(string([]byte{byte(a), byte(b)}), yes)
			}
		}
		for _, input := range []string{"yes", "YES\n", " no\nyes\n", " yes \n", strings.Repeat("x", 65536)} {
			check(input, yes)
		}
	}
}
