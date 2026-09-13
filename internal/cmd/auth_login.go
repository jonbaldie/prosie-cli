package cmd

import (
	"context"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/jonbaldie/prosie-cli/internal/auth"
	"github.com/jonbaldie/prosie-cli/internal/client"
	"github.com/jonbaldie/prosie-cli/internal/config"
)

func (c *RootCmd) executeAuthLogin(args []string) int {
	fs := flag.NewFlagSet("login", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	tokenFlag := fs.String("token", "", "Authenticate with a personal access token")
	noBrowserFlag := fs.Bool("no-browser", false, "Do not open the web browser automatically")
	scopesFlag := fs.String("scopes", "read write generate", "Requested token scopes")
	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(c.Err, "error parsing flags: %v\n", err)
		return 1
	}

	cfg, err := config.Load(c.ConfigPath)
	if err != nil {
		cfg = &config.Config{}
	}
	apiURL := config.ResolveApiURL(cfg)

	// Flow 1: Manual personal access token
	if *tokenFlag != "" {
		res, err := auth.LoginWithToken(context.Background(), c.ConfigPath, apiURL, *tokenFlag, c.HTTPClient)
		if err != nil {
			fmt.Fprintf(c.Err, "authentication failed: %v\n", err)
			return 1
		}

		if *jsonFlag {
			_ = c.WriteJSON(res)
			return 0
		}

		fmt.Fprintf(c.Out, "Successfully authenticated to %s!\n", apiURL)
		if res.User != nil {
			fmt.Fprintf(c.Out, "Logged in as %s (%s)\n", res.User.Name, res.User.Email)
		}
		return 0
	}

	// Flow 2: OAuth 2.0 Device Flow
	dcr, err := auth.RequestDeviceCode(context.Background(), c.HTTPClient, apiURL, auth.DefaultClientID, *scopesFlag)
	if err != nil {
		fmt.Fprintf(c.Err, "failed to initiate device authorization: %v\n", err)
		return 1
	}

	verificationURL := dcr.VerificationURLFull()

	if !*jsonFlag {
		fmt.Fprintf(c.Out, "First, copy your one-time code: %s\n", dcr.UserCode)
		fmt.Fprintf(c.Out, "Open this URL in your browser to approve authorization:\n  %s\n\n", verificationURL)
	}

	if !*noBrowserFlag && c.BrowserOpener != nil {
		if err := c.BrowserOpener(verificationURL); err != nil {
			if !*jsonFlag {
				fmt.Fprintf(c.Err, "Failed to open browser automatically: %v\nPlease visit the URL above manually.\n\n", err)
			}
		}
	}

	if !*jsonFlag {
		fmt.Fprintln(c.Out, "Waiting for authorization in browser...")
	}

	interval := time.Duration(dcr.Interval) * time.Second
	if interval <= 0 {
		interval = 5 * time.Second
	}
	expiresIn := time.Duration(dcr.ExpiresIn) * time.Second
	if expiresIn <= 0 {
		expiresIn = 900 * time.Second
	}

	tokenResp, err := auth.PollForToken(context.Background(), c.HTTPClient, apiURL, auth.DefaultClientID, dcr.DeviceCode, interval, expiresIn)
	if err != nil {
		fmt.Fprintf(c.Err, "authorization failed: %v\n", err)
		return 1
	}

	// Save token to config
	cfg.Token = tokenResp.AccessToken
	cfg.ApiURL = apiURL
	cfg.Scopes = tokenResp.Scopes()
	if err := config.Save(c.ConfigPath, cfg); err != nil {
		fmt.Fprintf(c.Err, "failed to save configuration: %v\n", err)
		return 1
	}

	cli := client.New(apiURL, cfg.Token, c.HTTPClient)
	user, err := cli.GetUser(context.Background())
	if err != nil {
		// Logged in successfully, but user info fetch had an issue
		user = nil
	}

	if *jsonFlag {
		_ = c.WriteJSON(&auth.LoginResult{
			Status: "authenticated",
			ApiURL: apiURL,
			User:   user,
			Scopes: cfg.Scopes,
		})
		return 0
	}

	fmt.Fprintf(c.Out, "\nSuccessfully authenticated to %s!\n", apiURL)
	if user != nil {
		fmt.Fprintf(c.Out, "Logged in as %s (%s)\n", user.Name, user.Email)
	}
	return 0
}
