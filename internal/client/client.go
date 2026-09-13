package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jonbaldie/prosie-cli/internal/version"
)

// User represents an authenticated Prosie user account.
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// ApiError represents an error response from the Prosie API.
type ApiError struct {
	StatusCode int    `json:"status_code"`
	ErrorCode  string `json:"error,omitempty"`
	Message    string `json:"message,omitempty"`
}

func (e *ApiError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("API error (%d): %s", e.StatusCode, e.Message)
	}
	if e.ErrorCode != "" {
		return fmt.Sprintf("API error (%d): %s", e.StatusCode, e.ErrorCode)
	}
	return fmt.Sprintf("API request failed with status code %d", e.StatusCode)
}

// Client provides authenticated HTTP communication with the Prosie API.
type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
	UserAgent  string
}

// New creates a new API client configured with the given base URL and token.
func New(baseURL, token string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 30 * time.Second,
		}
	}
	return &Client{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		Token:      token,
		HTTPClient: httpClient,
		UserAgent:  "prosie-cli/" + version.Version,
	}
}

// NewRequest creates an HTTP request targeting the given relative API endpoint.
func (c *Client) NewRequest(ctx context.Context, method, path string, body any) (*http.Request, error) {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	url := c.BaseURL + path

	var bodyReader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.UserAgent)

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	return req, nil
}

// Do sends an HTTP request using the configured client and processes the response.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	return c.HTTPClient.Do(req)
}

// CheckResponse parses non-2xx HTTP responses into an ApiError.
func CheckResponse(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		return nil
	}

	apiErr := &ApiError{
		StatusCode: resp.StatusCode,
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err == nil && len(bodyBytes) > 0 {
		var raw map[string]any
		if json.Unmarshal(bodyBytes, &raw) == nil {
			if msg, ok := raw["message"].(string); ok {
				apiErr.Message = msg
			}
			if errCode, ok := raw["error"].(string); ok {
				apiErr.ErrorCode = errCode
			}
			if errDesc, ok := raw["error_description"].(string); ok {
				if apiErr.Message == "" {
					apiErr.Message = errDesc
				}
			}
		} else {
			apiErr.Message = strings.TrimSpace(string(bodyBytes))
		}
	}

	if apiErr.StatusCode == http.StatusUnauthorized && apiErr.Message == "" {
		apiErr.Message = "unauthorized: invalid or missing authentication token"
	}

	return apiErr
}

// GetUser fetches the authenticated user profile from GET /api/user.
func (c *Client) GetUser(ctx context.Context) (*User, error) {
	req, err := c.NewRequest(ctx, http.MethodGet, "/api/user", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to /api/user failed: %w", err)
	}
	defer resp.Body.Close()

	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	var user User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode user response: %w", err)
	}

	return &user, nil
}
