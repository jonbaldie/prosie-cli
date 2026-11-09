package generatecmd

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jonbaldie/prosie-cli/internal/client"
	"github.com/jonbaldie/prosie-cli/internal/command"
)

func TestFormalGenerationJSONMode(t *testing.T) {
	for _, stream := range []bool{false, true} {
		for _, noStream := range []bool{false, true} {
			for _, jsonOutput := range []bool{false, true} {
				got := useStreaming(stream, noStream, jsonOutput)
				if got != (stream && !noStream && !jsonOutput) {
					t.Errorf("stream=%v no-stream=%v json=%v: got %v", stream, noStream, jsonOutput, got)
				}
			}
		}
	}
}

func TestFormalGenerationOutput(t *testing.T) {
	for _, tc := range []struct {
		tokens      []string
		prose, want string
	}{
		{nil, "result", ""},
		{[]string{""}, "result", ""},
		{[]string{"hello"}, "result", "hello\n"},
		{[]string{"hello\n"}, "result", "hello\n"},
		{[]string{"hello"}, "result\n", "hello"},
		{[]string{"hello", ""}, "result", "hello"},
	} {
		var out bytes.Buffer
		output := newTokenOutput(&out)
		for _, token := range tc.tokens {
			output.write(token)
		}
		output.finish(tc.prose)
		if out.String() != tc.want {
			t.Errorf("tokens %q prose %q: got %q want %q", tc.tokens, tc.prose, out.String(), tc.want)
		}
	}
}

func TestFormalGenerationCancellation(t *testing.T) {
	received := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received <- r.Method + " " + r.URL.Path
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var stderr bytes.Buffer
	env := &command.Environment{Err: &stderr}
	generation := client.New(server.URL, "token", server.Client()).Generation()
	code := continuationError(env, generation, ctx, "42", context.Canceled)
	if code != 1 || !strings.Contains(stderr.String(), "Generation cancelled.") {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	select {
	case got := <-received:
		if got != "POST /api/scenes/42/continue/cancel" {
			t.Fatalf("unexpected request: %s", got)
		}
	default:
		t.Fatal("no cancellation request reached server")
	}
}
