package command

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jonbaldie/prosie-cli/internal/auth"
	"github.com/jonbaldie/prosie-cli/internal/client"
	"github.com/jonbaldie/prosie-cli/internal/config"
	"io"
	"net/http"
	"strings"
)

type Environment struct {
	In            io.Reader
	Out           io.Writer
	Err           io.Writer
	ConfigPath    string
	HTTPClient    *http.Client
	BrowserOpener auth.BrowserOpener
	Context       context.Context
}

func Client(c *Environment) (*client.Client, error) {
	cfg, err := config.Load(c.ConfigPath)
	if err != nil {
		cfg = &config.Config{}
	}

	apiURL := config.ResolveApiURL(cfg)
	token, _ := config.ResolveToken(cfg)
	if token == "" {
		return nil, fmt.Errorf("you are not logged in. Run 'prosie auth login' or set PROSIE_API_TOKEN")
	}

	return client.New(apiURL, token, c.HTTPClient), nil
}

func WriteJSON(c *Environment, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = c.Out.Write(data)
	return err
}

func HasJSONFlag(args []string) bool {
	for _, arg := range args {
		if arg == "--json" || strings.HasPrefix(arg, "--json=") {
			return true
		}
	}
	return false
}
