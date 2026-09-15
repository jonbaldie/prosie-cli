package authcmd

import (
	"context"
	"flag"
	"fmt"
	"github.com/jonbaldie/prosie-cli/internal/command"
	"io"
	"time"

	"github.com/jonbaldie/prosie-cli/internal/auth"
	"github.com/jonbaldie/prosie-cli/internal/client"
	"github.com/jonbaldie/prosie-cli/internal/config"
)

func executeAuthLogin(c *command.Environment, args []string) int {
	fs := flag.NewFlagSet("login", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	tokenFlag := fs.String("token", "", "Authenticate with a personal access token")
	noBrowserFlag := fs.Bool("no-browser", false, "Do not open the web browser automatically")
	scopesFlag := fs.String("scopes", "read write generate", "Requested token scopes")
	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	if err := fs.Parse(args); err != nil {
		return command.FlagError(c, err, printAuthLoginHelp)
	}

	cfg, err := config.Load(c.ConfigPath)
	if err != nil {
		cfg = &config.Config{}
	}
	apiURL := config.ResolveApiURL(cfg)

	if *tokenFlag != "" {
		return loginWithToken(c, apiURL, *tokenFlag, *jsonFlag)
	}
	return loginWithDevice(c, cfg, apiURL, *scopesFlag, *noBrowserFlag, *jsonFlag)
}

func loginWithToken(c *command.Environment, apiURL, token string, jsonOutput bool) int {
	res, err := auth.LoginWithToken(context.Background(), c.ConfigPath, apiURL, token, c.HTTPClient)
	if err != nil {
		fmt.Fprintf(c.Err, "authentication failed: %v\n", err)
		return 1
	}

	if jsonOutput {
		_ = command.WriteJSON(c, res)
		return 0
	}

	fmt.Fprintf(c.Out, "Successfully authenticated to %s!\n", apiURL)
	if res.User != nil {
		fmt.Fprintf(c.Out, "Logged in as %s (%s)\n", res.User.Name, res.User.Email)
	}
	return 0
}

func loginWithDevice(c *command.Environment, cfg *config.Config, apiURL, scopes string, noBrowser, jsonOutput bool) int {
	dcr, err := auth.RequestDeviceCode(context.Background(), c.HTTPClient, apiURL, auth.DefaultClientID, scopes)
	if err != nil {
		fmt.Fprintf(c.Err, "failed to initiate device authorization: %v\n", err)
		return 1
	}

	showDeviceAuthorization(c, dcr, noBrowser, jsonOutput)
	interval, expiresIn := deviceTiming(dcr)

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

	if jsonOutput {
		_ = command.WriteJSON(c, &auth.LoginResult{
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

func showDeviceAuthorization(c *command.Environment, dcr *auth.DeviceCodeResponse, noBrowser, jsonOutput bool) {
	verificationURL := dcr.VerificationURLFull()

	if !jsonOutput {
		fmt.Fprintf(c.Out, "First, copy your one-time code: %s\n", dcr.UserCode)
		fmt.Fprintf(c.Out, "Open this URL in your browser to approve authorization:\n  %s\n\n", verificationURL)
	}

	if !noBrowser && c.BrowserOpener != nil {
		if err := c.BrowserOpener(verificationURL); err != nil {
			if !jsonOutput {
				fmt.Fprintf(c.Err, "Failed to open browser automatically: %v\nPlease visit the URL above manually.\n\n", err)
			}
		}
	}

	if !jsonOutput {
		fmt.Fprintln(c.Out, "Waiting for authorization in browser...")
	}

}

func deviceTiming(dcr *auth.DeviceCodeResponse) (time.Duration, time.Duration) {
	interval := time.Duration(dcr.Interval) * time.Second
	if interval <= 0 {
		interval = 5 * time.Second
	}
	expiresIn := time.Duration(dcr.ExpiresIn) * time.Second
	if expiresIn <= 0 {
		expiresIn = 900 * time.Second
	}

	return interval, expiresIn
}

func printAuthLoginHelp(c *command.Environment) {
	fmt.Fprint(c.Out, `Log in through OAuth device flow or a personal access token.

Usage:
  prosie auth login [flags]

Flags:
  -h, --help            Show help for command
      --json            Format output as JSON
      --token string    Authenticate with a personal access token
      --no-browser      Do not open the browser automatically
      --scopes string   Requested scopes (default "read write generate")
`)
}
