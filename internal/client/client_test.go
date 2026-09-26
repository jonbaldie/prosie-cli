package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jonbaldie/prosie-cli/internal/version"
)

func TestClientHeadersAndToken(t *testing.T) {
	var capturedAuth string
	var capturedAccept string
	var capturedUA string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		capturedAccept = r.Header.Get("Accept")
		capturedUA = r.Header.Get("User-Agent")

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 1, "name": "Jane", "email": "jane@example.com"})
	}))
	defer server.Close()

	cli := New(server.URL, "secret-token-xyz", server.Client())
	user, err := cli.GetUser(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if user.ID != 1 || user.Name != "Jane" || user.Email != "jane@example.com" {
		t.Fatalf("unexpected user data: %+v", user)
	}

	if capturedAuth != "Bearer secret-token-xyz" {
		t.Fatalf("expected Authorization header 'Bearer secret-token-xyz', got %q", capturedAuth)
	}
	if capturedAccept != "application/json" {
		t.Fatalf("expected Accept header 'application/json', got %q", capturedAccept)
	}
	wantUA := "prosie-cli/" + version.Version
	if capturedUA != wantUA {
		t.Fatalf("expected User-Agent %q, got %q", wantUA, capturedUA)
	}
}

func TestClientUnauthorizedError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "Unauthenticated."})
	}))
	defer server.Close()

	cli := New(server.URL, "bad-token", server.Client())
	_, err := cli.GetUser(context.Background())
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	apiErr, ok := err.(*ApiError)
	if !ok {
		t.Fatalf("expected *ApiError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != 401 {
		t.Fatalf("expected status 401, got %d", apiErr.StatusCode)
	}
	if apiErr.Message != "Unauthenticated." {
		t.Fatalf("expected 'Unauthenticated.', got %q", apiErr.Message)
	}
}
