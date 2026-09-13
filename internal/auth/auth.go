package auth

import (
	"context"
	"fmt"
	"net/http"

	"github.com/jonbaldie/prosie-cli/internal/client"
	"github.com/jonbaldie/prosie-cli/internal/config"
)

// StatusResult contains user authentication status and metadata.
type StatusResult struct {
	Authenticated bool         `json:"authenticated"`
	ApiURL        string       `json:"api_url"`
	TokenSource   string       `json:"token_source"`
	Scopes        []string     `json:"scopes,omitempty"`
	User          *client.User `json:"user,omitempty"`
}

// LoginResult contains output details from a successful login action.
type LoginResult struct {
	Status string       `json:"status"`
	ApiURL string       `json:"api_url"`
	User   *client.User `json:"user,omitempty"`
	Scopes []string     `json:"scopes,omitempty"`
}

// LogoutResult contains the status of a logout action.
type LogoutResult struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// LoginWithToken validates a personal access token against GET /api/user and stores it in config.
func LoginWithToken(ctx context.Context, cfgPath, baseURL, token string, httpClient *http.Client) (*LoginResult, error) {
	if token == "" {
		return nil, fmt.Errorf("token cannot be empty")
	}

	cli := client.New(baseURL, token, httpClient)
	user, err := cli.GetUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("token verification failed: %w", err)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		cfg = &config.Config{}
	}

	cfg.Token = token
	cfg.ApiURL = baseURL
	// Default scopes for personal access tokens
	if len(cfg.Scopes) == 0 {
		cfg.Scopes = []string{"read", "write", "generate"}
	}

	if err := config.Save(cfgPath, cfg); err != nil {
		return nil, fmt.Errorf("failed to save configuration: %w", err)
	}

	return &LoginResult{
		Status: "authenticated",
		ApiURL: baseURL,
		User:   user,
		Scopes: cfg.Scopes,
	}, nil
}

// InspectStatus resolves the active credentials and queries the server for user details.
func InspectStatus(ctx context.Context, cfgPath string, httpClient *http.Client) (*StatusResult, error) {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		cfg = &config.Config{}
	}

	apiURL := config.ResolveApiURL(cfg)
	token, source := config.ResolveToken(cfg)

	if token == "" {
		return &StatusResult{
			Authenticated: false,
			ApiURL:        apiURL,
			TokenSource:   source,
		}, nil
	}

	cli := client.New(apiURL, token, httpClient)
	user, err := cli.GetUser(ctx)
	if err != nil {
		return &StatusResult{
			Authenticated: false,
			ApiURL:        apiURL,
			TokenSource:   source,
		}, fmt.Errorf("failed to verify token with %s: %w", apiURL, err)
	}

	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{"read", "write", "generate"}
	}

	return &StatusResult{
		Authenticated: true,
		ApiURL:        apiURL,
		TokenSource:   source,
		Scopes:        scopes,
		User:          user,
	}, nil
}

// Logout removes stored credentials from the configuration file.
func Logout(cfgPath string) (*LogoutResult, error) {
	if err := config.Clear(cfgPath); err != nil {
		return nil, fmt.Errorf("failed to clear stored credentials: %w", err)
	}

	return &LogoutResult{
		Status:  "logged_out",
		Message: "Logged out successfully",
	}, nil
}
