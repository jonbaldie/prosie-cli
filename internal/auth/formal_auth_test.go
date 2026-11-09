package auth

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jonbaldie/prosie-cli/internal/client"
	"github.com/jonbaldie/prosie-cli/internal/config"
)

func TestFormalAuthRejectedTokenDoesNotSave(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"token":"old"}`), 0600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/user" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"message":"denied"}`)
	}))
	defer server.Close()
	_, err := LoginWithToken(context.Background(), path, server.URL, "bad", server.Client())
	if err == nil {
		t.Fatal("rejected token accepted")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"token":"old"}` {
		t.Fatalf("rejected token changed config: %s", data)
	}
}

func TestFormalAuthLogoutRetainsEnvironmentToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer env-token" {
			t.Errorf("wrong token: %q", r.Header.Get("Authorization"))
		}
		fmt.Fprint(w, `{"name":"Alice"}`)
	}))
	defer server.Close()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := config.Save(path, &config.Config{Token: "saved-token"}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PROSIE_API_TOKEN", "env-token")
	t.Setenv("PROSIE_API_URL", server.URL)
	if _, err := Logout(path); err != nil {
		t.Fatal(err)
	}
	status, err := InspectStatus(context.Background(), path, server.Client())
	if err != nil || status == nil || !status.Authenticated || status.TokenSource != "environment" {
		t.Fatalf("environment token lost after logout: status=%+v err=%v", status, err)
	}
}

type formalAuthTransport func(*http.Request) (*http.Response, error)

func (f formalAuthTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestFormalAuthCancelledBeforePolling(t *testing.T) {
	calls := 0
	httpClient := &http.Client{Transport: formalAuthTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"access_token":"late"}`))}, nil
	})}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	token, err := PollForToken(ctx, client.New("http://example.invalid", "", httpClient), "", "code", time.Millisecond, time.Second)
	if err != context.Canceled || token != nil || calls != 0 {
		t.Fatalf("cancelled poll: token=%+v err=%v calls=%d", token, err, calls)
	}
}

func TestFormalAuthDeniedStopsPolling(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, `{"error":"access_denied"}`)
			return
		}
		fmt.Fprint(w, `{"access_token":"unexpected"}`)
	}))
	defer server.Close()
	token, err := PollForToken(context.Background(), client.New(server.URL, "", server.Client()), "", "code", time.Millisecond, time.Second)
	if err != ErrAccessDenied || token != nil || calls != 1 {
		t.Fatalf("denial must stop polling: token=%+v err=%v calls=%d", token, err, calls)
	}
}
