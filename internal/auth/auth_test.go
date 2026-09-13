package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/jonbaldie/prosie-cli/internal/config"
)

func TestRequestDeviceCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth/device/code" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var payload map[string]string
		_ = json.NewDecoder(r.Body).Decode(&payload)

		if payload["client_id"] != "prosie-cli" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_client"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(DeviceCodeResponse{
			DeviceCode:              "dev-12345",
			UserCode:                "ABCD-EFGH",
			VerificationURI:         "https://prosie.app/oauth/device",
			VerificationURIComplete: "https://prosie.app/oauth/device?user_code=ABCD-EFGH",
			ExpiresIn:               900,
			Interval:                1,
		})
	}))
	defer server.Close()

	dcr, err := RequestDeviceCode(context.Background(), server.Client(), server.URL, "prosie-cli", "read write")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if dcr.DeviceCode != "dev-12345" || dcr.UserCode != "ABCD-EFGH" {
		t.Fatalf("unexpected device response: %+v", dcr)
	}
	if dcr.VerificationURLFull() != "https://prosie.app/oauth/device?user_code=ABCD-EFGH" {
		t.Fatalf("unexpected verification url: %s", dcr.VerificationURLFull())
	}
}

func TestPollForToken_PendingThenApproved(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth/token" {
			http.NotFound(w, r)
			return
		}

		attempts++
		w.Header().Set("Content-Type", "application/json")

		if attempts < 2 {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(TokenErrorResponse{
				Error:            "authorization_pending",
				ErrorDescription: "Waiting for user approval",
			})
			return
		}

		_ = json.NewEncoder(w).Encode(TokenResponse{
			AccessToken: "granted-access-token",
			TokenType:   "Bearer",
			Scope:       "read write generate",
		})
	}))
	defer server.Close()

	resp, err := PollForToken(context.Background(), server.Client(), server.URL, "prosie-cli", "dev-123", 10*time.Millisecond, 2*time.Second)
	if err != nil {
		t.Fatalf("unexpected polling error: %v", err)
	}

	if resp.AccessToken != "granted-access-token" {
		t.Fatalf("expected token 'granted-access-token', got %q", resp.AccessToken)
	}
	if len(resp.Scopes()) != 3 {
		t.Fatalf("expected 3 scopes, got %v", resp.Scopes())
	}
}

func TestPollForToken_AccessDenied(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(TokenErrorResponse{
			Error:            "access_denied",
			ErrorDescription: "User denied the request",
		})
	}))
	defer server.Close()

	_, err := PollForToken(context.Background(), server.Client(), server.URL, "prosie-cli", "dev-123", 10*time.Millisecond, 1*time.Second)
	if err != ErrAccessDenied {
		t.Fatalf("expected ErrAccessDenied, got %v", err)
	}
}

func TestPollForToken_ExpiredToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(TokenErrorResponse{
			Error: "expired_token",
		})
	}))
	defer server.Close()

	_, err := PollForToken(context.Background(), server.Client(), server.URL, "prosie-cli", "dev-123", 10*time.Millisecond, 1*time.Second)
	if err != ErrExpiredToken {
		t.Fatalf("expected ErrExpiredToken, got %v", err)
	}
}

func TestLoginWithToken(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/user" {
			auth := r.Header.Get("Authorization")
			if auth != "Bearer valid-token" {
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{"message": "Unauthenticated."})
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 42, "name": "Author", "email": "author@example.com"})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	// Fail with bad token
	_, err := LoginWithToken(context.Background(), cfgPath, server.URL, "bad-token", server.Client())
	if err == nil {
		t.Fatalf("expected error with bad token, got nil")
	}

	// Succeed with valid token
	res, err := LoginWithToken(context.Background(), cfgPath, server.URL, "valid-token", server.Client())
	if err != nil {
		t.Fatalf("unexpected error with valid token: %v", err)
	}
	if res.User.Name != "Author" || res.User.Email != "author@example.com" {
		t.Fatalf("unexpected user in result: %+v", res.User)
	}

	// Verify saved config
	saved, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("failed to load saved config: %v", err)
	}
	if saved.Token != "valid-token" || saved.ApiURL != server.URL {
		t.Fatalf("saved config mismatch: %+v", saved)
	}
}

func TestInspectStatus(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "Bearer active-token" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 1, "name": "Alice", "email": "alice@example.com"})
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "Unauthenticated."})
	}))
	defer server.Close()

	// 1. Unauthenticated state
	status, err := InspectStatus(context.Background(), cfgPath, server.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Authenticated {
		t.Fatalf("expected unauthenticated status")
	}

	// 2. Authenticated via saved config
	if err := config.Save(cfgPath, &config.Config{
		ApiURL: server.URL,
		Token:  "active-token",
		Scopes: []string{"read", "write"},
	}); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	status, err = InspectStatus(context.Background(), cfgPath, server.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !status.Authenticated || status.User.Name != "Alice" || status.TokenSource != "config" {
		t.Fatalf("unexpected status: %+v", status)
	}

	// 3. Authenticated via PROSIE_API_TOKEN override
	t.Setenv("PROSIE_API_TOKEN", "active-token")
	status, err = InspectStatus(context.Background(), cfgPath, server.Client())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !status.Authenticated || status.TokenSource != "environment" {
		t.Fatalf("expected token source 'environment', got %+v", status)
	}
}

func TestLogout(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")

	if err := config.Save(cfgPath, &config.Config{Token: "some-token"}); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	res, err := Logout(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != "logged_out" {
		t.Fatalf("unexpected status: %s", res.Status)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("error loading config: %v", err)
	}
	if cfg.Token != "" {
		t.Fatalf("expected empty token after logout, got %q", cfg.Token)
	}
}
