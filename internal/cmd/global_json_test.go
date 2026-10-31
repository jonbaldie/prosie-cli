package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

func TestLeadingJSONPreservesChatMessageAfterSeparator(t *testing.T) {
	var received string
	_, cfgPath, httpClient := setupTestChatEnv(t, func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		received = body.Content
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":{"message":{"id":2,"conversation_id":777,"role":"assistant","content":"Reply"},"model":"test"}}`)
	})
	root, out, errOut := newTestRootCmd(cfgPath, httpClient)
	code := root.Execute([]string{"--json", "chat", "send", "--conversation", "777", "--", "Hello"})
	if code != 0 || errOut.Len() != 0 {
		t.Fatalf("exit %d, stderr %q", code, errOut.String())
	}
	if received != "Hello" {
		t.Errorf("received message %q, want Hello", received)
	}
	var response map[string]any
	if err := json.Unmarshal(out.Bytes(), &response); err != nil {
		t.Fatalf("invalid JSON output %q: %v", out.String(), err)
	}
	if response["conversation_id"] != float64(777) {
		t.Errorf("conversation_id = %v, want 777", response["conversation_id"])
	}
}

func TestInheritedJSONArguments(t *testing.T) {
	t.Setenv("PROSIE_API_URL", "https://example.test")
	t.Setenv("PROSIE_API_TOKEN", "test-token")
	var content string
	client := &http.Client{Transport: deleteTransport(func(r *http.Request) (*http.Response, error) {
		var body struct {
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode: %v", err)
		}
		content = body.Content
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"data":{"message":{"id":2,"conversation_id":777,"role":"assistant","content":"Reply"},"model":"test"}}`))}, nil
	})}
	for _, tc := range []struct {
		name    string
		args    []string
		content string
	}{
		{"root", []string{"--json", "chat", "send", "--conversation", "777", "hello"}, "hello"},
		{"family and literal", []string{"chat", "--json", "send", "--conversation", "777", "--", "--json"}, "--json"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			content = ""
			root, out, errOut := newTestRootCmd(filepath.Join(t.TempDir(), "config.json"), client)
			code := root.Execute(tc.args)
			if code != 0 || errOut.Len() != 0 || !json.Valid(out.Bytes()) || content != tc.content {
				t.Fatalf("exit %d, stdout %q, stderr %q, content %q", code, out.String(), errOut.String(), content)
			}
		})
	}
}

func TestInheritedJSONOverrideAndIsolation(t *testing.T) {
	root, out, errOut := newTestRootCmd("", nil)
	for _, tc := range []struct {
		args []string
		json bool
	}{
		{[]string{"--json", "--json", "version"}, true},
		{[]string{"version"}, false},
		{[]string{"--json", "version", "--json=false"}, false},
		{[]string{"version", "--json"}, true},
	} {
		out.Reset()
		errOut.Reset()
		code := root.Execute(tc.args)
		if code != 0 || errOut.Len() != 0 || json.Valid(out.Bytes()) != tc.json {
			t.Fatalf("%v: exit %d stdout %q stderr %q", tc.args, code, out.String(), errOut.String())
		}
	}
}

func TestInheritedJSONMissingValueAndAuth(t *testing.T) {
	t.Setenv("PROSIE_API_TOKEN", "")
	root, out, errOut := newTestRootCmd(filepath.Join(t.TempDir(), "missing.json"), nil)
	code := root.Execute([]string{"--json", "chat", "send", "hello", "--conversation"})
	if code != 1 || out.Len() != 0 || !strings.Contains(errOut.String(), "flag needs an argument") {
		t.Fatalf("exit %d stdout %q stderr %q", code, out.String(), errOut.String())
	}
	out.Reset()
	errOut.Reset()
	code = root.Execute([]string{"auth", "--json", "status"})
	if code != 1 || errOut.Len() != 0 || !json.Valid(out.Bytes()) {
		t.Fatalf("exit %d stdout %q stderr %q", code, out.String(), errOut.String())
	}
	out.Reset()
	errOut.Reset()
	code = root.Execute([]string{"--json", "auth", "status", "--json=false"})
	if code != 1 || out.Len() != 0 || !strings.Contains(errOut.String(), "not logged in") {
		t.Fatalf("exit %d stdout %q stderr %q", code, out.String(), errOut.String())
	}
}

func TestInheritedJSONControlsGenerationStreaming(t *testing.T) {
	t.Setenv("PROSIE_API_URL", "https://example.test")
	t.Setenv("PROSIE_API_TOKEN", "test-token")
	var path, instruction string
	client := &http.Client{Transport: deleteTransport(func(r *http.Request) (*http.Response, error) {
		path = r.URL.Path
		var body struct {
			Instruction string `json:"instruction"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode: %v", err)
		}
		instruction = body.Instruction
		response := `{"data":{"prose":"Hello","persisted":true}}`
		if strings.HasSuffix(path, "/stream") {
			response = "event: done\ndata: {\"prose\":\"Hello\",\"persisted\":true}\n\n"
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(response))}, nil
	})}
	root, out, errOut := newTestRootCmd(filepath.Join(t.TempDir(), "config.json"), client)
	for _, tc := range []struct {
		args []string
		path string
		json bool
	}{
		{[]string{"--json", "generate", "continue", "101", "--instruction", "--json"}, "/api/scenes/101/continue", true},
		{[]string{"generate", "--json", "continue", "101", "--json=false"}, "/api/scenes/101/continue/stream", false},
	} {
		out.Reset()
		errOut.Reset()
		path = ""
		code := root.Execute(tc.args)
		if code != 0 || errOut.Len() != 0 || path != tc.path || (tc.json && instruction != "--json") || json.Valid(out.Bytes()) != tc.json {
			t.Fatalf("%v: exit %d stdout %q stderr %q path %q", tc.args, code, out.String(), errOut.String(), path)
		}
	}
}
