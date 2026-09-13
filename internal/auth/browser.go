package auth

import (
	"fmt"
	"os/exec"
	"runtime"
)

// BrowserOpener defines a function signature for opening a URL in the user's default browser.
type BrowserOpener func(url string) error

// DefaultBrowserOpener opens a URL using the default system handler.
func DefaultBrowserOpener(targetURL string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", targetURL)
	case "linux":
		cmd = exec.Command("xdg-open", targetURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", targetURL)
	default:
		return fmt.Errorf("automatic browser launch is not supported on %s", runtime.GOOS)
	}

	return cmd.Start()
}
