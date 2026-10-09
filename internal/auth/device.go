package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/jonbaldie/prosie-cli/internal/client"
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

// RequestDeviceCode sends a request to POST /oauth/device/code through the unauthenticated client cli.
func RequestDeviceCode(ctx context.Context, cli *client.Client, clientID, scopes string) (*DeviceCodeResponse, error) {
	if clientID == "" {
		clientID = DefaultClientID
	}

	payload := map[string]string{
		"client_id": clientID,
	}
	if scopes != "" {
		payload["scope"] = scopes
	}

	respBytes, err := cli.PostOAuth(ctx, "/oauth/device/code", payload)
	if err != nil {
		return nil, oauthFailure(err, "device authorization failed:", "device authorization failed with HTTP status")
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

// PollForToken polls POST /oauth/token through the unauthenticated client cli until
// authorization is granted, denied, expired, or timed out.
func PollForToken(ctx context.Context, cli *client.Client, clientID, deviceCode string, interval, expiresIn time.Duration) (*TokenResponse, error) {
	if clientID == "" {
		clientID = DefaultClientID
	}
	if interval <= 0 {
		interval = 5 * time.Second
	}
	if expiresIn <= 0 {
		expiresIn = 900 * time.Second
	}

	return pollUntilAuthorized(ctx, cli, clientID, deviceCode, interval, expiresIn)
}

func pollUntilAuthorized(ctx context.Context, cli *client.Client, clientID, deviceCode string, interval, expiresIn time.Duration) (*TokenResponse, error) {
	deadline := time.Now().Add(expiresIn)

	for {
		if time.Now().After(deadline) {
			return nil, ErrExpiredToken
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		token, backoff, err := pollTokenOnce(ctx, cli, clientID, deviceCode)
		if err != nil || token != nil {
			return token, err
		}
		interval += backoff

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
		}
	}
}

func pollTokenOnce(ctx context.Context, cli *client.Client, clientID, deviceCode string) (*TokenResponse, time.Duration, error) {
	payload := map[string]string{
		"grant_type":  GrantTypeDeviceCode,
		"client_id":   clientID,
		"device_code": deviceCode,
	}

	respBytes, err := cli.PostOAuth(ctx, "/oauth/token", payload)
	if err != nil {
		return tokenAttemptFailure(err)
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(respBytes, &tokenResp); err != nil {
		return nil, 0, fmt.Errorf("failed to parse token response: %w", err)
	}
	return &tokenResp, 0, nil
}

// tokenAttemptFailure maps an RFC 8628 token error to the next polling step.
func tokenAttemptFailure(err error) (*TokenResponse, time.Duration, error) {
	var apiErr *client.ApiError
	if !errors.As(err, &apiErr) {
		return nil, 0, err
	}

	switch apiErr.ErrorCode {
	case "authorization_pending":
		return nil, 0, nil
	case "slow_down":
		return nil, 5 * time.Second, nil
	case "access_denied":
		return nil, 0, ErrAccessDenied
	case "expired_token":
		return nil, 0, ErrExpiredToken
	default:
		return nil, 0, oauthFailure(err, "oauth error:", "unexpected status")
	}
}

// oauthFailure describes an OAuth endpoint failure. An OAuth error code follows
// codePrefix; a response with no error code reports its status after statusPrefix.
func oauthFailure(err error, codePrefix, statusPrefix string) error {
	var apiErr *client.ApiError
	if !errors.As(err, &apiErr) {
		return err
	}
	switch {
	case apiErr.ErrorCode != "" && apiErr.Message != "":
		return fmt.Errorf("%s %s (%s)", codePrefix, apiErr.ErrorCode, apiErr.Message)
	case apiErr.ErrorCode != "":
		return fmt.Errorf("%s %s", codePrefix, apiErr.ErrorCode)
	case apiErr.Message != "":
		return fmt.Errorf("%s %d: %s", statusPrefix, apiErr.StatusCode, apiErr.Message)
	default:
		return fmt.Errorf("%s %d", statusPrefix, apiErr.StatusCode)
	}
}
