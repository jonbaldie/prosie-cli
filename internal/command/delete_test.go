package command

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jonbaldie/prosie-cli/internal/client"
)

func TestExecuteDelete(t *testing.T) {
	for _, tc := range []struct {
		name, input, stdout, stderr string
		id                          any
		yes, json                   bool
		deleteErr                   error
		code, calls                 int
	}{
		{"cancel", "n\n", "", "Are you sure you want to delete chapter 7? [y/N]: Deletion cancelled.\n", "7", false, true, nil, 0, 0},
		{"confirmed", "yes\n", "Deleted chapter 7.\n", "Are you sure you want to delete chapter 7? [y/N]: ", "7", false, false, nil, 0, 1},
		{"failure", "", "", "error deleting chapter: denied\n", "7", true, false, errors.New("denied"), 1, 1},
		{"numeric JSON", "", "{\n  \"deleted\": true,\n  \"id\": 7\n}\n", "", "007", true, true, nil, 0, 1},
		{"string JSON", "", "{\n  \"deleted\": true,\n  \"id\": \"abc\"\n}\n", "", "abc", true, true, nil, 0, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("PROSIE_API_TOKEN", "token")
			var out, errOut bytes.Buffer
			env := &Environment{In: strings.NewReader(tc.input), Out: &out, Err: &errOut, ConfigPath: filepath.Join(t.TempDir(), "config.json")}
			calls := 0
			code := ExecuteDelete(env, DeleteOptions{Resource: "chapter", ID: tc.id, Yes: tc.yes, JSON: tc.json}, func(context.Context, *client.Client) error {
				calls++
				return tc.deleteErr
			})
			if code != tc.code || calls != tc.calls || out.String() != tc.stdout || errOut.String() != tc.stderr {
				t.Fatalf("code=%d calls=%d stdout=%q stderr=%q", code, calls, out.String(), errOut.String())
			}
		})
	}
}
