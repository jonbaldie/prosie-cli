package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jonbaldie/prosie-cli/internal/version"
)

// DefaultClientID is the registered public client ID for the Prosie CLI.
const DefaultClientID = "prosie-cli"

// GrantTypeDeviceCode is the RFC 8628 grant type identifier.
const GrantTypeDeviceCode = "urn:ietf:params:oauth:grant-type:device_code"

var (
	ErrAccessDenied = errors.New("authorization denied by user")
	ErrExpiredToken = errors.New("device authorization code has expired")
)

// DeviceCodeResponse represents the RFC 8628 device authorization response.
type DeviceCodeResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURL         string `json:"verification_url"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	VerificationURLComplete string `json:"verification_url_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

// VerificationURLFull returns the complete verification URL including the user code.
func (r *DeviceCodeResponse) VerificationURLFull() string {
	if r.VerificationURIComplete != "" {
		return r.VerificationURIComplete
	}
	if r.VerificationURLComplete != "" {
		return r.VerificationURLComplete
	}
	base := r.VerificationURIPrimary()
	if base == "" {
		return ""
	}
	return fmt.Sprintf("%s?user_code=%s", base, url.QueryEscape(r.UserCode))
}

// VerificationURIPrimary returns the primary base verification URI.
func (r *DeviceCodeResponse) VerificationURIPrimary() string {
	if r.VerificationURI != "" {
		return r.VerificationURI
	}
	return r.VerificationURL
}

// TokenResponse represents a successful token issuance response.
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

// Scopes returns the granted scopes as a slice of strings.
func (t *TokenResponse) Scopes() []string {
	if t.Scope == "" {
		return nil
	}
	return strings.Fields(t.Scope)
}

// TokenErrorResponse represents an OAuth error response during token polling.
type TokenErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// RequestDeviceCode sends a request to POST /oauth/device/code.
func RequestDeviceCode(ctx context.Context, httpClient *http.Client, baseURL, clientID, scopes string) (*DeviceCodeResponse, error) {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	if clientID == "" {
		clientID = DefaultClientID
	}

	payload := map[string]string{
		"client_id": clientID,
	}
	if scopes != "" {
		payload["scope"] = scopes
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to encode device code request: %w", err)
	}

	endpoint := strings.TrimRight(baseURL, "/") + "/oauth/device/code"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "prosie-cli/"+version.Version)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send device authorization request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp TokenErrorResponse
		if json.Unmarshal(respBytes, &errResp) == nil && errResp.Error != "" {
			if errResp.ErrorDescription != "" {
				return nil, fmt.Errorf("device authorization failed: %s (%s)", errResp.Error, errResp.ErrorDescription)
			}
			return nil, fmt.Errorf("device authorization failed: %s", errResp.Error)
		}
		return nil, fmt.Errorf("device authorization failed with HTTP status %d: %s", resp.StatusCode, string(respBytes))
	}

	var dcr DeviceCodeResponse
	if err := json.Unmarshal(respBytes, &dcr); err != nil {
		return nil, fmt.Errorf("failed to parse device authorization response: %w", err)
	}

	if dcr.Interval <= 0 {
		dcr.Interval = 5
	}
	if dcr.ExpiresIn <= 0 {
		dcr.ExpiresIn = 900
	}

	return &dcr, nil
}

// PollForToken polls POST /oauth/token until authorization is granted, denied, expired, or timed out.
func PollForToken(ctx context.Context, httpClient *http.Client, baseURL, clientID, deviceCode string, interval, expiresIn time.Duration) (*TokenResponse, error) {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	if clientID == "" {
		clientID = DefaultClientID
	}
	if interval <= 0 {
		interval = 5 * time.Second
	}
	if expiresIn <= 0 {
		expiresIn = 900 * time.Second
	}

	deadline := time.Now().Add(expiresIn)
	endpoint := strings.TrimRight(baseURL, "/") + "/oauth/token"

	for {
		if time.Now().After(deadline) {
			return nil, ErrExpiredToken
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		payload := map[string]string{
			"grant_type":  GrantTypeDeviceCode,
			"client_id":   clientID,
			"device_code": deviceCode,
		}
		bodyBytes, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal token request: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, err
		}

		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "prosie-cli/"+version.Version)

		resp, err := httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("token request failed: %w", err)
		}

		respBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read token response: %w", err)
		}

		if resp.StatusCode == http.StatusOK {
			var tokenResp TokenResponse
			if err := json.Unmarshal(respBytes, &tokenResp); err != nil {
				return nil, fmt.Errorf("failed to parse token response: %w", err)
			}
			return &tokenResp, nil
		}

		var errResp TokenErrorResponse
		_ = json.Unmarshal(respBytes, &errResp)

		switch errResp.Error {
		case "authorization_pending":
			// User has not yet completed authorization; continue polling.
		case "slow_down":
			// Polling too frequently; RFC 8628 specifies interval must increase by 5 seconds.
			interval += 5 * time.Second
		case "access_denied":
			return nil, ErrAccessDenied
		case "expired_token":
			return nil, ErrExpiredToken
		default:
			if errResp.ErrorDescription != "" {
				return nil, fmt.Errorf("oauth error: %s (%s)", errResp.Error, errResp.ErrorDescription)
			}
			if errResp.Error != "" {
				return nil, fmt.Errorf("oauth error: %s", errResp.Error)
			}
			return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(respBytes))
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
		}
	}
}
