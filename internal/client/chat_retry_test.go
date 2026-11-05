package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRetryConversation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.EscapedPath() != "/api/conversations/part%2F42/retry" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, `{"data":{"message":{"id":9,"role":"assistant","content":"Try again"},"model":"test-model","usage":{"total_tokens":5}}}`)
	}))
	defer server.Close()
	cli := New(server.URL, "token", server.Client())
	got, err := cli.Conversations().RetryConversation(context.Background(), " part/42 ")
	if err != nil {
		t.Fatal(err)
	}
	if got.Message == nil || got.Message.Content != "Try again" || got.Model != "test-model" || got.Usage.TotalTokens != 5 {
		t.Fatalf("unexpected response %+v", got)
	}
	_, err = cli.Conversations().RetryConversation(context.Background(), " ")
	if err == nil || !strings.Contains(err.Error(), "conversation ID is required") {
		t.Fatalf("unexpected error %v", err)
	}
}
