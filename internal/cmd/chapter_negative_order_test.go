package cmd

import (
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
)

func TestChapterRejectsNegativeOrder(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"create", []string{"chapter", "create", "1", "--title", "X", "--order", "-1"}},
		{"update", []string{"chapter", "update", "41", "--order", "-1"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var requests atomic.Int32
			config, client := chapterMetadataEnv(t, func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"data":{"id":1,"name":"X"}}`))
			})
			cmd, out, errOut := newTestRootCmd(config, client)
			code := cmd.Execute(tc.args)
			if code != 1 || out.Len() != 0 || !strings.Contains(errOut.String(), "error: --order must be greater than or equal to 0") || requests.Load() != 0 {
				t.Fatalf("exit = %d, stdout = %q, stderr = %q, requests = %d; want 1, empty stdout, order error, 0", code, out.String(), errOut.String(), requests.Load())
			}
		})
	}
}
